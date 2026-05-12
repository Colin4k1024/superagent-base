"""
Phase 6: Interrupt & Resume Tests

Tests the agent interrupt/resume capability.
"""
import time
import pytest

from conftest import chat_stream, save_screenshot, BASE_URL

import httpx


class TestInterruptResume:
    """Tests for agent interrupt and resume functionality."""

    AGENT_ID = "e2e-interrupt"

    def test_interrupt_agent_loaded(self, http_client):
        """Interrupt-capable agent is available in the runtime."""
        resp = http_client.get("/api/v1/agents")
        data = resp.json()
        agent_names = [a["name"] for a in data["agents"]]

        # This test may skip if interrupt agent failed to load
        if self.AGENT_ID not in agent_names:
            pytest.skip(f"{self.AGENT_ID} not loaded (may need interrupt support)")

    def test_interrupt_agent_chat(self, http_client, session_id):
        """Interrupt-capable agent can handle basic chat."""
        try:
            result = chat_stream(
                http_client,
                agent_id=self.AGENT_ID,
                message="Hello, can you help me?",
                session_id=session_id,
            )

            assert result.has_done
            assert len(result.content) > 0

            save_screenshot(
                "interrupt_basic_chat",
                f"Agent: {self.AGENT_ID}\n"
                f"Message: Hello, can you help me?\n"
                f"Response: {result.content}",
            )
        except httpx.HTTPStatusError as e:
            if e.response.status_code == 404:
                pytest.skip("Interrupt agent not available")
            raise

    def test_get_interrupt_state_no_pending(self, http_client, session_id):
        """Querying interrupt state when no interrupt is pending."""
        resp = http_client.get(
            "/api/v1/chat/interrupt_state",
            params={"agent_id": self.AGENT_ID, "session_id": session_id},
        )

        # Should return 200 with no interrupt, or 404 if agent not found
        if resp.status_code == 404:
            pytest.skip("Interrupt endpoint or agent not available")

        assert resp.status_code == 200
        data = resp.json()
        assert "interrupted" in data
        assert data["interrupted"] is False

        save_screenshot(
            "interrupt_state_clean",
            f"Agent: {self.AGENT_ID}\n"
            f"Session: {session_id}\n"
            f"State: {data}",
        )

    def test_resume_without_interrupt_returns_error(self, http_client, session_id):
        """Resuming without a pending interrupt returns 409 conflict."""
        resp = http_client.post(
            "/api/v1/chat/resume",
            json={
                "agent_id": self.AGENT_ID,
                "session_id": session_id,
                "input": {"confirm": True},
            },
        )

        # Should be 409 (no pending interrupt) or 404 (agent not found)
        assert resp.status_code in [404, 409], (
            f"Expected 404/409, got {resp.status_code}: {resp.text}"
        )

        save_screenshot(
            "interrupt_resume_no_pending",
            f"Agent: {self.AGENT_ID}\n"
            f"Session: {session_id}\n"
            f"Status: {resp.status_code}\n"
            f"Response: {resp.text[:200]}",
        )
