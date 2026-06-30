"""Tests for superagent.admin module."""

import httpx
import pytest
import respx

from superagent.admin import AdminClient, _raise_for_status
from superagent.exceptions import (
    AuthenticationError,
    NotFoundError,
    RateLimitError,
    ServerError,
    SuperagentError,
    ValidationError,
)
from superagent.types import AgentInfo, ApplyResult, ValidateResult


BASE = "http://testserver"


# ---------------------------------------------------------------------------
# _raise_for_status
# ---------------------------------------------------------------------------


class TestRaiseForStatus:
    """Tests for the _raise_for_status helper function."""

    def test_success_response_does_not_raise(self) -> None:
        resp = httpx.Response(200, json={"ok": True})
        _raise_for_status(resp)  # should not raise

    def test_201_does_not_raise(self) -> None:
        resp = httpx.Response(201, json={"ok": True})
        _raise_for_status(resp)

    def test_401_raises_auth_error(self) -> None:
        resp = httpx.Response(401, json={"msg": "unauthorized"})
        with pytest.raises(AuthenticationError) as exc_info:
            _raise_for_status(resp)
        assert exc_info.value.status_code == 401
        assert "unauthorized" in exc_info.value.message

    def test_403_raises_auth_error(self) -> None:
        resp = httpx.Response(403, json={"msg": "forbidden"})
        with pytest.raises(AuthenticationError) as exc_info:
            _raise_for_status(resp)
        assert exc_info.value.status_code == 403

    def test_404_raises_not_found(self) -> None:
        resp = httpx.Response(404, json={"msg": "not found"})
        with pytest.raises(NotFoundError) as exc_info:
            _raise_for_status(resp)
        assert exc_info.value.status_code == 404

    def test_422_raises_validation(self) -> None:
        resp = httpx.Response(422, json={"msg": "invalid"})
        with pytest.raises(ValidationError) as exc_info:
            _raise_for_status(resp)
        assert exc_info.value.status_code == 422

    def test_429_raises_rate_limit(self) -> None:
        resp = httpx.Response(429, json={"msg": "too many"})
        with pytest.raises(RateLimitError) as exc_info:
            _raise_for_status(resp)
        assert exc_info.value.status_code == 429

    def test_500_raises_server_error(self) -> None:
        resp = httpx.Response(500, json={"msg": "internal"})
        with pytest.raises(ServerError) as exc_info:
            _raise_for_status(resp)
        assert exc_info.value.status_code == 500

    def test_503_raises_server_error(self) -> None:
        resp = httpx.Response(503, json={"msg": "unavailable"})
        with pytest.raises(ServerError) as exc_info:
            _raise_for_status(resp)
        assert exc_info.value.status_code == 503

    def test_other_4xx_raises_base_error(self) -> None:
        resp = httpx.Response(418, json={"msg": "teapot"})
        with pytest.raises(SuperagentError) as exc_info:
            _raise_for_status(resp)
        assert exc_info.value.status_code == 418

    def test_fallback_to_message_key(self) -> None:
        """Uses 'message' key if 'msg' is absent."""
        resp = httpx.Response(500, json={"message": "something broke"})
        with pytest.raises(ServerError) as exc_info:
            _raise_for_status(resp)
        assert "something broke" in exc_info.value.message

    def test_fallback_to_text_when_no_json(self) -> None:
        """Non-JSON body falls back to response.text."""
        resp = httpx.Response(500, text="raw error text")
        with pytest.raises(ServerError) as exc_info:
            _raise_for_status(resp)
        assert "raw error text" in exc_info.value.message

    def test_code_field_from_body(self) -> None:
        resp = httpx.Response(500, json={"msg": "err", "code": "ERR_INTERNAL"})
        with pytest.raises(ServerError) as exc_info:
            _raise_for_status(resp)
        assert exc_info.value.code == "ERR_INTERNAL"

    def test_code_falls_back_to_status(self) -> None:
        resp = httpx.Response(500, json={"msg": "err"})
        with pytest.raises(ServerError) as exc_info:
            _raise_for_status(resp)
        assert exc_info.value.code == "500"


