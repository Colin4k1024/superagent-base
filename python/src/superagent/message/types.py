"""AgentScope 2.0 Message types.

Provides Msg and ContentBlock variants for agent communication and persistence.
Compatible with the AgentScope 2.0 message protocol.
"""

from __future__ import annotations

import time
import uuid
from dataclasses import dataclass, field
from typing import Any, Literal, Union


# ──────────────────────────────────────────────
# Content Block types
# ──────────────────────────────────────────────


@dataclass
class TextBlock:
    """A text content block."""

    type: Literal["text"] = "text"
    text: str = ""


@dataclass
class DataBlock:
    """A binary/URL data content block (images, files, etc.)."""

    type: Literal["data"] = "data"
    source: dict[str, str] = field(default_factory=dict)  # {"base64": "..."} or {"url": "..."}
    media_type: str = ""


@dataclass
class ThinkingBlock:
    """A thinking/reasoning content block."""

    type: Literal["thinking"] = "thinking"
    thinking: str = ""


@dataclass
class ToolCallBlock:
    """A tool call content block."""

    type: Literal["tool_call"] = "tool_call"
    name: str = ""
    arguments: dict[str, Any] = field(default_factory=dict)
    status: str = "pending"  # pending, running, done, error


@dataclass
class ToolResultBlock:
    """A tool result content block."""

    type: Literal["tool_result"] = "tool_result"
    name: str = ""
    output: str = ""
    state: str = "success"  # success, error, interrupted, denied


@dataclass
class HintBlock:
    """A hint content block for injecting context or suggestions."""

    type: Literal["hint"] = "hint"
    hint: str | list[TextBlock | DataBlock] = ""
    source: str | None = None


# Union type for all content blocks
ContentBlock = Union[TextBlock, DataBlock, ThinkingBlock, ToolCallBlock, ToolResultBlock, HintBlock]


# ──────────────────────────────────────────────
# Usage tracking
# ──────────────────────────────────────────────


@dataclass
class Usage:
    """Token usage statistics for a message."""

    input_tokens: int = 0
    output_tokens: int = 0


# ──────────────────────────────────────────────
# Msg — the core message type
# ──────────────────────────────────────────────


