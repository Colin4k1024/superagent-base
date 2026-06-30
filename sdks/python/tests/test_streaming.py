"""Tests for superagent.streaming module."""

import asyncio
from typing import List
from unittest.mock import AsyncMock, MagicMock, patch

import httpx
import pytest

from superagent.exceptions import StreamDisconnectedError
from superagent.streaming import A2UIStream
from superagent.types import A2UIEvent, A2UIEventType


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_sse_lines(events: list[tuple[str, str]]) -> str:
    """Build a raw SSE text block from (event_type, data_json) pairs.

    Each pair becomes:
        event: <type>\\n
        data: <json>\\n
        \\n

    A trailing blank line is included so httpx's ``aiter_lines()`` yields
    the empty string that the SSE parser needs to dispatch the final event.
    """
    parts: list[str] = []
    for etype, data in events:
        parts.append(f"event: {etype}")
        parts.append(f"data: {data}")
        parts.append("")
    # Join with newlines and add a final trailing newline so aiter_lines()
    # yields the empty-string separator for the last event block.
    return "\n".join(parts) + "\n"


def _make_response(text: str) -> httpx.Response:
    """Create a mock httpx.Response backed by plain text for streaming."""
    return httpx.Response(
        status_code=200,
        stream=httpx.ByteStream(text.encode()),
        headers={"content-type": "text/event-stream"},
    )


# ---------------------------------------------------------------------------
# _parse_event (static method)
# ---------------------------------------------------------------------------


class TestParseEvent:
    """Unit tests for A2UIStream._parse_event."""

    def test_parse_json_dict(self) -> None:
        evt = A2UIStream._parse_event("text", '{"delta": "hello"}')
        assert evt.event_type == "text"
        assert evt.data["delta"] == "hello"

    def test_parse_with_type_in_payload(self) -> None:
        """When event_type is 'message' and payload has 'type', it is used."""
        evt = A2UIStream._parse_event("message", '{"type": "tool_call", "name": "search"}')
        assert evt.event_type == "tool_call"
        assert evt.data["name"] == "search"
        assert "type" not in evt.data  # popped

    def test_parse_message_type_preserved_when_explicit(self) -> None:
        """When event_type is NOT 'message', payload 'type' is not overridden."""
        evt = A2UIStream._parse_event("text", '{"type": "ignored", "delta": "x"}')
        assert evt.event_type == "text"
        assert evt.data["type"] == "ignored"

    def test_parse_non_dict_json(self) -> None:
        evt = A2UIStream._parse_event("data", '[1, 2, 3]')
        assert evt.event_type == "data"
        assert evt.data == {"value": [1, 2, 3]}

    def test_parse_invalid_json(self) -> None:
        evt = A2UIStream._parse_event("text", "not json")
        assert evt.event_type == "text"
        assert evt.data == {"raw": "not json"}

    def test_parse_empty_json_object(self) -> None:
        evt = A2UIStream._parse_event("done", '{}')
        assert evt.event_type == "done"
        assert evt.data == {}


# ---------------------------------------------------------------------------
# A2UIStream async iteration
# ---------------------------------------------------------------------------


class TestA2UIStreamIteration:
    """Tests for the async iterator protocol."""

    async def test_iterate_single_text_event(self) -> None:
        raw = _make_sse_lines([("text", '{"delta": "hi"}')])
        resp = _make_response(raw)
        stream = A2UIStream(resp)
        events: List[A2UIEvent] = []
        async for evt in stream:
            events.append(evt)
        assert len(events) == 1
        assert events[0].event_type == "text"
        assert events[0].data["delta"] == "hi"

    async def test_iterate_multiple_events(self) -> None:
        raw = _make_sse_lines([
            ("text", '{"delta": "hello"}'),
            ("text", '{"delta": " world"}'),
            ("done", '{}'),
        ])
        resp = _make_response(raw)
        stream = A2UIStream(resp)
        events: List[A2UIEvent] = []
        async for evt in stream:
            events.append(evt)
        assert len(events) == 3
        assert events[2].event_type == "done"

    async def test_empty_stream_raises_stop(self) -> None:
        resp = _make_response("")
        stream = A2UIStream(resp)
        events: List[A2UIEvent] = []
        async for evt in stream:
            events.append(evt)
        assert events == []

    async def test_done_event_sets_done_flag(self) -> None:
        raw = _make_sse_lines([("done", '{}')])
        resp = _make_response(raw)
        stream = A2UIStream(resp)
        async for _ in stream:
            pass
        assert stream._done is True

    async def test_comments_and_unknown_lines_ignored(self) -> None:
        """Lines not starting with 'event:' or 'data:' are ignored."""
        raw = (
            ": this is a comment\n"
            "event: text\n"
            "retry: 3000\n"
            "data: {\"delta\": \"ok\"}\n"
            "\n"
        )
        resp = _make_response(raw)
        stream = A2UIStream(resp)
        events: List[A2UIEvent] = []
        async for evt in stream:
            events.append(evt)
        assert len(events) == 1
        assert events[0].data["delta"] == "ok"


# ---------------------------------------------------------------------------
# collect() helper
# ---------------------------------------------------------------------------


