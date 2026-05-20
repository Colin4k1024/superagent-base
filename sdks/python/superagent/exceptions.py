"""Superagent SDK exceptions."""

from __future__ import annotations


class SuperagentError(Exception):
    """Base exception for all Superagent SDK errors."""

    def __init__(self, message: str, status_code: int = 0, code: str = "") -> None:
        super().__init__(message)
        self.message = message
        self.status_code = status_code
        self.code = code

    def __repr__(self) -> str:
        return f"{type(self).__name__}(status_code={self.status_code}, code={self.code!r}, message={self.message!r})"


class AuthenticationError(SuperagentError):
    """Raised when authentication fails (HTTP 401 / 403)."""


class NotFoundError(SuperagentError):
    """Raised when a resource is not found (HTTP 404)."""


class ValidationError(SuperagentError):
    """Raised when request validation fails (HTTP 422)."""


class RateLimitError(SuperagentError):
    """Raised when the server rate-limits the client (HTTP 429)."""


class ServerError(SuperagentError):
    """Raised when the server returns a 5xx response after all retries."""


class StreamDisconnectedError(SuperagentError):
    """Raised when the SSE stream disconnects unexpectedly."""

    def __init__(self, message: str = "SSE stream disconnected unexpectedly") -> None:
        super().__init__(message, status_code=0, code="stream_disconnected")


class InterruptConflictError(SuperagentError):
    """Raised when a resume is attempted while no interrupt is pending."""

    def __init__(self, message: str = "No pending interrupt for this session") -> None:
        super().__init__(message, status_code=409, code="interrupt_conflict")
