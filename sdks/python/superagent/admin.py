"""Admin API client for Superagent."""

from __future__ import annotations

from typing import Any, Dict, List

import httpx

from .exceptions import SuperagentError
from .types import AgentInfo, ApplyResult, ValidateResult


class AdminClient:
    """Provides access to the Superagent admin API endpoints.

    Obtain an instance via :attr:`SuperagentClient.admin` rather than
    constructing this class directly.
    """

    def __init__(self, http: httpx.AsyncClient) -> None:
        self._http = http

    # ------------------------------------------------------------------
    # System
    # ------------------------------------------------------------------

    async def status(self) -> Dict[str, Any]:
        """Return system status.

        Calls ``GET /api/v2/admin/status``.
        """
        resp = await self._http.get("/api/v2/admin/status")
        _raise_for_status(resp)
        return resp.json()

    async def reload(self) -> Dict[str, Any]:
        """Hot-reload all agent YAML files.

        Calls ``POST /api/v2/admin/reload``.
        """
        resp = await self._http.post("/api/v2/admin/reload")
        _raise_for_status(resp)
        return resp.json()

    # ------------------------------------------------------------------
    # Agent CRUD
    # ------------------------------------------------------------------

    async def list_agents(self) -> List[AgentInfo]:
        """Return all registered agents.

        Calls ``GET /api/v2/admin/agents``.
        """
        resp = await self._http.get("/api/v2/admin/agents")
        _raise_for_status(resp)
        payload = resp.json()
        agents_raw = payload.get("agents", payload) if isinstance(payload, dict) else payload
        return [AgentInfo(**a) for a in agents_raw]

    async def get_agent(self, name: str) -> Dict[str, Any]:
        """Return full definition and raw YAML for a single agent.

        Calls ``GET /api/v2/admin/agents/:name``.
        """
        resp = await self._http.get(f"/api/v2/admin/agents/{name}")
        _raise_for_status(resp)
        return resp.json()

    async def create_agent(self, yaml_content: str) -> ApplyResult:
        """Create a new agent from YAML.

        Calls ``POST /api/v2/admin/agents``.

        Args:
            yaml_content: Full agent YAML definition string.
        """
        resp = await self._http.post(
            "/api/v2/admin/agents",
            json={"yaml": yaml_content},
        )
        _raise_for_status(resp)
        return ApplyResult(**resp.json())

    async def update_agent(self, name: str, yaml_content: str) -> ApplyResult:
        """Update an existing agent's YAML definition.

        Calls ``PUT /api/v2/admin/agents/:name``.

        Args:
            name: Agent name (must match ``metadata.name`` in the YAML).
            yaml_content: Updated agent YAML definition string.
        """
        resp = await self._http.put(
            f"/api/v2/admin/agents/{name}",
            json={"yaml": yaml_content},
        )
        _raise_for_status(resp)
        return ApplyResult(**resp.json())

    async def delete_agent(self, name: str) -> None:
        """Delete an agent by name.

        Calls ``DELETE /api/v2/admin/agents/:name``.
        """
        resp = await self._http.delete(f"/api/v2/admin/agents/{name}")
        _raise_for_status(resp)

    async def validate_agent(self, yaml_content: str) -> ValidateResult:
        """Validate an agent YAML without persisting it.

        Calls ``POST /api/v2/admin/agents/validate``.

        Args:
            yaml_content: Agent YAML definition string to validate.
        """
        resp = await self._http.post(
            "/api/v2/admin/agents/validate",
            json={"yaml": yaml_content},
        )
        _raise_for_status(resp)
        return ValidateResult(**resp.json())


# ---------------------------------------------------------------------------
# Internal helpers
# ---------------------------------------------------------------------------

def _raise_for_status(response: httpx.Response) -> None:
    """Raise the appropriate :class:`SuperagentError` subclass for non-2xx."""
    from .exceptions import (
        AuthenticationError,
        NotFoundError,
        RateLimitError,
        ServerError,
        ValidationError,
    )

    if response.is_success:
        return

    try:
        body = response.json()
        message = body.get("msg") or body.get("message") or response.text
        code = str(body.get("code", response.status_code))
    except Exception:
        message = response.text
        code = str(response.status_code)

    status = response.status_code
    if status in (401, 403):
        raise AuthenticationError(message, status_code=status, code=code)
    if status == 404:
        raise NotFoundError(message, status_code=status, code=code)
    if status == 422:
        raise ValidationError(message, status_code=status, code=code)
    if status == 429:
        raise RateLimitError(message, status_code=status, code=code)
    if status >= 500:
        raise ServerError(message, status_code=status, code=code)
    raise SuperagentError(message, status_code=status, code=code)
