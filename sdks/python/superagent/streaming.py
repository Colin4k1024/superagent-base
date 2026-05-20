"""A2UI SSE streaming support."""

from __future__ import annotations

import asyncio
import json
from typing import Any, AsyncIterator, Callable, Coroutine, Dict, List, Optional

import httpx

from .exceptions import StreamDisconnectedError
from .types import A2UIEvent, A2UIEventType


class A2UIStream:
    """Async iterator over A2UI Server-Sent Events.

    Usage::

        stream = client.chat_stream("my-agent", "Hello")
        async for event in stream:
            if event.event_type == "text":
                print(event.text_delta, end="", flush=True)

    Alternatively use the high-level helpers::

        text = await stream.collect()
    """

    def __init__(self, response: httpx.Response) -> None:
        self._response = response
        self._iter: AsyncIterator[str] = response.aiter_lines()
        self._done = False
        # pending SSE fields before the blank-line flush
        self._pending_event: Optional[str] = None
        self._pending_data: Optional[str] = None
        # registered callbacks: event_type -> list[callback]
        self._callbacks: Dict[str, List[Callable[[A2UIEvent], Any]]] = {}

    # ------------------------------------------------------------------
    # Async iterator protocol
    # ------------------------------------------------------------------

    def __aiter__(self) -> "A2UIStream":
        return self

    async def __anext__(self) -> A2UIEvent:
        """Return the next parsed A2UI event, or raise StopAsyncIteration."""
        if self._done:
            raise StopAsyncIteration

        try:
            return await self._next_event()
        except StopAsyncIteration:
            self._done = True
            raise
        except httpx.RemoteProtocolError as exc:
            self._done = True
            raise StreamDisconnectedError(str(exc)) from exc

    async def _next_event(self) -> A2UIEvent:
        """Read SSE lines until a complete event is assembled."""
        event_type: Optional[str] = None
        data_lines: List[str] = []

        async for line in self._iter:
            line = line.rstrip("\r")

            if line == "":
                # Blank line: dispatch accumulated event if any
                if data_lines:
                    raw_data = "\n".join(data_lines)
                    evt = self._parse_event(event_type or "message", raw_data)
                    event_type = None
                    data_lines = []
                    await self._fire_callbacks(evt)
                    # Stop iteration after done event
                    if evt.event_type == A2UIEventType.done:
                        self._done = True
                    return evt
                # Empty flush — keep reading
                continue

            if line.startswith("event:"):
                event_type = line[6:].strip()
            elif line.startswith("data:"):
                data_lines.append(line[5:].strip())
            # Ignore id:, retry:, and comment lines

        raise StopAsyncIteration

    @staticmethod
    def _parse_event(event_type: str, raw_data: str) -> A2UIEvent:
        """Parse a raw SSE data payload into an A2UIEvent."""
        try:
            parsed = json.loads(raw_data)
            if isinstance(parsed, dict):
                # Server may embed event type inside the payload
                if "type" in parsed and event_type == "message":
                    event_type = parsed.pop("type")
                data: Dict[str, Any] = parsed
            else:
                data = {"value": parsed}
        except (json.JSONDecodeError, ValueError):
            data = {"raw": raw_data}

        return A2UIEvent(event_type=event_type, data=data)

    # ------------------------------------------------------------------
    # High-level helpers
    # ------------------------------------------------------------------

    async def collect(self) -> str:
        """Consume the entire stream and return concatenated text content."""
        parts: List[str] = []
        async for event in self:
            if event.event_type in (A2UIEventType.text, "text"):
                delta = event.data.get("delta", "") or event.data.get("content", "")
                if delta:
                    parts.append(delta)
        return "".join(parts)

    def on(
        self,
        event_type: str | A2UIEventType,
        callback: Callable[[A2UIEvent], Any],
    ) -> "A2UIStream":
        """Register a callback for a specific event type.

        The callback receives the :class:`A2UIEvent` and may be a plain
        function or a coroutine function.  Returns *self* for chaining.

        Example::

            stream.on("tool_call", lambda e: print("tool:", e.data))
        """
        key = event_type.value if isinstance(event_type, A2UIEventType) else event_type
        self._callbacks.setdefault(key, []).append(callback)
        return self

    async def _fire_callbacks(self, event: A2UIEvent) -> None:
        handlers = self._callbacks.get(event.event_type, [])
        for handler in handlers:
            result = handler(event)
            if asyncio.iscoroutine(result):
                await result

    async def aclose(self) -> None:
        """Close the underlying HTTP response."""
        await self._response.aclose()

    async def __aenter__(self) -> "A2UIStream":
        return self

    async def __aexit__(self, *_: Any) -> None:
        await self.aclose()
