"""YAML configuration loader for agent definitions."""

from __future__ import annotations

import logging
import os
from pathlib import Path
from typing import Any

import yaml

from superagent.config.schema import AgentDefinition, SuperagentConfig

logger = logging.getLogger(__name__)


def load_config(path: str | Path) -> SuperagentConfig:
    """Load the top-level Superagent configuration from a YAML file."""
    with open(path) as f:
        data = yaml.safe_load(f)
    return SuperagentConfig.model_validate(data)


def load_agent_yaml(path: str | Path) -> AgentDefinition:
    """Load a single agent definition from a YAML file."""
    with open(path) as f:
        data = yaml.safe_load(f)
    return AgentDefinition.model_validate(data)


def load_agents_from_dir(directory: str | Path) -> list[AgentDefinition]:
    """Load all agent YAML definitions from a directory."""
    agents: list[AgentDefinition] = []
    dirpath = Path(directory)
    if not dirpath.is_dir():
        logger.warning("Agent directory %s does not exist", directory)
        return agents
    for p in sorted(dirpath.glob("*.yaml")):
        try:
            agents.append(load_agent_yaml(p))
        except Exception:
            logger.exception("Failed to load agent from %s", p)
    for p in sorted(dirpath.glob("*.yml")):
        try:
            agents.append(load_agent_yaml(p))
        except Exception:
            logger.exception("Failed to load agent from %s", p)
    return agents


def build_agent_from_def(
    definition: AgentDefinition,
    agent_registry: dict[str, Any] | None = None,
) -> Any:
    """Build a BaseAgent instance from an AgentDefinition.

    Returns the appropriate agent type based on spec.type.
    The agent_registry is used for supervisor/sequential/parallel/workflow
    agents to look up child agents by name.
    """
    from superagent.agents.base import BaseAgent
    from superagent.agents.chat import ChatModelAgent
    from superagent.agents.sequential import SequentialAgent
    from superagent.agents.parallel import ParallelAgent
    from superagent.agents.supervisor import SupervisorAgent
    from superagent.agents.workflow import WorkflowAgent

    spec = definition.spec
    name = definition.metadata.name
    agent_id = name
    registry = agent_registry or {}

    model_name = spec.get_model_name()

    if spec.type == "chat_model_agent":
        api_key = os.getenv("MODEL_API_KEY_0", "")
        base_url = os.getenv("MODEL_BASE_URL_0", "")
        provider = os.getenv("MODEL_PROVIDER_0", "openai")
        return ChatModelAgent(
            agent_id=agent_id,
            name=name,
            system_prompt=spec.system_prompt,
            model_name=model_name,
            max_steps=spec.max_steps,
            tools=spec.tools,
            model_provider=provider,
            api_key=api_key,
            base_url=base_url,
        )

    elif spec.type == "supervisor":
        children = [registry[n] for n in spec.sub_agents if n in registry]
        return SupervisorAgent(
            agent_id=agent_id,
            name=name,
            children=children,
            system_prompt=spec.system_prompt,
        )

    elif spec.type == "sequential":
        children = [registry[n] for n in spec.sub_agents if n in registry]
        return SequentialAgent(
            agent_id=agent_id,
            name=name,
            children=children,
            system_prompt=spec.system_prompt,
        )

    elif spec.type == "parallel":
        children = [registry[n] for n in spec.sub_agents if n in registry]
        return ParallelAgent(
            agent_id=agent_id,
            name=name,
            children=children,
            system_prompt=spec.system_prompt,
        )

    elif spec.type == "workflow":
        nodes: list[dict[str, Any]] = []
        edges: list[dict[str, Any]] = []

        if spec.workflow:
            nodes = [n.model_dump() for n in spec.workflow.nodes]
            edges = [
                {"source": e.get_source(), "target": e.get_target(), "condition": e.condition}
                for e in spec.workflow.edges
            ]
        else:
            nodes = spec.workflow_nodes
            edges = spec.workflow_edges

        children_map = {
            n: registry[n] for n in spec.sub_agents if n in registry
        }
        return WorkflowAgent(
            agent_id=agent_id,
            name=name,
            nodes=nodes,
            edges=edges,
            children=children_map,
            system_prompt=spec.system_prompt,
        )

    else:
        logger.warning("Unknown agent type %s for %s, defaulting to ChatModelAgent", spec.type, name)
        return ChatModelAgent(
            agent_id=agent_id,
            name=name,
            system_prompt=spec.system_prompt,
            model_name=model_name,
        )