class TestCollect:
    """Tests for A2UIStream.collect()."""

    async def test_collects_text_deltas(self) -> None:
        raw = _make_sse_lines([
            ("text", '{"delta": "Hello"}'),
            ("text", '{"delta": " World"}'),
            ("done", '{}'),
        ])
        resp = _make_response(raw)
        stream = A2UIStream(resp)
        result = await stream.collect()
        assert result == "Hello World"

    async def test_collects_content_field(self) -> None:
        raw = _make_sse_lines([
            ("text", '{"content": "from content"}'),
            ("done", '{}'),
        ])
        resp = _make_response(raw)
        stream = A2UIStream(resp)
        result = await stream.collect()
        assert result == "from content"

    async def test_ignores_non_text_events(self) -> None:
        raw = _make_sse_lines([
            ("tool_call", '{"name": "search"}'),
            ("text", '{"delta": "answer"}'),
            ("done", '{}'),
        ])
        resp = _make_response(raw)
        stream = A2UIStream(resp)
        result = await stream.collect()
        assert result == "answer"

    async def test_collect_empty_stream(self) -> None:
        resp = _make_response("")
        stream = A2UIStream(resp)
        result = await stream.collect()
        assert result == ""

    async def test_collect_skips_empty_deltas(self) -> None:
        raw = _make_sse_lines([
            ("text", '{"delta": ""}'),
            ("text", '{"delta": "real"}'),
            ("text", '{"delta": ""}'),
            ("done", '{}'),
        ])
        resp = _make_response(raw)
        stream = A2UIStream(resp)
        result = await stream.collect()
        assert result == "real"


# ---------------------------------------------------------------------------
# Callbacks
# ---------------------------------------------------------------------------


class TestCallbacks:
    """Tests for the on() callback registration."""

    async def test_sync_callback_fired(self) -> None:
        raw = _make_sse_lines([("text", '{"delta": "x"}')])
        resp = _make_response(raw)
        stream = A2UIStream(resp)
        received: List[A2UIEvent] = []
        stream.on("text", lambda e: received.append(e))
        async for _ in stream:
            pass
        assert len(received) == 1
        assert received[0].data["delta"] == "x"

    async def test_async_callback_fired(self) -> None:
        raw = _make_sse_lines([("text", '{"delta": "y"}')])
        resp = _make_response(raw)
        stream = A2UIStream(resp)
        received: List[A2UIEvent] = []

        async def handler(evt: A2UIEvent) -> None:
            received.append(evt)

        stream.on("text", handler)
        async for _ in stream:
            pass
        assert len(received) == 1

    async def test_callback_with_enum_key(self) -> None:
        raw = _make_sse_lines([("done", '{}')])
        resp = _make_response(raw)
        stream = A2UIStream(resp)
        called = False

        def handler(e: A2UIEvent) -> None:
            nonlocal called
            called = True

        stream.on(A2UIEventType.done, handler)
        async for _ in stream:
            pass
        assert called is True

    async def test_multiple_callbacks_same_event(self) -> None:
        raw = _make_sse_lines([("text", '{"delta": "z"}')])
        resp = _make_response(raw)
        stream = A2UIStream(resp)
        count = {"n": 0}

        stream.on("text", lambda e: count.update(n=count["n"] + 1))
        stream.on("text", lambda e: count.update(n=count["n"] + 1))
        async for _ in stream:
            pass
        assert count["n"] == 2

    async def test_callback_not_fired_for_other_events(self) -> None:
        raw = _make_sse_lines([("done", '{}')])
        resp = _make_response(raw)
        stream = A2UIStream(resp)
        called = False
        stream.on("text", lambda e: setattr(e, "_called", True))
        async for _ in stream:
            pass
        # No text event, so text callback never fires

    async def test_on_returns_self_for_chaining(self) -> None:
        resp = _make_response("")
        stream = A2UIStream(resp)
        result = stream.on("text", lambda e: None).on("done", lambda e: None)
        assert result is stream


# ---------------------------------------------------------------------------
# Context manager
# ---------------------------------------------------------------------------


class TestContextManager:
    """Tests for async context manager protocol."""

    async def test_aenter_aexit(self) -> None:
        resp = _make_response("")
        stream = A2UIStream(resp)
        async with stream as s:
            assert s is stream

    async def test_aexit_calls_aclose(self) -> None:
        resp = _make_response("")
        stream = A2UIStream(resp)
        with patch.object(stream, "aclose", new_callable=AsyncMock) as mock_close:
            async with stream:
                pass
            mock_close.assert_awaited_once()


# ---------------------------------------------------------------------------
# RemoteProtocolError -> StreamDisconnectedError
# ---------------------------------------------------------------------------


class TestRemoteProtocolError:
    """RemoteProtocolError from httpx should become StreamDisconnectedError."""

    async def test_remote_protocol_error_converted(self) -> None:
        """When the underlying iterator raises RemoteProtocolError,
        __anext__ should convert it to StreamDisconnectedError."""
        resp = _make_response("")
        stream = A2UIStream(resp)
        # Monkey-patch the iterator to raise RemoteProtocolError
        async def _broken_iter():
            raise httpx.RemoteProtocolError("connection lost")
            yield  # make it an async generator  # noqa: E501

        stream._iter = _broken_iter()
        stream._done = False
        with pytest.raises(StreamDisconnectedError, match="connection lost"):
            async for _ in stream:
                pass
        assert stream._done is True
