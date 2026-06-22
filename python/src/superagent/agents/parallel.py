"""Parallel agent — runs child agents concurrently and merges results.

Each child receives the same input message.  Results are joined with a
separator.  If any child raises, its error is captured rather than
failing the whole batch.
"""

from __future__ import annotations

import asyncio
import logging
import uuid
from typing import Any, AsyncGenerator

from agentscope.agent import Agent
from agentscope.event import (
    AgentEvent,
    ReplyEndEvent,
    ReplyStartEvent,
    TextBlockDeltaEvent,
    TextBlockEndEvent,
    TextBlockStartEvent,
)

from superagent.agents.base import BaseAgent

logger = logging.getLogger(__name__)


class ParallelAgent(BaseAgent):
    """Fan-out: runs all children concurrently, collects and merges results.

    Each child receives the same input message.  Results are joined with a
    separator.  If any child raises, its error is captured rather than
    failing the whole batch.
    """

    def __init__(
        self,
        agent_id: str,
        name: str,
        children: list[BaseAgent] | None = None,
        merge_separator: str = "\n---\n",
        **kwargs: Any,
    ) -> None:
        super().__init__(agent_id, name, **kwargs)
        self.children: list[BaseAgent] = children or []
        self.merge_separator = merge_separator

    def _build_agent(self) -> Agent:
        """Build a passthrough AgentScope agent for the parallel orchestrator.

        The actual execution logic is in run()/run_stream() which fan out to
        child agents.  This agent is used only if someone calls .agent directly.
        """
        from superagent.agents.chat import ChatModelAgent

        helper = ChatModelAgent(
            agent_id=f"{self.agent_id}-par",
            name=f"{self.name}-par",
            system_prompt=(
                f"{self.system_prompt}\n\n"
                "You are a parallel orchestrator. Merge results from multiple agents."
            ),
            model_name=self.model_name,
        )
        return helper._build_agent()

    @property
    def agent(self) -> Agent:
        """Lazy-initialise the underlying agent."""
        if self._agent is None:
            self._agent = self._build_agent()
        return self._agent

    async def _run_child(self, child: BaseAgent, message: str, **kwargs: Any) -> str:
        """Run a single child, capturing errors."""
        try:
            return await child.run(message, **kwargs)
        except Exception as exc:
            logger.error("Child %s failed: %s", child.name, exc)
            return f"[error:{child.name}] {exc}"

    async def run(self, message: str, **kwargs: Any) -> str:
        """Execute all children concurrently and merge results."""
        if not self.children:
            return f"[ParallelAgent:{self.name}] No children registered"

        tasks = [self._run_child(child, message, **kwargs) for child in self.children]
        results = await asyncio.gather(*tasks)
        return self.merge_separator.join(results)

    async def run_stream(self, message: str, **kwargs: Any) -> AsyncGenerator[AgentEvent, None]:
        """Execute children concurrently; yield merged result as text events."""
        reply_id = uuid.uuid4().hex

        yield ReplyStartEvent(session_id="", reply_id=reply_id, name=self.name)

        result = await self.run(message, **kwargs)

        block_id = uuid.uuid4().hex
        yield TextBlockStartEvent(reply_id=reply_id, block_id=block_id)
        yield TextBlockDeltaEvent(reply_id=reply_id, block_id=block_id, delta=result)
        yield TextBlockEndEvent(reply_id=reply_id, block_id=block_id)

        yield ReplyEndEvent(session_id="", reply_id=reply_id)

    def describe(self) -> dict[str, Any]:
        base = super().describe()
        base["children"] = [c.describe() for c in self.children]
        base["merge_separator"] = self.merge_separator
        return base
