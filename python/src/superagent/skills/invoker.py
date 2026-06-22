"""Skill invokers — execute skills via local functions, HTTP, or composite chains.

Mirrors Go base ``pkg/skill.LocalInvoker``, ``HTTPInvoker``, and composite patterns.
"""

from __future__ import annotations

import logging
from abc import ABC, abstractmethod
from typing import Any, Callable, Protocol

import httpx

logger = logging.getLogger(__name__)

# Type alias matching Go's SkillFunc
SkillFunc = Callable[..., Any]


class Invoker(Protocol):
    """Protocol for skill invokers."""

    async def invoke(self, skill_name: str, input_data: dict[str, Any]) -> dict[str, Any]: ...


class LocalInvoker:
    """Invokes skills via locally registered Python callables.

    Mirrors Go base ``skill.LocalInvoker``.
    """

    def __init__(self) -> None:
        self._skills: dict[str, SkillFunc] = {}

    def register(self, name: str, fn: SkillFunc) -> None:
        """Register a local skill function."""
        self._skills[name] = fn
        logger.debug("Registered local skill: %s", name)

    def has_skill(self, name: str) -> bool:
        return name in self._skills

    def list_skills(self) -> list[str]:
        return list(self._skills.keys())

    async def invoke(self, skill_name: str, input_data: dict[str, Any]) -> dict[str, Any]:
        """Invoke a locally registered skill."""
        fn = self._skills.get(skill_name)
        if fn is None:
            raise KeyError(f"Local skill not found: {skill_name}")

        import asyncio
        if asyncio.iscoroutinefunction(fn):
            result = await fn(**input_data)
        else:
            result = fn(**input_data)

        if isinstance(result, dict):
            return result
        return {"result": result}


class HTTPInvoker:
    """Invokes skills via HTTP REST endpoints.

    Mirrors Go base ``skill.HTTPInvoker``.
    """

    def __init__(self, timeout: float = 30.0) -> None:
        self._endpoints: dict[str, str] = {}
        self._timeout = timeout

    def register_endpoint(self, skill_name: str, endpoint: str) -> None:
        """Register an HTTP endpoint for a skill."""
        self._endpoints[skill_name] = endpoint
        logger.debug("Registered HTTP skill endpoint: %s -> %s", skill_name, endpoint)

    def has_skill(self, name: str) -> bool:
        return name in self._endpoints

    def list_skills(self) -> list[str]:
        return list(self._endpoints.keys())

    async def invoke(self, skill_name: str, input_data: dict[str, Any]) -> dict[str, Any]:
        """Invoke a skill via its HTTP endpoint."""
        endpoint = self._endpoints.get(skill_name)
        if endpoint is None:
            raise KeyError(f"HTTP skill endpoint not found: {skill_name}")

        async with httpx.AsyncClient(timeout=self._timeout) as client:
            resp = await client.post(endpoint, json=input_data)
            resp.raise_for_status()
            return resp.json()


class CompositeInvoker:
    """Chains multiple invokers — tries local first, then HTTP.

    Mirrors Go base composite invoker pattern.
    """

    def __init__(self, invokers: list[Any] | None = None) -> None:
        self._invokers: list[Any] = invokers or []

    def add_invoker(self, invoker: Any) -> None:
        """Add an invoker to the chain."""
        self._invokers.append(invoker)

    async def invoke(self, skill_name: str, input_data: dict[str, Any]) -> dict[str, Any]:
        """Try each invoker in order; return first successful result."""
        last_error: Exception | None = None
        for invoker in self._invokers:
            if hasattr(invoker, "has_skill") and not invoker.has_skill(skill_name):
                continue
            try:
                return await invoker.invoke(skill_name, input_data)
            except (KeyError, Exception) as exc:
                last_error = exc
                logger.debug("Invoker %s failed for %s: %s", type(invoker).__name__, skill_name, exc)
                continue

        if last_error:
            raise last_error
        raise KeyError(f"No invoker found for skill: {skill_name}")
