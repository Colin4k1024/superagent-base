"""Model registry — manages available LLM providers and models.

Provides a central registry for model configurations and a factory
method (``create_model``) that returns real AgentScope 2.0 model instances
(``OpenAIChatModel`` / ``DashScopeChatModel``).
"""

from __future__ import annotations

import logging
import os
from dataclasses import dataclass, field
from typing import Any

from agentscope.credential import DashScopeCredential, OpenAICredential
from agentscope.model import DashScopeChatModel, OpenAIChatModel
from pydantic import SecretStr

logger = logging.getLogger(__name__)


@dataclass
class ModelConfig:
    """Configuration for a single model endpoint."""
    provider: str
    model_name: str
    base_url: str = ""
    api_key: str = ""
    max_tokens: int = 4096
    cost_per_1k_input: float = 0.0
    cost_per_1k_output: float = 0.0
    latency_ms: int = 0
    capabilities: list[str] = field(default_factory=list)
    extra: dict[str, Any] = field(default_factory=dict)


class ModelRegistry:
    """Central registry of model providers.

    Models are registered by name and looked up when agents need to call an LLM.
    Supports loading from environment variables (MODEL_API_KEY_0, etc.).
    """

    def __init__(self) -> None:
        self._models: dict[str, ModelConfig] = {}

    def register(self, name: str, config: ModelConfig) -> None:
        self._models[name] = config
        logger.info("Registered model %s (%s/%s)", name, config.provider, config.model_name)

    def get(self, name: str) -> ModelConfig | None:
        return self._models.get(name)

    def list_models(self) -> list[str]:
        return list(self._models.keys())

    def list_configs(self) -> list[tuple[str, ModelConfig]]:
        return list(self._models.items())

    def remove(self, name: str) -> bool:
        return self._models.pop(name, None) is not None

    def create_model(
        self,
        name: str,
        api_key: str = "",
        base_url: str = "",
    ) -> OpenAIChatModel | DashScopeChatModel:
        """Create a real AgentScope model instance from a registered config.

        Falls back to env vars (MODEL_API_KEY_0, MODEL_BASE_URL_0) if the
        config has no api_key/base_url.

        Raises ValueError if the model name is not registered.
        """
        config = self._models.get(name)
        if config is None:
            raise ValueError(f"Model {name!r} not registered")

        key = api_key or config.api_key or os.getenv("MODEL_API_KEY_0", "")
        url = base_url or config.base_url or os.getenv("MODEL_BASE_URL_0", "")
        secret_key = SecretStr(key)

        if config.provider == "dashscope":
            return DashScopeChatModel(
                credential=DashScopeCredential(api_key=secret_key),
                model=config.model_name,
            )
        else:
            # OpenAI-compatible (default)
            model_kwargs: dict[str, Any] = {}
            if url:
                model_kwargs["base_url"] = url
            return OpenAIChatModel(
                credential=OpenAICredential(api_key=secret_key),
                model=config.model_name,
                **model_kwargs,
            )

    @classmethod
    def from_env(cls) -> ModelRegistry:
        """Build a registry from environment variables.

        Reads MODEL_API_KEY_N, MODEL_BASE_URL_N, MODEL_NAME_N for N=0,1,2...
        """
        registry = cls()
        for i in range(20):
            api_key = os.getenv(f"MODEL_API_KEY_{i}")
            if not api_key:
                continue
            base_url = os.getenv(f"MODEL_BASE_URL_{i}", "https://api.openai.com/v1")
            model_name = os.getenv(f"MODEL_NAME_{i}", "gpt-4o")
            provider = os.getenv(f"MODEL_PROVIDER_{i}", "openai")
            name = os.getenv(f"MODEL_ID_{i}", f"model-{i}")
            registry.register(name, ModelConfig(
                provider=provider,
                model_name=model_name,
                base_url=base_url,
                api_key=api_key,
            ))
        return registry
