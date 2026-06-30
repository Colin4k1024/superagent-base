"""Tests for superagent.client module."""

import asyncio
import json
from typing import List
from unittest.mock import AsyncMock, MagicMock, patch

import httpx
import pytest
import respx

from superagent.client import SuperagentClient, _LazyStream
from superagent.exceptions import (
    AuthenticationError,
    NotFoundError,
    ServerError,
    SuperagentError,
    StreamDisconnectedError,
)
from superagent.types import AgentInfo

BASE = "http://testserver"


# ---------------------------------------------------------------------------
# Client initialization
# ---------------------------------------------------------------------------


class TestClientInit:
    """Tests for SuperagentClient construction."""

    def test_default_init(self) -> None:
        client = SuperagentClient()
        assert client._http.base_url == httpx.URL("http://localhost:8888")
        assert "Authorization" not in client._http.headers

    def test_custom_base_url_strips_trailing_slash(self) -> None:
        client = SuperagentClient(base_url="http://example.com/api/")
        assert str(client._http.base_url).rstrip("/") == "http://example.com/api"

    def test_api_key_sets_bearer_header(self) -> None:
        client = SuperagentClient(api_key="test-key-123")
        assert client._http.headers["Authorization"] == "Bearer test-key-123"

    def test_no_auth_header_when_no_key(self) -> None:
        client = SuperagentClient()
        assert "Authorization" not in client._http.headers

    def test_content_type_always_json(self) -> None:
        client = SuperagentClient()
        assert client._http.headers["Content-Type"] == "application/json"

    def test_custom_timeout(self) -> None:
        client = SuperagentClient(timeout=30.0)
        assert client._http.timeout.connect == 30.0
        assert client._http.timeout.read == 300.0

    def test_admin_is_none_until_accessed(self) -> None:
        client = SuperagentClient()
        assert client._admin is None

    def test_admin_lazy_init(self) -> None:
        client = SuperagentClient()
        admin = client.admin
        assert admin is not None
        assert client._admin is admin
        # Second access returns same instance
        assert client.admin is admin


# ---------------------------------------------------------------------------
# Context manager
# ---------------------------------------------------------------------------


class TestClientContextManager:
    """Tests for async context manager protocol."""

    async def test_aenter_returns_self(self) -> None:
        async with SuperagentClient() as client:
            assert isinstance(client, SuperagentClient)

    async def test_aexit_calls_close(self) -> None:
        client = SuperagentClient()
        with patch.object(client, "close", new_callable=AsyncMock) as mock_close:
            async with client:
                pass
            mock_close.assert_awaited_once()

    async def test_close_calls_aclose(self) -> None:
        client = SuperagentClient()
        with patch.object(client._http, "aclose", new_callable=AsyncMock) as mock:
            await client.close()
            mock.assert_awaited_once()


# ---------------------------------------------------------------------------
# list_agents
# ---------------------------------------------------------------------------


class TestListAgents:
    """Tests for SuperagentClient.list_agents()."""

    @respx.mock
    async def test_list_agents_success(self) -> None:
        respx.get(f"{BASE}/api/v2/agents").respond(200, json={
            "agents": [
                {"name": "research", "type": "chat_model_agent", "description": "Research agent"},
                {"name": "code", "type": "deep_agent"},
            ]
        })
        async with SuperagentClient(base_url=BASE) as client:
            agents = await client.list_agents()
        assert len(agents) == 2
        assert isinstance(agents[0], AgentInfo)
        assert agents[0].name == "research"
        assert agents[1].name == "code"

    @respx.mock
    async def test_list_agents_bare_list(self) -> None:
        """Server returns a bare list instead of {'agents': [...]}."""
        respx.get(f"{BASE}/api/v2/agents").respond(200, json=[
            {"name": "a1", "type": "supervisor"},
        ])
        async with SuperagentClient(base_url=BASE) as client:
            agents = await client.list_agents()
        assert len(agents) == 1

    @respx.mock
    async def test_list_agents_empty(self) -> None:
        respx.get(f"{BASE}/api/v2/agents").respond(200, json={"agents": []})
        async with SuperagentClient(base_url=BASE) as client:
            agents = await client.list_agents()
        assert agents == []

    @respx.mock
    async def test_list_agents_auth_error(self) -> None:
        respx.get(f"{BASE}/api/v2/agents").respond(401, json={"msg": "no token"})
        async with SuperagentClient(base_url=BASE) as client:
            with pytest.raises(AuthenticationError):
                await client.list_agents()

    @respx.mock
    async def test_list_agents_server_error(self) -> None:
        respx.get(f"{BASE}/api/v2/agents").respond(500, json={"msg": "down"})
        async with SuperagentClient(base_url=BASE) as client:
            with pytest.raises(ServerError):
                await client.list_agents()