# ---------------------------------------------------------------------------
# AdminClient
# ---------------------------------------------------------------------------


@pytest.fixture
def http() -> httpx.AsyncClient:
    return httpx.AsyncClient(base_url=BASE)


@pytest.fixture
def admin(http: httpx.AsyncClient) -> AdminClient:
    return AdminClient(http)


class TestAdminStatus:
    """Tests for AdminClient.status()."""

    @respx.mock
    async def test_status_success(self, admin: AdminClient) -> None:
        respx.get(f"{BASE}/api/v2/admin/status").respond(
            200, json={"status": "ok", "agents": 3}
        )
        result = await admin.status()
        assert result["status"] == "ok"
        assert result["agents"] == 3

    @respx.mock
    async def test_status_server_error(self, admin: AdminClient) -> None:
        respx.get(f"{BASE}/api/v2/admin/status").respond(
            500, json={"msg": "down"}
        )
        with pytest.raises(ServerError):
            await admin.status()


class TestAdminReload:
    """Tests for AdminClient.reload()."""

    @respx.mock
    async def test_reload_success(self, admin: AdminClient) -> None:
        respx.post(f"{BASE}/api/v2/admin/reload").respond(
            200, json={"reloaded": 5}
        )
        result = await admin.reload()
        assert result["reloaded"] == 5

    @respx.mock
    async def test_reload_server_error(self, admin: AdminClient) -> None:
        respx.post(f"{BASE}/api/v2/admin/reload").respond(
            500, json={"msg": "reload failed"}
        )
        with pytest.raises(ServerError):
            await admin.reload()


class TestAdminListAgents:
    """Tests for AdminClient.list_agents()."""

    @respx.mock
    async def test_list_agents_from_dict(self, admin: AdminClient) -> None:
        respx.get(f"{BASE}/api/v2/admin/agents").respond(200, json={
            "agents": [
                {"name": "a1", "type": "chat_model_agent"},
                {"name": "a2", "type": "supervisor"},
            ]
        })
        agents = await admin.list_agents()
        assert len(agents) == 2
        assert isinstance(agents[0], AgentInfo)
        assert agents[0].name == "a1"
        assert agents[1].name == "a2"

    @respx.mock
    async def test_list_agents_from_list(self, admin: AdminClient) -> None:
        """Server returns a bare list instead of {'agents': [...]}."""
        respx.get(f"{BASE}/api/v2/admin/agents").respond(200, json=[
            {"name": "b1", "type": "deep_agent"},
        ])
        agents = await admin.list_agents()
        assert len(agents) == 1
        assert agents[0].name == "b1"

    @respx.mock
    async def test_list_agents_empty(self, admin: AdminClient) -> None:
        respx.get(f"{BASE}/api/v2/admin/agents").respond(200, json={"agents": []})
        agents = await admin.list_agents()
        assert agents == []

    @respx.mock
    async def test_list_agents_auth_error(self, admin: AdminClient) -> None:
        respx.get(f"{BASE}/api/v2/admin/agents").respond(
            401, json={"msg": "no token"}
        )
        with pytest.raises(AuthenticationError):
            await admin.list_agents()


class TestAdminGetAgent:
    """Tests for AdminClient.get_agent()."""

    @respx.mock
    async def test_get_agent_success(self, admin: AdminClient) -> None:
        respx.get(f"{BASE}/api/v2/admin/agents/my-agent").respond(200, json={
            "name": "my-agent",
            "type": "chat_model_agent",
            "yaml": "apiVersion: v1\nkind: Agent",
        })
        result = await admin.get_agent("my-agent")
        assert result["name"] == "my-agent"
        assert "yaml" in result

    @respx.mock
    async def test_get_agent_not_found(self, admin: AdminClient) -> None:
        respx.get(f"{BASE}/api/v2/admin/agents/ghost").respond(
            404, json={"msg": "not found"}
        )
        with pytest.raises(NotFoundError):
            await admin.get_agent("ghost")


