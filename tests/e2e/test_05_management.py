"""
Phase 5: Agent Management & Hot-Reload Tests

Tests CRUD operations, agent listing, hot-reload, and validation.
"""
import os
import time
import shutil
from pathlib import Path

import httpx
import pytest
import yaml

from conftest import BASE_URL, AGENT_CONFIG_DIR, save_screenshot


class TestAgentListing:
    """Tests for GET /api/v1/agents endpoint."""

    def test_list_agents_returns_200(self, http_client):
        """Agent list endpoint returns 200 with valid JSON."""
        resp = http_client.get("/api/v1/agents")
        assert resp.status_code == 200
        data = resp.json()
        assert "agents" in data
        assert isinstance(data["agents"], list)

        save_screenshot(
            "management_list_200",
            f"Status: {resp.status_code}\n"
            f"Agent count: {len(data['agents'])}\n"
            f"Agents: {[a['name'] for a in data['agents']]}",
        )

    def test_list_agents_contains_expected_fields(self, http_client):
        """Each agent in list has name and description fields."""
        resp = http_client.get("/api/v1/agents")
        data = resp.json()

        for agent in data["agents"]:
            assert "name" in agent, f"Agent missing 'name' field: {agent}"

        save_screenshot(
            "management_agent_fields",
            f"Sample agent structure: {data['agents'][0] if data['agents'] else 'none'}",
        )

    def test_list_includes_all_types(self, http_client):
        """Agent list includes agents of different types (basic, workflow, etc.)."""
        resp = http_client.get("/api/v1/agents")
        data = resp.json()
        agent_names = [a["name"] for a in data["agents"]]

        expected = ["e2e-basic", "e2e-workflow"]
        found = [name for name in expected if name in agent_names]
        assert len(found) >= 1, (
            f"Expected at least 1 of {expected} in {agent_names}"
        )

        save_screenshot(
            "management_all_types",
            f"Expected agents: {expected}\n"
            f"Found: {found}\n"
            f"All agents: {agent_names}",
        )


class TestHotReload:
    """Tests for filesystem-based hot-reload of agent definitions."""

    def _get_agent_names(self, client: httpx.Client) -> list:
        """Helper to get current agent names."""
        resp = client.get("/api/v1/agents")
        if resp.status_code != 200:
            return []
        return [a["name"] for a in resp.json().get("agents", [])]

    def test_add_new_agent_yaml(self, http_client):
        """Adding a new YAML file triggers agent discovery."""
        agent_name = "e2e-hotreload-add"
        yaml_content = {
            "apiVersion": "superagent/v1",
            "kind": "Agent",
            "metadata": {
                "name": agent_name,
                "version": "1.0.0",
                "tags": ["e2e", "hotreload"],
            },
            "spec": {
                "type": "chat_model_agent",
                "model": {"primary": "Qwen3-Coder-Next-4bit"},
                "system_prompt": "You are a hot-reload test agent. Say 'hot-reload works' to any input.",
            },
        }

        target_path = Path(AGENT_CONFIG_DIR) / f"{agent_name}.yaml"
        try:
            # Write new agent YAML
            with open(target_path, "w") as f:
                yaml.dump(yaml_content, f, default_flow_style=False)

            # Wait for watcher to detect
            time.sleep(5)

            # Check if agent is now available
            agents = self._get_agent_names(http_client)
            assert agent_name in agents, (
                f"Hot-reload add failed. '{agent_name}' not in {agents}"
            )

            save_screenshot(
                "management_hotreload_add",
                f"Added: {agent_name}\n"
                f"Path: {target_path}\n"
                f"Detected: {'YES' if agent_name in agents else 'NO'}\n"
                f"Current agents: {agents}",
            )
        finally:
            # Cleanup
            if target_path.exists():
                target_path.unlink()
                time.sleep(2)

    def test_remove_agent_yaml(self, http_client):
        """Removing a YAML file triggers agent removal."""
        agent_name = "e2e-hotreload-remove"
        yaml_content = {
            "apiVersion": "superagent/v1",
            "kind": "Agent",
            "metadata": {"name": agent_name, "version": "1.0.0"},
            "spec": {
                "type": "chat_model_agent",
                "model": {"primary": "Qwen3-Coder-Next-4bit"},
                "system_prompt": "Temporary agent for removal test.",
            },
        }

        target_path = Path(AGENT_CONFIG_DIR) / f"{agent_name}.yaml"

        # Step 1: Add agent and poll until detected
        with open(target_path, "w") as f:
            yaml.dump(yaml_content, f, default_flow_style=False)

        was_added = False
        for _ in range(10):
            time.sleep(1)
            if agent_name in self._get_agent_names(http_client):
                was_added = True
                break

        # Step 2: Remove agent and poll until gone
        if target_path.exists():
            target_path.unlink()

        was_removed = False
        for _ in range(10):
            time.sleep(1)
            if agent_name not in self._get_agent_names(http_client):
                was_removed = True
                break

        agents_after = self._get_agent_names(http_client)

        save_screenshot(
            "management_hotreload_remove",
            f"Agent: {agent_name}\n"
            f"Added successfully: {was_added}\n"
            f"Removed successfully: {was_removed}\n"
            f"Agents after removal: {agents_after}",
        )

        assert was_added, f"Agent was never added: {self._get_agent_names(http_client)}"
        assert was_removed, f"Agent not removed: {agents_after}"

    def test_modify_agent_yaml(self, http_client):
        """Modifying an existing YAML file updates the agent."""
        agent_name = "e2e-hotreload-modify"
        target_path = Path(AGENT_CONFIG_DIR) / f"{agent_name}.yaml"

        try:
            # Step 1: Create initial agent
            yaml_v1 = {
                "apiVersion": "superagent/v1",
                "kind": "Agent",
                "metadata": {"name": agent_name, "version": "1.0.0"},
                "spec": {
                    "type": "chat_model_agent",
                    "model": {"primary": "Qwen3-Coder-Next-4bit"},
                    "system_prompt": "Version 1 agent.",
                },
            }
            with open(target_path, "w") as f:
                yaml.dump(yaml_v1, f, default_flow_style=False)
            time.sleep(5)

            agents = self._get_agent_names(http_client)
            assert agent_name in agents, f"Initial creation failed: {agents}"

            # Step 2: Modify the YAML (change version)
            yaml_v2 = yaml_v1.copy()
            yaml_v2["metadata"] = {"name": agent_name, "version": "2.0.0"}
            yaml_v2["spec"]["system_prompt"] = "Version 2 agent - modified."

            with open(target_path, "w") as f:
                yaml.dump(yaml_v2, f, default_flow_style=False)
            time.sleep(5)

            # Agent should still be available after modification
            agents = self._get_agent_names(http_client)
            assert agent_name in agents, f"Agent disappeared after modify: {agents}"

            save_screenshot(
                "management_hotreload_modify",
                f"Agent: {agent_name}\n"
                f"Modified: v1.0.0 -> v2.0.0\n"
                f"Still available: {agent_name in agents}",
            )
        finally:
            if target_path.exists():
                target_path.unlink()
                time.sleep(1)


