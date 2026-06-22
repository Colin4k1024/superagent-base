"""Tests for FastAPI server endpoints."""

import pytest
from httpx import ASGITransport, AsyncClient

from superagent.server import app


@pytest.fixture
def client():
    """Create an async test client."""
    transport = ASGITransport(app=app)
    return AsyncClient(transport=transport, base_url="http://test")


@pytest.mark.asyncio
async def test_health(client):
    async with client as c:
        resp = await c.get("/health")
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "ok"


@pytest.mark.asyncio
async def test_metrics(client):
    async with client as c:
        resp = await c.get("/metrics")
        assert resp.status_code == 200


@pytest.mark.asyncio
async def test_list_agents(client):
    async with client as c:
        resp = await c.get("/api/v2/agents")
        assert resp.status_code == 200
        assert isinstance(resp.json(), list)


@pytest.mark.asyncio
async def test_list_conversations(client):
    async with client as c:
        resp = await c.get("/api/v2/conversations")
        assert resp.status_code == 200
        assert isinstance(resp.json(), list)


@pytest.mark.asyncio
async def test_admin_reload(client):
    async with client as c:
        resp = await c.post("/api/v2/admin/reload")
        assert resp.status_code == 200
        assert resp.json()["status"] == "ok"


@pytest.mark.asyncio
async def test_chat_stream_not_found(client):
    """Verify SSE streaming returns 404 for unknown agent."""
    async with client as c:
        resp = await c.post(
            "/api/v2/chat/stream",
            json={"agent_id": "nonexistent", "message": "hello"},
        )
        assert resp.status_code == 404


@pytest.mark.asyncio
async def test_chat_resume_not_found(client):
    """Verify resume returns 404 for unknown agent."""
    async with client as c:
        resp = await c.post(
            "/api/v2/chat/resume",
            json={"agent_id": "nonexistent", "session_id": "s1", "resume_payload": {}},
        )
        assert resp.status_code == 404
