"""FastAPI application with SSE streaming endpoints.

API-compatible with the Go base (superagent-base) HTTP layer.
"""

from __future__ import annotations

import asyncio
import json
import logging
import os
import time
import uuid
from contextlib import asynccontextmanager
from pathlib import Path
from typing import Any, AsyncIterator

from fastapi import FastAPI, File, HTTPException, Query, Request, UploadFile
from fastapi.responses import JSONResponse, PlainTextResponse, Response
from pydantic import BaseModel, Field
from sse_starlette.sse import EventSourceResponse

from superagent.agents.base import BaseAgent
from superagent.config.loader import build_agent_from_def, load_agents_from_dir
from superagent.config.schema import AgentDefinition
from superagent.event import (
    ReplyStartEvent,
    ReplyEndEvent,
    TextBlockStartEvent,
    TextBlockDeltaEvent,
    TextBlockEndEvent,
    ThinkingBlockStartEvent,
    ThinkingBlockDeltaEvent,
    ThinkingBlockEndEvent,
    ToolCallStartEvent,
    ToolCallDeltaEvent,
    ToolCallEndEvent,
    ToolResultStartEvent,
    ToolResultTextDeltaEvent,
    ToolResultEndEvent,
    ModelCallEndEvent,
    HintBlockEvent,
    ExceedMaxItersEvent,
    RequireUserConfirmEvent,
)
from superagent.memory.backends.builtin import BuiltinMemory
from superagent.message import Msg, AssistantMsg, TextBlock
from superagent.tools.mcp.registry import MCPRegistry
from superagent.skills.manager import SkillManager, SkillMeta

logger = logging.getLogger(__name__)

_agents: dict[str, BaseAgent] = {}
_agent_defs: dict[str, AgentDefinition] = {}
_memory = BuiltinMemory()
_start_time = time.time()
_agents_dir = os.getenv("AGENTS_DIR", "configs/agents")

_mcp_registry = MCPRegistry()
_skill_manager = SkillManager()

_REQUEST_COUNT = 0
_ERROR_COUNT = 0

# --- In-memory stores for stub endpoints ---
_agent_states: dict[str, dict[str, Any]] = {}
_interrupt_states: dict[str, dict[str, Any]] = {}
_files: dict[str, dict[str, Any]] = {}
_workflows: dict[str, dict[str, Any]] = {}
_long_term_memory: list[dict[str, Any]] = []
_conversations: dict[str, dict[str, Any]] = {}
_skills: list[dict[str, Any]] = [
    {"name": "calculator", "description": "Basic arithmetic calculator", "version": "1.0.0"},
    {"name": "datetime", "description": "Date and time utilities", "version": "1.0.0"},
    {"name": "uuid", "description": "UUID generation", "version": "1.0.0"},
]
_tools: list[dict[str, Any]] = [
    {"name": "web_search", "type": "builtin", "description": "Search the web"},
    {"name": "http_request", "type": "builtin", "description": "Make HTTP requests"},
    {"name": "code_execute", "type": "builtin", "description": "Execute code snippets"},
]


def _reload_agents() -> int:
    global _agents, _agent_defs
    agents_dir = Path(_agents_dir)
    if not agents_dir.is_dir():
        logger.warning("Agents directory %s does not exist", agents_dir)
        return 0

    defs = load_agents_from_dir(agents_dir)
    _agent_defs.clear()
    _agents.clear()

    # First pass: build leaf agents (chat_model_agent, deep_agent)
    for d in defs:
        if d.spec.type in ("chat_model_agent", "deep_agent"):
            try:
                agent = build_agent_from_def(d)
                _agents[d.metadata.name] = agent
                _agent_defs[d.metadata.name] = d
            except Exception:
                logger.exception("Failed to build agent %s", d.metadata.name)

    # Second pass: build orchestration agents (they reference leaf agents)
    for d in defs:
        if d.spec.type not in ("chat_model_agent", "deep_agent"):
            try:
                agent = build_agent_from_def(d, agent_registry=_agents)
                _agents[d.metadata.name] = agent
                _agent_defs[d.metadata.name] = d
            except Exception:
                logger.exception("Failed to build agent %s", d.metadata.name)

    logger.info("Loaded %d agents from %s", len(_agents), agents_dir)
    return len(_agents)


def _ok(data: Any = None, msg: str = "ok") -> dict[str, Any]:
    """Standard success response envelope matching Go base."""
    return {"code": 0, "msg": msg, "data": data}


@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info("Superagent Python base starting — version %s", "0.1.0")
    _reload_agents()

    for s in _skills:
        _skill_manager.register_local(
            SkillMeta(name=s["name"], description=s["description"], version=s["version"]),
            lambda **kw: {"result": f"Skill {s['name']} executed"},
        )

    yield
    await _mcp_registry.disconnect_all()
    logger.info("Superagent Python base shutting down")


app = FastAPI(
    title="Superagent Python Base",
    version="0.1.0",
    description="Python agent platform — API-compatible with superagent-base Go",
    lifespan=lifespan,
)


