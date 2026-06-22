"""Signal collector — captures agent execution signals for evolution (stub)."""

from __future__ import annotations

import logging
from typing import Any

logger = logging.getLogger(__name__)


class SignalCollector:
    """Collects signals from agent runs for experience replay.

    Signals include:
    - Task success/failure
    - Tool call outcomes
    - Model latency and token usage
    - User feedback

    Stub: logs signals but does not persist them.
    """

    def __init__(self) -> None:
        self._signals: list[dict[str, Any]] = []

    def record(self, signal_type: str, payload: dict[str, Any]) -> None:
        """Record a signal."""
        signal = {"type": signal_type, **payload}
        self._signals.append(signal)
        logger.debug("Recorded signal: %s", signal_type)

    def get_signals(self, signal_type: str | None = None) -> list[dict[str, Any]]:
        """Return collected signals, optionally filtered by type."""
        if signal_type is None:
            return list(self._signals)
        return [s for s in self._signals if s.get("type") == signal_type]

    def clear(self) -> None:
        """Clear all collected signals."""
        self._signals.clear()
