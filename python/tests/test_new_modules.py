"""Tests for new advanced modules: MCP Registry, Skills, Middleware, AgentLoop, Context Injection."""

import asyncio
import logging
import time

import pytest

from superagent.tools.mcp.registry import MCPRegistry, MCPServerConfig
from superagent.tools.mcp.client import MCPClient
from superagent.skills.invoker import LocalInvoker, HTTPInvoker, CompositeInvoker
from superagent.skills.manager import SkillManager, SkillMeta, SkillInstance
from superagent.tools.middleware import (
    chain,
    retry_middleware,
    timeout_middleware,
    rate_limit_middleware,
    cache_middleware,
    log_middleware,
    ToolInvoker,
)
from superagent.agents.agentloop import AgentLoopAgent, DONE_MARKER
from superagent.context.injection import ContextInjectionMiddleware
from superagent.agents.base import BaseAgent


# ═══════════════════════════════════════════════════════════════════════════════
# Helpers
# ═══════════════════════════════════════════════════════════════════════════════

class StubAgent(BaseAgent):
    """Test-only agent that returns deterministic responses."""

    def __init__(self, agent_id: str, name: str, response: str | None = None, **kwargs):
        super().__init__(agent_id, name, **kwargs)
        self._responses = [response] if response else []
        self._call_count = 0

    def _build_agent(self):
        raise NotImplementedError("StubAgent does not use AgentScope")

    async def run(self, message: str, **kwargs) -> str:
        if self._call_count < len(self._responses):
            resp = self._responses[self._call_count]
        else:
            resp = f"[{self.name}] {message}"
        self._call_count += 1
        return resp


class DoneAfterNTurnsAgent(BaseAgent):
    """Agent that returns normal responses for N turns, then includes [DONE]."""

    def __init__(self, agent_id: str, name: str, turns_before_done: int = 2, **kwargs):
        super().__init__(agent_id, name, **kwargs)
        self.turns_before_done = turns_before_done
        self._call_count = 0

    def _build_agent(self):
        raise NotImplementedError

    async def run(self, message: str, **kwargs) -> str:
        self._call_count += 1
        if self._call_count >= self.turns_before_done:
            return f"Final answer after {self._call_count} turns {DONE_MARKER}"
        return f"Turn {self._call_count} thinking..."


# ═══════════════════════════════════════════════════════════════════════════════
# MCP Registry Tests
# ═══════════════════════════════════════════════════════════════════════════════

class TestMCPRegistry:

    def test_registry_starts_empty(self):
        reg = MCPRegistry()
        assert reg.list_servers() == []
        assert len(reg) == 0

    def test_registry_contains_check(self):
        reg = MCPRegistry()
        assert "foo" not in reg

    def test_get_client_returns_none_for_missing(self):
        reg = MCPRegistry()
        assert reg.get_client("nonexistent") is None

    def test_server_config_defaults(self):
        cfg = MCPServerConfig(name="test")
        assert cfg.name == "test"
        assert cfg.transport == "stdio"
        assert cfg.endpoint == ""
        assert cfg.command == ""
        assert cfg.args == []
        assert cfg.env == {}

    def test_server_config_with_values(self):
        cfg = MCPServerConfig(
            name="my-server",
            transport="sse",
            endpoint="http://localhost:3000/sse",
        )
        assert cfg.name == "my-server"
        assert cfg.transport == "sse"
        assert cfg.endpoint == "http://localhost:3000/sse"


# ═══════════════════════════════════════════════════════════════════════════════
# Skills System Tests
# ═══════════════════════════════════════════════════════════════════════════════