# ──────────────────────────────────────────────
# ASGI v1→v2 path rewrite middleware.
# Inserted into the ASGI stack before FastAPI's router
# so route matching sees /api/v2/* paths.
# ──────────────────────────────────────────────

_V1_PREFIX = "/api/v1/"
_V2_PREFIX = "/api/v2/"


class _V1RewriteMiddleware:
    """Rewrites /api/v1/* paths to /api/v2/* before routing."""
    def __init__(self, app):
        self._app = app

    async def __call__(self, scope, receive, send):
        if scope.get("type") == "http":
            path: str = scope.get("path", "")
            if path.startswith(_V1_PREFIX):
                new_path = _V2_PREFIX + path[len(_V1_PREFIX):]
                scope = dict(scope)
                scope["path"] = new_path
                scope["raw_path"] = new_path.encode()
        await self._app(scope, receive, send)


app.add_middleware(_V1RewriteMiddleware)  # type: ignore[arg-type]


# ──────────────────────────────────────────────
# Pydantic models — existing
# ──────────────────────────────────────────────

class ChatRequest(BaseModel):
    agent_id: str = Field(..., description="Target agent ID")
    message: str = Field(..., description="User message")
    session_id: str | None = Field(None, description="Session/conversation ID")
    stream: bool = Field(True, description="Enable SSE streaming")


class ChatResumeRequest(BaseModel):
    agent_id: str
    session_id: str
    resume_payload: dict[str, Any] = Field(default_factory=dict)


class AgentCreateRequest(BaseModel):
    yaml_definition: str = Field(..., description="Agent YAML definition")


class AgentInfo(BaseModel):
    id: str
    name: str
    type: str
    model: str = ""
    system_prompt: str = ""


class ConversationInfo(BaseModel):
    id: str
    message_count: int


# ──────────────────────────────────────────────
# Pydantic models — new (for missing endpoints)
# ──────────────────────────────────────────────

class AbortRequest(BaseModel):
    session_id: str = Field(..., description="Session ID to abort")


class AgentStateSetRequest(BaseModel):
    key: str = Field(..., description="State key")
    value: Any = Field(..., description="State value")


class ConversationCreateRequest(BaseModel):
    title: str = Field("", description="Conversation title")
    agent_id: str = Field("", description="Associated agent ID")
    metadata: dict[str, Any] = Field(default_factory=dict)


class ConversationUpdateRequest(BaseModel):
    title: str | None = None
    metadata: dict[str, Any] | None = None


class MemoryCreateRequest(BaseModel):
    user_id: str = Field(..., description="User ID")
    content: str = Field(..., description="Memory content")
    metadata: dict[str, Any] = Field(default_factory=dict)


class MemoryUpdateRequest(BaseModel):
    content: str | None = None
    metadata: dict[str, Any] | None = None


class WorkflowRunRequest(BaseModel):
    workflow_id: str = Field(..., description="Workflow ID")
    inputs: dict[str, Any] = Field(default_factory=dict)


class WorkflowStreamRunRequest(BaseModel):
    workflow_id: str = Field(..., description="Workflow ID")
    inputs: dict[str, Any] = Field(default_factory=dict)


class WorkflowStreamResumeRequest(BaseModel):
    workflow_id: str = Field(..., description="Workflow ID")
    session_id: str = Field(..., description="Session ID")
    resume_payload: dict[str, Any] = Field(default_factory=dict)


class WorkflowChatRequest(BaseModel):
    workflow_id: str = Field(..., description="Workflow ID")
    message: str = Field(..., description="User message")
    session_id: str | None = None


# ──────────────────────────────────────────────
# Passport auth stubs — compatible with Coze Studio frontend
# Accepts any email/password; returns deterministic mock user.
# ──────────────────────────────────────────────

_FIXED_USER_ID = 7657359490462777344

def _build_user_response(email: str) -> dict[str, Any]:
    name = email.split("@")[0] if "@" in email else email
    return {
        "code": 0,
        "msg": "",
        "data": {
            "user_id_str": str(_FIXED_USER_ID),
            "name": name,
            "user_unique_name": name,
            "email": email,
            "description": "",
            "avatar_url": "default_icon/user_default_icon.png",
            "screen_name": name,
            "app_user_info": {"user_unique_name": name},
            "locale": "zh-CN",
            "user_create_time": int(time.time()),
        },
    }


@app.post("/api/passport/web/email/register/v2/")
async def passport_register(request: Request) -> dict[str, Any]:
    body = await request.json()
    email = body.get("email", "user@example.com")
    return _build_user_response(email)


@app.post("/api/passport/web/email/login/")
async def passport_login(request: Request) -> dict[str, Any]:
    body = await request.json()
    email = body.get("email", "user@example.com")
    return _build_user_response(email)


@app.get("/api/passport/web/logout/")
async def passport_logout() -> dict[str, Any]:
    return {"code": 0, "msg": ""}


# ──────────────────────────────────────────────
# Existing endpoints (unchanged)
# ──────────────────────────────────────────────

