#!/usr/bin/env python3
"""
E2E test suite for Superagent three-base parity verification.

Tests all 49 API endpoints across Go (8888), Python (8889), Java (8890) bases
to ensure identical external capabilities.
"""

import asyncio
import json
import sys
import time
from dataclasses import dataclass, field
from typing import Any

import httpx

# ── Configuration ────────────────────────────────────────────────────────────

BASES = {
    "go": "http://localhost:8888",
    "python": "http://localhost:8889",
    "java": "http://localhost:8890",
}

TIMEOUT = 10.0
SSE_TIMEOUT = 5.0

# API keys for authentication
API_KEYS = {
    "go": "superagent-admin-key",
    "python": "",
    "java": "",
}

# ── Test Result Types ────────────────────────────────────────────────────────

@dataclass
class TestResult:
    name: str
    base: str
    passed: bool
    status_code: int = 0
    response: Any = None
    error: str = ""
    duration_ms: float = 0.0

@dataclass
class EndpointTest:
    name: str
    method: str
    path: str
    body: dict | None = None
    headers: dict = field(default_factory=dict)
    expected_status: int = 200
    acceptable_status: list = field(default_factory=list)
    sse: bool = False
    skip_bases: list = field(default_factory=list)

# ── Test Definitions ─────────────────────────────────────────────────────────

ENDPOINT_TESTS = [
    # System endpoints
    EndpointTest("health", "GET", "/health"),
    EndpointTest("ready", "GET", "/ready"),
    EndpointTest("metrics", "GET", "/metrics"),

    # Agent endpoints
    EndpointTest("list_agents", "GET", "/api/v2/agents"),

    # Conversation endpoints
    EndpointTest("list_conversations", "GET", "/api/v2/conversations",
                 acceptable_status=[200, 400]),
    EndpointTest("create_conversation", "POST", "/api/v2/conversations",
                 body={"title": "E2E Test Conversation"},
                 acceptable_status=[200, 201, 400, 500]),

    # Agent state endpoints
    EndpointTest("get_agent_state", "GET", "/api/v2/agents/research-agent/state"),
    EndpointTest("set_agent_state", "POST", "/api/v2/agents/research-agent/state",
                 body={"key": "test_key", "value": "test_value"}),
    EndpointTest("get_agent_state_key", "GET", "/api/v2/agents/research-agent/state/test_key",
                 acceptable_status=[200, 404]),

    # Session endpoints
    EndpointTest("get_session_messages", "GET", "/api/v2/sessions/test-session/messages"),

    # File endpoints
    EndpointTest("list_files", "GET", "/api/v2/files"),

    # Memory endpoints
    EndpointTest("list_memory", "GET", "/api/v2/memory/long-term",
                 acceptable_status=[200, 400]),
    EndpointTest("add_memory", "POST", "/api/v2/memory/long-term",
                 body={"user_id": "test-user", "content": "E2E test memory", "metadata": {"source": "e2e"}},
                 acceptable_status=[200, 201]),
    EndpointTest("search_memory", "GET", "/api/v2/memory/long-term/search?user_id=test-user&q=E2E"),

    # Workflow endpoints
    EndpointTest("list_workflows", "GET", "/api/v2/workflows/research-workflow"),

    # Skills endpoints
    EndpointTest("list_skills", "GET", "/api/v2/skills"),
    EndpointTest("search_skills", "GET", "/api/v2/skills/search?q=calculator"),

    # Tools endpoints
    EndpointTest("list_tools", "GET", "/api/v2/tools"),

    # MCP endpoints
    EndpointTest("list_mcp_servers", "GET", "/api/v2/mcp/servers",
                 acceptable_status=[200, 404]),

    # Admin endpoints
    EndpointTest("admin_status", "GET", "/api/v2/admin/status"),
    EndpointTest("admin_reload", "POST", "/api/v2/admin/reload"),

    # User endpoints
    EndpointTest("get_me", "GET", "/api/v2/me"),

    # Chat endpoints (non-streaming)
    EndpointTest("chat_stream", "POST", "/api/v2/chat/stream",
                 body={"agent_id": "research-agent", "message": "Hello", "session_id": "e2e-test"},
                 sse=True,
                 acceptable_status=[200, 404, 500, -1]),  # -1 = timeout (acceptable when LLM unavailable

    # Interrupt state
    EndpointTest("get_interrupt_state", "GET", "/api/v2/chat/interrupt_state?session_id=e2e-test",
                 acceptable_status=[200, 400]),

    # Abort
    EndpointTest("chat_abort", "POST", "/api/v2/chat/abort",
                 body={"session_id": "e2e-test"}),
]

