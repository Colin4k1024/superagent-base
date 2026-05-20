"""Superagent Python SDK.

Quick start::

    import asyncio
    from superagent import SuperagentClient

    async def main():
        async with SuperagentClient(base_url="http://localhost:8888") as client:
            reply = await client.chat("research-agent", "What is quantum computing?")
            print(reply)

    asyncio.run(main())
"""

from .client import SuperagentClient
from .exceptions import (
    AuthenticationError,
    InterruptConflictError,
    NotFoundError,
    RateLimitError,
    ServerError,
    StreamDisconnectedError,
    SuperagentError,
    ValidationError,
)
from .streaming import A2UIStream
from .types import (
    A2UIEvent,
    A2UIEventType,
    AgentInfo,
    ApplyResult,
    ChatRequest,
    ValidateResult,
)

__all__ = [
    # Client
    "SuperagentClient",
    # Streaming
    "A2UIStream",
    # Types
    "A2UIEvent",
    "A2UIEventType",
    "AgentInfo",
    "ApplyResult",
    "ChatRequest",
    "ValidateResult",
    # Exceptions
    "SuperagentError",
    "AuthenticationError",
    "NotFoundError",
    "ValidationError",
    "RateLimitError",
    "ServerError",
    "StreamDisconnectedError",
    "InterruptConflictError",
]

__version__ = "0.1.0"