@app.get("/health")
async def health() -> dict[str, Any]:
    return {
        "status": "ok",
        "version": "0.1.0",
        "uptime_seconds": int(time.time() - _start_time),
        "agents_loaded": len(_agents),
    }


@app.get("/metrics")
async def metrics() -> PlainTextResponse:
    lines = [
        "# HELP superagent_requests_total Total chat requests",
        "# TYPE superagent_requests_total counter",
        f"superagent_requests_total {_REQUEST_COUNT}",
        "# HELP superagent_errors_total Total errors",
        "# TYPE superagent_errors_total counter",
        f"superagent_errors_total {_ERROR_COUNT}",
        "# HELP superagent_agents_loaded Number of loaded agents",
        "# TYPE superagent_agents_loaded gauge",
        f"superagent_agents_loaded {len(_agents)}",
        "# HELP superagent_uptime_seconds Uptime in seconds",
        "# TYPE superagent_uptime_seconds gauge",
        f"superagent_uptime_seconds {int(time.time() - _start_time)}",
    ]
    return PlainTextResponse("\n".join(lines) + "\n", media_type="text/plain")


@app.get("/api/v2/agents")
async def list_agents() -> list[dict[str, Any]]:
    result = []
    for name, agent in _agents.items():
        result.append(agent.describe())
    return result


@app.get("/api/v2/conversations")
async def list_conversations() -> list[dict[str, Any]]:
    conv_ids = await _memory.list_conversations()
    conversations = []
    for cid in conv_ids:
        msgs = await _memory.get(cid)
        conversations.append({"id": cid, "message_count": len(msgs)})
    return conversations