class TestLocalInvoker:

    @pytest.mark.asyncio
    async def test_register_and_invoke(self):
        invoker = LocalInvoker()
        invoker.register("add", lambda a, b: {"sum": a + b})
        result = await invoker.invoke("add", {"a": 3, "b": 4})
        assert result == {"sum": 7}

    @pytest.mark.asyncio
    async def test_invoke_async_function(self):
        invoker = LocalInvoker()

        async def async_skill(x: int) -> dict:
            return {"doubled": x * 2}

        invoker.register("double", async_skill)
        result = await invoker.invoke("double", {"x": 5})
        assert result == {"doubled": 10}

    @pytest.mark.asyncio
    async def test_invoke_non_dict_result(self):
        invoker = LocalInvoker()
        invoker.register("greet", lambda name: f"Hello {name}")
        result = await invoker.invoke("greet", {"name": "World"})
        assert result == {"result": "Hello World"}

    @pytest.mark.asyncio
    async def test_invoke_missing_skill_raises(self):
        invoker = LocalInvoker()
        with pytest.raises(KeyError, match="Local skill not found"):
            await invoker.invoke("missing", {})

    def test_has_skill(self):
        invoker = LocalInvoker()
        invoker.register("foo", lambda: "bar")
        assert invoker.has_skill("foo") is True
        assert invoker.has_skill("bar") is False

    def test_list_skills(self):
        invoker = LocalInvoker()
        invoker.register("a", lambda: None)
        invoker.register("b", lambda: None)
        assert sorted(invoker.list_skills()) == ["a", "b"]


class TestHTTPInvoker:

    def test_register_endpoint(self):
        invoker = HTTPInvoker()
        invoker.register_endpoint("search", "http://localhost:8080/search")
        assert invoker.has_skill("search") is True
        assert invoker.has_skill("other") is False

    def test_list_skills(self):
        invoker = HTTPInvoker()
        invoker.register_endpoint("a", "http://a.com")
        invoker.register_endpoint("b", "http://b.com")
        assert sorted(invoker.list_skills()) == ["a", "b"]

    @pytest.mark.asyncio
    async def test_invoke_missing_endpoint_raises(self):
        invoker = HTTPInvoker()
        with pytest.raises(KeyError, match="HTTP skill endpoint not found"):
            await invoker.invoke("missing", {})


class TestCompositeInvoker:

    @pytest.mark.asyncio
    async def test_local_first(self):
        local = LocalInvoker()
        local.register("calc", lambda a, b: {"result": a + b})
        http = HTTPInvoker()

        composite = CompositeInvoker([local, http])
        result = await composite.invoke("calc", {"a": 1, "b": 2})
        assert result == {"result": 3}

    @pytest.mark.asyncio
    async def test_fallback_to_http(self):
        """When local doesn't have the skill, falls through to HTTP (which will fail)."""
        local = LocalInvoker()
        http = HTTPInvoker()
        composite = CompositeInvoker([local, http])

        with pytest.raises(KeyError):
            await composite.invoke("missing", {})

    @pytest.mark.asyncio
    async def test_add_invoker(self):
        composite = CompositeInvoker()
        local = LocalInvoker()
        local.register("test", lambda: {"ok": True})
        composite.add_invoker(local)
        result = await composite.invoke("test", {})
        assert result == {"ok": True}


