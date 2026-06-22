"""AgentLoop agent — runs a child agent in a loop until it signals completion.

Mirrors Go base ``pkg/agentdef/agentloop.go``:
- Calls the child agent repeatedly
- Checks for a ``[DONE]`` marker in the response
- Accumulates context across turns
- Persists initial input and final output to memory
"""

from __future__ import annotations

import logging
from typing import Any

from superagent.agents.base import BaseAgent

logger = logging.getLogger(__name__)

DONE_MARKER = "[DONE]"


class AgentLoopAgent(BaseAgent):
    """Agent that wraps another agent and runs it in a loop.

    Each turn:
    1. Calls the child agent with accumulated context
    2. Checks if the response contains the done marker ``[DONE]``
    3. If done, returns the final output (marker stripped)
    4. If not done, appends the response to context and loops

    Attributes:
        max_turns: Maximum loop iterations (default 25, matching Go base).
        done_marker: String that signals completion (default ``[DONE]``).
        child_agent: The agent to call each turn.
    """

    def __init__(
        self,
        agent_id: str,
        name: str,
        child_agent: BaseAgent | None = None,
        max_turns: int = 25,
        done_marker: str = DONE_MARKER,
        **kwargs: Any,
    ) -> None:
        super().__init__(agent_id, name, **kwargs)
        self.child_agent = child_agent
        self.max_turns = max_turns
        self.done_marker = done_marker

    def _build_agent(self) -> Any:
        """Not used — AgentLoopAgent delegates to child_agent."""
        raise NotImplementedError("AgentLoopAgent delegates to child_agent")

    async def run(self, message: str, **kwargs: Any) -> str:
        """Run the child agent in a loop until done marker or max turns."""
        if self.child_agent is None:
            return f"[{self.name}] No child agent configured"

        session_id = kwargs.get("session_id", "")
        context_parts: list[str] = [message]
        final_output = ""

        for turn in range(self.max_turns):
            # Build accumulated prompt
            if turn == 0:
                prompt = message
            else:
                prompt = "\n\n".join(context_parts)

            logger.debug("AgentLoop %s turn %d/%d", self.name, turn + 1, self.max_turns)

            try:
                response = await self.child_agent.run(prompt, **kwargs)
            except Exception as exc:
                logger.error("AgentLoop %s child agent failed on turn %d: %s", self.name, turn + 1, exc)
                return f"[{self.name}] Error on turn {turn + 1}: {exc}"

            # Check for done marker
            if self.done_marker in response:
                # Strip the marker and return final output
                final_output = response.replace(self.done_marker, "").strip()
                logger.info("AgentLoop %s completed after %d turns", self.name, turn + 1)
                break

            # Accumulate context for next turn
            context_parts.append(response)
            final_output = response
        else:
            logger.warning("AgentLoop %s reached max turns (%d)", self.name, self.max_turns)

        return final_output

    async def run_stream(self, message: str, **kwargs: Any):
        """Run in loop mode, streaming events from each turn."""
        if self.child_agent is None:
            return

        context_parts: list[str] = [message]

        for turn in range(self.max_turns):
            prompt = "\n\n".join(context_parts) if turn > 0 else message

            turn_text = ""
            done = False
            async for event in self.child_agent.run_stream(prompt, **kwargs):
                delta = getattr(event, "delta", None)
                if delta:
                    turn_text += delta
                    if self.done_marker in turn_text:
                        done = True
                yield event

            if done or self.done_marker in turn_text:
                logger.info("AgentLoop %s stream completed after %d turns", self.name, turn + 1)
                return

            context_parts.append(turn_text)

        logger.warning("AgentLoop %s stream reached max turns (%d)", self.name, self.max_turns)

    def describe(self) -> dict[str, Any]:
        info = super().describe()
        info["type"] = "AgentLoopAgent"
        info["max_turns"] = self.max_turns
        info["done_marker"] = self.done_marker
        info["child_agent"] = self.child_agent.name if self.child_agent else None
        return info