async def _stream_agent(agent: BaseAgent, message: str, session_id: str) -> AsyncIterator[dict[str, Any]]:
    """Stream agent events as A2UI-compatible SSE.

    Maps AgentScope 2.0 event types to the Superagent A2UI protocol:
    - TextBlockDeltaEvent → event: "text" with delta
    - ToolCallStartEvent  → event: "tool_call"
    - ToolResultEndEvent  → event: "tool_result"
    - ReplyEndEvent       → event: "done"
    - ThinkingBlockDeltaEvent → event: "thinking" with delta
    - ModelCallEndEvent   → event: "model_call_end" with usage
    - HintBlockEvent      → event: "hint"
    """
    global _REQUEST_COUNT, _ERROR_COUNT
    _REQUEST_COUNT += 1

    # Store user message as Msg
    user_msg = Msg(name="user", role="user", content=[TextBlock(text=message)])
    await _memory.append_msg(session_id, user_msg)

    try:
        # Track reply state for message reconstruction
        reply_msg: Msg | None = None
        full_text = ""

        async for event in agent.run_stream(message, session_id=session_id):
            event_type = type(event).__name__

            # Reply lifecycle
            if isinstance(event, ReplyStartEvent):
                reply_msg = AssistantMsg(
                    name=event.name or agent.name,
                    content=[],
                    id=event.reply_id,
                )
                yield {
                    "event": "progress",
                    "data": json.dumps({"status": "started", "reply_id": event.reply_id}),
                }
                continue

            if isinstance(event, ReplyEndEvent):
                if reply_msg is not None:
                    reply_msg.finished_at = time.strftime("%Y-%m-%dT%H:%M:%S", time.gmtime()) + "Z"
                yield {
                    "event": "done",
                    "data": json.dumps({"finish_reason": "stop", "reply_id": event.reply_id}),
                }
                continue

            # Text block events
            if isinstance(event, TextBlockStartEvent):
                if reply_msg is not None:
                    reply_msg.append_event(event)
                yield {
                    "event": "text_start",
                    "data": json.dumps({"block_id": event.block_id}),
                }
                continue

            if isinstance(event, TextBlockDeltaEvent):
                if reply_msg is not None:
                    reply_msg.append_event(event)
                full_text += event.delta
                yield {
                    "event": "text",
                    "data": json.dumps({"delta": event.delta, "block_id": event.block_id}),
                }
                continue

            if isinstance(event, TextBlockEndEvent):
                yield {
                    "event": "text_end",
                    "data": json.dumps({"block_id": event.block_id}),
                }
                continue

            # Thinking block events
            if isinstance(event, ThinkingBlockStartEvent):
                if reply_msg is not None:
                    reply_msg.append_event(event)
                yield {
                    "event": "thinking_start",
                    "data": json.dumps({"block_id": event.block_id}),
                }
                continue

            if isinstance(event, ThinkingBlockDeltaEvent):
                if reply_msg is not None:
                    reply_msg.append_event(event)
                yield {
                    "event": "thinking",
                    "data": json.dumps({"delta": event.delta, "block_id": event.block_id}),
                }
                continue

            if isinstance(event, ThinkingBlockEndEvent):
                yield {
                    "event": "thinking_end",
                    "data": json.dumps({"block_id": event.block_id}),
                }
                continue

            # Tool call events
            if isinstance(event, ToolCallStartEvent):
                if reply_msg is not None:
                    reply_msg.append_event(event)
                yield {
                    "event": "tool_call",
                    "data": json.dumps({
                        "tool": event.tool_call_name,
                        "tool_call_id": event.tool_call_id,
                        "status": "start",
                    }),
                }
                continue

            if isinstance(event, ToolCallDeltaEvent):
                yield {
                    "event": "tool_call_delta",
                    "data": json.dumps({
                        "tool_call_id": event.tool_call_id,
                        "delta": event.delta,
                    }),
                }
                continue

            if isinstance(event, ToolCallEndEvent):
                if reply_msg is not None:
                    reply_msg.append_event(event)
                yield {
                    "event": "tool_call_end",
                    "data": json.dumps({"tool_call_id": event.tool_call_id}),
                }
                continue

            # Tool result events
            if isinstance(event, ToolResultStartEvent):
                if reply_msg is not None:
                    reply_msg.append_event(event)
                yield {
                    "event": "tool_result_start",
                    "data": json.dumps({
                        "tool": event.tool_call_name,
                        "tool_call_id": event.tool_call_id,
                    }),
                }
                continue

            if isinstance(event, ToolResultTextDeltaEvent):
                if reply_msg is not None:
                    reply_msg.append_event(event)
                yield {
                    "event": "tool_result_delta",
                    "data": json.dumps({
                        "tool_call_id": event.tool_call_id,
                        "delta": event.delta,
                    }),
                }
                continue

            if isinstance(event, ToolResultEndEvent):
                if reply_msg is not None:
                    reply_msg.append_event(event)
                yield {
                    "event": "tool_result",
                    "data": json.dumps({
                        "tool_call_id": event.tool_call_id,
                        "status": event.state,
                    }),
                }
                continue

            # Model call events
            if isinstance(event, ModelCallEndEvent):
                if reply_msg is not None:
                    reply_msg.append_event(event)
                yield {
                    "event": "model_call_end",
                    "data": json.dumps({
                        "input_tokens": event.input_tokens,
                        "output_tokens": event.output_tokens,
                    }),
                }
                continue

            # Hint events
            if isinstance(event, HintBlockEvent):
                if reply_msg is not None:
                    reply_msg.append_event(event)
                hint_val = event.hint if isinstance(event.hint, str) else json.dumps(event.hint)
                yield {
                    "event": "hint",
                    "data": json.dumps({"hint": hint_val, "source": event.source}),
                }
                continue

            # Exceed max iterations
            if isinstance(event, ExceedMaxItersEvent):
                yield {
                    "event": "error",
                    "data": json.dumps({"error": "exceeded_max_iterations", "name": event.name}),
                }
                continue

            # Require user confirmation (interrupt)
            if isinstance(event, RequireUserConfirmEvent):
                yield {
                    "event": "interrupt",
                    "data": json.dumps({
                        "type": "require_user_confirm",
                        "tool_calls": event.tool_calls,
                    }),
                }
                continue

            # Fallback: try delta attribute (legacy compatibility)
            delta = getattr(event, "delta", None)
            if delta:
                full_text += delta
                yield {
                    "event": "text",
                    "data": json.dumps({"delta": delta}),
                }
                continue

            data = getattr(event, "data", None)
            if data:
                yield {"event": "text", "data": json.dumps({"delta": str(data)})}

        # Store assistant message
        if reply_msg is not None:
            await _memory.append_msg(session_id, reply_msg)
        elif full_text:
            reply_msg = AssistantMsg(name=agent.name, content=full_text)
            await _memory.append_msg(session_id, reply_msg)
        else:
            reply_text = await agent.run(message)
            reply_msg = AssistantMsg(name=agent.name, content=reply_text)
            await _memory.append_msg(session_id, reply_msg)
            yield {"event": "text", "data": json.dumps({"delta": reply_text})}
            yield {"event": "done", "data": json.dumps({"finish_reason": "stop"})}

    except Exception as exc:
        _ERROR_COUNT += 1
        logger.exception("Agent %s failed", agent.name)
        yield {
            "event": "error",
            "data": json.dumps({"error": str(exc)}),
        }


@app.post("/api/v2/chat/stream")
async def chat_stream(req: ChatRequest, request: Request) -> EventSourceResponse:
    agent = _agents.get(req.agent_id)
    if agent is None:
        raise HTTPException(status_code=404, detail=f"Agent {req.agent_id!r} not found")

    session_id = req.session_id or uuid.uuid4().hex

    return EventSourceResponse(
        _stream_agent(agent, req.message, session_id),
        media_type="text/event-stream",
    )


@app.post("/api/v2/chat/resume")
async def chat_resume(req: ChatResumeRequest) -> JSONResponse:
    agent = _agents.get(req.agent_id)
    if agent is None:
        raise HTTPException(status_code=404, detail=f"Agent {req.agent_id!r} not found")

    try:
        result = await agent.resume(req.resume_payload)
    except NotImplementedError:
        raise HTTPException(status_code=400, detail=f"Agent {req.agent_id} does not support resume")

    return JSONResponse(content={
        "agent_id": req.agent_id,
        "session_id": req.session_id,
        "status": "resumed",
        "result": result,
    })


