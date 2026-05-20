"""Pydantic models and enums for the Superagent SDK."""

from __future__ import annotations

from enum import Enum
from typing import Any, Dict, List, Optional

from pydantic import BaseModel, Field


class A2UIEventType(str, Enum):
    """All event types emitted by the A2UI streaming protocol."""

    text = "text"
    thinking = "thinking"
    tool_call = "tool_call"
    tool_result = "tool_result"
    code_block = "code_block"
    interrupt = "interrupt"
    error = "error"
    done = "done"
    progress = "progress"
    agent_switch = "agent_switch"


class A2UIEvent(BaseModel):
    """A single event emitted over an A2UI SSE stream."""

    event_type: str
    data: Dict[str, Any] = Field(default_factory=dict)

    @property
    def type(self) -> A2UIEventType | str:
        """Return the event type as an enum value when recognised."""
        try:
            return A2UIEventType(self.event_type)
        except ValueError:
            return self.event_type

    @property
    def text_delta(self) -> str:
        """Convenience accessor for text/delta from a *text* event."""
        return self.data.get("delta", "") or self.data.get("content", "")


class AgentInfo(BaseModel):
    """Basic metadata about an agent returned by list/get endpoints."""

    name: str
    type: str = ""
    description: str = ""
    status: str = ""


class ChatRequest(BaseModel):
    """Request body sent to the chat endpoints."""

    agent_id: str
    message: str
    session_id: str = "default"
    stream: bool = True


class ApplyResult(BaseModel):
    """Result of a create-or-update agent operation."""

    name: str
    status: str  # "created" | "updated" | "unchanged"
    message: str = ""


class ValidateResult(BaseModel):
    """Result of a YAML validation request."""

    valid: bool
    errors: List[str] = Field(default_factory=list)
    warnings: List[str] = Field(default_factory=list)