class TestSkillManager:

    def test_register_local(self):
        mgr = SkillManager()
        meta = SkillMeta(name="calculator", description="Math ops")
        mgr.register_local(meta, lambda a, b: a + b)

        installed = mgr.list_installed()
        assert len(installed) == 1
        assert installed[0].meta.name == "calculator"
        assert installed[0].source == "local"

    def test_register_http(self):
        mgr = SkillManager()
        meta = SkillMeta(name="translator", description="Translation")
        mgr.register_http(meta, "http://translate.api/call")

        installed = mgr.list_installed()
        assert len(installed) == 1
        assert installed[0].source == "http"

    @pytest.mark.asyncio
    async def test_invoke_local_skill(self):
        mgr = SkillManager()
        meta = SkillMeta(name="add", description="Addition")
        mgr.register_local(meta, lambda a, b: {"sum": a + b})

        result = await mgr.invoke("add", {"a": 10, "b": 20})
        assert result == {"sum": 30}

    @pytest.mark.asyncio
    async def test_search_by_name(self):
        mgr = SkillManager()
        mgr.register_local(SkillMeta(name="calculator", description="Math"), lambda: None)
        mgr.register_local(SkillMeta(name="translator", description="Lang"), lambda: None)

        results = await mgr.search("calc")
        assert len(results) == 1
        assert results[0].name == "calculator"

    @pytest.mark.asyncio
    async def test_search_by_description(self):
        mgr = SkillManager()
        mgr.register_local(SkillMeta(name="foo", description="Translation service"), lambda: None)

        results = await mgr.search("translation")
        assert len(results) == 1

    @pytest.mark.asyncio
    async def test_search_by_tag(self):
        mgr = SkillManager()
        meta = SkillMeta(name="bar", description="", tags=["math", "arithmetic"])
        mgr.register_local(meta, lambda: None)

        results = await mgr.search("math")
        assert len(results) == 1

    @pytest.mark.asyncio
    async def test_install_stub(self):
        mgr = SkillManager()
        await mgr.install("remote-skill", "2.0.0")
        installed = mgr.list_installed()
        assert len(installed) == 1
        assert installed[0].meta.name == "remote-skill"
        assert installed[0].meta.version == "2.0.0"
        assert installed[0].source == "hub"

    def test_get_tool(self):
        mgr = SkillManager()
        meta = SkillMeta(name="test")
        mgr.register_local(meta, lambda: None)
        assert mgr.get_tool("test") is not None
        assert mgr.get_tool("nonexistent") is None

    def test_uninstall(self):
        mgr = SkillManager()
        mgr.register_local(SkillMeta(name="temp"), lambda: None)
        assert mgr.uninstall("temp") is True
        assert mgr.uninstall("temp") is False
        assert len(mgr.list_installed()) == 0


# ═══════════════════════════════════════════════════════════════════════════════
# Tool Middleware Tests
# ═══════════════════════════════════════════════════════════════════════════════

class TestToolMiddleware:

    @pytest.mark.asyncio
    async def test_chain_no_middleware(self):
        async def invoker(name: str, args: dict) -> dict:
            return {"name": name, "args": args}

        chained = chain()(invoker)
        result = await chained("test", {"x": 1})
        assert result == {"name": "test", "args": {"x": 1}}

    @pytest.mark.asyncio
    async def test_log_middleware(self):
        async def invoker(name: str, args: dict) -> dict:
            return {"ok": True}

        logged = log_middleware()(invoker)
        result = await logged("test", {})
        assert result == {"ok": True}

    @pytest.mark.asyncio
    async def test_cache_middleware_hit(self):
        call_count = 0

        async def invoker(name: str, args: dict) -> dict:
            nonlocal call_count
            call_count += 1
            return {"count": call_count}

        cached = cache_middleware(ttl=60.0)(invoker)
        r1 = await cached("test", {"key": "value"})
        r2 = await cached("test", {"key": "value"})
        assert r1 == {"count": 1}
        assert r2 == {"count": 1}  # cached
        assert call_count == 1

    @pytest.mark.asyncio
    async def test_cache_middleware_different_args(self):
        call_count = 0

        async def invoker(name: str, args: dict) -> dict:
            nonlocal call_count
            call_count += 1
            return {"count": call_count}

        cached = cache_middleware(ttl=60.0)(invoker)
        await cached("test", {"a": 1})
        await cached("test", {"a": 2})
        assert call_count == 2  # different args, no cache hit

    @pytest.mark.asyncio
    async def test_timeout_middleware_success(self):
        async def invoker(name: str, args: dict) -> dict:
            return {"ok": True}

        timed = timeout_middleware(timeout=5.0)(invoker)
        result = await timed("test", {})
        assert result == {"ok": True}

    @pytest.mark.asyncio
    async def test_timeout_middleware_exceeded(self):
        async def slow_invoker(name: str, args: dict) -> dict:
            await asyncio.sleep(10)
            return {}

        timed = timeout_middleware(timeout=0.01)(slow_invoker)
        with pytest.raises(TimeoutError, match="timed out"):
            await timed("test", {})

    @pytest.mark.asyncio
    async def test_retry_middleware_success_first_try(self):
        call_count = 0

        async def invoker(name: str, args: dict) -> dict:
            nonlocal call_count
            call_count += 1
            return {"attempts": call_count}

        retried = retry_middleware(max_retries=3, backoff=0.01)(invoker)
        result = await retried("test", {})
        assert result == {"attempts": 1}

    @pytest.mark.asyncio
    async def test_retry_middleware_success_after_failures(self):
        call_count = 0

        async def flaky_invoker(name: str, args: dict) -> dict:
            nonlocal call_count
            call_count += 1
            if call_count < 3:
                raise ValueError(f"fail {call_count}")
            return {"attempts": call_count}

        retried = retry_middleware(max_retries=3, backoff=0.01)(flaky_invoker)
        result = await retried("test", {})
        assert result == {"attempts": 3}

    @pytest.mark.asyncio
    async def test_retry_middleware_all_failures(self):
        async def always_fail(name: str, args: dict) -> dict:
            raise ValueError("always fails")

        retried = retry_middleware(max_retries=2, backoff=0.01)(always_fail)
        with pytest.raises(ValueError, match="always fails"):
            await retried("test", {})

    @pytest.mark.asyncio
    async def test_rate_limit_middleware(self):
        async def invoker(name: str, args: dict) -> dict:
            return {"ok": True}

        limited = rate_limit_middleware(rpm=100)(invoker)
        result = await limited("test", {})
        assert result == {"ok": True}

    @pytest.mark.asyncio
    async def test_chain_multiple_middlewares(self):
        call_count = 0

        async def invoker(name: str, args: dict) -> dict:
            nonlocal call_count
            call_count += 1
            return {"count": call_count}

        pipeline = chain(
            log_middleware(),
            cache_middleware(ttl=60.0),
            retry_middleware(max_retries=1, backoff=0.01),
        )(invoker)

        r1 = await pipeline("test", {"k": "v"})
        r2 = await pipeline("test", {"k": "v"})
        assert r1["count"] == 1
        assert r2["count"] == 1  # cached by cache middleware


