# tests/matrix/test_chat.py
"""流式对话测试：验证 A2UI 事件流和 collect() 行为。"""
import pytest
from superagent.client import SuperagentClient
from superagent.types import AgentInfo


def _first_agent_name(agents: list[AgentInfo]) -> str | None:
    """Return the name of the first loaded agent, or None."""
    return agents[0].name if agents else None


@pytest.mark.asyncio
async def test_chat_stream_returns_text(client: SuperagentClient) -> None:
    """chat().collect() 必须返回非空字符串。"""
    async with client:
        agents = await client.list_agents()
        agent_name = _first_agent_name(agents)
        if not agent_name:
            pytest.skip("No agents loaded — cannot test chat")

        text = await client.chat(agent_name, "Say hello in one word.", session_id="matrix-test-1")
    assert isinstance(text, str) and len(text) > 0, \
        f"Expected non-empty text response, got: {repr(text)}"


@pytest.mark.asyncio
async def test_chat_stream_emits_events(client: SuperagentClient) -> None:
    """chat_stream() 必须发出至少一个事件。"""
    async with client:
        agents = await client.list_agents()
        agent_name = _first_agent_name(agents)
        if not agent_name:
            pytest.skip("No agents loaded — cannot test chat")

        events = []
        stream = client.chat_stream(agent_name, "Say yes.", session_id="matrix-test-2")
        async with stream:
            async for event in stream:
                events.append(event)
                if len(events) >= 10:
                    break

    assert len(events) > 0, "Expected at least one SSE event from chat_stream()"


@pytest.mark.asyncio
async def test_chat_session_continuity(client: SuperagentClient) -> None:
    """同 session_id 的第二条消息不应报错（验证会话管理）。"""
    async with client:
        agents = await client.list_agents()
        agent_name = _first_agent_name(agents)
        if not agent_name:
            pytest.skip("No agents loaded — cannot test chat")

        session = "matrix-continuity-test"
        await client.chat(agent_name, "My name is Matrix.", session_id=session)
        response2 = await client.chat(agent_name, "What is my name?", session_id=session)
    assert isinstance(response2, str), "Second message in session should succeed"
