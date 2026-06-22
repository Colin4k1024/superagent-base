"""builtin/http_request — HTTP request tool using httpx.

Provides both a plain async function (backward-compatible) and a
``ToolBase`` subclass (``HttpRequest``) for use with AgentScope 2.0 toolkits.
"""

from __future__ import annotations

import json
import logging
from typing import Any

import httpx
from agentscope.message import TextBlock
from agentscope.permission import PermissionBehavior, PermissionContext, PermissionDecision
from agentscope.tool import ToolBase, ToolChunk

logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Plain async function — backward-compatible, used by tests
# ---------------------------------------------------------------------------

async def http_request(
    url: str,
    method: str = "GET",
    headers: dict[str, str] | None = None,
    body: str | None = None,
    timeout: float = 30.0,
    **kwargs: Any,
) -> dict[str, Any]:
    """Make an HTTP request and return status, headers, and body.

    Supports GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS.
    Response body is truncated to 100 KB for safety.
    """
    try:
        async with httpx.AsyncClient(
            timeout=timeout,
            follow_redirects=True,
            verify=True,
        ) as client:
            resp = await client.request(
                method=method.upper(),
                url=url,
                headers=headers,
                content=body,
            )
            resp_text = resp.text[:100_000]
            return {
                "status_code": resp.status_code,
                "headers": dict(resp.headers),
                "body": resp_text,
            }
    except httpx.TimeoutException:
        logger.warning("HTTP request timed out: %s %s", method, url)
        return {
            "status_code": 0,
            "headers": {},
            "body": f"[timeout] Request to {url} timed out after {timeout}s",
        }
    except Exception as exc:
        logger.exception("HTTP request failed: %s %s", method, url)
        return {
            "status_code": 0,
            "headers": {},
            "body": f"[error] {exc}",
        }


# ---------------------------------------------------------------------------
# AgentScope 2.0 ToolBase subclass
# ---------------------------------------------------------------------------

class HttpRequest(ToolBase):
    """AgentScope tool for making HTTP requests."""

    name = "HttpRequest"
    description = "Make an HTTP request (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS) and return the response status, headers, and body."
    input_schema = {
        "type": "object",
        "properties": {
            "url": {
                "type": "string",
                "description": "The URL to send the request to.",
            },
            "method": {
                "type": "string",
                "description": "HTTP method (default GET).",
                "default": "GET",
            },
            "headers": {
                "type": "object",
                "description": "Optional request headers as key-value pairs.",
            },
            "body": {
                "type": "string",
                "description": "Optional request body string.",
            },
            "timeout": {
                "type": "number",
                "description": "Request timeout in seconds (default 30).",
                "default": 30.0,
            },
        },
        "required": ["url"],
    }
    is_concurrency_safe = True
    is_read_only = True

    async def check_permissions(
        self, tool_input: dict[str, Any], context: PermissionContext
    ) -> PermissionDecision:
        return PermissionDecision(behavior=PermissionBehavior.ALLOW, message="HTTP requests are allowed")

    async def __call__(
        self,
        url: str,
        method: str = "GET",
        headers: dict[str, str] | None = None,
        body: str | None = None,
        timeout: float = 30.0,
    ) -> ToolChunk:
        result = await http_request(url, method=method, headers=headers, body=body, timeout=timeout)
        return ToolChunk(content=[TextBlock(text=json.dumps(result, ensure_ascii=False, default=str))])
