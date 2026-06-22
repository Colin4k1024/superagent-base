"""Workflow agent — DAG-based execution with topological sort and template vars."""

from __future__ import annotations

import asyncio
import logging
import re
import uuid
from collections import defaultdict, deque
from typing import Any, AsyncGenerator

from agentscope.agent import Agent
from agentscope.event import (
    AgentEvent,
    ReplyEndEvent,
    ReplyStartEvent,
    TextBlockDeltaEvent,
    TextBlockEndEvent,
    TextBlockStartEvent,
)

from superagent.agents.base import BaseAgent

logger = logging.getLogger(__name__)

_TEMPLATE_RE = re.compile(r"\{\{\.(\w+)\}\}")

_SAFE_BUILTINS: dict[str, Any] = {
    "str": str, "int": int, "float": float, "bool": bool,
    "len": len, "range": range, "enumerate": enumerate,
    "zip": zip, "map": map, "filter": filter, "sorted": sorted,
    "list": list, "dict": dict, "set": set, "tuple": tuple,
    "True": True, "False": False, "None": None,
    "print": print, "isinstance": isinstance, "type": type,
    "min": min, "max": max, "sum": sum, "abs": abs, "round": round,
    "any": any, "all": all, "reversed": reversed,
}


class WorkflowNode:
    """A single node in a workflow DAG."""

    __slots__ = ("node_id", "node_type", "agent_id", "tool_name", "code", "condition", "config")

    def __init__(
        self,
        node_id: str,
        node_type: str = "agent_call",
        agent_id: str | None = None,
        tool_name: str | None = None,
        code: str | None = None,
        condition: str | None = None,
        config: dict[str, Any] | None = None,
    ) -> None:
        self.node_id = node_id
        self.node_type = node_type
        self.agent_id = agent_id
        self.tool_name = tool_name
        self.code = code
        self.condition = condition
        self.config = config or {}


class WorkflowEdge:
    """A directed edge between two workflow nodes."""

    __slots__ = ("source", "target", "condition", "variable_map")

    def __init__(
        self,
        source: str,
        target: str,
        condition: str | None = None,
        variable_map: dict[str, str] | None = None,
    ) -> None:
        self.source = source
        self.target = target
        self.condition = condition
        self.variable_map = variable_map or {}


def _substitute_templates(template: str, context: dict[str, Any]) -> str:
    """Replace {{.varName}} placeholders with values from context."""
    def _replace(match: re.Match[str]) -> str:
        key = match.group(1)
        val = context.get(key, match.group(0))
        return str(val)
    return _TEMPLATE_RE.sub(_replace, template)


def _topological_layers(
    nodes: list[WorkflowNode],
    edges: list[WorkflowEdge],
) -> list[list[str]]:
    """Return nodes grouped by topological layer for parallel execution.

    Raises ValueError if the graph contains cycles.
    """
    node_ids = {n.node_id for n in nodes}
    in_degree: dict[str, int] = {nid: 0 for nid in node_ids}
    adjacency: dict[str, list[str]] = defaultdict(list)

    for edge in edges:
        if edge.source in node_ids and edge.target in node_ids:
            adjacency[edge.source].append(edge.target)
            in_degree[edge.target] = in_degree.get(edge.target, 0) + 1

    queue: deque[str] = deque(nid for nid, deg in in_degree.items() if deg == 0)
    layers: list[list[str]] = []
    visited: set[str] = set()

    while queue:
        layer = list(queue)
        layers.append(layer)
        next_queue: deque[str] = deque()
        for nid in layer:
            visited.add(nid)
            for child in adjacency.get(nid, []):
                in_degree[child] -= 1
                if in_degree[child] == 0 and child not in visited:
                    next_queue.append(child)
        queue = next_queue

    if len(visited) != len(node_ids):
        unvisited = node_ids - visited
        raise ValueError(f"Workflow DAG has cycles involving nodes: {unvisited}")

    return layers