class TestAdminCreateAgent:
    """Tests for AdminClient.create_agent()."""

    @respx.mock
    async def test_create_agent_success(self, admin: AdminClient) -> None:
        respx.post(f"{BASE}/api/v2/admin/agents").respond(200, json={
            "name": "new-agent",
            "status": "created",
            "message": "OK",
        })
        result = await admin.create_agent("apiVersion: v1\nkind: Agent")
        assert isinstance(result, ApplyResult)
        assert result.name == "new-agent"
        assert result.status == "created"

    @respx.mock
    async def test_create_agent_validation_error(self, admin: AdminClient) -> None:
        respx.post(f"{BASE}/api/v2/admin/agents").respond(
            422, json={"msg": "invalid yaml"}
        )
        with pytest.raises(ValidationError, match="invalid yaml"):
            await admin.create_agent("bad yaml")

    @respx.mock
    async def test_create_agent_sends_yaml(self, admin: AdminClient) -> None:
        route = respx.post(f"{BASE}/api/v2/admin/agents").respond(
            200, json={"name": "x", "status": "created"}
        )
        await admin.create_agent("my yaml content")
        req = route.calls.last.request
        import json
        body = json.loads(req.content)
        assert body["yaml"] == "my yaml content"


class TestAdminUpdateAgent:
    """Tests for AdminClient.update_agent()."""

    @respx.mock
    async def test_update_success(self, admin: AdminClient) -> None:
        respx.put(f"{BASE}/api/v2/admin/agents/existing").respond(200, json={
            "name": "existing",
            "status": "updated",
            "message": "changed",
        })
        result = await admin.update_agent("existing", "new yaml")
        assert isinstance(result, ApplyResult)
        assert result.status == "updated"

    @respx.mock
    async def test_update_not_found(self, admin: AdminClient) -> None:
        respx.put(f"{BASE}/api/v2/admin/agents/ghost").respond(
            404, json={"msg": "not found"}
        )
        with pytest.raises(NotFoundError):
            await admin.update_agent("ghost", "yaml")


class TestAdminDeleteAgent:
    """Tests for AdminClient.delete_agent()."""

    @respx.mock
    async def test_delete_success(self, admin: AdminClient) -> None:
        respx.delete(f"{BASE}/api/v2/admin/agents/to-delete").respond(200)
        await admin.delete_agent("to-delete")  # should not raise

    @respx.mock
    async def test_delete_not_found(self, admin: AdminClient) -> None:
        respx.delete(f"{BASE}/api/v2/admin/agents/ghost").respond(
            404, json={"msg": "not found"}
        )
        with pytest.raises(NotFoundError):
            await admin.delete_agent("ghost")

    @respx.mock
    async def test_delete_server_error(self, admin: AdminClient) -> None:
        respx.delete(f"{BASE}/api/v2/admin/agents/x").respond(
            500, json={"msg": "cannot delete"}
        )
        with pytest.raises(ServerError):
            await admin.delete_agent("x")


class TestAdminValidateAgent:
    """Tests for AdminClient.validate_agent()."""

    @respx.mock
    async def test_validate_valid(self, admin: AdminClient) -> None:
        respx.post(f"{BASE}/api/v2/admin/agents/validate").respond(200, json={
            "valid": True,
            "errors": [],
            "warnings": [],
        })
        result = await admin.validate_agent("good yaml")
        assert isinstance(result, ValidateResult)
        assert result.valid is True

    @respx.mock
    async def test_validate_invalid(self, admin: AdminClient) -> None:
        respx.post(f"{BASE}/api/v2/admin/agents/validate").respond(200, json={
            "valid": False,
            "errors": ["missing metadata.name"],
            "warnings": ["deprecated field"],
        })
        result = await admin.validate_agent("bad yaml")
        assert result.valid is False
        assert "missing metadata.name" in result.errors

    @respx.mock
    async def test_validate_server_error(self, admin: AdminClient) -> None:
        respx.post(f"{BASE}/api/v2/admin/agents/validate").respond(
            500, json={"msg": "internal"}
        )
        with pytest.raises(ServerError):
            await admin.validate_agent("yaml")