# ═══════════════════════════════════════════════════════════════════════════════
# AgentLoop Tests
# ═══════════════════════════════════════════════════════════════════════════════

class TestAgentLoopAgent:

    def test_agentloop_creation(self):
        child = StubAgent(agent_id="c", name="child")
        agent = AgentLoopAgent(
            agent_id="loop-1", name="loop",
            child_agent=child, max_turns=10,
        )
        assert agent.max_turns == 10
        assert agent.done_marker == DONE_MARKER
        assert agent.child_agent is child

    def test_agentloop_describe(self):
        child = StubAgent(agent_id="c", name="child")
        agent = AgentLoopAgent(agent_id="loop-1", name="loop", child_agent=child)
        info = agent.describe()
        assert info["type"] == "AgentLoopAgent"
        assert info["max_turns"] == 25
        assert info["child_agent"] == "child"

    @pytest.mark.asyncio
    async def test_agentloop_no_child(self):
        agent = AgentLoopAgent(agent_id="loop-1", name="loop")
        result = await agent.run("hello")
        assert "No child agent" in result

    @pytest.mark.asyncio
    async def test_agentloop_done_immediately(self):
        child = StubAgent(agent_id="c", name="child", response=f"answer {DONE_MARKER}")
        agent = AgentLoopAgent(agent_id="loop-1", name="loop", child_agent=child)
        result = await agent.run("question")
        assert "answer" in result
        assert DONE_MARKER not in result

    @pytest.mark.asyncio
    async def test_agentloop_multi_turn(self):
        child = DoneAfterNTurnsAgent(agent_id="c", name="child", turns_before_done=3)
        agent = AgentLoopAgent(
            agent_id="loop-1", name="loop",
            child_agent=child, max_turns=10,
        )
        result = await agent.run("start")
        assert "Final answer" in result
        assert child._call_count == 3

    @pytest.mark.asyncio
    async def test_agentloop_max_turns_reached(self):
        child = StubAgent(agent_id="c", name="never-done", response="thinking...")
        agent = AgentLoopAgent(
            agent_id="loop-1", name="loop",
            child_agent=child, max_turns=3,
        )
        result = await agent.run("start")
        assert child._call_count == 3
        assert "thinking..." in result

    @pytest.mark.asyncio
    async def test_agentloop_custom_done_marker(self):
        child = StubAgent(agent_id="c", name="child", response="result [FINISHED]")
        agent = AgentLoopAgent(
            agent_id="loop-1", name="loop",
            child_agent=child, done_marker="[FINISHED]",
        )
        result = await agent.run("go")
        assert "result" in result
        assert "[FINISHED]" not in result


