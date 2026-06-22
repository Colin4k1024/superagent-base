"""AgentScope 2.0 Message types.

Provides Msg, ContentBlock variants, and convenience constructors for
agent communication and persistence.
"""

from superagent.message.types import (
    ContentBlock,
    DataBlock,
    HintBlock,
    Msg,
    TextBlock,
    ThinkingBlock,
    ToolCallBlock,
    ToolResultBlock,
    Usage,
    AssistantMsg,
    SystemMsg,
    UserMsg,
)

__all__ = [
    "ContentBlock",
    "DataBlock",
    "HintBlock",
    "Msg",
    "TextBlock",
    "ThinkingBlock",
    "ToolCallBlock",
    "ToolResultBlock",
    "Usage",
    "AssistantMsg",
    "SystemMsg",
    "UserMsg",
]
