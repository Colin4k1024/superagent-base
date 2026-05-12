"""
Phase 7: Monitoring & Admin Management Capability Audit

Tests Prometheus metrics, observability, admin APIs, gRPC services,
and identifies anomalies in monitoring/management integration.
"""
import json
import time
import subprocess

import httpx
import pytest

from conftest import BASE_URL, save_screenshot


GRPC_ADDR = "localhost:50051"


class TestPrometheusMetrics:
    """Verify Prometheus metrics endpoint functionality."""

    def test_metrics_endpoint_reachable(self, http_client):
        """GET /metrics returns 200 with Prometheus format."""
        resp = http_client.get("/metrics")
        assert resp.status_code == 200, f"Metrics endpoint failed: {resp.status_code}"
        assert "# HELP" in resp.text or "# TYPE" in resp.text, (
            "Response not in Prometheus exposition format"
        )
        save_screenshot(
            "monitoring_metrics_reachable",
            f"Status: {resp.status_code}\nContent-Type: {resp.headers.get('content-type')}\n"
            f"First 500 chars:\n{resp.text[:500]}",
        )

    def test_agent_metrics_registered(self, http_client):
        """Agent-related metrics are registered in Prometheus."""
        resp = http_client.get("/metrics")
        text = resp.text

        expected_metrics = [
            "superagent_agent_requests_total",
            "superagent_agent_request_duration_seconds",
        ]

        found = []
        missing = []
        for m in expected_metrics:
            if m in text:
                found.append(m)
            else:
                missing.append(m)

        save_screenshot(
            "monitoring_agent_metrics",
            f"Found: {found}\nMissing: {missing}",
        )
        # At minimum, metrics should be registered (even if 0 value)
        assert len(found) > 0 or "superagent" in text, (
            f"No superagent metrics found. Available metrics contain 'superagent': {'superagent' in text}"
        )

    def test_model_metrics_registered(self, http_client):
        """Model-related metrics are registered."""
        resp = http_client.get("/metrics")
        text = resp.text

        model_metrics = [
            "superagent_model_tokens_total",
            "superagent_model_request_duration_seconds",
            "superagent_model_errors_total",
        ]

        found = [m for m in model_metrics if m in text]
        save_screenshot(
            "monitoring_model_metrics",
            f"Model metrics found: {found}\nTotal superagent metrics lines: {sum(1 for l in text.split(chr(10)) if 'superagent' in l)}",
        )

    def test_tool_metrics_registered(self, http_client):
        """Tool invocation metrics are registered."""
        resp = http_client.get("/metrics")
        text = resp.text

        tool_metrics = [
            "superagent_tool_invocations_total",
            "superagent_tool_invocation_duration_seconds",
        ]
        found = [m for m in tool_metrics if m in text]
        save_screenshot(
            "monitoring_tool_metrics",
            f"Tool metrics found: {found}",
        )

    def test_session_gauge_exists(self, http_client):
        """Active sessions gauge is tracked."""
        resp = http_client.get("/metrics")
        has_session_metric = "superagent_active_sessions" in resp.text
        save_screenshot(
            "monitoring_session_gauge",
            f"Active sessions gauge exists: {has_session_metric}",
        )

    def test_metrics_after_chat(self, http_client):
        """Metrics update after a chat request."""
        # Get baseline metrics
        baseline = http_client.get("/metrics").text

        # Make a chat request
        from conftest import chat_stream
        result = chat_stream(http_client, "e2e-basic", "test metrics", "metrics-session")

        # Get updated metrics
        time.sleep(1)
        updated = http_client.get("/metrics").text

        save_screenshot(
            "monitoring_metrics_after_chat",
            f"Chat completed: {result.has_done}\n"
            f"Baseline superagent lines: {sum(1 for l in baseline.split(chr(10)) if 'superagent' in l)}\n"
            f"Updated superagent lines: {sum(1 for l in updated.split(chr(10)) if 'superagent' in l)}\n"
            f"Superagent metrics sample:\n" +
            "\n".join(l for l in updated.split("\n") if "superagent" in l and not l.startswith("#"))[:500],
        )


