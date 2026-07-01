# tests/matrix/test_agents.py
"""Agent 列表 + Admin status 测试：三后端对齐验证。"""
import pytest
from superagent.client import SuperagentClient


@pytest.mark.asyncio
async def test_list_agents_returns_list(client: SuperagentClient) -> None:
    """GET /api/v2/agents 必须返回列表（可为空）。"""
    async with client:
        agents = await client.list_agents()
    assert isinstance(agents, list), "list_agents() should return a list"


@pytest.mark.asyncio
async def test_list_agents_items_have_name(client: SuperagentClient) -> None:
    """Agent 列表中每一项都有 name 字段。"""
    async with client:
        agents = await client.list_agents()
    for agent in agents:
        assert hasattr(agent, 'name') and agent.name, \
            f"Agent item missing name: {agent}"


@pytest.mark.asyncio
async def test_admin_status(client: SuperagentClient) -> None:
    """GET /api/v2/admin/status 必须返回含 uptime 或 status 字段的对象。"""
    async with client:
        status = await client.admin.status()
    assert isinstance(status, dict), "admin.status() should return a dict"
    assert any(k in status for k in ("uptime", "status", "version", "agents_loaded")), \
        f"admin.status() missing expected fields: {status}"


@pytest.mark.asyncio
async def test_admin_validate_agent(client: SuperagentClient) -> None:
    """POST /api/v2/admin/agents/validate 对有效 YAML 应返回 valid=True。"""
    valid_yaml = """apiVersion: superagent/v1
kind: Agent
metadata:
  name: test-validate-agent
spec:
  type: chat_model_agent
  system_prompt: "You are a test agent."
  model:
    primary: gpt-4o-mini
"""
    async with client:
        result = await client.admin.validate_agent(valid_yaml)
    assert result.valid is True, f"Valid YAML should pass validation: {result}"
