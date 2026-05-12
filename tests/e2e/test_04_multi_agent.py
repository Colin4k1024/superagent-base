"""
Phase 4: Multi-Agent Collaboration Tests

Tests supervisor, sequential, and parallel orchestration modes.
"""
import pytest

from conftest import chat_stream, save_screenshot


class TestSupervisorAgent:
    """Tests for supervisor-type multi-agent orchestration."""

    AGENT_ID = "e2e-supervisor"

    def test_supervisor_responds(self, http_client, session_id):
        """Supervisor agent coordinates sub-agents and produces a response."""
        result = chat_stream(
            http_client,
            agent_id=self.AGENT_ID,
            message="What are the benefits of exercise?",
            session_id=session_id,
        )

        assert result.has_done, "Supervisor did not complete"
        assert len(result.content) > 10, (
            f"Supervisor output too short: '{result.content}'"
        )

        save_screenshot(
            "supervisor_basic",
            f"Agent: {self.AGENT_ID}\n"
            f"Question: What are the benefits of exercise?\n"
            f"Elapsed: {result.elapsed_ms:.0f}ms\n"
            f"Response:\n{result.content}",
        )

    def test_supervisor_delegation(self, http_client, session_id):
        """Supervisor delegates to sub-agent for factual questions."""
        result = chat_stream(
            http_client,
            agent_id=self.AGENT_ID,
            message="Research: What is the capital of France?",
            session_id=session_id,
        )

        assert result.has_done
        # Should contain the answer (directly or via delegation)
        content_lower = result.content.lower()
        assert "paris" in content_lower, (
            f"Expected 'paris' in response: {result.content}"
        )

        save_screenshot(
            "supervisor_delegation",
            f"Delegation test\n"
            f"Question: What is the capital of France?\n"
            f"Response: {result.content}",
        )


class TestSequentialAgent:
    """Tests for sequential multi-agent pipeline."""

    AGENT_ID = "e2e-sequential"

    def test_sequential_pipeline_completes(self, http_client, session_id):
        """Sequential agent runs sub-agents in order and returns final output."""
        result = chat_stream(
            http_client,
            agent_id=self.AGENT_ID,
            message="Explain what a database is.",
            session_id=session_id,
        )

        assert result.has_done, "Sequential pipeline did not complete"
        assert len(result.content) > 5, (
            f"Sequential output too short: '{result.content}'"
        )

        save_screenshot(
            "sequential_basic",
            f"Agent: {self.AGENT_ID}\n"
            f"Pipeline: e2e-basic (single step)\n"
            f"Input: Explain what a database is.\n"
            f"Elapsed: {result.elapsed_ms:.0f}ms\n"
            f"Output:\n{result.content}",
        )

    def test_sequential_handles_complex_input(self, http_client, session_id):
        """Sequential pipeline handles multi-sentence input."""
        result = chat_stream(
            http_client,
            agent_id=self.AGENT_ID,
            message="I need to understand microservices architecture. What are the key patterns?",
            session_id=session_id,
        )

        assert result.has_done
        assert len(result.content) > 20

        save_screenshot(
            "sequential_complex",
            f"Complex input test\n"
            f"Response length: {len(result.content)} chars\n"
            f"Response: {result.content[:300]}",
        )


class TestParallelAgent:
    """Tests for parallel multi-agent execution."""

    AGENT_ID = "e2e-parallel"

    def test_parallel_execution_completes(self, http_client, session_id):
        """Parallel agent runs sub-agents concurrently and combines results."""
        result = chat_stream(
            http_client,
            agent_id=self.AGENT_ID,
            message="What is cloud computing?",
            session_id=session_id,
        )

        assert result.has_done, "Parallel execution did not complete"
        assert len(result.content) > 5, (
            f"Parallel output too short: '{result.content}'"
        )

        save_screenshot(
            "parallel_basic",
            f"Agent: {self.AGENT_ID}\n"
            f"Sub-agents: e2e-basic (parallel)\n"
            f"Input: What is cloud computing?\n"
            f"Elapsed: {result.elapsed_ms:.0f}ms\n"
            f"Output:\n{result.content}",
        )

    def test_parallel_performance_reasonable(self, http_client, session_id):
        """Parallel execution should not be significantly slower than single agent."""
        result = chat_stream(
            http_client,
            agent_id=self.AGENT_ID,
            message="Briefly define REST API.",
            session_id=session_id,
        )

        assert result.has_done
        # Parallel with 1 sub-agent should finish within single-agent timeframe
        assert result.elapsed_ms < 90000, (
            f"Parallel too slow: {result.elapsed_ms:.0f}ms"
        )

        save_screenshot(
            "parallel_performance",
            f"Performance: {result.elapsed_ms:.0f}ms\n"
            f"Content: {result.content[:200]}",
        )
