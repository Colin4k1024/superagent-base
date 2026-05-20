"""Primary Superagent client."""

from __future__ import annotations

import asyncio
import logging
from typing import Any, Dict, List, Optional

import httpx

from .admin import AdminClient, _raise_for_status
from .exceptions import ServerError, StreamDisconnectedError
from .streaming import A2UIStream
from .types import AgentInfo

logger = logging.getLogger(__name__)

# Retry configuration for 5xx responses
_MAX_RETRIES = 3
_RETRY_DELAYS = (1.0, 2.0, 4.0)  # seconds — exponential back-off


class SuperagentClient:
    """Async HTTP client for the Superagent platform.

    Supports both streaming and non-streaming chat, agent administration,
    and interrupt/resume workflows.

    Usage (async context manager — recommended)::

        async with SuperagentClient(base_url="http://localhost:8888") as client:
            response = await client.chat("research-agent", "Hello!")
            print(response)

    Usage (manual lifecycle)::

        client = SuperagentClient()
        try:
            response = await client.chat("research-agent", "Hello!")
        finally:
            await client.close()

    Args:
        base_url: Base URL of the Superagent HTTP server.
        api_key: Optional API key sent as the ``Authorization: Bearer`` header.
        timeout: Request timeout in seconds (default 60 s; streaming uses 300 s).
    """

    def __init__(
        self,
        base_url: str = "http://localhost:8888",
        api_key: Optional[str] = None,
        timeout: float = 60.0,
    ) -> None:
        headers: Dict[str, str] = {"Content-Type": "application/json"}
        if api_key:
            headers["Authorization"] = f"Bearer {api_key}"

        self._http = httpx.AsyncClient(
            base_url=base_url.rstrip("/"),
            headers=headers,
            timeout=httpx.Timeout(timeout, read=300.0),
        )
        self._admin: Optional[AdminClient] = None

    # ------------------------------------------------------------------
    # Chat
    # ------------------------------------------------------------------

    async def chat(
        self,
        agent_id: str,
        message: str,
        session_id: str = "default",
    ) -> str:
        """Send a message and return the complete text response.

        Internally opens a streaming request and collects all ``text`` events.
        5xx responses are retried up to three times with exponential back-off.

        Args:
            agent_id: Name of the target agent.
            message: User message text.
            session_id: Conversation session identifier.

        Returns:
            The agent's complete text reply as a single string.
        """
        stream = self.chat_stream(agent_id, message, session_id)
        async with stream:
            return await stream.collect()

    def chat_stream(
        self,
        agent_id: str,
        message: str,
        session_id: str = "default",
    ) -> A2UIStream:
        """Return a lazy :class:`~superagent.streaming.A2UIStream`.

        The HTTP request is not sent until the stream is iterated.  Use
        ``async for`` or :meth:`~superagent.streaming.A2UIStream.collect`
        to consume events.

        Calls ``POST /api/v1/chat/stream`` with the ``X-A2UI: true`` header.

        Args:
            agent_id: Name of the target agent.
            message: User message text.
            session_id: Conversation session identifier.
        """
        return _LazyStream(
            client=self._http,
            method="POST",
            url="/api/v1/chat/stream",
            json={
                "agent_id": agent_id,
                "message": message,
                "session_id": session_id,
            },
            headers={"X-A2UI": "true", "Accept": "text/event-stream"},
        )

    async def resume(
        self,
        agent_id: str,
        session_id: str,
        input: Dict[str, Any],
    ) -> A2UIStream:
        """Resume a paused agent after an interrupt.

        Calls ``POST /api/v1/chat/resume``.

        Args:
            agent_id: Name of the agent that raised the interrupt.
            session_id: Session that is in the interrupted state.
            input: User-supplied form values to satisfy the interrupt fields.

        Returns:
            An :class:`~superagent.streaming.A2UIStream` for the resumed run.
        """
        req = self._http.build_request(
            "POST",
            "/api/v1/chat/resume",
            json={"agent_id": agent_id, "session_id": session_id, "input": input},
            headers={"X-A2UI": "true", "Accept": "text/event-stream"},
        )
        response = await self._http.send(req, stream=True)
        _raise_for_status(response)
        return A2UIStream(response)

    # ------------------------------------------------------------------
    # Agents
    # ------------------------------------------------------------------

    async def list_agents(self) -> List[AgentInfo]:
        """Return all live agents exposed by the server.

        Calls ``GET /api/v1/agents``.
        """
        resp = await self._http.get("/api/v1/agents")
        _raise_for_status(resp)
        payload = resp.json()
        agents_raw = payload.get("agents", payload) if isinstance(payload, dict) else payload
        return [AgentInfo(**a) for a in agents_raw]

    # ------------------------------------------------------------------
    # Admin
    # ------------------------------------------------------------------

    @property
    def admin(self) -> AdminClient:
        """Lazily-initialised :class:`~superagent.admin.AdminClient`."""
        if self._admin is None:
            self._admin = AdminClient(self._http)
        return self._admin

    # ------------------------------------------------------------------
    # Lifecycle
    # ------------------------------------------------------------------

    async def close(self) -> None:
        """Close the underlying HTTP connection pool."""
        await self._http.aclose()

    async def __aenter__(self) -> "SuperagentClient":
        return self

    async def __aexit__(self, *_: Any) -> None:
        await self.close()


# ---------------------------------------------------------------------------
# Internal: lazy stream that performs the request on first iteration
# ---------------------------------------------------------------------------

class _LazyStream(A2UIStream):
    """A2UIStream whose HTTP request is deferred until iteration begins.

    This allows ``client.chat_stream(...)`` to return synchronously while
    the actual connection is made lazily, consistent with the httpx streaming
    pattern.  5xx responses trigger up to three retries before raising.
    """

    def __init__(
        self,
        client: httpx.AsyncClient,
        method: str,
        url: str,
        json: Dict[str, Any],
        headers: Dict[str, str],
    ) -> None:
        # We cannot call super().__init__ yet because we have no response.
        # We store the parameters and initialise properly on first iteration.
        self._client = client
        self._method = method
        self._url = url
        self._json = json
        self._extra_headers = headers
        self._response: Optional[httpx.Response] = None  # type: ignore[assignment]
        self._done = False
        self._callbacks: Dict[str, List[Any]] = {}

    async def _ensure_connected(self) -> None:
        if self._response is not None:
            return

        last_exc: Optional[Exception] = None
        for attempt, delay in enumerate((*_RETRY_DELAYS, None)):
            try:
                req = self._client.build_request(
                    self._method,
                    self._url,
                    json=self._json,
                    headers=self._extra_headers,
                )
                resp = await self._client.send(req, stream=True)
                if resp.status_code >= 500 and attempt < _MAX_RETRIES - 1:
                    await resp.aclose()
                    raise ServerError(
                        f"Server returned {resp.status_code}",
                        status_code=resp.status_code,
                    )
                _raise_for_status(resp)
                self._response = resp
                self._iter = resp.aiter_lines()
                return
            except ServerError as exc:
                last_exc = exc
                if delay is not None:
                    logger.warning(
                        "Server error on attempt %d/%d, retrying in %.0fs…",
                        attempt + 1,
                        _MAX_RETRIES,
                        delay,
                    )
                    await asyncio.sleep(delay)

        raise last_exc  # type: ignore[misc]

    async def __anext__(self) -> Any:
        await self._ensure_connected()
        return await super().__anext__()

    async def aclose(self) -> None:
        if self._response is not None:
            await self._response.aclose()