class WorkflowAgent(BaseAgent):
    """Executes a DAG of nodes with topological-layer parallel execution.

    Node types: agent_call, tool_call, code, condition.
    Template variables {{.varName}} are substituted from shared context.
    """

    def __init__(
        self,
        agent_id: str,
        name: str,
        nodes: list[dict[str, Any]] | None = None,
        edges: list[dict[str, Any]] | None = None,
        children: dict[str, BaseAgent] | None = None,
        **kwargs: Any,
    ) -> None:
        super().__init__(agent_id, name, **kwargs)
        self.workflow_nodes = [self._parse_node(n) for n in (nodes or [])]
        self.workflow_edges = [self._parse_edge(e) for e in (edges or [])]
        self.children: dict[str, BaseAgent] = children or {}

    @staticmethod
    def _parse_node(raw: dict[str, Any]) -> WorkflowNode:
        return WorkflowNode(
            node_id=raw.get("id", raw.get("node_id", "")),
            node_type=raw.get("type", "agent_call"),
            agent_id=raw.get("agent_id"),
            tool_name=raw.get("tool"),
            code=raw.get("code"),
            condition=raw.get("condition"),
            config=raw.get("config", {}),
        )

    @staticmethod
    def _parse_edge(raw: dict[str, Any]) -> WorkflowEdge:
        return WorkflowEdge(
            source=raw.get("source", raw.get("from", "")),
            target=raw.get("target", raw.get("to", "")),
            condition=raw.get("condition"),
            variable_map=raw.get("variable_map", {}),
        )

    def _build_agent(self) -> Agent:
        """Build a passthrough AgentScope agent for the workflow orchestrator.

        The actual execution logic is in run()/run_stream() which execute the
        DAG.  This agent is used only if someone calls .agent directly.
        """
        from superagent.agents.chat import ChatModelAgent

        helper = ChatModelAgent(
            agent_id=f"{self.agent_id}-wf",
            name=f"{self.name}-wf",
            system_prompt=(
                f"{self.system_prompt}\n\n"
                "You are a workflow orchestrator executing a DAG of nodes."
            ),
            model_name=self.model_name,
        )
        return helper._build_agent()

    @property
    def agent(self) -> Agent:
        """Lazy-initialise the underlying agent."""
        if self._agent is None:
            self._agent = self._build_agent()
        return self._agent

    async def _execute_node(
        self,
        node: WorkflowNode,
        context: dict[str, Any],
        message: str,
    ) -> str:
        if node.node_type == "agent_call":
            child = self.children.get(node.agent_id or node.node_id)
            if child is None:
                return f"[workflow] agent {node.agent_id!r} not found"
            input_text = _substitute_templates(message, context)
            return await child.run(input_text)

        if node.node_type == "tool_call":
            return f"[workflow] tool_call {node.tool_name!r} not implemented"

        if node.node_type == "code":
            code = _substitute_templates(node.code or "", context)
            local_vars: dict[str, Any] = {"input": message, "context": context}
            exec(code, {"__builtins__": _SAFE_BUILTINS}, local_vars)  # noqa: S102
            return str(local_vars.get("output", ""))

        if node.node_type == "condition":
            expr = _substitute_templates(node.condition or "True", context)
            result = eval(expr, {"__builtins__": _SAFE_BUILTINS}, context)  # noqa: S307
            return str(result)

        return f"[workflow] unknown node type: {node.node_type}"

    async def run(self, message: str, **kwargs: Any) -> str:
        if not self.workflow_nodes:
            return f"[WorkflowAgent:{self.name}] No nodes defined"

        layers = _topological_layers(self.workflow_nodes, self.workflow_edges)
        node_map = {n.node_id: n for n in self.workflow_nodes}
        context: dict[str, Any] = {"input": message}
        last_output = message

        for layer in layers:
            tasks = [
                self._execute_node(node_map[nid], context, last_output)
                for nid in layer
            ]
            results = await asyncio.gather(*tasks, return_exceptions=True)

            for nid, result in zip(layer, results):
                if isinstance(result, Exception):
                    logger.error("Workflow node %s failed: %s", nid, result)
                    context[nid] = f"[error] {result}"
                else:
                    context[nid] = result

            if results:
                last_result = results[-1]
                last_output = (
                    str(last_result)
                    if not isinstance(last_result, Exception)
                    else f"[error] {last_result}"
                )

        return last_output

    async def run_stream(self, message: str, **kwargs: Any) -> AsyncGenerator[AgentEvent, None]:
        reply_id = uuid.uuid4().hex
        yield ReplyStartEvent(session_id="", reply_id=reply_id, name=self.name)

        result = await self.run(message, **kwargs)

        block_id = uuid.uuid4().hex
        yield TextBlockStartEvent(reply_id=reply_id, block_id=block_id)
        yield TextBlockDeltaEvent(reply_id=reply_id, block_id=block_id, delta=result)
        yield TextBlockEndEvent(reply_id=reply_id, block_id=block_id)
        yield ReplyEndEvent(session_id="", reply_id=reply_id)

    def describe(self) -> dict[str, Any]:
        base = super().describe()
        base["nodes"] = [{"id": n.node_id, "type": n.node_type} for n in self.workflow_nodes]
        base["edges"] = [{"from": e.source, "to": e.target} for e in self.workflow_edges]
        return base
