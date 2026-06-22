"""Workspace configuration — manages runtime workspace state."""

from __future__ import annotations

import logging
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)


class Workspace:
    """Runtime workspace configuration.

    Manages the working directory, agent definitions path, and runtime state.
    """

    def __init__(
        self,
        root: str | Path = ".",
        agents_dir: str | Path = "configs/agents",
        tools_dir: str | Path | None = None,
    ) -> None:
        self.root = Path(root).resolve()
        self.agents_dir = Path(agents_dir)
        self.tools_dir = Path(tools_dir) if tools_dir else None
        logger.info("Workspace root: %s", self.root)

    @property
    def agents_path(self) -> Path:
        """Return the resolved agents directory path."""
        if self.agents_dir.is_absolute():
            return self.agents_dir
        return self.root / self.agents_dir

    def describe(self) -> dict[str, Any]:
        """Return workspace state as a dict."""
        return {
            "root": str(self.root),
            "agents_dir": str(self.agents_path),
            "tools_dir": str(self.tools_dir) if self.tools_dir else None,
        }
