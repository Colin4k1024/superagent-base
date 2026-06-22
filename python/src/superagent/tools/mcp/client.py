"""MCP client — connects to MCP servers via stdio or SSE transport.

Wraps ``agentscope.mcp.MCPClient`` for SSE/HTTP transports, and provides
a lightweight stdio JSON-RPC implementation as fallback.
"""

from __future__ import annotations

import asyncio
import json
import logging
from typing import Any

logger = logging.getLogger(__name__)


class MCPClient:
    """Client for connecting to MCP (Model Context Protocol) servers.

    Supports two transports:
    - ``stdio`` — launches a subprocess and communicates via stdin/stdout
    - ``sse``  — connects to a remote SSE endpoint via AgentScope's MCPClient

    When ``transport="sse"``, the real ``agentscope.mcp.MCPClient`` is used
    internally.  For ``stdio``, a lightweight JSON-RPC implementation is used.
    """

    def __init__(
        self,
        server_name: str,
        transport: str = "stdio",
        endpoint: str = "",
        command: str = "",
        args: list[str] | None = None,
        env: dict[str, str] | None = None,
    ) -> None:
        self.server_name = server_name
        self.transport = transport
        self.endpoint = endpoint
        self.command = command
        self.args = args or []
        self.env = env or {}
        self._connected = False
        self._process: asyncio.subprocess.Process | None = None
        self._tools: list[dict[str, Any]] = []
        self._request_id = 0
        self._agentscope_mcp: Any = None  # agentscope.mcp.MCPClient instance

    async def connect(self) -> None:
        """Establish connection to the MCP server."""
        if self.transport == "stdio":
            await self._connect_stdio()
        elif self.transport == "sse":
            await self._connect_sse()
        else:
            raise ValueError(f"Unsupported transport: {self.transport}")
        self._connected = True
        logger.info("MCP client connected to %s via %s", self.server_name, self.transport)

    async def _connect_stdio(self) -> None:
        if not self.command:
            raise ValueError("command is required for stdio transport")
        cmd = [self.command] + self.args
        self._process = await asyncio.create_subprocess_exec(
            *cmd,
            stdin=asyncio.subprocess.PIPE,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
            env=self.env or None,
        )
        await self._send_request("initialize", {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "superagent-python", "version": "0.1.0"},
        })

    async def _connect_sse(self) -> None:
        """Connect using AgentScope's real MCPClient for HTTP/SSE transport."""
        if not self.endpoint:
            raise ValueError("endpoint is required for SSE transport")

        try:
            from agentscope.mcp import MCPClient as AgentScopeMCP
            from agentscope.mcp._config import HttpMCPConfig

            config = HttpMCPConfig(url=self.endpoint)
            self._agentscope_mcp = AgentScopeMCP(
                name=self.server_name,
                mcp_config=config,
                is_stateful=False,
            )
            await self._agentscope_mcp.connect()
            logger.info("AgentScope MCPClient connected to %s", self.endpoint)
        except ImportError:
            logger.warning("agentscope.mcp not available, using stub SSE connection")
        except Exception as exc:
            logger.error("Failed to connect AgentScope MCPClient: %s", exc)
            raise

    async def _send_request(self, method: str, params: dict[str, Any] | None = None) -> dict[str, Any]:
        self._request_id += 1
        request = {
            "jsonrpc": "2.0",
            "id": self._request_id,
            "method": method,
        }
        if params:
            request["params"] = params

        if self._process is None or self._process.stdin is None:
            raise RuntimeError("MCP process not started")

        payload = json.dumps(request) + "\n"
        self._process.stdin.write(payload.encode())
        await self._process.stdin.drain()

        if self._process.stdout is None:
            raise RuntimeError("MCP process stdout not available")

        line = await asyncio.wait_for(self._process.stdout.readline(), timeout=30.0)
        response = json.loads(line.decode().strip())
        if "error" in response:
            raise RuntimeError(f"MCP error: {response['error']}")
        return response.get("result", {})

    async def disconnect(self) -> None:
        """Close the connection."""
        if self._agentscope_mcp is not None:
            try:
                await self._agentscope_mcp.close()
            except Exception:
                pass
            self._agentscope_mcp = None

        if self._process is not None:
            try:
                self._process.terminate()
                await asyncio.wait_for(self._process.wait(), timeout=5.0)
            except Exception:
                self._process.kill()
            self._process = None
        self._connected = False

    async def list_tools(self) -> list[dict[str, Any]]:
        """Return available tools from the MCP server."""
        if self._agentscope_mcp is not None:
            try:
                tools = await self._agentscope_mcp.list_tools()
                return tools if isinstance(tools, list) else []
            except Exception as exc:
                logger.error("AgentScope MCP list_tools failed: %s", exc)
                return []

        if self._tools:
            return self._tools
        result = await self._send_request("tools/list")
        self._tools = result.get("tools", [])
        return self._tools

    async def call_tool(self, tool_name: str, arguments: dict[str, Any]) -> dict[str, Any]:
        """Invoke a tool on the MCP server."""
        if self._agentscope_mcp is not None:
            try:
                tool = self._agentscope_mcp.get_tool(tool_name)
                if tool is not None:
                    result = await tool(**arguments)
                    return {"content": str(result)}
            except Exception as exc:
                logger.error("AgentScope MCP call_tool failed: %s", exc)
                return {"error": str(exc)}

        result = await self._send_request("tools/call", {
            "name": tool_name,
            "arguments": arguments,
        })
        return result

    def to_agentscope_mcp(self) -> Any:
        """Return the underlying AgentScope MCPClient (if using SSE transport).

        Useful for registering with an AgentScope Toolkit:
        ``toolkit.register_mcp_client(mcp_client.to_agentscope_mcp())``
        """
        return self._agentscope_mcp

    @property
    def is_connected(self) -> bool:
        return self._connected