@dataclass
class Msg:
    """A message in the AgentScope 2.0 protocol.

    Messages are the basic unit for agent communication and persistence.
    They contain ordered typed content blocks and metadata.
    """

    id: str = ""
    name: str = ""
    role: str = "assistant"  # user, assistant, system
    content: list[ContentBlock] = field(default_factory=list)
    metadata: dict[str, Any] = field(default_factory=dict)
    created_at: str = ""
    finished_at: str | None = None
    usage: Usage = field(default_factory=Usage)

    def __post_init__(self) -> None:
        if not self.id:
            self.id = uuid.uuid4().hex
        if not self.created_at:
            self.created_at = _iso_now()

    def get_text_content(self, separator: str = "\n") -> str | None:
        """Get concatenated text from all TextBlocks.

        Returns None if there are no TextBlocks.
        """
        texts = [block.text for block in self.content if isinstance(block, TextBlock) and block.text]
        if not texts:
            return None
        return separator.join(texts)

    def get_content_blocks(self, block_type: str | type) -> list[ContentBlock]:
        """Filter content blocks by type name or class.

        Args:
            block_type: Either a string like "text" or a class like TextBlock.
        """
        if isinstance(block_type, str):
            return [block for block in self.content if getattr(block, "type", None) == block_type]
        return [block for block in self.content if isinstance(block, block_type)]

    def has_content_blocks(self, block_type: str | type) -> bool:
        """Check if message has blocks of given type."""
        return len(self.get_content_blocks(block_type)) > 0

    def append_event(self, event: Any) -> None:
        """Append a streaming event to reconstruct message state.

        This implements the AgentScope 2.0 append_event pattern where
        events from reply_stream can be accumulated into a full Msg.
        """
        from superagent.event.types import (
            TextBlockStartEvent,
            TextBlockDeltaEvent,
            TextBlockEndEvent,
            ThinkingBlockStartEvent,
            ThinkingBlockDeltaEvent,
            ThinkingBlockEndEvent,
            ToolCallStartEvent,
            ToolCallDeltaEvent,
            ToolCallEndEvent,
            ToolResultStartEvent,
            ToolResultTextDeltaEvent,
            ToolResultEndEvent,
            HintBlockEvent,
            ModelCallEndEvent,
            ReplyEndEvent,
        )

        if isinstance(event, TextBlockStartEvent):
            self.content.append(TextBlock(text=""))

        elif isinstance(event, TextBlockDeltaEvent):
            # Find last TextBlock and append delta
            for block in reversed(self.content):
                if isinstance(block, TextBlock):
                    block.text += event.delta
                    break

        elif isinstance(event, TextBlockEndEvent):
            pass  # TextBlock is complete, no action needed

        elif isinstance(event, ThinkingBlockStartEvent):
            self.content.append(ThinkingBlock(thinking=""))

        elif isinstance(event, ThinkingBlockDeltaEvent):
            for block in reversed(self.content):
                if isinstance(block, ThinkingBlock):
                    block.thinking += event.delta
                    break

        elif isinstance(event, ThinkingBlockEndEvent):
            pass

        elif isinstance(event, ToolCallStartEvent):
            self.content.append(ToolCallBlock(
                name=event.tool_call_name,
                arguments={},
                status="running",
            ))

        elif isinstance(event, ToolCallDeltaEvent):
            # Tool call deltas typically stream JSON arguments
            for block in reversed(self.content):
                if isinstance(block, ToolCallBlock):
                    # Accumulate raw delta; caller may parse JSON separately
                    block.arguments["_raw_delta"] = block.arguments.get("_raw_delta", "") + event.delta
                    break

        elif isinstance(event, ToolCallEndEvent):
            for block in reversed(self.content):
                if isinstance(block, ToolCallBlock):
                    block.status = "done"
                    break

        elif isinstance(event, ToolResultStartEvent):
            self.content.append(ToolResultBlock(
                name=event.tool_call_name,
                output="",
                state="success",
            ))

        elif isinstance(event, ToolResultTextDeltaEvent):
            for block in reversed(self.content):
                if isinstance(block, ToolResultBlock):
                    block.output += event.delta
                    break

        elif isinstance(event, ToolResultEndEvent):
            for block in reversed(self.content):
                if isinstance(block, ToolResultBlock):
                    block.state = event.state
                    break

        elif isinstance(event, HintBlockEvent):
            self.content.append(HintBlock(
                hint=event.hint,
                source=event.source,
            ))

        elif isinstance(event, ModelCallEndEvent):
            self.usage.input_tokens += event.input_tokens
            self.usage.output_tokens += event.output_tokens

        elif isinstance(event, ReplyEndEvent):
            self.finished_at = _iso_now()

    def to_dict(self) -> dict[str, Any]:
        """Serialize Msg to a plain dict for storage/transport."""
        return {
            "id": self.id,
            "name": self.name,
            "role": self.role,
            "content": [_block_to_dict(b) for b in self.content],
            "metadata": self.metadata,
            "created_at": self.created_at,
            "finished_at": self.finished_at,
            "usage": {"input_tokens": self.usage.input_tokens, "output_tokens": self.usage.output_tokens},
        }

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> Msg:
        """Deserialize a Msg from a plain dict."""
        content_blocks: list[ContentBlock] = []
        for block_data in data.get("content", []):
            block = _block_from_dict(block_data)
            if block is not None:
                content_blocks.append(block)

        usage_data = data.get("usage", {})
        usage = Usage(
            input_tokens=usage_data.get("input_tokens", 0),
            output_tokens=usage_data.get("output_tokens", 0),
        )

        return cls(
            id=data.get("id", ""),
            name=data.get("name", ""),
            role=data.get("role", "assistant"),
            content=content_blocks,
            metadata=data.get("metadata", {}),
            created_at=data.get("created_at", ""),
            finished_at=data.get("finished_at"),
            usage=usage,
        )