# ═══════════════════════════════════════════════════════════════════════════════
# Context Injection Tests
# ═══════════════════════════════════════════════════════════════════════════════

class TestContextInjectionMiddleware:

    def test_no_injection(self):
        mw = ContextInjectionMiddleware()
        messages = [{"role": "user", "content": "hello"}]
        result = mw.inject(messages)
        assert result == messages

    def test_static_context(self):
        mw = ContextInjectionMiddleware(static_context="You are a helpful assistant.")
        messages = [{"role": "user", "content": "hello"}]
        result = mw.inject(messages)
        assert len(result) == 2
        assert result[0]["role"] == "system"
        assert "helpful assistant" in result[0]["content"]
        assert result[1]["content"] == "hello"

    def test_inject_timestamp(self):
        mw = ContextInjectionMiddleware(inject_timestamp=True)
        messages = [{"role": "user", "content": "hello"}]
        result = mw.inject(messages)
        assert len(result) == 2
        assert result[0]["role"] == "system"
        assert "Current timestamp:" in result[0]["content"]

    def test_inject_session_metadata(self):
        mw = ContextInjectionMiddleware(inject_session_metadata=True)
        messages = [{"role": "user", "content": "hello"}]
        result = mw.inject(messages, session_id="sess-123")
        assert len(result) == 2
        assert "Session ID: sess-123" in result[0]["content"]

    def test_inject_extra_metadata(self):
        mw = ContextInjectionMiddleware(inject_session_metadata=True)
        result = mw.inject(
            [{"role": "user", "content": "hi"}],
            session_id="s1",
            extra_metadata={"user_role": "admin", "locale": "zh-CN"},
        )
        assert "user_role: admin" in result[0]["content"]
        assert "locale: zh-CN" in result[0]["content"]

    def test_all_injections_combined(self):
        mw = ContextInjectionMiddleware(
            inject_timestamp=True,
            inject_session_metadata=True,
            static_context="System context.",
        )
        result = mw.build_context_messages(session_id="s1")
        assert len(result) == 1
        content = result[0]["content"]
        assert "System context." in content
        assert "Current timestamp:" in content
        assert "Session ID: s1" in content

    def test_build_context_messages_empty(self):
        mw = ContextInjectionMiddleware()
        result = mw.build_context_messages()
        assert result == []


# ═══════════════════════════════════════════════════════════════════════════════
# Server Endpoint Tests
# ═══════════════════════════════════════════════════════════════════════════════

class TestServerEndpoints:

    @pytest.fixture
    def client(self):
        from httpx import ASGITransport, AsyncClient
        from superagent.server import app
        transport = ASGITransport(app=app)
        return AsyncClient(transport=transport, base_url="http://test")

    @pytest.mark.asyncio
    async def test_mcp_servers_endpoint(self, client):
        async with client as c:
            resp = await c.get("/api/v2/mcp/servers")
            assert resp.status_code == 200
            data = resp.json()
            assert data["code"] == 0
            assert isinstance(data["data"], list)

    @pytest.mark.asyncio
    async def test_skills_endpoint_uses_manager(self, client):
        async with client as c:
            resp = await c.get("/api/v2/skills")
            assert resp.status_code == 200
            data = resp.json()
            assert data["code"] == 0
            skills = data["data"]
            assert isinstance(skills, list)

    @pytest.mark.asyncio
    async def test_skills_search_endpoint(self, client):
        async with client as c:
            resp = await c.get("/api/v2/skills/search?q=calc")
            assert resp.status_code == 200
            data = resp.json()
            assert data["code"] == 0
            assert isinstance(data["data"], list)

    @pytest.mark.asyncio
    async def test_tools_endpoint(self, client):
        async with client as c:
            resp = await c.get("/api/v2/tools")
            assert resp.status_code == 200
            data = resp.json()
            assert data["code"] == 0
