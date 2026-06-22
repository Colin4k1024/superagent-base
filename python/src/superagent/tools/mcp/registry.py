"""MCP Registry — manages named MCP client connections.

Mirrors the Go base ``pkg/mcp.Registry`` which maps server names to
``MCPClient`` instances supporting stdio and SSE transports.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass, field
from typing import Any

from superagent.tools.mcp.client import MCPClient

logger = logging.getLogger(__name__)


@dataclass
class MCPServerConfig:
    """Configuration for an MCP server connection."""

    name: str
    transport: str = "stdio"  # "stdio" or "sse"
    endpoint: str = ""  # URL for SSE transport
    command: str = ""  # executable for stdio transport
    args: list[str] = field(default_factory=list)
    env: dict[str, str] = field(default_factory=dict)


class MCPRegistry:
    """Registry that manages named MCP client connections.

    Mirrors Go base ``mcp.Registry``:
    - ``connect(cfg)`` — create and connect a client for the given config
    - ``get_client(name)`` — retrieve a connected client by server name
    - ``list_servers()`` — return names of all registered servers
    """

    def __init__(self) -> None:
        self._clients: dict[str, MCPClient] = {}

    async def connect(self, config: MCPServerConfig) -> None:
        """Create an MCPClient from *config*, connect it, and register it."""
        if config.name in self._clients:
            logger.warning("MCP server %s already registered, disconnecting old client", config.name)
            await self._clients[config.name].disconnect()

        client = MCPClient(
            server_name=config.name,
            transport=config.transport,
            endpoint=config.endpoint,
            command=config.command,
            args=config.args,
            env=config.env,
        )
        await client.connect()
        self._clients[config.name] = client
        logger.info("Registered MCP server %s (%s)", config.name, config.transport)

    def get_client(self, name: str) -> MCPClient | None:
        """Return the connected client for *name*, or ``None``."""
        return self._clients.get(name)

    def list_servers(self) -> list[str]:
        """Return names of all registered MCP servers."""
        return list(self._clients.keys())

    async def disconnect_all(self) -> None:
        """Disconnect and remove all clients."""
        for name, client in list(self._clients.items()):
            try:
                await client.disconnect()
            except Exception:
                logger.exception("Error disconnecting MCP server %s", name)
        self._clients.clear()

    async def disconnect(self, name: str) -> bool:
        """Disconnect a specific server. Returns True if it existed."""
        client = self._clients.pop(name, None)
        if client is None:
            return False
        await client.disconnect()
        return True

    def __len__(self) -> int:
        return len(self._clients)

    def __contains__(self, name: str) -> bool:
        return name in self._clients
