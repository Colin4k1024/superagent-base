"""Tool middleware chain — composable middleware for tool invocations.

Mirrors Go base ``pkg/tool/`` middleware pattern:
    RetryMiddleware, TimeoutMiddleware, RateLimitMiddleware,
    CacheMiddleware, LogMiddleware, Chain.
"""

from __future__ import annotations

import asyncio
import hashlib
import json
import logging
import time
from collections import defaultdict
from typing import Any, Callable, Awaitable

logger = logging.getLogger(__name__)

# Type aliases matching Go base
ToolInvoker = Callable[[str, dict[str, Any]], Awaitable[dict[str, Any]]]
Middleware = Callable[[ToolInvoker], ToolInvoker]


def chain(*middlewares: Middleware) -> Middleware:
    """Compose multiple middlewares into a single middleware.

    The first middleware in the list is the outermost (executes first).
    """
    def combined(invoker: ToolInvoker) -> ToolInvoker:
        for mw in reversed(middlewares):
            invoker = mw(invoker)
        return invoker
    return combined


def retry_middleware(max_retries: int = 3, backoff: float = 1.0) -> Middleware:
    """Retry failed tool calls with exponential backoff.

    Args:
        max_retries: Maximum number of retry attempts.
        backoff: Base backoff in seconds (doubled each retry).
    """
    def middleware(next_invoker: ToolInvoker) -> ToolInvoker:
        async def wrapper(name: str, args: dict[str, Any]) -> dict[str, Any]:
            last_exc: Exception | None = None
            for attempt in range(max_retries + 1):
                try:
                    return await next_invoker(name, args)
                except Exception as exc:
                    last_exc = exc
                    if attempt < max_retries:
                        delay = backoff * (2 ** attempt)
                        logger.warning(
                            "Tool %s attempt %d/%d failed: %s — retrying in %.1fs",
                            name, attempt + 1, max_retries, exc, delay,
                        )
                        await asyncio.sleep(delay)
            raise last_exc  # type: ignore[misc]
        return wrapper
    return middleware


def timeout_middleware(timeout: float = 30.0) -> Middleware:
    """Enforce a timeout on tool invocations.

    Args:
        timeout: Maximum seconds to wait for a tool call.
    """
    def middleware(next_invoker: ToolInvoker) -> ToolInvoker:
        async def wrapper(name: str, args: dict[str, Any]) -> dict[str, Any]:
            try:
                return await asyncio.wait_for(next_invoker(name, args), timeout=timeout)
            except asyncio.TimeoutError:
                raise TimeoutError(f"Tool {name} timed out after {timeout}s")
        return wrapper
    return middleware


def rate_limit_middleware(rpm: int = 60) -> Middleware:
    """Limit tool invocations per minute.

    Args:
        rpm: Maximum requests per minute across all tools.
    """
    timestamps: list[float] = []

    def middleware(next_invoker: ToolInvoker) -> ToolInvoker:
        async def wrapper(name: str, args: dict[str, Any]) -> dict[str, Any]:
            now = time.monotonic()
            # Purge entries older than 60 seconds
            cutoff = now - 60.0
            while timestamps and timestamps[0] < cutoff:
                timestamps.pop(0)

            if len(timestamps) >= rpm:
                wait = timestamps[0] + 60.0 - now
                if wait > 0:
                    logger.warning("Rate limit reached for tool %s — waiting %.1fs", name, wait)
                    await asyncio.sleep(wait)

            timestamps.append(time.monotonic())
            return await next_invoker(name, args)
        return wrapper
    return middleware


def cache_middleware(ttl: float = 300.0) -> Middleware:
    """Cache tool results by (name, args) for *ttl* seconds.

    Args:
        ttl: Time-to-live in seconds for cached results.
    """
    _cache: dict[str, tuple[float, dict[str, Any]]] = {}

    def _cache_key(name: str, args: dict[str, Any]) -> str:
        raw = json.dumps({"name": name, "args": args}, sort_keys=True, default=str)
        return hashlib.sha256(raw.encode()).hexdigest()

    def middleware(next_invoker: ToolInvoker) -> ToolInvoker:
        async def wrapper(name: str, args: dict[str, Any]) -> dict[str, Any]:
            key = _cache_key(name, args)
            now = time.monotonic()

            if key in _cache:
                ts, result = _cache[key]
                if now - ts < ttl:
                    logger.debug("Cache hit for tool %s", name)
                    return result

            result = await next_invoker(name, args)
            _cache[key] = (now, result)
            return result
        return wrapper
    return middleware


def log_middleware(logger_instance: logging.Logger | None = None) -> Middleware:
    """Log tool invocations with timing.

    Args:
        logger_instance: Logger to use (defaults to module logger).
    """
    log = logger_instance or logger

    def middleware(next_invoker: ToolInvoker) -> ToolInvoker:
        async def wrapper(name: str, args: dict[str, Any]) -> dict[str, Any]:
            start = time.monotonic()
            log.info("Tool %s invoked with args keys: %s", name, list(args.keys()))
            try:
                result = await next_invoker(name, args)
                elapsed = time.monotonic() - start
                log.info("Tool %s completed in %.3fs", name, elapsed)
                return result
            except Exception as exc:
                elapsed = time.monotonic() - start
                log.error("Tool %s failed after %.3fs: %s", name, elapsed, exc)
                raise
        return wrapper
    return middleware