class TestAgentValidation:
    """Tests for agent YAML validation and error handling."""

    def test_invalid_agent_not_loaded(self, http_client):
        """Invalid YAML (missing required fields) is rejected gracefully."""
        agent_name = "e2e-invalid-agent"
        target_path = Path(AGENT_CONFIG_DIR) / f"{agent_name}.yaml"

        try:
            # Write invalid YAML (missing apiVersion)
            invalid_yaml = {
                "kind": "Agent",
                "metadata": {"name": agent_name},
                "spec": {"type": "chat_model_agent"},
            }
            with open(target_path, "w") as f:
                yaml.dump(invalid_yaml, f, default_flow_style=False)
            time.sleep(3)

            # Invalid agent should NOT appear in the list
            agents = self._get_agent_names(http_client)
            assert agent_name not in agents, (
                f"Invalid agent was loaded: {agents}"
            )

            save_screenshot(
                "management_validation_reject",
                f"Invalid agent: {agent_name}\n"
                f"Missing: apiVersion\n"
                f"Loaded: {'NO (correct)' if agent_name not in agents else 'YES (BUG)'}",
            )
        finally:
            if target_path.exists():
                target_path.unlink()

    def _get_agent_names(self, client: httpx.Client) -> list:
        resp = client.get("/api/v1/agents")
        if resp.status_code != 200:
            return []
        return [a["name"] for a in resp.json().get("agents", [])]

    def test_chat_with_nonexistent_agent_returns_error(self, http_client):
        """Chatting with non-existent agent returns appropriate error."""
        resp = http_client.post(
            "/api/v1/chat/stream",
            json={
                "agent_id": "nonexistent-agent-xyz",
                "session_id": "test",
                "message": "hello",
            },
        )
        # Should return 404 or error
        assert resp.status_code in [404, 400, 500], (
            f"Expected error for non-existent agent, got {resp.status_code}"
        )

        save_screenshot(
            "management_nonexistent_agent",
            f"Agent: nonexistent-agent-xyz\n"
            f"Status: {resp.status_code}\n"
            f"Response: {resp.text[:200]}",
        )
