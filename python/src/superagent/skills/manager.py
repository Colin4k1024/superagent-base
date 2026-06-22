"""Skill Manager — handles skill lifecycle (install, search, invoke).

Mirrors Go base ``pkg/skill.Manager`` with local cache and invoker delegation.
"""

from __future__ import annotations

import logging
import time
from dataclasses import dataclass, field
from typing import Any

from superagent.skills.invoker import CompositeInvoker, HTTPInvoker, LocalInvoker

logger = logging.getLogger(__name__)


@dataclass
class SkillMeta:
    """Metadata for a skill (mirrors Go ``skill.SkillMeta``)."""

    name: str
    description: str = ""
    version: str = "1.0.0"
    author: str = ""
    tags: list[str] = field(default_factory=list)
    installed_at: float = field(default_factory=time.time)


@dataclass
class SkillInstance:
    """An installed skill with its metadata."""

    meta: SkillMeta
    source: str = "local"  # "local", "http", "hub"


class SkillManager:
    """Manages skill lifecycle: install, search, list, invoke.

    Mirrors Go base ``skill.Manager`` with:
    - ``install(name, version)`` — register a skill
    - ``get_tool(name)`` — retrieve skill callable
    - ``list_installed()`` — list all installed skills
    - ``search(query, opts)`` — search available skills
    - ``register_local(meta, fn)`` — register a local skill function
    """

    def __init__(self) -> None:
        self._installed: dict[str, SkillInstance] = {}
        self._local_invoker = LocalInvoker()
        self._http_invoker = HTTPInvoker()
        self._composite = CompositeInvoker([self._local_invoker, self._http_invoker])

    @property
    def invoker(self) -> CompositeInvoker:
        """The composite invoker chain."""
        return self._composite

    @property
    def local_invoker(self) -> LocalInvoker:
        return self._local_invoker

    @property
    def http_invoker(self) -> HTTPInvoker:
        return self._http_invoker

    def register_local(self, meta: SkillMeta, fn: Any) -> None:
        """Register a local skill with metadata and callable."""
        self._local_invoker.register(meta.name, fn)
        self._installed[meta.name] = SkillInstance(meta=meta, source="local")
        logger.info("Registered local skill: %s v%s", meta.name, meta.version)

    def register_http(self, meta: SkillMeta, endpoint: str) -> None:
        """Register an HTTP-backed skill."""
        self._http_invoker.register_endpoint(meta.name, endpoint)
        self._installed[meta.name] = SkillInstance(meta=meta, source="http")
        logger.info("Registered HTTP skill: %s -> %s", meta.name, endpoint)

    async def install(self, name: str, version: str = "latest") -> None:
        """Install a skill by name (stub — registers a placeholder).

        In production this would fetch from SkillHub.
        """
        if name in self._installed:
            logger.warning("Skill %s already installed", name)
            return

        meta = SkillMeta(name=name, version=version)
        self._installed[name] = SkillInstance(meta=meta, source="hub")
        logger.info("Installed skill: %s v%s", name, version)

    def get_tool(self, name: str) -> Any | None:
        """Return the callable for a skill, or None."""
        if self._local_invoker.has_skill(name):
            return self._local_invoker
        if self._http_invoker.has_skill(name):
            return self._http_invoker
        return None

    def list_installed(self) -> list[SkillInstance]:
        """Return all installed skill instances."""
        return list(self._installed.values())

    async def search(self, query: str, limit: int = 20) -> list[SkillMeta]:
        """Search installed skills by name or description.

        In production, this would also query SkillHub.
        """
        query_lower = query.lower()
        results: list[SkillMeta] = []
        for inst in self._installed.values():
            meta = inst.meta
            if (
                query_lower in meta.name.lower()
                or query_lower in meta.description.lower()
                or any(query_lower in tag.lower() for tag in meta.tags)
            ):
                results.append(meta)
                if len(results) >= limit:
                    break
        return results

    async def invoke(self, skill_name: str, input_data: dict[str, Any]) -> dict[str, Any]:
        """Invoke a skill through the composite invoker chain."""
        return await self._composite.invoke(skill_name, input_data)

    def uninstall(self, name: str) -> bool:
        """Remove an installed skill. Returns True if it existed."""
        inst = self._installed.pop(name, None)
        if inst is None:
            return False
        # Note: we don't remove from invokers as they may have shared state
        logger.info("Uninstalled skill: %s", name)
        return True
