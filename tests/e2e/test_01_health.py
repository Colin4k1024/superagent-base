"""
Phase 1: Environment & Health Check Tests

Validates that LLM service, backend API, and agent runtime are operational.
"""
import httpx
import pytest

from conftest import BASE_URL, LLM_URL, LLM_API_KEY, MODEL_ID, save_screenshot


class TestHealthCheck:
    """Verify all services are reachable before running real tests."""

    def test_llm_service_reachable(self):
        """LLM service at port 8000 responds to /v1/models."""
        resp = httpx.get(
            f"{LLM_URL}/v1/models",
            headers={"Authorization": f"Bearer {LLM_API_KEY}"},
            timeout=10,
        )
        assert resp.status_code == 200, f"LLM service not reachable: {resp.status_code}"
        data = resp.json()
        assert "data" in data
        model_ids = [m["id"] for m in data["data"]]
        assert MODEL_ID in model_ids, f"Model {MODEL_ID} not found in {model_ids}"

        save_screenshot("llm_models", resp.text)

    def test_llm_chat_completion(self):
        """LLM can generate a basic chat completion."""
        resp = httpx.post(
            f"{LLM_URL}/v1/chat/completions",
            headers={"Authorization": f"Bearer {LLM_API_KEY}"},
            json={
                "model": MODEL_ID,
                "messages": [{"role": "user", "content": "Say hello in one word."}],
                "max_tokens": 10,
            },
            timeout=30,
        )
        assert resp.status_code == 200, f"LLM chat failed: {resp.text}"
        data = resp.json()
        assert "choices" in data
        assert len(data["choices"]) > 0
        content = data["choices"][0]["message"]["content"]
        assert len(content) > 0, "Empty response from LLM"

        save_screenshot("llm_chat_completion", resp.text)

    def test_backend_service_reachable(self, http_client):
        """Backend API at port 8888 responds."""
        resp = http_client.get("/api/v1/agents")
        assert resp.status_code == 200, f"Backend not reachable: {resp.status_code}"

        save_screenshot("backend_agents_list", resp.text)

    def test_agents_loaded(self, http_client):
        """At least one agent is loaded in the runtime."""
        resp = http_client.get("/api/v1/agents")
        assert resp.status_code == 200
        data = resp.json()
        assert "agents" in data
        assert len(data["agents"]) > 0, "No agents loaded"

        agent_names = [a["name"] for a in data["agents"]]
        save_screenshot("loaded_agents", f"Agents: {agent_names}")

    def test_e2e_test_agents_loaded(self, http_client):
        """E2E test agents are detected by the hot-reload watcher."""
        resp = http_client.get("/api/v1/agents")
        data = resp.json()
        agent_names = [a["name"] for a in data["agents"]]

        # At minimum, e2e-basic should be loaded
        assert "e2e-basic" in agent_names, (
            f"e2e-basic agent not found. Available: {agent_names}"
        )
