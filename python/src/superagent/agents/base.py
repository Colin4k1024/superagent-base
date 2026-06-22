"""Base agent class — wrapper around AgentScope 2.0 agents."""

from __future__ import annotations

import logging
from abc import ABC, abstractmethod
from typing import Any

from agentscope.agent import Agent
from agentscope.message import UserMsg
from agentscope.state import AgentState

from superagent.message import Msg, AssistantMsg, TextBlock

logger = logging.getLogger(__name__)


class BaseAgent(ABC):
    """Abstract base for all Superagent agent types.

    Provides a unified interface over AgentScope's native agent classes.
    Wraps ``agentscope.agent.Agent`` with Superagent-specific conventions.
    """

    def __init__(
        self,
        agent_id: str,
        name: str,
        system_prompt: str = "",
        model_name: str = "gpt-4o",
        **kwargs: Any,
    ) -> None:
        self.agent_id = agent_id
        self.name = name
        self.system_prompt = system_prompt
        self.model_name = model_name
        self._config = kwargs
        self._agent: Agent | None = None
        self._state = AgentState()
        logger.info("Initialised %s (%s)", name, self.__class__.__name__)

    @abstractmethod
    def _build_agent(self) -> Agent:
        """Build and return the underlying AgentScope agent."""
        ...

    @property
    def agent(self) -> Agent:
        """Lazy-initialise the underlying agent."""
        if self._agent is None:
            self._agent = self._build_agent()
        return self._agent

    async def run(self, message: str, **kwargs: Any) -> str:
        """Execute the agent and return the final response text."""
        user_msg = UserMsg(name="user", content=message)
        reply_msg = await self.agent.reply(user_msg)
        text = reply_msg.get_text_content()
        return text if text is not None else ""

    async def run_msg(self, message: str, **kwargs: Any) -> Msg:
        """Execute the agent and return a structured Msg object.

        Unlike run() which returns a plain string, run_msg() returns
        a full Msg with metadata, content blocks, and usage tracking.
        """
        text = await self.run(message, **kwargs)
        return AssistantMsg(name=self.name, content=text)

    async def run_stream(self, message: str, **kwargs: Any):
        """Execute the agent and yield streaming events."""
        user_msg = UserMsg(name="user", content=message)
        async for event in self.agent.reply_stream(user_msg):
            yield event

    async def resume(self, payload: dict[str, Any]) -> str:
        """Resume an interrupted conversation (default: not supported)."""
        raise NotImplementedError("resume is not supported by this agent type")

    def get_state(self) -> AgentState:
        """Return the current agent state."""
        return self._state

    def set_state(self, state: AgentState) -> None:
        """Restore agent state."""
        self._state = state
        if self._agent is not None:
            self._agent.state = state

    def describe(self) -> dict[str, Any]:
        """Return a JSON-serialisable description of this agent."""
        return {
            "id": self.agent_id,
            "name": self.name,
            "type": self.__class__.__name__,
            "model": self.model_name,
            "system_prompt": self.system_prompt[:200] if self.system_prompt else "",
        }