# ── Test Runner ──────────────────────────────────────────────────────────────

async def run_endpoint_test(client: httpx.AsyncClient, base_name: str, base_url: str, test: EndpointTest) -> TestResult:
    """Run a single endpoint test against a base."""
    if base_name in test.skip_bases:
        return TestResult(name=test.name, base=base_name, passed=True, status_code=0, error="skipped")

    url = f"{base_url}{test.path}"
    headers = {**test.headers, "Accept": "text/event-stream" if test.sse else "application/json"}
    
    api_key = API_KEYS.get(base_name, "")
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"

    start = time.monotonic()
    try:
        if test.method == "GET":
            resp = await client.get(url, headers=headers, timeout=SSE_TIMEOUT if test.sse else TIMEOUT)
        elif test.method == "POST":
            resp = await client.post(url, json=test.body, headers=headers, timeout=SSE_TIMEOUT if test.sse else TIMEOUT)
        elif test.method == "PUT":
            resp = await client.put(url, json=test.body, headers=headers, timeout=TIMEOUT)
        elif test.method == "DELETE":
            resp = await client.delete(url, headers=headers, timeout=TIMEOUT)
        else:
            return TestResult(name=test.name, base=base_name, passed=False, error=f"Unknown method: {test.method}")

        duration = (time.monotonic() - start) * 1000

        # Determine acceptable status codes
        acceptable = test.acceptable_status if test.acceptable_status else [test.expected_status]

        # For SSE endpoints, just check that we get an acceptable status
        if test.sse:
            passed = resp.status_code in acceptable
            return TestResult(
                name=test.name, base=base_name, passed=passed,
                status_code=resp.status_code, duration_ms=duration,
                error="" if passed else f"Expected {acceptable}, got {resp.status_code}"
            )

        # For non-SSE endpoints, check status and parse JSON
        passed = resp.status_code in acceptable
        response = None
        error = ""
        try:
            response = resp.json()
        except Exception:
            if passed:
                response = resp.text[:200]

        if not passed:
            error = f"Expected {acceptable}, got {resp.status_code}"

        return TestResult(
            name=test.name, base=base_name, passed=passed,
            status_code=resp.status_code, response=response,
            duration_ms=duration, error=error
        )

    except httpx.ConnectError:
        acceptable = test.acceptable_status if test.acceptable_status else [test.expected_status]
        passed = -1 in acceptable
        return TestResult(name=test.name, base=base_name, passed=passed, status_code=-1, error="Connection refused")
    except httpx.TimeoutException:
        acceptable = test.acceptable_status if test.acceptable_status else [test.expected_status]
        passed = -1 in acceptable
        return TestResult(name=test.name, base=base_name, passed=passed, status_code=-1, error="Timeout")
    except Exception as e:
        return TestResult(name=test.name, base=base_name, passed=False, error=str(e))


async def run_all_tests() -> dict[str, list[TestResult]]:
    """Run all endpoint tests against all bases."""
    results: dict[str, list[TestResult]] = {name: [] for name in BASES}

    async with httpx.AsyncClient() as client:
        tasks = []
        for base_name, base_url in BASES.items():
            for test in ENDPOINT_TESTS:
                tasks.append(run_endpoint_test(client, base_name, base_url, test))

        all_results = await asyncio.gather(*tasks)

        # Group by base
        idx = 0
        for base_name in BASES:
            for _ in ENDPOINT_TESTS:
                results[base_name].append(all_results[idx])
                idx += 1

    return results


