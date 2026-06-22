"""builtin/web_search — web search via DuckDuckGo HTML scraping.

Provides both a plain async function (backward-compatible) and a
``ToolBase`` subclass (``WebSearch``) for use with AgentScope 2.0 toolkits.
"""

from __future__ import annotations

import json
import logging
import re
from typing import Any
from urllib.parse import quote_plus

import httpx
from agentscope.message import TextBlock
from agentscope.permission import PermissionBehavior, PermissionContext, PermissionDecision
from agentscope.tool import ToolBase, ToolChunk

logger = logging.getLogger(__name__)

_DDG_URL = "https://html.duckduckgo.com/html/"
_USER_AGENT = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
    "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)
_RESULT_RE = re.compile(
    r'<a[^>]+class="result__a"[^>]*href="([^"]*)"[^>]*>(.*?)</a>.*?'
    r'<a[^>]+class="result__snippet"[^>]*>(.*?)</a>',
    re.DOTALL,
)
_TAG_RE = re.compile(r"<[^>]+>")


def _strip_html(text: str) -> str:
    return _TAG_RE.sub("", text).strip()


# ---------------------------------------------------------------------------
# Plain async function — backward-compatible, used by tests
# ---------------------------------------------------------------------------

async def web_search(
    query: str,
    max_results: int = 5,
    **kwargs: Any,
) -> list[dict[str, str]]:
    """Search the web via DuckDuckGo HTML endpoint.

    Returns a list of ``{title, url, snippet}`` dicts.
    Falls back to an empty list on network errors.
    """
    try:
        async with httpx.AsyncClient(timeout=15.0, follow_redirects=True) as client:
            resp = await client.post(
                _DDG_URL,
                data={"q": query, "b": ""},
                headers={"User-Agent": _USER_AGENT},
            )
            resp.raise_for_status()
            html = resp.text
    except Exception:
        logger.exception("web_search request failed for query=%r", query)
        return []

    results: list[dict[str, str]] = []
    for match in _RESULT_RE.finditer(html):
        url, title, snippet = match.group(1), match.group(2), match.group(3)
        results.append({
            "title": _strip_html(title),
            "url": url,
            "snippet": _strip_html(snippet),
        })
        if len(results) >= max_results:
            break
    return results


# ---------------------------------------------------------------------------
# AgentScope 2.0 ToolBase subclass
# ---------------------------------------------------------------------------

class WebSearch(ToolBase):
    """AgentScope tool for web search via DuckDuckGo."""

    name = "WebSearch"
    description = "Search the web for information using DuckDuckGo. Returns a list of results with title, URL, and snippet."
    input_schema = {
        "type": "object",
        "properties": {
            "query": {
                "type": "string",
                "description": "The search query string.",
            },
            "max_results": {
                "type": "integer",
                "description": "Maximum number of results to return (default 5).",
                "default": 5,
            },
        },
        "required": ["query"],
    }
    is_concurrency_safe = True
    is_read_only = True

    async def check_permissions(
        self, tool_input: dict[str, Any], context: PermissionContext
    ) -> PermissionDecision:
        return PermissionDecision(behavior=PermissionBehavior.ALLOW, message="Web search is allowed")

    async def __call__(self, query: str, max_results: int = 5) -> ToolChunk:
        results = await web_search(query, max_results=max_results)
        return ToolChunk(content=[TextBlock(text=json.dumps(results, ensure_ascii=False))])
