"""builtin/code_execute — code execution via subprocess sandbox.

Provides both a plain async function (backward-compatible) and a
``ToolBase`` subclass (``CodeExecute``) for use with AgentScope 2.0 toolkits.
"""

from __future__ import annotations

import asyncio
import json
import logging
import tempfile
from pathlib import Path
from typing import Any

from agentscope.message import TextBlock
from agentscope.permission import PermissionBehavior, PermissionContext, PermissionDecision
from agentscope.tool import ToolBase, ToolChunk

logger = logging.getLogger(__name__)

_LANGUAGE_COMMANDS: dict[str, list[str]] = {
    "python": ["python3", "-c"],
    "python3": ["python3", "-c"],
    "bash": ["bash", "-c"],
    "sh": ["sh", "-c"],
    "node": ["node", "-e"],
}

_DEFAULT_TIMEOUT = 30
_MAX_OUTPUT = 100_000


# ---------------------------------------------------------------------------
# Plain async function — backward-compatible, used by tests
# ---------------------------------------------------------------------------

async def code_execute(
    code: str,
    language: str = "python",
    timeout: int = _DEFAULT_TIMEOUT,
    **kwargs: Any,
) -> dict[str, Any]:
    """Execute code in a subprocess with timeout.

    Supported languages: python, python3, bash, sh, node.
    Output (stdout + stderr) is truncated to 100 KB.
    """
    lang = language.lower()
    cmd_parts = _LANGUAGE_COMMANDS.get(lang)
    if cmd_parts is None:
        return {
            "exit_code": -1,
            "stdout": "",
            "stderr": f"[error] Unsupported language: {language}. Supported: {', '.join(_LANGUAGE_COMMANDS)}",
        }

    cmd = cmd_parts + [code]

    try:
        proc = await asyncio.create_subprocess_exec(
            *cmd,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        try:
            stdout_bytes, stderr_bytes = await asyncio.wait_for(
                proc.communicate(), timeout=timeout
            )
        except asyncio.TimeoutError:
            proc.kill()
            await proc.wait()
            return {
                "exit_code": -1,
                "stdout": "",
                "stderr": f"[timeout] Execution timed out after {timeout}s",
            }

        stdout = stdout_bytes.decode("utf-8", errors="replace")[:_MAX_OUTPUT]
        stderr = stderr_bytes.decode("utf-8", errors="replace")[:_MAX_OUTPUT]

        return {
            "exit_code": proc.returncode or 0,
            "stdout": stdout,
            "stderr": stderr,
        }
    except FileNotFoundError:
        return {
            "exit_code": -1,
            "stdout": "",
            "stderr": f"[error] Interpreter not found: {cmd_parts[0]}",
        }
    except Exception as exc:
        logger.exception("code_execute failed for language=%s", language)
        return {
            "exit_code": -1,
            "stdout": "",
            "stderr": f"[error] {exc}",
        }


# ---------------------------------------------------------------------------
# AgentScope 2.0 ToolBase subclass
# ---------------------------------------------------------------------------

class CodeExecute(ToolBase):
    """AgentScope tool for executing code in a subprocess sandbox."""

    name = "CodeExecute"
    description = "Execute code in a sandboxed subprocess. Supported languages: python, python3, bash, sh, node. Returns exit_code, stdout, and stderr."
    input_schema = {
        "type": "object",
        "properties": {
            "code": {
                "type": "string",
                "description": "The code to execute.",
            },
            "language": {
                "type": "string",
                "description": "Programming language (default python). Supported: python, python3, bash, sh, node.",
                "default": "python",
            },
            "timeout": {
                "type": "integer",
                "description": "Execution timeout in seconds (default 30).",
                "default": 30,
            },
        },
        "required": ["code"],
    }
    is_concurrency_safe = False
    is_read_only = False

    async def check_permissions(
        self, tool_input: dict[str, Any], context: PermissionContext
    ) -> PermissionDecision:
        return PermissionDecision(behavior=PermissionBehavior.ALLOW, message="Code execution is allowed")

    async def __call__(self, code: str, language: str = "python", timeout: int = _DEFAULT_TIMEOUT) -> ToolChunk:
        result = await code_execute(code, language=language, timeout=timeout)
        return ToolChunk(content=[TextBlock(text=json.dumps(result, ensure_ascii=False))])
