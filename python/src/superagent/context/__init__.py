"""Context injection — inject metadata into agent prompts.

Mirrors Go base ``pkg/agentdef/mw_context_injection.go``.
"""

from superagent.context.injection import ContextInjectionMiddleware

__all__ = ["ContextInjectionMiddleware"]