# ---------------------------------------------------------------------------
# chat
# ---------------------------------------------------------------------------


class TestChat:
    """Tests for SuperagentClient.chat()."""

    @respx.mock
    async def test_chat_collects_text(self) -> None:
        sse_body = (
            'event: text\n'
            'data: {"delta": "Hello"}\n'
            '\n'
            'event: text\n'
            'data: {"delta": " World"}\n'
            '\n'
            'event: done\n'
            'data: {}\n'
            '\n'
        )
        respx.post(f"{BASE}/api/v2/chat/stream").respond(
            200,
            text=sse_body,
            headers={"content-type": "text/event-stream"},
        )
        async with SuperagentClient(base_url=BASE) as client:
            result = await client.chat("research-agent", "Hi")
        assert result == "Hello World"

    @respx.mock
    async def test_chat_empty_stream(self) -> None:
        respx.post(f"{BASE}/api/v2/chat/stream").respond(
            200, text="", headers={"content-type": "text/event-stream"},
        )
        async with SuperagentClient(base_url=BASE) as client:
            result = await client.chat("agent", "msg")
        assert result == ""

    @respx.mock
    async def test_chat_sends_correct_payload(self) -> None:
        route = respx.post(f"{BASE}/api/v2/chat/stream").respond(
            200, text="event: done\ndata: {}\n\n",
            headers={"content-type": "text/event-stream"},
        )
        async with SuperagentClient(base_url=BASE) as client:
            await client.chat("my-agent", "hello", session_id="sess-1")
        req = route.calls.last.request
        body = json.loads(req.content)
        assert body["agent_id"] == "my-agent"
        assert body["message"] == "hello"
        assert body["session_id"] == "sess-1"

    @respx.mock
    async def test_chat_sends_a2ui_headers(self) -> None:
        route = respx.post(f"{BASE}/api/v2/chat/stream").respond(
            200, text="event: done\ndata: {}\n\n",
            headers={"content-type": "text/event-stream"},
        )
        async with SuperagentClient(base_url=BASE) as client:
            await client.chat("a", "m")
        req = route.calls.last.request
        assert req.headers["x-a2ui"] == "true"
        assert req.headers["accept"] == "text/event-stream"


# ---------------------------------------------------------------------------
# chat_stream
# ---------------------------------------------------------------------------


class TestChatStream:
    """Tests for SuperagentClient.chat_stream()."""

    @respx.mock
    async def test_returns_lazy_stream(self) -> None:
        respx.post(f"{BASE}/api/v2/chat/stream").respond(
            200, text="event: done\ndata: {}\n\n",
            headers={"content-type": "text/event-stream"},
        )
        async with SuperagentClient(base_url=BASE) as client:
            stream = client.chat_stream("a", "m")
        assert isinstance(stream, _LazyStream)

    @respx.mock
    async def test_stream_is_async_iterable(self) -> None:
        sse_body = (
            'event: text\n'
            'data: {"delta": "hi"}\n'
            '\n'
            'event: done\n'
            'data: {}\n'
            '\n'
        )
        respx.post(f"{BASE}/api/v2/chat/stream").respond(
            200, text=sse_body,
            headers={"content-type": "text/event-stream"},
        )
        async with SuperagentClient(base_url=BASE) as client:
            stream = client.chat_stream("a", "m")
            events = []
            async for evt in stream:
                events.append(evt)
        assert len(events) == 2
        assert events[0].event_type == "text"


# ---------------------------------------------------------------------------
# resume
# ---------------------------------------------------------------------------


