from superagent.tools.builtin.web_search import WebSearch, web_search
from superagent.tools.builtin.http_request import HttpRequest, http_request
from superagent.tools.builtin.code_execute import CodeExecute, code_execute
from superagent.tools.mcp.client import MCPClient

# AgentScope ToolBase instances — used by ChatModelAgent._build_toolkit()
TOOL_REGISTRY: dict[str, type] = {
    "builtin/web_search": WebSearch,
    "builtin/http_request": HttpRequest,
    "builtin/code_execute": CodeExecute,
    "web_search": WebSearch,
    "http_request": HttpRequest,
    "code_execute": CodeExecute,
}

__all__ = [
    "web_search",
    "http_request",
    "code_execute",
    "WebSearch",
    "HttpRequest",
    "CodeExecute",
    "MCPClient",
    "TOOL_REGISTRY",
]
