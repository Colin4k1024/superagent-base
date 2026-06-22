"""Context injection middleware — prepends context messages to agent prompts.

Mirrors Go base ``pkg/agentdef/mw_context_injection.go`` which injects
timestamp, session metadata, and static context into the message list
before sending to the model.
"""

from __future__ import annotations

import time
from dataclasses import dataclass, field
from typing import Any


@dataclass
class ContextInjectionMiddleware:
    """Injects contextual information into agent message lists.

    Attributes:
        inject_timestamp: Prepend current UTC timestamp.
        inject_session_metadata: Prepend session-level metadata (session_id, etc.).
        static_context: Static text injected into every request.
    """

    inject_timestamp: bool = False
    inject_session_metadata: bool = False
    static_context: str = ""

    def build_context_messages(
        self,
        session_id: str = "",
        extra_metadata: dict[str, Any] | None = None,
    ) -> list[dict[str, str]]:
        """Build a list of system messages to prepend.

        Returns a list of ``{"role": "system", "content": "..."}`` messages.
        """
        parts: list[str] = []

        if self.static_context:
            parts.append(self.static_context)

        if self.inject_timestamp:
            ts = time.strftime("%Y-%m-%d %H:%M:%S UTC", time.gmtime())
            parts.append(f"Current timestamp: {ts}")

        if self.inject_session_metadata and session_id:
            parts.append(f"Session ID: {session_id}")
            if extra_metadata:
                for k, v in extra_metadata.items():
                    parts.append(f"{k}: {v}")

        if not parts:
            return []

        return [{"role": "system", "content": "\n".join(parts)}]

    def inject(
        self,
        messages: list[dict[str, str]],
        session_id: str = "",
        extra_metadata: dict[str, Any] | None = None,
    ) -> list[dict[str, str]]:
        """Prepend context messages to the given message list.

        Returns a new list with context messages at the beginning.
        """
        context = self.build_context_messages(session_id, extra_metadata)
        return context + list(messages)
