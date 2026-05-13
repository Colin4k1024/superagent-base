"""
E2E Test Configuration and Fixtures for Superagent-Base
"""
import json
import os
import time
import shutil
from pathlib import Path
from datetime import datetime

import httpx
import pytest
import sseclient
import yaml

# ----- Configuration -----

BASE_URL = os.getenv("E2E_BASE_URL", "http://localhost:8888")
LLM_URL = os.getenv("E2E_LLM_URL", "http://localhost:8000")
LLM_API_KEY = os.getenv("E2E_LLM_API_KEY", "123456")
MODEL_ID = os.getenv("E2E_MODEL_ID", "Qwen3-Coder-Next-4bit")
AGENT_CONFIG_DIR = os.getenv(
    "E2E_AGENT_CONFIG_DIR",
    str(Path(__file__).parent.parent.parent / "backend" / "configs" / "agents"),
)
REPORTS_DIR = Path(__file__).parent / "reports"
SCREENSHOTS_DIR = REPORTS_DIR / "screenshots"
TIMEOUT = 60  # seconds per request


# ----- Fixtures -----


@pytest.fixture(scope="session")
def base_url():
    return BASE_URL


@pytest.fixture(scope="session")
def llm_url():
    return LLM_URL


@pytest.fixture(scope="session")
def http_client():
    """Shared HTTP client for the test session."""
    with httpx.Client(base_url=BASE_URL, timeout=TIMEOUT) as client:
        yield client


@pytest.fixture(scope="session")
def agent_config_dir():
    return AGENT_CONFIG_DIR


@pytest.fixture(scope="session", autouse=True)
def setup_test_agents():
    """Copy test agent YAMLs into the backend config directory before tests."""
    test_agents_dir = Path(__file__).parent / "agents"
    target_dir = Path(AGENT_CONFIG_DIR)

    if not target_dir.exists():
        target_dir.mkdir(parents=True, exist_ok=True)

    copied_files = []
    for yaml_file in test_agents_dir.glob("*.yaml"):
        dest = target_dir / yaml_file.name
        shutil.copy2(yaml_file, dest)
        copied_files.append(dest)

    # Give the file watcher time to detect and build new agents.
    # The watcher debounces events (~1s) then builds each agent sequentially.
    time.sleep(5)

    yield

    # Cleanup: remove test agents after all tests
    for f in copied_files:
        if f.exists():
            f.unlink()


@pytest.fixture
def session_id():
    """Generate a unique session ID for each test."""
    return f"e2e-session-{int(time.time() * 1000)}"


# ----- Helpers -----


class SSEResponse:
    """Parsed SSE response with tokens and metadata."""

    def __init__(self, tokens: list, raw_events: list, elapsed_ms: float):
        self.tokens = tokens
        self.raw_events = raw_events
        self.elapsed_ms = elapsed_ms
        self.full_text = "".join(tokens)
        self.has_done = any(t == "[DONE]" for t in tokens)

    @property
    def content(self):
        """Full response content without [DONE] marker."""
        return "".join(t for t in self.tokens if t != "[DONE]")

    def __repr__(self):
        return f"SSEResponse(tokens={len(self.tokens)}, elapsed={self.elapsed_ms:.0f}ms, content_len={len(self.content)})"


def chat_stream(
    client: httpx.Client,
    agent_id: str,
    message: str,
    session_id: str = "default",
) -> SSEResponse:
    """Send a chat message and collect all SSE tokens."""
    start = time.time()
    tokens = []
    raw_events = []

    with client.stream(
        "POST",
        "/api/v1/chat/stream",
        json={
            "agent_id": agent_id,
            "session_id": session_id,
            "message": message,
        },
        headers={"Accept": "text/event-stream"},
    ) as response:
        response.raise_for_status()
        sse_client = sseclient.SSEClient(response.iter_bytes())
        for event in sse_client.events():
            raw_events.append({"event": event.event, "data": event.data})
            tokens.append(event.data)
            if event.data == "[DONE]":
                break

    elapsed = (time.time() - start) * 1000
    return SSEResponse(tokens=tokens, raw_events=raw_events, elapsed_ms=elapsed)


def save_screenshot(name: str, content: str):
    """Save test evidence as a text file (API response screenshot)."""
    SCREENSHOTS_DIR.mkdir(parents=True, exist_ok=True)
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    filepath = SCREENSHOTS_DIR / f"{timestamp}_{name}.txt"
    filepath.write_text(content, encoding="utf-8")
    return filepath


# ----- Pytest Hooks -----


def pytest_html_report_title(report):
    report.title = "Superagent-Base E2E Test Report"


@pytest.fixture(autouse=True)
def test_evidence(request):
    """Collect test evidence for reporting."""
    evidence = {}
    yield evidence
    # After test, save evidence if any
    if evidence:
        name = request.node.name.replace(" ", "_").replace("[", "_").replace("]", "")
        content = json.dumps(evidence, indent=2, ensure_ascii=False)
        save_screenshot(name, content)