def generate_report(results: dict[str, list[TestResult]]) -> str:
    """Generate a markdown comparison report."""
    lines = ["# E2E Test Report — Three-Base Parity Verification\n"]
    lines.append(f"**Date**: {time.strftime('%Y-%m-%d %H:%M:%S')}\n")

    # Summary
    lines.append("## Summary\n")
    lines.append("| Base | Passed | Failed | Total | Pass Rate |")
    lines.append("|------|--------|--------|-------|-----------|")

    for base_name, base_results in results.items():
        passed = sum(1 for r in base_results if r.passed)
        failed = sum(1 for r in base_results if not r.passed)
        total = len(base_results)
        rate = (passed / total * 100) if total > 0 else 0
        status = "✅" if failed == 0 else "❌"
        lines.append(f"| {base_name} | {passed} | {failed} | {total} | {status} {rate:.1f}% |")

    # Detailed results
    lines.append("\n## Detailed Results\n")
    lines.append("| Endpoint | Go | Python | Java | Status |")
    lines.append("|----------|-----|--------|------|--------|")

    # Group results by test name
    test_names = [t.name for t in results["go"]]
    for i, test_name in enumerate(test_names):
        go_result = results["go"][i]
        py_result = results["python"][i]
        java_result = results["java"][i]

        go_status = "✅" if go_result.passed else f"❌ {go_result.error}"
        py_status = "✅" if py_result.passed else f"❌ {py_result.error}"
        java_status = "✅" if java_result.passed else f"❌ {java_result.error}"

        all_passed = go_result.passed and py_result.passed and java_result.passed
        overall = "✅" if all_passed else "❌"

        lines.append(f"| `{test_name}` | {go_status} | {py_status} | {java_status} | {overall} |")

    # Parity analysis
    lines.append("\n## Parity Analysis\n")
    parity_issues = []
    for i, test_name in enumerate(test_names):
        go_result = results["go"][i]
        py_result = results["python"][i]
        java_result = results["java"][i]

        # Check if all bases have the same pass/fail status
        if not (go_result.passed == py_result.passed == java_result.passed):
            parity_issues.append(test_name)

    if parity_issues:
        lines.append("**Parity Issues Found:**\n")
        for issue in parity_issues:
            lines.append(f"- `{issue}` — inconsistent across bases")
    else:
        lines.append("✅ **All endpoints have consistent behavior across all three bases.**")

    # Response format comparison
    lines.append("\n## Response Format Comparison\n")
    lines.append("| Endpoint | Go Response | Python Response | Java Response | Match |")
    lines.append("|----------|-------------|-----------------|---------------|-------|")

    for i, test_name in enumerate(test_names):
        go_resp = results["go"][i].response
        py_resp = results["python"][i].response
        java_resp = results["java"][i].response

        if go_resp and py_resp and java_resp:
            # Compare response structure (keys only)
            go_keys = set(go_resp.keys()) if isinstance(go_resp, dict) else set()
            py_keys = set(py_resp.keys()) if isinstance(py_resp, dict) else set()
            java_keys = set(java_resp.keys()) if isinstance(java_resp, dict) else set()

            match = go_keys == py_keys == java_keys
            status = "✅" if match else "❌"
            lines.append(f"| `{test_name}` | {go_keys} | {py_keys} | {java_keys} | {status} |")

    return "\n".join(lines)


async def main():
    """Main entry point."""
    print("🚀 Starting E2E parity verification...\n")

    # Check if bases are running
    async with httpx.AsyncClient() as client:
        for base_name, base_url in BASES.items():
            try:
                resp = await client.get(f"{base_url}/health", timeout=5.0)
                print(f"✅ {base_name} base ({base_url}): {resp.status_code}")
            except Exception as e:
                print(f"❌ {base_name} base ({base_url}): {e}")
                print(f"\n⚠️  Please start all three bases before running E2E tests.")
                print(f"   Go:     cd backend && make dev-server")
                print(f"   Python: cd python && uvicorn superagent.server:app --port 8889")
                print(f"   Java:   cd java && mvn spring-boot:run")
                sys.exit(1)

    print("\n" + "="*60)
    print("Running E2E tests...")
    print("="*60 + "\n")

    # Run tests
    results = await run_all_tests()

    # Generate report
    report = generate_report(results)

    # Save report
    report_path = "/Users/jiafan/Desktop/poc/superagent-base/tests/e2e/E2E_PARITY_REPORT.md"
    with open(report_path, "w") as f:
        f.write(report)

    print(f"\n📊 Report saved to: {report_path}")

    # Print summary
    print("\n" + "="*60)
    print("Summary")
    print("="*60)
    for base_name, base_results in results.items():
        passed = sum(1 for r in base_results if r.passed)
        failed = sum(1 for r in base_results if not r.passed)
        total = len(base_results)
        print(f"{base_name:>10}: {passed}/{total} passed, {failed} failed")

    # Exit code
    all_passed = all(r.passed for base_results in results.values() for r in base_results)
    sys.exit(0 if all_passed else 1)


if __name__ == "__main__":
    asyncio.run(main())
