"""AgentScope 2.0 Event types.

Events are the streaming counterpart of messages. They represent incremental
updates as an agent generates a reply, enabling real-time streaming to clients.
"""

from __future__ import annotations

import time
import uuid
from dataclasses import dataclass, field
from typing import Any


# ──────────────────────────────────────────────
# Base event
# ──────────────────────────────────────────────


@dataclass
class EventBase:
    """Base class for all agent events."""

    id: str = ""
    created_at: str = ""

    def __post_init__(self) -> None:
        if not self.id:
            self.id = uuid.uuid4().hex
        if not self.created_at:
            self.created_at = time.strftime("%Y-%m-%dT%H:%M:%S", time.gmtime()) + f".{int(time.time() * 1000) % 1000:03d}Z"


# ──────────────────────────────────────────────
# Reply lifecycle events
# ──────────────────────────────────────────────


@dataclass
class ReplyStartEvent(EventBase):
    """Signals the start of a reply."""

    reply_id: str = ""
    session_id: str = ""
    name: str = ""
    role: str = "assistant"


@dataclass
class ReplyEndEvent(EventBase):
    """Signals the end of a reply."""

    reply_id: str = ""
    session_id: str = ""


@dataclass
class ExceedMaxItersEvent(EventBase):
    """Signals that the agent exceeded maximum iterations."""

    reply_id: str = ""
    name: str = ""


# ──────────────────────────────────────────────
# Text block events
# ──────────────────────────────────────────────


@dataclass
class TextBlockStartEvent(EventBase):
    """Signals the start of a text block."""

    reply_id: str = ""
    block_id: str = ""


@dataclass
class TextBlockDeltaEvent(EventBase):
    """A text chunk within a text block."""

    reply_id: str = ""
    block_id: str = ""
    delta: str = ""


@dataclass
class TextBlockEndEvent(EventBase):
    """Signals the end of a text block."""

    reply_id: str = ""
    block_id: str = ""


# ──────────────────────────────────────────────
# Thinking block events
# ──────────────────────────────────────────────


@dataclass
class ThinkingBlockStartEvent(EventBase):
    """Signals the start of a thinking/reasoning block."""

    reply_id: str = ""
    block_id: str = ""


@dataclass
class ThinkingBlockDeltaEvent(EventBase):
    """A thinking chunk within a thinking block."""

    reply_id: str = ""
    block_id: str = ""
    delta: str = ""


@dataclass
class ThinkingBlockEndEvent(EventBase):
    """Signals the end of a thinking block."""

    reply_id: str = ""
    block_id: str = ""


# ──────────────────────────────────────────────
# Data block events
# ──────────────────────────────────────────────


@dataclass
class DataBlockStartEvent(EventBase):
    """Signals the start of a data block (image, file, etc.)."""

    reply_id: str = ""
    block_id: str = ""
    media_type: str = ""


@dataclass
class DataBlockDeltaEvent(EventBase):
    """A data chunk within a data block."""

    reply_id: str = ""
    block_id: str = ""
    delta: str = ""


@dataclass
class DataBlockEndEvent(EventBase):
    """Signals the end of a data block."""

    reply_id: str = ""
    block_id: str = ""


# ──────────────────────────────────────────────
# Tool call events
# ──────────────────────────────────────────────


@dataclass
class ToolCallStartEvent(EventBase):
    """Signals the start of a tool call."""

    reply_id: str = ""
    tool_call_id: str = ""
    tool_call_name: str = ""


@dataclass
class ToolCallDeltaEvent(EventBase):
    """A chunk of tool call arguments (typically streamed JSON)."""

    reply_id: str = ""
    tool_call_id: str = ""
    delta: str = ""


@dataclass
class ToolCallEndEvent(EventBase):
    """Signals the end of a tool call."""

    reply_id: str = ""
    tool_call_id: str = ""


# ──────────────────────────────────────────────
# Tool result events
# ──────────────────────────────────────────────


@dataclass
class ToolResultStartEvent(EventBase):
    """Signals the start of a tool result."""

    reply_id: str = ""
    tool_call_id: str = ""
    tool_call_name: str = ""


@dataclass
class ToolResultTextDeltaEvent(EventBase):
    """A text chunk within a tool result."""

    reply_id: str = ""
    tool_call_id: str = ""
    delta: str = ""


@dataclass
class ToolResultDataDeltaEvent(EventBase):
    """A data chunk within a tool result."""

    reply_id: str = ""
    tool_call_id: str = ""
    delta: str = ""
    media_type: str = ""


@dataclass
class ToolResultEndEvent(EventBase):
    """Signals the end of a tool result."""

    reply_id: str = ""
    tool_call_id: str = ""
    state: str = "success"  # success, error, interrupted, denied


# ──────────────────────────────────────────────
# Model call events
# ──────────────────────────────────────────────


@dataclass
class ModelCallStartEvent(EventBase):
    """Signals the start of a model call."""

    reply_id: str = ""
    model_name: str = ""


@dataclass
class ModelCallEndEvent(EventBase):
    """Signals the end of a model call with token usage."""

    reply_id: str = ""
    input_tokens: int = 0
    output_tokens: int = 0


# ──────────────────────────────────────────────
# Interrupt / confirmation events
# ──────────────────────────────────────────────


@dataclass
class RequireUserConfirmEvent(EventBase):
    """Signals that the agent requires user confirmation before proceeding."""

    reply_id: str = ""
    tool_calls: list[dict[str, Any]] = field(default_factory=list)


@dataclass
class RequireExternalExecutionEvent(EventBase):
    """Signals that external execution is required."""

    reply_id: str = ""
    execution_id: str = ""
    payload: dict[str, Any] = field(default_factory=dict)


@dataclass
class UserConfirmResultEvent(EventBase):
    """Carries the user's confirmation result."""

    reply_id: str = ""
    confirmed: bool = False
    payload: dict[str, Any] = field(default_factory=dict)


@dataclass
class ExternalExecutionResultEvent(EventBase):
    """Carries the result of external execution."""

    reply_id: str = ""
    execution_id: str = ""
    result: dict[str, Any] = field(default_factory=dict)


# ──────────────────────────────────────────────
# Hint and custom events
# ──────────────────────────────────────────────


@dataclass
class HintBlockEvent(EventBase):
    """A hint event for injecting context or suggestions."""

    reply_id: str = ""
    block_id: str = ""
    hint: str | list[Any] = ""
    source: str | None = None


@dataclass
class CustomEvent(EventBase):
    """A custom event for extensibility."""

    reply_id: str = ""
    name: str = ""
    value: dict[str, Any] = field(default_factory=dict)
