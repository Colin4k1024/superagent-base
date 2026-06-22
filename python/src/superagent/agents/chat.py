"""chat_model_agent — single-model ReAct agent powered by AgentScope 2.0."""

from __future__ import annotations

import os
from typing import Any

from agentscope.agent import Agent
from agentscope.model import DashScopeChatModel, OpenAIChatModel
from agentscope.credential import DashScopeCredential, OpenAICredential
from agentscope.tool import Toolkit
from pydantic import SecretStr

from superagent.agents.base import BaseAgent


class ChatModelAgent(BaseAgent):
    """Single-model ReAct agent.

    Wraps ``agentscope.agent.Agent`` with tool-calling and step-limited reasoning.
    Supports both DashScope and OpenAI-compatible endpoints.
    """

    def __init__(
        self,
        agent_id: str,
        name: str,
        system_prompt: str = "You are a helpful assistant.",
        model_name: str = "gpt-4o",
        max_steps: int = 10,
        tools: list[str] | None = None,
        model_provider: str = "openai",
        api_key: str = "",
        base_url: str = "",
        **kwargs: Any,
    ) -> None:
        super().__init__(agent_id, name, system_prompt, model_name, **kwargs)
        self.max_steps = max_steps
        self.tools = tools or []
        self.model_provider = model_provider
        self.api_key = api_key or os.getenv("MODEL_API_KEY_0", "")
        self.base_url = base_url or os.getenv("MODEL_BASE_URL_0", "")

    def _build_model(self):
        """Build the appropriate model based on provider."""
        secret_key = SecretStr(self.api_key)
        if self.model_provider == "dashscope":
            return DashScopeChatModel(
                credential=DashScopeCredential(api_key=secret_key),
                model=self.model_name,
            )
        else:
            # OpenAI-compatible (default)
            model_kwargs: dict[str, Any] = {}
            if self.base_url:
                model_kwargs["base_url"] = self.base_url
            return OpenAIChatModel(
                credential=OpenAICredential(api_key=secret_key),
                model=self.model_name,
                **model_kwargs,
            )

    def _build_toolkit(self) -> Toolkit:
        """Build toolkit by resolving tool names to AgentScope ToolBase instances.

        Uses ``TOOL_REGISTRY`` from ``superagent.tools`` to map tool refs
        (e.g. ``builtin/web_search``) to concrete ``ToolBase`` subclasses.
        """
        if not self.tools:
            return Toolkit()

        from superagent.tools import TOOL_REGISTRY

        tool_instances = []
        for tool_ref in self.tools:
            # Strip ``ref:`` prefix if present (from YAML ``tools: [{ref: builtin/web_search}]``)
            ref = tool_ref
            if isinstance(tool_ref, dict):
                ref = tool_ref.get("ref", str(tool_ref))
            ref = str(ref)

            cls = TOOL_REGISTRY.get(ref)
            if cls is not None:
                tool_instances.append(cls())
            else:
                # Unknown tool ref — skip with warning
                import logging
                logging.getLogger(__name__).warning("Unknown tool ref %r, skipping", ref)

        return Toolkit(tools=tool_instances if tool_instances else None)

    def _build_agent(self) -> Agent:
        """Build the AgentScope Agent."""
        return Agent(
            name=self.name,
            system_prompt=self.system_prompt,
            model=self._build_model(),
            toolkit=self._build_toolkit(),
        )

    def describe(self) -> dict[str, Any]:
        """Return agent description with additional metadata."""
        base = super().describe()
        base.update({
            "max_steps": self.max_steps,
            "tools": self.tools,
            "model_provider": self.model_provider,
        })
        return base
