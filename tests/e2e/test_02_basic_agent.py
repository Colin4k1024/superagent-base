"""
Phase 2: Basic Agent (chat_model_agent) Tests

Tests single-turn and multi-turn conversations with a simple LLM agent.
"""
import time
import pytest

from conftest import chat_stream, save_screenshot


class TestBasicAgent:
    """End-to-end tests for chat_model_agent type."""

    AGENT_ID = "e2e-basic"

    def test_single_message_response(self, http_client, session_id):
        """Agent responds to a simple message with streaming tokens."""
        result = chat_stream(
            http_client,
            agent_id=self.AGENT_ID,
            message="What is 2 + 2?",
            session_id=session_id,
        )

        assert len(result.tokens) > 0, "No tokens received"
        assert result.has_done, "Missing [DONE] termination signal"
        assert len(result.content) > 0, "Empty response content"
        assert result.elapsed_ms < 60000, f"Response too slow: {result.elapsed_ms}ms"

        save_screenshot(
            "basic_single_message",
            f"Question: What is 2 + 2?\n"
            f"Response ({result.elapsed_ms:.0f}ms, {len(result.tokens)} tokens):\n"
            f"{result.content}",
        )

    def test_streaming_produces_multiple_tokens(self, http_client, session_id):
        """SSE stream produces multiple token events, not a single blob."""
        result = chat_stream(
            http_client,
            agent_id=self.AGENT_ID,
            message="Explain what Python is in 2 sentences.",
            session_id=session_id,
        )

        # Should have multiple streaming tokens (not just 1 big chunk + DONE)
        content_tokens = [t for t in result.tokens if t != "[DONE]"]
        assert len(content_tokens) >= 2, (
            f"Expected streaming tokens, got {len(content_tokens)} chunk(s)"
        )

        save_screenshot(
            "basic_streaming_tokens",
            f"Token count: {len(content_tokens)}\n"
            f"First 5 tokens: {content_tokens[:5]}\n"
            f"Full response: {result.content}",
        )

    def test_multi_turn_conversation(self, http_client):
        """Agent maintains context across multiple turns in same session."""
        session = f"e2e-multiturn-{int(time.time() * 1000)}"

        # Turn 1: Establish context
        r1 = chat_stream(
            http_client,
            agent_id=self.AGENT_ID,
            message="My name is Alice. Remember this.",
            session_id=session,
        )
        assert r1.has_done

        # Turn 2: Ask about context
        r2 = chat_stream(
            http_client,
            agent_id=self.AGENT_ID,
            message="What is my name?",
            session_id=session,
        )
        assert r2.has_done
        # The model should recall "Alice"
        assert "alice" in r2.content.lower(), (
            f"Agent didn't recall name. Response: {r2.content}"
        )

        save_screenshot(
            "basic_multi_turn",
            f"Turn 1: My name is Alice. Remember this.\n"
            f"Response 1: {r1.content}\n\n"
            f"Turn 2: What is my name?\n"
            f"Response 2: {r2.content}",
        )

    def test_long_message_handling(self, http_client, session_id):
        """Agent handles longer input messages without error."""
        long_message = "Please analyze: " + ("This is a test sentence. " * 50)
        result = chat_stream(
            http_client,
            agent_id=self.AGENT_ID,
            message=long_message,
            session_id=session_id,
        )

        assert result.has_done
        assert len(result.content) > 0

    def test_empty_like_message(self, http_client, session_id):
        """Agent handles minimal input gracefully."""
        result = chat_stream(
            http_client,
            agent_id=self.AGENT_ID,
            message="hi",
            session_id=session_id,
        )

        assert result.has_done
        assert len(result.content) > 0
