"""Supervisor agent — routes tasks to child agents based on capability.

Uses a real AgentScope 2.0 Agent with a FunctionTool for LLM-based routing.
The supervisor agent analyses the incoming message and delegates to the most
appropriate child agent.
"""

from __future__ import annotations

import json
import logging
import os
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
from agentscope.model import DashScopeChatModel, OpenAIChatModel
from agentscope.credential import DashScopeCredential, OpenAICredential
from agentscope.tool import FunctionTool, Toolkit
from pydantic import SecretStr

from superagent.agents.base import BaseAgent

logger = logging.getLogger(__name__)


def _build_route_tool(children: list[BaseAgent]) -> Any:
    """Build a routing function that the supervisor LLM can call.

    The function accepts ``agent_name`` and ``task``, and delegates to the
    named child agent synchronously (via ``asyncio.run``).
    """
    child_map = {c.name: c for c in children}
    child_names = list(child_map.keys())
    children_desc = json.dumps([
        {"name": c.name, "description": c.describe().get("system_prompt", "")[:200]}
        for c in children
    ], ensure_ascii=False)

    async def route_to_agent(agent_name: str, task: str) -> str:
        """Route a task to a named child agent and return its response.

        Args:
            agent_name: Name of the child agent to delegate to.
            task: The task/message to send to the child agent.
        """
        child = child_map.get(agent_name)
        if child is None:
            available = ", ".join(child_names)
            return json.dumps({
                "error": f"Agent {agent_name!r} not found. Available: {available}",
                "available_agents": child_names,
            })
        result = await child.run(task)
        return json.dumps({"agent": agent_name, "result": result}, ensure_ascii=False)

    # Set metadata for AgentScope
    route_to_agent.__name__ = "route_to_agent"
    route_to_agent.__doc__ = (
        f"Route a task to one of the available child agents. "
        f"Available agents: {children_desc}"
    )
    return route_to_agent


class SupervisorAgent(BaseAgent):
    """Routes incoming tasks to the most appropriate child agent.

    Uses a real AgentScope Agent with an LLM to decide which child agent
    should handle each task.  Supports multi-round delegation: the LLM
    can call the routing tool multiple times before producing a final answer.
    """

    def __init__(
        self,
        agent_id: str,
        name: str,
        children: list[BaseAgent] | None = None,
        model_provider: str = "",
        api_key: str = "",
        base_url: str = "",
        **kwargs: Any,
    ) -> None:
        super().__init__(agent_id, name, **kwargs)
        self.children: list[BaseAgent] = children or []
        self.model_provider = model_provider or os.getenv("MODEL_PROVIDER_0", "openai")
        self.api_key = api_key or os.getenv("MODEL_API_KEY_0", "")
        self.base_url = base_url or os.getenv("MODEL_BASE_URL_0", "")

    def _build_model(self):
        """Build the LLM model for the supervisor's routing decisions."""
        secret_key = SecretStr(self.api_key)
        if self.model_provider == "dashscope":
            return DashScopeChatModel(
                credential=DashScopeCredential(api_key=secret_key),
                model=self.model_name,
            )
        else:
            model_kwargs: dict[str, Any] = {}
            if self.base_url:
                model_kwargs["base_url"] = self.base_url
            return OpenAIChatModel(
                credential=OpenAICredential(api_key=secret_key),
                model=self.model_name,
                **model_kwargs,
            )

    def _build_agent(self) -> Agent:
        """Build the AgentScope Agent for supervisor routing.

        Creates a FunctionTool wrapping the routing logic and an Agent
        with a system prompt that instructs the LLM to use the tool.
        """
        if not self.children:
            # No children — build a minimal agent that explains the situation
            return Agent(
                name=self.name,
                system_prompt=(
                    f"{self.system_prompt}\n\n"
                    "You are a supervisor agent with no child agents registered. "
                    "Explain that no children are available to delegate to."
                ),
                model=self._build_model(),
            )

        route_fn = _build_route_tool(self.children)
        toolkit = Toolkit(tools=[FunctionTool(route_fn)])

        children_desc = "\n".join(
            f"- {c.name}: {c.describe().get('system_prompt', 'No description')[:200]}"
            for c in self.children
        )

        prompt = (
            f"{self.system_prompt}\n\n"
            f"You are a supervisor agent. Your job is to analyse the user's request "
            f"and delegate it to the most appropriate child agent using the "
            f"route_to_agent tool.\n\n"
            f"Available child agents:\n{children_desc}\n\n"
            f"Always call route_to_agent with the chosen agent_name and a clear task. "
            f"If multiple agents are needed, call the tool multiple times."
        )

        return Agent(
            name=self.name,
            system_prompt=prompt,
            model=self._build_model(),
            toolkit=toolkit,
        )

    @property
    def agent(self) -> Agent:
        """Lazy-initialise the underlying agent."""
        if self._agent is None:
            self._agent = self._build_agent()
        return self._agent

    async def run(self, message: str, **kwargs: Any) -> str:
        """Execute supervisor: use LLM to route, fall back to first child."""
        if not self.children:
            return f"[SupervisorAgent:{self.name}] No children registered"

        if not self.api_key:
            # No API key — fall back to first child (backward-compatible)
            child = self.children[0]
            return await child.run(message, **kwargs)

        # Use the real AgentScope agent for LLM-based routing
        try:
            from agentscope.message import UserMsg
            user_msg = UserMsg(name="user", content=message)
            reply_msg = await self.agent.reply(user_msg)
            text = reply_msg.get_text_content()
            return text if text is not None else ""
        except Exception as exc:
            logger.warning("Supervisor LLM routing failed (%s), falling back to first child", exc)
            child = self.children[0]
            return await child.run(message, **kwargs)

    async def run_stream(self, message: str, **kwargs: Any) -> AsyncGenerator[AgentEvent, None]:
        """Execute supervisor with streaming.

        If an API key is available, streams through the AgentScope agent.
        Otherwise falls back to the first child's stream.
        """
        if not self.children:
            reply_id = uuid.uuid4().hex
            yield ReplyStartEvent(session_id="", reply_id=reply_id, name=self.name)
            block_id = uuid.uuid4().hex
            yield TextBlockStartEvent(reply_id=reply_id, block_id=block_id)
            yield TextBlockDeltaEvent(
                reply_id=reply_id, block_id=block_id,
                delta=f"[SupervisorAgent:{self.name}] No children registered",
            )
            yield TextBlockEndEvent(reply_id=reply_id, block_id=block_id)
            yield ReplyEndEvent(session_id="", reply_id=reply_id)
            return

        if not self.api_key:
            # No API key — fall back to first child
            child = self.children[0]
            async for event in child.run_stream(message, **kwargs):
                yield event
            return

        # Stream through the real AgentScope agent
        try:
            from agentscope.message import UserMsg
            user_msg = UserMsg(name="user", content=message)
            async for event in self.agent.reply_stream(user_msg):
                yield event
        except Exception as exc:
            logger.warning("Supervisor streaming failed (%s), falling back to first child", exc)
            child = self.children[0]
            async for event in child.run_stream(message, **kwargs):
                yield event

    def describe(self) -> dict[str, Any]:
        base = super().describe()
        base["children"] = [c.describe() for c in self.children]
        base["model_provider"] = self.model_provider
        return base
