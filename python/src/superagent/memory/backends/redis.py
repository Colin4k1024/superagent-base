"""Redis-backed memory backend for conversation history."""

from __future__ import annotations

import json
import logging
import time
from typing import Any

logger = logging.getLogger(__name__)


class RedisMemory:
    """Redis-backed conversation history backend using redis.asyncio.

    Stores each conversation as a JSON-serialized list under a prefixed key.
    Supports append, get, clear, search, and TTL-based expiration.
    """

    def __init__(
        self,
        redis_url: str = "redis://localhost:6379/0",
        prefix: str = "sa:conv:",
        ttl_seconds: int = 86400 * 7,
    ) -> None:
        self.redis_url = redis_url
        self.prefix = prefix
        self.ttl_seconds = ttl_seconds
        self._client: Any = None

    async def _ensure_client(self) -> Any:
        if self._client is None:
            import redis.asyncio as aioredis
            self._client = aioredis.from_url(self.redis_url, decode_responses=True)
        return self._client

    def _key(self, conversation_id: str) -> str:
        return f"{self.prefix}{conversation_id}"

    async def get(self, conversation_id: str) -> list[dict[str, Any]]:
        client = await self._ensure_client()
        raw = await client.get(self._key(conversation_id))
        return json.loads(raw) if raw else []

    async def append(self, conversation_id: str, message: dict[str, Any]) -> None:
        client = await self._ensure_client()
        key = self._key(conversation_id)
        history = await self.get(conversation_id)
        if "timestamp" not in message:
            message["timestamp"] = time.time()
        history.append(message)
        await client.set(key, json.dumps(history), ex=self.ttl_seconds)

    async def clear(self, conversation_id: str) -> None:
        client = await self._ensure_client()
        await client.delete(self._key(conversation_id))

    async def list_conversations(self) -> list[str]:
        client = await self._ensure_client()
        keys = []
        async for key in client.scan_iter(match=f"{self.prefix}*"):
            keys.append(key.removeprefix(self.prefix))
        return keys

    async def search(self, conversation_id: str, query: str, limit: int = 10) -> list[dict[str, Any]]:
        history = await self.get(conversation_id)
        query_lower = query.lower()
        results = [
            msg for msg in history
            if query_lower in str(msg.get("content", "")).lower()
        ]
        return results[-limit:]

    async def get_last_n(self, conversation_id: str, n: int = 10) -> list[dict[str, Any]]:
        history = await self.get(conversation_id)
        return history[-n:]