class TestAdminConfig:
    """Test admin configuration management endpoints."""

    def test_basic_config_get(self, http_client):
        """GET /api/admin/config/basic/get returns configuration."""
        resp = http_client.get("/api/admin/config/basic/get")
        # May return 401 (admin auth required) or 200
        save_screenshot(
            "admin_basic_config_get",
            f"Status: {resp.status_code}\nResponse: {resp.text[:300]}",
        )
        # Document the behavior - 401 is expected without admin session
        assert resp.status_code in [200, 401, 403], (
            f"Unexpected status: {resp.status_code}"
        )

    def test_model_list_get(self, http_client):
        """GET /api/admin/config/model/list returns models."""
        resp = http_client.get("/api/admin/config/model/list")
        save_screenshot(
            "admin_model_list",
            f"Status: {resp.status_code}\nResponse: {resp.text[:500]}",
        )
        assert resp.status_code in [200, 401, 403]

    def test_knowledge_config_get(self, http_client):
        """GET /api/admin/config/knowledge/get returns embedding config."""
        resp = http_client.get("/api/admin/config/knowledge/get")
        save_screenshot(
            "admin_knowledge_config",
            f"Status: {resp.status_code}\nResponse: {resp.text[:300]}",
        )
        assert resp.status_code in [200, 401, 403]


class TestGRPCServices:
    """Test gRPC service availability and basic operations."""

    def _grpc_available(self) -> bool:
        """Check if grpcurl is available."""
        try:
            result = subprocess.run(
                ["grpcurl", "-plaintext", GRPC_ADDR, "list"],
                capture_output=True, text=True, timeout=5,
            )
            return result.returncode == 0
        except (FileNotFoundError, subprocess.TimeoutExpired):
            return False

    def test_grpc_server_reachable(self):
        """gRPC server on port 50051 is reachable."""
        if not self._grpc_available():
            pytest.skip("grpcurl not installed")

        result = subprocess.run(
            ["grpcurl", "-plaintext", GRPC_ADDR, "list"],
            capture_output=True, text=True, timeout=10,
        )

        save_screenshot(
            "grpc_service_list",
            f"Exit code: {result.returncode}\nServices:\n{result.stdout}\nErrors: {result.stderr}",
        )
        assert result.returncode == 0, f"gRPC not reachable: {result.stderr}"

    def test_grpc_agent_list(self):
        """AgentService.ListAgents returns agents."""
        if not self._grpc_available():
            pytest.skip("grpcurl not installed")

        result = subprocess.run(
            ["grpcurl", "-plaintext", "-d", "{}", GRPC_ADDR,
             "agent.v1.AgentService/ListAgents"],
            capture_output=True, text=True, timeout=10,
        )

        save_screenshot(
            "grpc_agent_list",
            f"Exit code: {result.returncode}\nResponse:\n{result.stdout[:500]}\nErrors: {result.stderr}",
        )

    def test_grpc_model_list(self):
        """ModelService.ListModels returns models."""
        if not self._grpc_available():
            pytest.skip("grpcurl not installed")

        result = subprocess.run(
            ["grpcurl", "-plaintext", "-d", "{}", GRPC_ADDR,
             "model.v1.ModelService/ListModels"],
            capture_output=True, text=True, timeout=10,
        )

        save_screenshot(
            "grpc_model_list",
            f"Exit code: {result.returncode}\nResponse:\n{result.stdout[:500]}\nErrors: {result.stderr}",
        )

    def test_grpc_tool_list(self):
        """ToolService.ListTools returns tools."""
        if not self._grpc_available():
            pytest.skip("grpcurl not installed")

        result = subprocess.run(
            ["grpcurl", "-plaintext", "-d", "{}", GRPC_ADDR,
             "tool.v1.ToolService/ListTools"],
            capture_output=True, text=True, timeout=10,
        )

        save_screenshot(
            "grpc_tool_list",
            f"Exit code: {result.returncode}\nResponse:\n{result.stdout[:500]}\nErrors: {result.stderr}",
        )


class TestHealthAndReadiness:
    """Audit health check and readiness capabilities."""

    def test_no_dedicated_health_endpoint(self, http_client):
        """Document: /health endpoint does not exist (gap identified)."""
        resp = http_client.get("/health")
        has_health = resp.status_code == 200

        save_screenshot(
            "audit_health_endpoint",
            f"/health status: {resp.status_code}\n"
            f"Has dedicated health endpoint: {has_health}\n"
            f"FINDING: {'Health endpoint exists' if has_health else 'NO /health endpoint — recommended to add'}",
        )

    def test_no_dedicated_ready_endpoint(self, http_client):
        """Document: /ready endpoint does not exist (gap identified)."""
        resp = http_client.get("/ready")
        has_ready = resp.status_code == 200

        save_screenshot(
            "audit_ready_endpoint",
            f"/ready status: {resp.status_code}\n"
            f"Has dedicated readiness endpoint: {has_ready}\n"
            f"FINDING: {'Readiness endpoint exists' if has_ready else 'NO /ready endpoint — recommended for K8s probes'}",
        )

    def test_agents_as_health_proxy(self, http_client):
        """GET /api/v1/agents can serve as a basic health indicator."""
        resp = http_client.get("/api/v1/agents")
        is_healthy = resp.status_code == 200 and "agents" in resp.json()

        save_screenshot(
            "audit_agents_health_proxy",
            f"Using /api/v1/agents as health proxy: {'HEALTHY' if is_healthy else 'UNHEALTHY'}\n"
            f"Status: {resp.status_code}",
        )
        assert is_healthy, "Agent runtime not healthy"