@app.post("/api/v2/admin/reload")
async def admin_reload() -> JSONResponse:
    count = _reload_agents()
    return JSONResponse(content={"status": "ok", "agents_loaded": count})


# ──────────────────────────────────────────────
# NEW: System endpoints
# ──────────────────────────────────────────────

@app.get("/ready")
async def ready() -> dict[str, Any]:
    """Readiness probe — returns ok when agents are loaded."""
    return _ok({"ready": len(_agents) > 0, "agents_loaded": len(_agents)})


@app.get("/api/v2/me")
async def me() -> dict[str, Any]:
    """Return current user / service identity."""
    return _ok({
        "user_id": "python-base",
        "name": "Superagent Python Base",
        "version": "0.1.0",
        "roles": ["admin"],
    })


@app.get("/api/v2/admin/status")
async def admin_status() -> dict[str, Any]:
    """System status overview — flat format matching Go/Java contract."""
    return {
        "status": "running",
        "version": "0.1.0",
        "uptime_seconds": int(time.time() - _start_time),
        "agents_loaded": len(_agents),
        "agent_names": list(_agents.keys()),
        "requests_total": _REQUEST_COUNT,
        "errors_total": _ERROR_COUNT,
        "conversations_count": len(_conversations),
        "files_count": len(_files),
        "memory_entries": len(_long_term_memory),
    }


@app.get("/api/v2/admin/agents")
async def admin_list_agents() -> dict[str, Any]:
    """Admin agent list — mirrors /api/v2/agents for v1 compatibility."""
    agent_list = [
        {"name": a.name, "type": getattr(a, "agent_type", "unknown"),
         "description": getattr(a, "description", "")}
        for a in _agents.values()
    ]
    return {"agents": agent_list, "count": len(agent_list)}


@app.post("/api/v2/admin/agents/validate")
async def admin_validate_agent(request: Request) -> dict[str, Any]:
    """Validate agent YAML definition."""
    try:
        body = await request.json()
        yaml_def = body.get("yaml_definition", "") or body.get("yaml", "")
        if not yaml_def.strip():
            return {"valid": False, "error": "yaml_definition is required"}
        import yaml as _yaml
        data = _yaml.safe_load(yaml_def)
        if not data or "metadata" not in data:
            return {"valid": False, "error": "Missing metadata section"}
        name = data.get("metadata", {}).get("name", "")
        agent_type = data.get("spec", {}).get("type", "")
        return {"valid": True, "name": name, "type": agent_type}
    except Exception as e:
        return {"valid": False, "error": str(e)}


# ──────────────────────────────────────────────
# NEW: Core Chat — interrupt_state / abort
# ──────────────────────────────────────────────

@app.get("/api/v2/chat/interrupt_state")
async def chat_interrupt_state(
    session_id: str = Query(..., description="Session ID"),
) -> dict[str, Any]:
    """Return interrupt state for a session, if any."""
    state = _interrupt_states.get(session_id)
    if state is None:
        return _ok(None, msg="no interrupt state")
    return _ok(state)


@app.post("/api/v2/chat/abort")
async def chat_abort(req: AbortRequest) -> dict[str, Any]:
    """Request abort of an in-flight chat session."""
    _interrupt_states.pop(req.session_id, None)
    return _ok({"session_id": req.session_id, "status": "abort_requested"})


# ──────────────────────────────────────────────
# NEW: Agent state CRUD
# ──────────────────────────────────────────────

@app.get("/api/v2/agents/{agent_id}/state")
async def get_agent_state(agent_id: str) -> dict[str, Any]:
    """Return all state key-value pairs for an agent."""
    if agent_id not in _agents:
        raise HTTPException(status_code=404, detail=f"Agent {agent_id!r} not found")
    state = _agent_states.get(agent_id, {})
    return _ok(state)


@app.get("/api/v2/agents/{agent_id}/state/{key}")
async def get_agent_state_key(agent_id: str, key: str) -> dict[str, Any]:
    """Return a single state value by key."""
    if agent_id not in _agents:
        raise HTTPException(status_code=404, detail=f"Agent {agent_id!r} not found")
    state = _agent_states.get(agent_id, {})
    if key not in state:
        raise HTTPException(status_code=404, detail=f"Key {key!r} not found")
    return _ok({key: state[key]})


@app.post("/api/v2/agents/{agent_id}/state")
async def set_agent_state(agent_id: str, req: AgentStateSetRequest) -> dict[str, Any]:
    """Set a state key-value pair for an agent."""
    if agent_id not in _agents:
        raise HTTPException(status_code=404, detail=f"Agent {agent_id!r} not found")
    _agent_states.setdefault(agent_id, {})[req.key] = req.value
    return _ok({req.key: req.value})


@app.delete("/api/v2/agents/{agent_id}/state/{key}")
async def delete_agent_state_key(agent_id: str, key: str) -> dict[str, Any]:
    """Delete a state key for an agent."""
    if agent_id not in _agents:
        raise HTTPException(status_code=404, detail=f"Agent {agent_id!r} not found")
    state = _agent_states.get(agent_id, {})
    if key not in state:
        raise HTTPException(status_code=404, detail=f"Key {key!r} not found")
    del state[key]
    return _ok({"deleted": key})


