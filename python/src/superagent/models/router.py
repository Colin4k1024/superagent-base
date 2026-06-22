"""Model routing strategies — select the best model for a given task."""

from __future__ import annotations

import logging
import random
from enum import Enum
from typing import Any

from superagent.models.registry import ModelConfig, ModelRegistry

logger = logging.getLogger(__name__)


class RoutingStrategy(str, Enum):
    COST = "cost"
    LATENCY = "latency"
    CAPABILITY = "capability"
    ROUND_ROBIN = "round_robin"


class ModelRouter:
    """Routes requests to the best model based on strategy.

    Mirrors the Go base's modelrouter package.
    """

    def __init__(
        self,
        registry: ModelRegistry,
        strategy: RoutingStrategy = RoutingStrategy.CAPABILITY,
    ) -> None:
        self.registry = registry
        self.strategy = strategy
        self._rr_index = 0

    def select_model(self, requirements: dict[str, Any] | None = None) -> ModelConfig | None:
        """Select the best model for the given requirements.

        Requirements can specify: min_capabilities, max_cost, max_latency.
        """
        models = self.registry.list_configs()
        if not models:
            return None

        reqs = requirements or {}
        candidates = self._filter_candidates(models, reqs)

        if not candidates:
            logger.warning("No models match requirements, falling back to first available")
            return models[0][1]

        if self.strategy == RoutingStrategy.COST:
            return min(candidates, key=lambda m: m[1].cost_per_1k_input)[1]
        elif self.strategy == RoutingStrategy.LATENCY:
            return min(candidates, key=lambda m: m[1].latency_ms)[1]
        elif self.strategy == RoutingStrategy.CAPABILITY:
            return max(candidates, key=lambda m: len(m[1].capabilities))[1]
        elif self.strategy == RoutingStrategy.ROUND_ROBIN:
            self._rr_index = (self._rr_index + 1) % len(candidates)
            return candidates[self._rr_index][1]
        else:
            return candidates[0][1]

    def _filter_candidates(
        self,
        models: list[tuple[str, ModelConfig]],
        reqs: dict[str, Any],
    ) -> list[tuple[str, ModelConfig]]:
        candidates = list(models)

        min_caps = reqs.get("min_capabilities", [])
        if min_caps:
            candidates = [
                (n, m) for n, m in candidates
                if all(cap in m.capabilities for cap in min_caps)
            ]

        max_cost = reqs.get("max_cost")
        if max_cost is not None:
            candidates = [
                (n, m) for n, m in candidates
                if m.cost_per_1k_input <= max_cost
            ]

        max_latency = reqs.get("max_latency")
        if max_latency is not None:
            candidates = [
                (n, m) for n, m in candidates
                if m.latency_ms <= max_latency
            ]

        return candidates
