"""Tests for agent type implementations."""

import pytest

from superagent.agents.base import BaseAgent
from superagent.agents.chat import ChatModelAgent
from superagent.agents.supervisor import SupervisorAgent
from superagent.agents.sequential import SequentialAgent
from superagent.agents.parallel import ParallelAgent
from superagent.agents.workflow import WorkflowAgent


class StubAgent(BaseAgent):
    """Test-only agent that returns deterministic responses without API calls."""

    def __init__(self, agent_id: str, name: str, response: str | None = None, **kwargs):
        super().__init__(agent_id, name, **kwargs)
        self._response = response

    def _build_agent(self):
        raise NotImplementedError("StubAgent does not use AgentScope")

    async def run(self, message: str, **kwargs) -> str:
        return self._response or f"[{self.name}] {message}"


# ── BaseAgent ────────────────────────────────────────────────────────────────

def test_base_agent_describe():
    agent = StubAgent(agent_id="d-1", name="desc-agent")
    info = agent.describe()
    assert info["id"] == "d-1"
    assert info["name"] == "desc-agent"
    assert info["type"] == "StubAgent"


@pytest.mark.asyncio
async def test_base_agent_run():
    agent = StubAgent(agent_id="b-1", name="base-stub")
    result = await agent.run("hello")
    assert result == "[base-stub] hello"


@pytest.mark.asyncio
async def test_base_agent_resume_not_supported():
    agent = StubAgent(agent_id="b-2", name="no-resume")
    with pytest.raises(NotImplementedError):
        await agent.resume({})


# ── ChatModelAgent ───────────────────────────────────────────────────────────

def test_chat_model_agent_creation():
    agent = ChatModelAgent(agent_id="c-1", name="chat", api_key="test-key")
    info = agent.describe()
    assert info["id"] == "c-1"
    assert info["type"] == "ChatModelAgent"
    assert info["model_provider"] == "openai"


def test_chat_model_agent_dashscope():
    agent = ChatModelAgent(
        agent_id="c-2", name="chat-ds",
        model_provider="dashscope",
        api_key="test-key",
    )
    assert agent.model_provider == "dashscope"


# ── SequentialAgent ──────────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_sequential_agent_run():
    c1 = StubAgent(agent_id="c1", name="step1")
    c2 = StubAgent(agent_id="c2", name="step2")
    agent = SequentialAgent(agent_id="seq-1", name="seq", children=[c1, c2])
    result = await agent.run("input")
    assert "step2" in result
    assert "input" in result


@pytest.mark.asyncio
async def test_sequential_agent_empty():
    agent = SequentialAgent(agent_id="seq-e", name="empty-seq")
    result = await agent.run("input")
    assert "No children" in result


@pytest.mark.asyncio
async def test_sequential_chains_outputs():
    c1 = StubAgent(agent_id="c1", name="upper", response="OUTPUT1")
    c2 = StubAgent(agent_id="c2", name="lower", response="OUTPUT2")
    agent = SequentialAgent(agent_id="seq-2", name="chain", children=[c1, c2])
    result = await agent.run("start")
    assert result == "OUTPUT2"


# ── ParallelAgent ────────────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_parallel_agent_run():
    c1 = StubAgent(agent_id="c1", name="child1")
    c2 = StubAgent(agent_id="c2", name="child2")
    agent = ParallelAgent(agent_id="par-1", name="par", children=[c1, c2])
    result = await agent.run("input")
    assert "---" in result
    assert "child1" in result
    assert "child2" in result


@pytest.mark.asyncio
async def test_parallel_agent_empty():
    agent = ParallelAgent(agent_id="par-e", name="empty-par")
    result = await agent.run("input")
    assert "No children" in result


@pytest.mark.asyncio
async def test_parallel_custom_separator():
    c1 = StubAgent(agent_id="c1", name="a", response="X")
    c2 = StubAgent(agent_id="c2", name="b", response="Y")
    agent = ParallelAgent(
        agent_id="par-2", name="par",
        children=[c1, c2],
        merge_separator=" | ",
    )
    result = await agent.run("input")
    assert result == "X | Y"


# ── SupervisorAgent ──────────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_supervisor_agent_run():
    child = StubAgent(agent_id="c1", name="worker")
    agent = SupervisorAgent(agent_id="sup-1", name="sup", children=[child])
    result = await agent.run("task")
    assert "worker" in result
    assert "task" in result


@pytest.mark.asyncio
async def test_supervisor_agent_empty():
    agent = SupervisorAgent(agent_id="sup-e", name="empty-sup")
    result = await agent.run("task")
    assert "No children" in result


# ── WorkflowAgent ────────────────────────────────────────────────────────────

@pytest.mark.asyncio
async def test_workflow_agent_empty():
    agent = WorkflowAgent(agent_id="wf-e", name="empty-wf")
    result = await agent.run("input")
    assert "No nodes" in result


@pytest.mark.asyncio
async def test_workflow_single_node():
    child = StubAgent(agent_id="n1", name="node1", response="workflow-result")
    agent = WorkflowAgent(
        agent_id="wf-1", name="wf",
        nodes=[{"id": "n1", "type": "agent_call", "agent_id": "n1"}],
        children={"n1": child},
    )
    result = await agent.run("input")
    assert result == "workflow-result"


@pytest.mark.asyncio
async def test_workflow_template_substitution():
    child = StubAgent(agent_id="n1", name="echo")
    agent = WorkflowAgent(
        agent_id="wf-tpl", name="wf-tpl",
        nodes=[{"id": "n1", "type": "agent_call", "agent_id": "n1"}],
        children={"n1": child},
    )
    result = await agent.run("hello world")
    assert "hello world" in result


@pytest.mark.asyncio
async def test_workflow_code_node():
    agent = WorkflowAgent(
        agent_id="wf-code", name="wf-code",
        nodes=[{
            "id": "n1",
            "type": "code",
            "code": "output = input.upper()",
        }],
    )
    result = await agent.run("hello")
    assert result == "HELLO"


@pytest.mark.asyncio
async def test_workflow_condition_node():
    agent = WorkflowAgent(
        agent_id="wf-cond", name="wf-cond",
        nodes=[{
            "id": "n1",
            "type": "condition",
            "condition": "len(input) > 3",
        }],
    )
    result = await agent.run("hello")
    assert result == "True"


def test_workflow_topological_sort():
    """Verify DAG layering: n1 -> n2 -> n3 produces 3 layers."""
    from superagent.agents.workflow import WorkflowNode, WorkflowEdge, _topological_layers

    nodes = [
        WorkflowNode("n1"), WorkflowNode("n2"), WorkflowNode("n3"),
    ]
    edges = [
        WorkflowEdge("n1", "n2"), WorkflowEdge("n2", "n3"),
    ]
    layers = _topological_layers(nodes, edges)
    assert len(layers) == 3
    assert layers[0] == ["n1"]
    assert layers[1] == ["n2"]
    assert layers[2] == ["n3"]


def test_workflow_cycle_detection():
    """Verify that cycles are detected."""
    from superagent.agents.workflow import WorkflowNode, WorkflowEdge, _topological_layers

    nodes = [WorkflowNode("n1"), WorkflowNode("n2")]
    edges = [WorkflowEdge("n1", "n2"), WorkflowEdge("n2", "n1")]
    with pytest.raises(ValueError, match="cycle"):
        _topological_layers(nodes, edges)