# ──────────────────────────────────────────────
# NEW: Sessions
# ──────────────────────────────────────────────

@app.get("/api/v2/sessions/{session_id}/messages")
async def get_session_messages(session_id: str) -> dict[str, Any]:
    """Return message history for a session."""
    msgs = await _memory.get(session_id)
    return _ok({"messages": msgs})


@app.delete("/api/v2/sessions/{session_id}")
async def delete_session(session_id: str) -> dict[str, Any]:
    """Clear / delete a session and its messages."""
    await _memory.clear(session_id)
    _interrupt_states.pop(session_id, None)
    return _ok({"deleted": session_id})


# ──────────────────────────────────────────────
# NEW: Conversations CRUD + messages
# ──────────────────────────────────────────────

@app.post("/api/v2/conversations")
async def create_conversation(req: ConversationCreateRequest) -> dict[str, Any]:
    """Create a new conversation."""
    conv_id = uuid.uuid4().hex
    conv = {
        "id": conv_id,
        "title": req.title,
        "agent_id": req.agent_id,
        "metadata": req.metadata,
        "message_count": 0,
        "created_at": time.time(),
        "updated_at": time.time(),
    }
    _conversations[conv_id] = conv
    return _ok(conv)


@app.get("/api/v2/conversations/{conversation_id}/messages")
async def get_conversation_messages(conversation_id: str) -> dict[str, Any]:
    """Return messages for a conversation."""
    msgs = await _memory.get(conversation_id)
    return _ok({"messages": msgs, "total": len(msgs)})


@app.delete("/api/v2/conversations/{conversation_id}/messages/{message_id}")
async def delete_conversation_message(conversation_id: str, message_id: str) -> dict[str, Any]:
    """Delete a specific message from a conversation (stub)."""
    return _ok({"deleted": message_id, "conversation_id": conversation_id})


@app.get("/api/v2/conversations/{conversation_id}")
async def get_conversation(conversation_id: str) -> dict[str, Any]:
    """Get a single conversation by ID."""
    conv = _conversations.get(conversation_id)
    if conv is None:
        # Also check memory-backed conversations
        msgs = await _memory.get(conversation_id)
        if msgs is not None:
            return _ok({"id": conversation_id, "message_count": len(msgs)})
        raise HTTPException(status_code=404, detail=f"Conversation {conversation_id!r} not found")
    return _ok(conv)


@app.put("/api/v2/conversations/{conversation_id}")
async def update_conversation(conversation_id: str, req: ConversationUpdateRequest) -> dict[str, Any]:
    """Update conversation metadata."""
    conv = _conversations.get(conversation_id)
    if conv is None:
        raise HTTPException(status_code=404, detail=f"Conversation {conversation_id!r} not found")
    if req.title is not None:
        conv["title"] = req.title
    if req.metadata is not None:
        conv["metadata"] = req.metadata
    conv["updated_at"] = time.time()
    return _ok(conv)


@app.delete("/api/v2/conversations/{conversation_id}")
async def delete_conversation(conversation_id: str) -> dict[str, Any]:
    """Delete a conversation."""
    _conversations.pop(conversation_id, None)
    await _memory.clear(conversation_id)
    return _ok({"deleted": conversation_id})


# ──────────────────────────────────────────────
# NEW: Files CRUD
# ──────────────────────────────────────────────

@app.post("/api/v2/files")
async def upload_file(file: UploadFile = File(...)) -> dict[str, Any]:
    """Upload a file and return its metadata."""
    file_id = uuid.uuid4().hex
    content = await file.read()
    entry = {
        "id": file_id,
        "filename": file.filename or "unnamed",
        "content_type": file.content_type or "application/octet-stream",
        "size": len(content),
        "created_at": time.time(),
    }
    _files[file_id] = {**entry, "_content": content}
    return _ok(entry)


@app.get("/api/v2/files")
async def list_files() -> dict[str, Any]:
    """List all uploaded files (metadata only)."""
    items = []
    for f in _files.values():
        items.append({k: v for k, v in f.items() if not k.startswith("_")})
    return _ok(items)


@app.get("/api/v2/files/{file_id}")
async def get_file(file_id: str) -> dict[str, Any]:
    """Get file metadata by ID."""
    entry = _files.get(file_id)
    if entry is None:
        raise HTTPException(status_code=404, detail=f"File {file_id!r} not found")
    return _ok({k: v for k, v in entry.items() if not k.startswith("_")})


@app.get("/api/v2/files/{file_id}/content")
async def get_file_content(file_id: str) -> Response:
    """Download raw file content."""
    entry = _files.get(file_id)
    if entry is None:
        raise HTTPException(status_code=404, detail=f"File {file_id!r} not found")
    return Response(
        content=entry.get("_content", b""),
        media_type=entry.get("content_type", "application/octet-stream"),
        headers={"Content-Disposition": f'attachment; filename="{entry["filename"]}"'},
    )


