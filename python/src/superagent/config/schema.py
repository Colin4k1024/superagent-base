"""Pydantic schemas for Superagent configuration and agent definitions.

Mirrors the Go base's agentdef package schema.
"""

from __future__ import annotations

from typing import Any

from pydantic import BaseModel, Field


class ModelSpec(BaseModel):
    primary: str = "gpt-4o"
    fallback: str = ""
    router: str = ""


class InterruptSpec(BaseModel):
    enabled: bool = False
    checkpoint_backend: str = "builtin"
    timeout_seconds: int = 300


class WorkflowNodeSpec(BaseModel):
    id: str
    type: str = "agent_call"
    agent_id: str | None = None
    tool: str | None = None
    code: str | None = None
    condition: str | None = None
    config: dict[str, Any] = Field(default_factory=dict)


class WorkflowEdgeSpec(BaseModel):
    source: str = ""
    target: str = ""
    from_id: str = ""
    to_id: str = ""
    condition: str | None = None
    variable_map: dict[str, str] = Field(default_factory=dict)

    def get_source(self) -> str:
        return self.source or self.from_id

    def get_target(self) -> str:
        return self.target or self.to_id


class WorkflowSpec(BaseModel):
    nodes: list[WorkflowNodeSpec] = Field(default_factory=list)
    edges: list[WorkflowEdgeSpec] = Field(default_factory=list)


class AgentMetadata(BaseModel):
    name: str
    version: str = "1.0.0"
    labels: dict[str, str] = Field(default_factory=dict)
    annotations: dict[str, str] = Field(default_factory=dict)


class ToolRef(BaseModel):
    """Tool reference — matches Go base's ToolRef struct."""
    ref: str
    config: dict[str, Any] = Field(default_factory=dict)


class SubAgentRef(BaseModel):
    """Sub-agent reference — matches Go base's SubAgentRef struct."""
    ref: str
    role: str = ""
    config: dict[str, Any] = Field(default_factory=dict)


def _normalize_tool_ref(value: str | dict[str, Any] | ToolRef) -> ToolRef:
    """Accept string, dict, or ToolRef and normalize to ToolRef."""
    if isinstance(value, ToolRef):
        return value
    if isinstance(value, str):
        return ToolRef(ref=value)
    if isinstance(value, dict):
        return ToolRef(ref=value.get("ref", ""), config=value.get("config", {}))
    return ToolRef(ref=str(value))


def _normalize_sub_agent_ref(value: str | dict[str, Any] | SubAgentRef) -> SubAgentRef:
    """Accept string, dict, or SubAgentRef and normalize to SubAgentRef."""
    if isinstance(value, SubAgentRef):
        return value
    if isinstance(value, str):
        return SubAgentRef(ref=value)
    if isinstance(value, dict):
        return SubAgentRef(
            ref=value.get("ref", ""),
            role=value.get("role", ""),
            config=value.get("config", {}),
        )
    return SubAgentRef(ref=str(value))


class AgentSpec(BaseModel):
    type: str = "chat_model_agent"
    model: str | ModelSpec = "gpt-4o"
    system_prompt: str = ""
    tools: list[str | dict[str, Any] | ToolRef] = Field(default_factory=list)
    max_steps: int = 10
    interrupt: InterruptSpec | None = None
    sub_agents: list[str | dict[str, Any] | SubAgentRef] = Field(default_factory=list)
    workflow: WorkflowSpec | None = None
    workflow_nodes: list[dict[str, Any]] = Field(default_factory=list)
    workflow_edges: list[dict[str, Any]] = Field(default_factory=list)

    def get_tools(self) -> list[ToolRef]:
        """Return normalized tool references."""
        return [_normalize_tool_ref(t) for t in self.tools]

    def get_sub_agents(self) -> list[SubAgentRef]:
        """Return normalized sub-agent references."""
        return [_normalize_sub_agent_ref(s) for s in self.sub_agents]

    def get_model_name(self) -> str:
        if isinstance(self.model, ModelSpec):
            return self.model.primary
        return self.model


class AgentDefinition(BaseModel):
    """Full agent definition — matches the Go base's AgentDefinition struct."""
    apiVersion: str = "superagent/v1"
    kind: str = "Agent"
    metadata: AgentMetadata
    spec: AgentSpec = Field(default_factory=AgentSpec)


class ModelProviderConfig(BaseModel):
    name: str
    provider: str
    base_url: str = ""
    api_key: str = ""
    models: list[str] = Field(default_factory=list)


class SuperagentConfig(BaseModel):
    version: str = "1"
    providers: list[ModelProviderConfig] = Field(default_factory=list)
    agents_dir: str = "configs/agents"
    redis_url: str = "redis://localhost:6379/0"
    log_level: str = "INFO"
