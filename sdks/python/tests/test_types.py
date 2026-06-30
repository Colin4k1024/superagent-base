"""Tests for superagent.types module."""

import json

import pytest

from superagent.types import (
    A2UIEvent,
    A2UIEventType,
    AgentInfo,
    ApplyResult,
    ChatRequest,
    ValidateResult,
)


# ---------------------------------------------------------------------------
# A2UIEventType
# ---------------------------------------------------------------------------


class TestA2UIEventType:
    """A2UIEventType is a str Enum with the expected members."""

    def test_known_types(self) -> None:
        expected = {
            "text",
            "thinking",
            "tool_call",
            "tool_result",
            "code_block",
            "interrupt",
            "error",
            "done",
            "progress",
            "agent_switch",
        }
        actual = {m.value for m in A2UIEventType}
        assert actual == expected

    def test_is_str_subclass(self) -> None:
        assert issubclass(A2UIEventType, str)
        assert A2UIEventType.text == "text"

    def test_member_count(self) -> None:
        assert len(A2UIEventType) == 10


# ---------------------------------------------------------------------------
# A2UIEvent
# ---------------------------------------------------------------------------


class TestA2UIEvent:
    """Tests for the A2UIEvent Pydantic model."""

    def test_basic_construction(self) -> None:
        evt = A2UIEvent(event_type="text", data={"delta": "hello"})
        assert evt.event_type == "text"
        assert evt.data["delta"] == "hello"

    def test_default_data_is_empty_dict(self) -> None:
        evt = A2UIEvent(event_type="done")
        assert evt.data == {}

    def test_type_property_known_enum(self) -> None:
        evt = A2UIEvent(event_type="text")
        assert evt.type == A2UIEventType.text

    def test_type_property_unknown_string(self) -> None:
        evt = A2UIEvent(event_type="custom_future_type")
        assert evt.type == "custom_future_type"
        assert not isinstance(evt.type, A2UIEventType)

    def test_text_delta_from_delta_key(self) -> None:
        evt = A2UIEvent(event_type="text", data={"delta": "hi"})
        assert evt.text_delta == "hi"

    def test_text_delta_from_content_key(self) -> None:
        evt = A2UIEvent(event_type="text", data={"content": "world"})
        assert evt.text_delta == "world"

    def test_text_delta_prefers_delta_over_content(self) -> None:
        evt = A2UIEvent(event_type="text", data={"delta": "d", "content": "c"})
        assert evt.text_delta == "d"

    def test_text_delta_empty_when_no_keys(self) -> None:
        evt = A2UIEvent(event_type="text", data={})
        assert evt.text_delta == ""

    def test_text_delta_falsy_delta_falls_through(self) -> None:
        """When delta is empty string, content is used."""
        evt = A2UIEvent(event_type="text", data={"delta": "", "content": "fallback"})
        assert evt.text_delta == "fallback"

    def test_serialization_roundtrip(self) -> None:
        evt = A2UIEvent(event_type="tool_call", data={"name": "search", "args": {}})
        d = evt.model_dump()
        evt2 = A2UIEvent(**d)
        assert evt2.event_type == evt.event_type
        assert evt2.data == evt.data

    def test_json_roundtrip(self) -> None:
        evt = A2UIEvent(event_type="text", data={"delta": "hello"})
        j = evt.model_dump_json()
        evt2 = A2UIEvent.model_validate_json(j)
        assert evt2.event_type == "text"
        assert evt2.data["delta"] == "hello"


# ---------------------------------------------------------------------------
# AgentInfo
# ---------------------------------------------------------------------------


class TestAgentInfo:
    """Tests for the AgentInfo Pydantic model."""

    def test_full_construction(self) -> None:
        info = AgentInfo(name="research", type="chat_model_agent", description="Research agent", status="active")
        assert info.name == "research"
        assert info.type == "chat_model_agent"
        assert info.description == "Research agent"
        assert info.status == "active"

    def test_defaults(self) -> None:
        info = AgentInfo(name="minimal")
        assert info.type == ""
        assert info.description == ""
        assert info.status == ""

    def test_serialization(self) -> None:
        info = AgentInfo(name="test", type="supervisor")
        d = info.model_dump()
        assert d == {"name": "test", "type": "supervisor", "description": "", "status": ""}

    def test_from_dict(self) -> None:
        data = {"name": "a", "type": "t", "description": "d", "status": "s"}
        info = AgentInfo(**data)
        assert info.name == "a"


# ---------------------------------------------------------------------------
# ChatRequest
# ---------------------------------------------------------------------------


class TestChatRequest:
    """Tests for the ChatRequest Pydantic model."""

    def test_defaults(self) -> None:
        req = ChatRequest(agent_id="agent", message="hello")
        assert req.session_id == "default"
        assert req.stream is True

    def test_custom_session(self) -> None:
        req = ChatRequest(agent_id="a", message="m", session_id="s1")
        assert req.session_id == "s1"

    def test_non_streaming(self) -> None:
        req = ChatRequest(agent_id="a", message="m", stream=False)
        assert req.stream is False

    def test_serialization(self) -> None:
        req = ChatRequest(agent_id="a", message="m")
        d = req.model_dump()
        assert d["agent_id"] == "a"
        assert d["message"] == "m"
        assert d["session_id"] == "default"
        assert d["stream"] is True


# ---------------------------------------------------------------------------
# ApplyResult
# ---------------------------------------------------------------------------


class TestApplyResult:
    """Tests for the ApplyResult Pydantic model."""

    def test_construction(self) -> None:
        r = ApplyResult(name="agent-x", status="created", message="OK")
        assert r.name == "agent-x"
        assert r.status == "created"
        assert r.message == "OK"

    def test_default_message(self) -> None:
        r = ApplyResult(name="a", status="updated")
        assert r.message == ""

    def test_from_json(self) -> None:
        data = {"name": "a", "status": "unchanged"}
        r = ApplyResult(**data)
        assert r.status == "unchanged"


# ---------------------------------------------------------------------------
# ValidateResult
# ---------------------------------------------------------------------------


class TestValidateResult:
    """Tests for the ValidateResult Pydantic model."""

    def test_valid_result(self) -> None:
        r = ValidateResult(valid=True)
        assert r.valid is True
        assert r.errors == []
        assert r.warnings == []

    def test_invalid_with_errors(self) -> None:
        r = ValidateResult(valid=False, errors=["missing name", "bad type"])
        assert r.valid is False
        assert len(r.errors) == 2

    def test_with_warnings(self) -> None:
        r = ValidateResult(valid=True, warnings=["deprecated field"])
        assert r.warnings == ["deprecated field"]

    def test_serialization_roundtrip(self) -> None:
        r = ValidateResult(valid=False, errors=["e1"], warnings=["w1"])
        d = r.model_dump()
        r2 = ValidateResult(**d)
        assert r2.valid == r.valid
        assert r2.errors == r.errors
        assert r2.warnings == r.warnings