@app.delete("/api/v2/files/{file_id}")
async def delete_file(file_id: str) -> dict[str, Any]:
    """Delete a file."""
    if file_id not in _files:
        raise HTTPException(status_code=404, detail=f"File {file_id!r} not found")
    del _files[file_id]
    return _ok({"deleted": file_id})


# ──────────────────────────────────────────────
# NEW: Long-term Memory CRUD + search
# ──────────────────────────────────────────────

@app.get("/api/v2/memory/long-term")
async def list_long_term_memory(
    user_id: str = Query("", description="Filter by user ID"),
    limit: int = Query(50, ge=1, le=500),
    offset: int = Query(0, ge=0),
) -> dict[str, Any]:
    """List long-term memory entries with pagination."""
    entries = _long_term_memory
    if user_id:
        entries = [m for m in entries if m.get("user_id") == user_id]
    total = len(entries)
    page = entries[offset: offset + limit]
    return _ok({"memories": page, "total": total, "limit": limit, "offset": offset})


@app.post("/api/v2/memory/long-term")
async def create_long_term_memory(req: MemoryCreateRequest) -> dict[str, Any]:
    """Add a new long-term memory entry."""
    entry = {
        "id": uuid.uuid4().hex,
        "user_id": req.user_id,
        "content": req.content,
        "metadata": req.metadata,
        "created_at": time.time(),
        "updated_at": time.time(),
    }
    _long_term_memory.append(entry)
    return _ok(entry)


@app.get("/api/v2/memory/long-term/search")
async def search_long_term_memory(
    user_id: str = Query("", description="Filter by user ID"),
    q: str = Query("", description="Search query"),
) -> dict[str, Any]:
    """Search long-term memory by keyword (simple substring match)."""
    entries = _long_term_memory
    if user_id:
        entries = [m for m in entries if m.get("user_id") == user_id]
    if q:
        q_lower = q.lower()
        entries = [m for m in entries if q_lower in m.get("content", "").lower()]
    return _ok({"memories": entries, "total": len(entries)})


@app.put("/api/v2/memory/long-term/{memory_id}")
async def update_long_term_memory(memory_id: str, req: MemoryUpdateRequest) -> dict[str, Any]:
    """Update a memory entry."""
    for m in _long_term_memory:
        if m["id"] == memory_id:
            if req.content is not None:
                m["content"] = req.content
            if req.metadata is not None:
                m["metadata"] = req.metadata
            m["updated_at"] = time.time()
            return _ok(m)
    raise HTTPException(status_code=404, detail=f"Memory {memory_id!r} not found")


@app.delete("/api/v2/memory/long-term/{memory_id}")
async def delete_long_term_memory(memory_id: str) -> dict[str, Any]:
    """Delete a memory entry."""
    for i, m in enumerate(_long_term_memory):
        if m["id"] == memory_id:
            _long_term_memory.pop(i)
            return _ok({"deleted": memory_id})
    raise HTTPException(status_code=404, detail=f"Memory {memory_id!r} not found")


# ──────────────────────────────────────────────
# NEW: Workflows
# ──────────────────────────────────────────────

@app.post("/api/v2/workflows/run")
async def workflow_run(req: WorkflowRunRequest) -> dict[str, Any]:
    """Execute a workflow synchronously (stub)."""
    return _ok({
        "workflow_id": req.workflow_id,
        "status": "completed",
        "outputs": {},
        "execution_time_ms": 0,
    })


async def _stream_workflow_stub(workflow_id: str) -> AsyncIterator[dict[str, Any]]:
    """Stub SSE stream for workflow execution."""
    yield {
        "event": "progress",
        "data": json.dumps({"workflow_id": workflow_id, "status": "started"}),
    }
    yield {
        "event": "done",
        "data": json.dumps({"workflow_id": workflow_id, "status": "completed"}),
    }


@app.post("/api/v2/workflows/stream_run")
async def workflow_stream_run(req: WorkflowStreamRunRequest) -> EventSourceResponse:
    """Execute a workflow with SSE streaming (stub)."""
    return EventSourceResponse(
        _stream_workflow_stub(req.workflow_id),
        media_type="text/event-stream",
    )


@app.post("/api/v2/workflows/stream_resume")
async def workflow_stream_resume(req: WorkflowStreamResumeRequest) -> EventSourceResponse:
    """Resume a paused workflow with SSE streaming (stub)."""
    return EventSourceResponse(
        _stream_workflow_stub(req.workflow_id),
        media_type="text/event-stream",
    )


@app.post("/api/v2/workflows/chat")
async def workflow_chat(req: WorkflowChatRequest) -> dict[str, Any]:
    """Run a workflow in chat mode (stub)."""
    return _ok({
        "workflow_id": req.workflow_id,
        "session_id": req.session_id or uuid.uuid4().hex,
        "reply": "Workflow chat response stub",
    })


@app.get("/api/v2/workflows/{workflow_id}")
async def get_workflow(workflow_id: str) -> dict[str, Any]:
    """Get workflow definition / status (stub)."""
    return _ok({
        "workflow_id": workflow_id,
        "name": workflow_id,
        "status": "ready",
        "nodes": [],
        "edges": [],
    })