# ──────────────────────────────────────────────
# Convenience constructors
# ──────────────────────────────────────────────


def UserMsg(name: str = "user", content: str | list[ContentBlock] = "", **kwargs: Any) -> Msg:
    """Create a user message."""
    blocks = _ensure_content_blocks(content)
    return Msg(name=name, role="user", content=blocks, **kwargs)


def AssistantMsg(
    name: str = "assistant",
    content: str | list[ContentBlock] = "",
    id: str = "",
    **kwargs: Any,
) -> Msg:
    """Create an assistant message."""
    blocks = _ensure_content_blocks(content)
    msg = Msg(name=name, role="assistant", content=blocks, **kwargs)
    if id:
        msg.id = id
    return msg


def SystemMsg(name: str = "system", content: str | list[ContentBlock] = "", **kwargs: Any) -> Msg:
    """Create a system message."""
    blocks = _ensure_content_blocks(content)
    return Msg(name=name, role="system", content=blocks, **kwargs)


# ──────────────────────────────────────────────
# Internal helpers
# ──────────────────────────────────────────────


def _iso_now() -> str:
    """Return current time as ISO 8601 string."""
    return time.strftime("%Y-%m-%dT%H:%M:%S", time.gmtime()) + f".{int(time.time() * 1000) % 1000:03d}Z"


def _ensure_content_blocks(content: str | list[ContentBlock]) -> list[ContentBlock]:
    """Convert string content to a list with a single TextBlock."""
    if isinstance(content, str):
        return [TextBlock(text=content)]
    return content


def _block_to_dict(block: ContentBlock) -> dict[str, Any]:
    """Serialize a ContentBlock to a dict."""
    if isinstance(block, TextBlock):
        return {"type": "text", "text": block.text}
    elif isinstance(block, DataBlock):
        return {"type": "data", "source": block.source, "media_type": block.media_type}
    elif isinstance(block, ThinkingBlock):
        return {"type": "thinking", "thinking": block.thinking}
    elif isinstance(block, ToolCallBlock):
        return {"type": "tool_call", "name": block.name, "arguments": block.arguments, "status": block.status}
    elif isinstance(block, ToolResultBlock):
        return {"type": "tool_result", "name": block.name, "output": block.output, "state": block.state}
    elif isinstance(block, HintBlock):
        hint_val: str | list[dict[str, Any]] = ""
        if isinstance(block.hint, str):
            hint_val = block.hint
        elif isinstance(block.hint, list):
            hint_val = [_block_to_dict(b) for b in block.hint]
        return {"type": "hint", "hint": hint_val, "source": block.source}
    return {"type": "unknown"}


def _block_from_dict(data: dict[str, Any]) -> ContentBlock | None:
    """Deserialize a ContentBlock from a dict."""
    block_type = data.get("type")
    if block_type == "text":
        return TextBlock(text=data.get("text", ""))
    elif block_type == "data":
        return DataBlock(source=data.get("source", {}), media_type=data.get("media_type", ""))
    elif block_type == "thinking":
        return ThinkingBlock(thinking=data.get("thinking", ""))
    elif block_type == "tool_call":
        return ToolCallBlock(name=data.get("name", ""), arguments=data.get("arguments", {}), status=data.get("status", "pending"))
    elif block_type == "tool_result":
        return ToolResultBlock(name=data.get("name", ""), output=data.get("output", ""), state=data.get("state", "success"))
    elif block_type == "hint":
        hint_data = data.get("hint", "")
        if isinstance(hint_data, list):
            hint_val: str | list[TextBlock | DataBlock] = [_block_from_dict(b) for b in hint_data if _block_from_dict(b) is not None]  # type: ignore[assignment]
        else:
            hint_val = str(hint_data)
        return HintBlock(hint=hint_val, source=data.get("source"))
    return None