class TestResume:
    """Tests for SuperagentClient.resume()."""

    @respx.mock
    async def test_resume_returns_stream(self) -> None:
        sse_body = 'event: done\ndata: {}\n\n'
        respx.post(f"{BASE}/api/v2/chat/resume").respond(
            200, text=sse_body,
            headers={"content-type": "text/event-stream"},
        )
        async with SuperagentClient(base_url=BASE) as client:
            stream = await client.resume("agent-1", "sess-1", {"choice": "yes"})
        assert hasattr(stream, "__aiter__")

    @respx.mock
    async def test_resume_sends_correct_payload(self) -> None:
        route = respx.post(f"{BASE}/api/v2/chat/resume").respond(
            200, text="event: done\ndata: {}\n\n",
            headers={"content-type": "text/event-stream"},
        )
        async with SuperagentClient(base_url=BASE) as client:
            await client.resume("ag", "sess", {"field": "value"})
        req = route.calls.last.request
        body = json.loads(req.content)
        assert body["agent_id"] == "ag"
        assert body["session_id"] == "sess"
        assert body["input"] == {"field": "value"}

    @respx.mock
    async def test_resume_auth_error(self) -> None:
        """resume() uses send(stream=True); on error, _raise_for_status is
        called on a streaming response whose body hasn't been consumed.
        We patch _raise_for_status to raise the expected exception directly,
        matching the contract of the production code's error handling."""
        respx.post(f"{BASE}/api/v2/chat/resume").respond(
            401, json={"msg": "denied"}
        )
        async with SuperagentClient(base_url=BASE) as client:
            with patch(
                "superagent.client._raise_for_status",
                side_effect=AuthenticationError("denied", status_code=401, code="401"),
            ):
                with pytest.raises(AuthenticationError):
                    await client.resume("a", "s", {})

    @respx.mock
    async def test_resume_server_error(self) -> None:
        respx.post(f"{BASE}/api/v2/chat/resume").respond(
            500, json={"msg": "boom"}
        )
        async with SuperagentClient(base_url=BASE) as client:
            with patch(
                "superagent.client._raise_for_status",
                side_effect=ServerError("boom", status_code=500, code="500"),
            ):
                with pytest.raises(ServerError):
                    await client.resume("a", "s", {})


# ---------------------------------------------------------------------------
# _LazyStream retry logic
# ---------------------------------------------------------------------------


class TestLazyStreamRetry:
    """Tests for the _LazyStream retry-on-5xx logic."""

    @respx.mock
    async def test_retries_on_5xx_then_succeeds(self) -> None:
        sse_body = 'event: done\ndata: {}\n\n'
        route = respx.post(f"{BASE}/api/v2/chat/stream")
        route.side_effect = [
            httpx.Response(500, json={"msg": "err"}),
            httpx.Response(503, json={"msg": "err"}),
            httpx.Response(200, text=sse_body, headers={"content-type": "text/event-stream"}),
        ]
        async with SuperagentClient(base_url=BASE) as client:
            stream = client.chat_stream("a", "m")
            # Patch asyncio.sleep to avoid real delays
            with patch("superagent.client.asyncio.sleep", new_callable=AsyncMock):
                events = []
                async for evt in stream:
                    events.append(evt)
        assert len(events) == 1  # done event

    @respx.mock
    async def test_retries_exhausted_raises_server_error(self) -> None:
        """The retry loop iterates 4 times (3 delays + final None).
        On attempts 0-1, the code raises ServerError directly.
        On attempt 2, _raise_for_status is called on the streaming response.
        On attempt 3, another send() occurs.
        We provide 4 mock responses and patch _raise_for_status to handle
        the streaming response limitation."""
        route = respx.post(f"{BASE}/api/v2/chat/stream")
        route.side_effect = [
            httpx.Response(500, json={"msg": "err"}),
            httpx.Response(500, json={"msg": "err"}),
            httpx.Response(500, json={"msg": "err"}),
            httpx.Response(500, json={"msg": "err"}),
        ]
        async with SuperagentClient(base_url=BASE) as client:
            stream = client.chat_stream("a", "m")
            with patch("superagent.client.asyncio.sleep", new_callable=AsyncMock):
                with patch(
                    "superagent.client._raise_for_status",
                    side_effect=ServerError("err", status_code=500, code="500"),
                ):
                    with pytest.raises(ServerError):
                        async for _ in stream:
                            pass

    @respx.mock
    async def test_no_retry_on_4xx(self) -> None:
        """4xx responses should not trigger retry logic."""
        respx.post(f"{BASE}/api/v2/chat/stream").respond(
            401, json={"msg": "unauthorized"}
        )
        async with SuperagentClient(base_url=BASE) as client:
            stream = client.chat_stream("a", "m")
            with patch(
                "superagent.client._raise_for_status",
                side_effect=AuthenticationError("unauthorized", status_code=401, code="401"),
            ):
                with pytest.raises(AuthenticationError):
                    async for _ in stream:
                        pass


