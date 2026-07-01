# tests/matrix/test_health.py
"""健康检查：每个后端的 /health 端点必须返回 200。"""
import httpx
import pytest

HEALTH_URLS = [
    pytest.param("http://localhost:8888/health", id="go"),
    pytest.param("http://localhost:8889/health", id="python"),
    pytest.param("http://localhost:8890/health", id="java"),
]


@pytest.mark.parametrize("url", HEALTH_URLS)
def test_health_returns_200(url: str) -> None:
    try:
        resp = httpx.get(url, timeout=10)
        assert resp.status_code == 200, \
            f"Expected 200 from {url}, got {resp.status_code}"
    except httpx.ConnectError:
        pytest.fail(f"Backend not reachable at {url} — is it running?")


@pytest.mark.parametrize("url", HEALTH_URLS)
def test_health_response_is_json(url: str) -> None:
    try:
        resp = httpx.get(url, timeout=10)
        assert resp.headers.get("content-type", "").startswith("application/json"), \
            f"Expected JSON content-type from {url}, got: {resp.headers.get('content-type')}"
    except httpx.ConnectError:
        pytest.fail(f"Backend not reachable at {url} — is it running?")
