"""
Phase 3: Workflow (DAG) Agent Tests

Tests the graph-based workflow execution with multiple nodes.
"""
import pytest

from conftest import chat_stream, save_screenshot


class TestWorkflowAgent:
    """End-to-end tests for workflow (DAG execution) agents."""

    AGENT_ID = "e2e-workflow"

    def test_workflow_executes_and_returns_result(self, http_client, session_id):
        """Workflow agent completes the DAG pipeline and returns final output."""
        result = chat_stream(
            http_client,
            agent_id=self.AGENT_ID,
            message="artificial intelligence",
            session_id=session_id,
        )

        assert result.has_done, "Workflow did not complete (missing [DONE])"
        assert len(result.content) > 10, (
            f"Workflow output too short: '{result.content}'"
        )

        save_screenshot(
            "workflow_basic_execution",
            f"Input: artificial intelligence\n"
            f"Pipeline: research -> summarize\n"
            f"Elapsed: {result.elapsed_ms:.0f}ms\n"
            f"Output:\n{result.content}",
        )

    def test_workflow_processes_different_topics(self, http_client, session_id):
        """Workflow produces different results for different inputs."""
        r1 = chat_stream(
            http_client,
            agent_id=self.AGENT_ID,
            message="quantum computing",
            session_id=f"{session_id}-1",
        )
        r2 = chat_stream(
            http_client,
            agent_id=self.AGENT_ID,
            message="ocean biology",
            session_id=f"{session_id}-2",
        )

        assert r1.has_done and r2.has_done
        # Results should differ for different topics
        assert r1.content != r2.content, "Same output for different inputs"

        save_screenshot(
            "workflow_different_topics",
            f"Topic 1: quantum computing\n"
            f"Output 1: {r1.content[:200]}\n\n"
            f"Topic 2: ocean biology\n"
            f"Output 2: {r2.content[:200]}",
        )

    def test_workflow_performance(self, http_client, session_id):
        """Workflow completes within reasonable time (multi-node overhead)."""
        result = chat_stream(
            http_client,
            agent_id=self.AGENT_ID,
            message="machine learning",
            session_id=session_id,
        )

        assert result.has_done
        # Workflow has 2 LLM calls, should finish within 2 minutes
        assert result.elapsed_ms < 120000, (
            f"Workflow too slow: {result.elapsed_ms:.0f}ms"
        )

        save_screenshot(
            "workflow_performance",
            f"Nodes: 2 (research -> summarize)\n"
            f"Total time: {result.elapsed_ms:.0f}ms\n"
            f"Avg per node: {result.elapsed_ms / 2:.0f}ms",
        )