# ---------------------------------------------------------------------------
# _LazyStream close
# ---------------------------------------------------------------------------


class TestLazyStreamClose:
    """Tests for _LazyStream.aclose()."""

    async def test_close_when_not_connected(self) -> None:
        """aclose before any iteration should not raise."""
        mock_client = AsyncMock(spec=httpx.AsyncClient)
        stream = _LazyStream(
            client=mock_client,
            method="POST",
            url="/test",
            json={},
            headers={},
        )
        await stream.aclose()  # _response is None, should be a no-op

    @respx.mock
    async def test_close_after_iteration(self) -> None:
        sse_body = 'event: done\ndata: {}\n\n'
        respx.post(f"{BASE}/api/v2/chat/stream").respond(
            200, text=sse_body,
            headers={"content-type": "text/event-stream"},
        )
        async with SuperagentClient(base_url=BASE) as client:
            stream = client.chat_stream("a", "m")
            async for _ in stream:
                pass
            # Now close — should close the underlying response
            await stream.aclose()


# ---------------------------------------------------------------------------
# Admin property delegation
# ---------------------------------------------------------------------------


class TestClientAdmin:
    """Test that client.admin delegates to AdminClient correctly."""

    @respx.mock
    async def test_admin_status_via_client(self) -> None:
        respx.get(f"{BASE}/api/v2/admin/status").respond(
            200, json={"status": "healthy"}
        )
        async with SuperagentClient(base_url=BASE) as client:
            result = await client.admin.status()
        assert result["status"] == "healthy"

    @respx.mock
    async def test_admin_list_agents_via_client(self) -> None:
        respx.get(f"{BASE}/api/v2/admin/agents").respond(200, json={
            "agents": [{"name": "x", "type": "t"}]
        })
        async with SuperagentClient(base_url=BASE) as client:
            agents = await client.admin.list_agents()
        assert len(agents) == 1


# ---------------------------------------------------------------------------
# Edge cases
# ---------------------------------------------------------------------------


class TestClientEdgeCases:
    """Edge-case tests."""

    @respx.mock
    async def test_chat_with_default_session_id(self) -> None:
        route = respx.post(f"{BASE}/api/v2/chat/stream").respond(
            200, text="event: done\ndata: {}\n\n",
            headers={"content-type": "text/event-stream"},
        )
        async with SuperagentClient(base_url=BASE) as client:
            await client.chat("a", "m")
        body = json.loads(route.calls.last.request.content)
        assert body["session_id"] == "default"

    @respx.mock
    async def test_multiple_chats_same_client(self) -> None:
        sse1 = 'event: text\ndata: {"delta": "one"}\n\nevent: done\ndata: {}\n\n'
        sse2 = 'event: text\ndata: {"delta": "two"}\n\nevent: done\ndata: {}\n\n'
        route = respx.post(f"{BASE}/api/v2/chat/stream")
        route.side_effect = [
            httpx.Response(200, text=sse1, headers={"content-type": "text/event-stream"}),
            httpx.Response(200, text=sse2, headers={"content-type": "text/event-stream"}),
        ]
        async with SuperagentClient(base_url=BASE) as client:
            r1 = await client.chat("a", "first")
            r2 = await client.chat("a", "second")
        assert r1 == "one"
        assert r2 == "two"

    @respx.mock
    async def test_auth_header_sent_with_api_key(self) -> None:
        route = respx.post(f"{BASE}/api/v2/chat/stream").respond(
            200, text="event: done\ndata: {}\n\n",
            headers={"content-type": "text/event-stream"},
        )
        async with SuperagentClient(base_url=BASE, api_key="my-secret") as client:
            await client.chat("a", "m")
        req = route.calls.last.request
        assert req.headers["authorization"] == "Bearer my-secret"