# ──────────────────────────────────────────────
# NEW: Skills & Tools
# ──────────────────────────────────────────────

@app.get("/api/v2/skills")
async def list_skills() -> dict[str, Any]:
    """List installed skills."""
    installed = _skill_manager.list_installed()
    return _ok([{"name": i.meta.name, "description": i.meta.description, "version": i.meta.version, "source": i.source} for i in installed])


@app.get("/api/v2/skills/search")
async def search_skills(q: str = Query("", description="Search query")) -> dict[str, Any]:
    """Search available skills by name or description."""
    results = await _skill_manager.search(q)
    return _ok([{"name": m.name, "description": m.description, "version": m.version} for m in results])


@app.get("/api/v2/tools")
async def list_tools() -> dict[str, Any]:
    """List all registered tools."""
    return _ok(_tools)


# ──────────────────────────────────────────────
# NEW: MCP Servers
# ──────────────────────────────────────────────

@app.get("/api/v2/mcp/servers")
async def list_mcp_servers() -> dict[str, Any]:
    """List all registered MCP servers."""
    servers = []
    for name in _mcp_registry.list_servers():
        client = _mcp_registry.get_client(name)
        servers.append({
            "name": name,
            "connected": client.is_connected if client else False,
        })
    return _ok(servers)


# ──────────────────────────────────────────────
# Task #3: Missing endpoints
# ──────────────────────────────────────────────

@app.get("/api/v2/admin/mcp/servers")
async def admin_list_mcp_servers() -> dict[str, Any]:
    """Admin MCP server list — mirrors /api/v2/mcp/servers for path compatibility."""
    servers = []
    for name in _mcp_registry.list_servers():
        client = _mcp_registry.get_client(name)
        servers.append({
            "name": name,
            "connected": client.is_connected if client else False,
        })
    return _ok(servers)


@app.get("/api/v2/admin/agents/{agent_id}")
async def admin_get_agent(agent_id: str) -> dict[str, Any]:
    agent = _agents.get(agent_id)
    if agent is None:
        raise HTTPException(status_code=404, detail=f"Agent {agent_id!r} not found")
    return {
        "name": agent.name,
        "type": getattr(agent, "agent_type", "unknown"),
        "description": getattr(agent, "description", ""),
        "tools": getattr(agent, "tools", []),
    }


_log_queue: asyncio.Queue = asyncio.Queue(maxsize=100)


@app.get("/api/v2/admin/logs")
async def admin_logs() -> EventSourceResponse:
    """SSE stream of structured log events."""
    async def generator():
        yield json.dumps({"level": "INFO", "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ"), "message": "Connected to log stream"})
        while True:
            try:
                msg = await asyncio.wait_for(_log_queue.get(), timeout=30.0)
                yield json.dumps(msg)
            except asyncio.TimeoutError:
                yield json.dumps({"level": "INFO", "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ"), "message": "heartbeat"})
    return EventSourceResponse(generator())


@app.post("/api/v2/skills/install")
async def install_skill(request: Request) -> dict[str, Any]:
    body = await request.json()
    name = body.get("name", "")
    return {"status": "ok", "name": name, "installed": True, "message": "skill install stub"}


@app.delete("/api/v2/skills/{name}")
async def uninstall_skill(name: str) -> dict[str, Any]:
    return {"status": "ok", "name": name, "uninstalled": True, "message": "skill uninstall stub"}


# ──────────────────────────────────────────────
# Task #5: Admin CRUD stubs + MCP management stubs
# ──────────────────────────────────────────────

@app.post("/api/v2/admin/agents")
async def admin_create_agent(request: Request) -> dict[str, Any]:
    body = await request.json()
    return {"status": "stub", "message": "Agent creation not yet implemented", "yaml_definition": body.get("yaml_definition", "")}


@app.put("/api/v2/admin/agents/{agent_id}")
async def admin_update_agent(agent_id: str, request: Request) -> dict[str, Any]:
    return {"status": "stub", "name": agent_id, "message": "Agent update not yet implemented"}


@app.delete("/api/v2/admin/agents/{agent_id}")
async def admin_delete_agent(agent_id: str) -> dict[str, Any]:
    return {"status": "stub", "name": agent_id, "message": "Agent deletion not yet implemented"}


@app.post("/api/v2/admin/mcp/servers")
async def admin_connect_mcp(request: Request) -> dict[str, Any]:
    body = await request.json()
    name = body.get("name", "")
    return {"status": "ok", "name": name, "connected": False, "message": "MCP connect stub"}


@app.delete("/api/v2/admin/mcp/servers/{name}")
async def admin_disconnect_mcp(name: str) -> dict[str, Any]:
    return {"status": "ok", "name": name, "disconnected": True}


@app.get("/api/v2/admin/mcp/servers/{name}/tools")
async def admin_mcp_server_tools(name: str) -> dict[str, Any]:
    return {"server": name, "tools": [], "count": 0}
