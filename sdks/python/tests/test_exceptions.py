"""Tests for superagent.exceptions module."""

from superagent.exceptions import (
    AuthenticationError,
    InterruptConflictError,
    NotFoundError,
    RateLimitError,
    ServerError,
    StreamDisconnectedError,
    SuperagentError,
    ValidationError,
)


class TestSuperagentError:
    """Tests for the base SuperagentError class."""

    def test_init_with_all_params(self) -> None:
        err = SuperagentError("something failed", status_code=500, code="ERR_500")
        assert err.message == "something failed"
        assert err.status_code == 500
        assert err.code == "ERR_500"
        assert str(err) == "something failed"

    def test_init_defaults(self) -> None:
        err = SuperagentError("oops")
        assert err.message == "oops"
        assert err.status_code == 0
        assert err.code == ""

    def test_repr(self) -> None:
        err = SuperagentError("bad", status_code=400, code="BAD")
        r = repr(err)
        assert "SuperagentError" in r
        assert "status_code=400" in r
        assert "code='BAD'" in r
        assert "message='bad'" in r

    def test_is_exception(self) -> None:
        err = SuperagentError("test")
        assert isinstance(err, Exception)

    def test_can_be_caught(self) -> None:
        try:
            raise ServerError("down", status_code=503, code="SVC")
        except SuperagentError as e:
            assert e.status_code == 503
        else:
            raise AssertionError("Should have been caught")


class TestSubclassHierarchy:
    """All SDK exceptions inherit from SuperagentError."""

    _subclasses = (
        AuthenticationError,
        NotFoundError,
        ValidationError,
        RateLimitError,
        ServerError,
        StreamDisconnectedError,
        InterruptConflictError,
    )

    def test_all_subclasses_inherit_base(self) -> None:
        for cls in self._subclasses:
            assert issubclass(cls, SuperagentError), f"{cls.__name__} should inherit SuperagentError"

    def test_instantiate_each(self) -> None:
        for cls in self._subclasses:
            err = cls("test message")
            assert err.message == "test message"
            assert isinstance(err, SuperagentError)


class TestStreamDisconnectedError:
    """Tests for StreamDisconnectedError defaults."""

    def test_default_message(self) -> None:
        err = StreamDisconnectedError()
        assert err.message == "SSE stream disconnected unexpectedly"
        assert err.status_code == 0
        assert err.code == "stream_disconnected"

    def test_custom_message(self) -> None:
        err = StreamDisconnectedError("connection reset")
        assert err.message == "connection reset"
        assert err.code == "stream_disconnected"


class TestInterruptConflictError:
    """Tests for InterruptConflictError defaults."""

    def test_default_message(self) -> None:
        err = InterruptConflictError()
        assert err.message == "No pending interrupt for this session"
        assert err.status_code == 409
        assert err.code == "interrupt_conflict"

    def test_custom_message(self) -> None:
        err = InterruptConflictError("already resumed")
        assert err.message == "already resumed"
        assert err.status_code == 409


class TestCatchByType:
    """Verify exceptions can be caught by their specific type."""

    def test_auth_error(self) -> None:
        try:
            raise AuthenticationError("unauthorized", status_code=401, code="AUTH")
        except AuthenticationError as e:
            assert e.status_code == 401

    def test_not_found(self) -> None:
        try:
            raise NotFoundError("not there", status_code=404, code="NF")
        except NotFoundError as e:
            assert e.status_code == 404

    def test_validation(self) -> None:
        try:
            raise ValidationError("bad input", status_code=422, code="VAL")
        except ValidationError as e:
            assert e.status_code == 422

    def test_rate_limit(self) -> None:
        try:
            raise RateLimitError("slow down", status_code=429, code="RL")
        except RateLimitError as e:
            assert e.status_code == 429

    def test_server_error(self) -> None:
        try:
            raise ServerError("boom", status_code=500, code="SVC")
        except ServerError as e:
            assert e.status_code == 500
