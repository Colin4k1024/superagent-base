"""Builtin in-memory backend for conversation history."""

from __future__ import annotations

import logging
import time
from typing import Any

from superagent.message import Msg

logger = logging.getLogger(__name__)


class BuiltinMemory:
    """In-memory conversation history backend.

    Stores messages in a dict of lists keyed by conversation_id.
    Supports both raw dict messages and typed Msg objects.
    Suitable for development and single-instance deployments.
    """

    def __init__(self, max_messages: int = 1000) -> None:
        self.max_messages = max_messages
        self._store: dict[str, list[dict[str, Any]]] = {}
        self._msg_store: dict[str, list[Msg]] = {}

    async def get(self, conversation_id: str) -> list[dict[str, Any]]:
        """Return message history for a conversation as raw dicts."""
        return list(self._store.get(conversation_id, []))

    async def get_msgs(self, conversation_id: str) -> list[Msg]:
        """Return message history for a conversation as Msg objects."""
        return list(self._msg_store.get(conversation_id, []))

    async def append(self, conversation_id: str, message: dict[str, Any]) -> None:
        """Append a raw dict message to the conversation history."""
        if conversation_id not in self._store:
            self._store[conversation_id] = []
        history = self._store[conversation_id]
        if "timestamp" not in message:
            message["timestamp"] = time.time()
        history.append(message)
        if len(history) > self.max_messages:
            self._store[conversation_id] = history[-self.max_messages:]

    async def append_msg(self, conversation_id: str, msg: Msg) -> None:
        """Append a typed Msg object to the conversation history.

        Also stores a serialized dict version for backward compatibility
        with endpoints that return raw dicts.
        """
        if conversation_id not in self._msg_store:
            self._msg_store[conversation_id] = []
        if conversation_id not in self._store:
            self._store[conversation_id] = []

        msg_history = self._msg_store[conversation_id]
        msg_history.append(msg)
        if len(msg_history) > self.max_messages:
            self._msg_store[conversation_id] = msg_history[-self.max_messages:]

        # Also store as dict for backward compatibility
        dict_history = self._store[conversation_id]
        dict_history.append(msg.to_dict())
        if len(dict_history) > self.max_messages:
            self._store[conversation_id] = dict_history[-self.max_messages:]

    async def clear(self, conversation_id: str) -> None:
        """Clear conversation history."""
        self._store.pop(conversation_id, None)
        self._msg_store.pop(conversation_id, None)

    async def list_conversations(self) -> list[str]:
        """Return all conversation IDs."""
        # Merge keys from both stores
        all_ids = set(self._store.keys()) | set(self._msg_store.keys())
        return list(all_ids)

    async def search(self, conversation_id: str, query: str, limit: int = 10) -> list[dict[str, Any]]:
        """Simple substring search within a conversation's messages."""
        history = self._store.get(conversation_id, [])
        query_lower = query.lower()
        results = [
            msg for msg in history
            if query_lower in str(msg.get("content", "")).lower()
        ]
        return results[-limit:]

    async def get_last_n(self, conversation_id: str, n: int = 10) -> list[dict[str, Any]]:
        """Return the last N messages for a conversation as raw dicts."""
        history = self._store.get(conversation_id, [])
        return list(history[-n:])

    async def get_last_n_msgs(self, conversation_id: str, n: int = 10) -> list[Msg]:
        """Return the last N messages for a conversation as Msg objects."""
        history = self._msg_store.get(conversation_id, [])
        return list(history[-n:])
