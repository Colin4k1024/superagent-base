"""Sequential agent — runs child agents in a pipeline, chaining outputs.

Each child agent receives the previous child's output as input.  Only the
final agent's output is streamed; intermediate steps run synchronously.
"""

from __future__ import annotations

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


class SequentialAgent(BaseAgent):
    """Executes child agents in order, passing each output as the next input.

    Only the final agent's output is streamed; intermediate steps run
    synchronously and feed forward as context.
    """

    def __init__(
        self,
        agent_id: str,
        name: str,
        children: list[BaseAgent] | None = None,
        **kwargs: Any,
    ) -> None:
        super().__init__(agent_id, name, **kwargs)
        self.children: list[BaseAgent] = children or []

    def _build_agent(self) -> Agent:
        """Build a passthrough AgentScope agent for the sequential orchestrator.

        The actual execution logic is in run()/run_stream() which chain child
        agents.  This agent is used only if someone calls .agent directly.
        """
        from superagent.agents.chat import ChatModelAgent

        # Build a minimal ChatModelAgent as the underlying agent
        helper = ChatModelAgent(
            agent_id=f"{self.agent_id}-seq",
            name=f"{self.name}-seq",
            system_prompt=(
                f"{self.system_prompt}\n\n"
                "You are a sequential orchestrator. Process the input step by step."
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

    async def run(self, message: str, **kwargs: Any) -> str:
        """Execute children in order, chaining outputs."""
        if not self.children:
            return f"[SequentialAgent:{self.name}] No children registered"

        current = message
        for i, child in enumerate(self.children):
            logger.debug(
                "Sequential step %d/%d: %s -> %s",
                i + 1,
                len(self.children),
                self.name,
                child.name,
            )
            current = await child.run(current, **kwargs)
        return current

    async def run_stream(self, message: str, **kwargs: Any) -> AsyncGenerator[AgentEvent, None]:
        """Execute children in order; only stream the final child's output.

        Intermediate children execute via ``run()`` (non-streaming), and the
        last child's stream is forwarded to the caller.
        """
        if not self.children:
            reply_id = uuid.uuid4().hex
            yield ReplyStartEvent(session_id="", reply_id=reply_id, name=self.name)
            block_id = uuid.uuid4().hex
            yield TextBlockStartEvent(reply_id=reply_id, block_id=block_id)
            yield TextBlockDeltaEvent(
                reply_id=reply_id, block_id=block_id,
                delta=f"[SequentialAgent:{self.name}] No children registered",
            )
            yield TextBlockEndEvent(reply_id=reply_id, block_id=block_id)
            yield ReplyEndEvent(session_id="", reply_id=reply_id)
            return

        current = message
        # Run all but the last child non-streaming
        for child in self.children[:-1]:
            current = await child.run(current, **kwargs)

        # Stream the last child's output
        async for event in self.children[-1].run_stream(current, **kwargs):
            yield event

    def describe(self) -> dict[str, Any]:
        base = super().describe()
        base["children"] = [c.describe() for c in self.children]
        return base