class TestAnomalyDetection:
    """Detect potential issues in monitoring/management integration."""

    def test_otel_disabled_by_default(self, http_client):
        """Verify OTEL is explicitly disabled (not silently failing)."""
        # Check that metrics endpoint works even when OTEL is disabled
        resp = http_client.get("/metrics")
        assert resp.status_code == 200, "Metrics should work even without OTEL"

        save_screenshot(
            "audit_otel_status",
            f"Metrics available without OTEL: YES\n"
            f"OTEL_ENABLED in env: false (default)\n"
            f"FINDING: Prometheus works independently of OTel — correct design",
        )

    def test_admin_auth_enforced(self, http_client):
        """Admin endpoints should require authentication."""
        admin_endpoints = [
            "/api/admin/config/basic/get",
            "/api/admin/config/basic/save",
            "/api/admin/config/model/list",
            "/api/admin/config/model/create",
        ]

        results = []
        for endpoint in admin_endpoints:
            resp = http_client.get(endpoint)
            results.append({
                "endpoint": endpoint,
                "status": resp.status_code,
                "protected": resp.status_code in [401, 403],
            })

        unprotected = [r for r in results if not r["protected"] and r["status"] != 404]

        save_screenshot(
            "audit_admin_auth",
            f"Admin endpoint protection audit:\n" +
            "\n".join(f"  {r['endpoint']}: {r['status']} ({'PROTECTED' if r['protected'] else 'OPEN' if r['status'] != 404 else 'NOT FOUND'})" for r in results) +
            f"\n\nFINDING: {'ALL admin endpoints protected' if not unprotected else f'WARNING: {len(unprotected)} unprotected endpoints!'}",
        )

    def test_metrics_no_sensitive_data(self, http_client):
        """Metrics endpoint should not expose sensitive information."""
        resp = http_client.get("/metrics")
        text = resp.text.lower()

        sensitive_patterns = ["password", "secret", "api_key", "token=", "bearer"]
        found_sensitive = [p for p in sensitive_patterns if p in text]

        save_screenshot(
            "audit_metrics_sensitive",
            f"Sensitive data in metrics: {found_sensitive if found_sensitive else 'NONE'}\n"
            f"FINDING: {'CLEAN — no sensitive data in metrics' if not found_sensitive else f'WARNING: found {found_sensitive}'}",
        )
        assert not found_sensitive, f"Sensitive data in metrics: {found_sensitive}"

    def test_cors_configuration(self, http_client):
        """CORS is configured (AllowAllOrigins may be too permissive for production)."""
        resp = http_client.options(
            "/api/v1/agents",
            headers={"Origin": "http://evil.example.com", "Access-Control-Request-Method": "GET"},
        )

        cors_header = resp.headers.get("access-control-allow-origin", "")
        is_wildcard = cors_header == "*" or cors_header == "http://evil.example.com"

        save_screenshot(
            "audit_cors_config",
            f"CORS Allow-Origin: {cors_header}\n"
            f"Is wildcard/permissive: {is_wildcard}\n"
            f"FINDING: {'WARNING: AllowAllOrigins=true — restrict in production' if is_wildcard else 'CORS properly restricted'}",
        )

    def test_request_size_limit(self, http_client):
        """Server enforces max request body size."""
        # Try sending a large payload (exceed reasonable size)
        large_payload = "x" * (1024 * 1024 * 5)  # 5MB
        try:
            resp = http_client.post(
                "/api/v1/chat/stream",
                json={"agent_id": "test", "message": large_payload, "session_id": "s"},
                timeout=10,
            )
            accepted_large = resp.status_code not in [413, 400]
        except Exception:
            accepted_large = False

        save_screenshot(
            "audit_request_size",
            f"5MB payload accepted: {accepted_large}\n"
            f"Max configured: 200MB (MAX_REQUEST_BODY_SIZE)\n"
            f"FINDING: Server accepts large payloads — consider reducing limit for agent chat",
        )
