# tests/matrix/conftest.py
"""Pytest fixtures for matrix tests.

Each fixture creates a SuperagentClient pointing at one of the three backends.
Tests parametrised with `client` run against all three automatically.
"""
import sys
import os
import pytest

# Allow importing the Python SDK from the repo root
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '../../sdks/python'))

from superagent.client import SuperagentClient  # noqa: E402

GO_URL     = os.getenv("MATRIX_GO_URL",     "http://localhost:8888")
PYTHON_URL = os.getenv("MATRIX_PYTHON_URL", "http://localhost:8889")
JAVA_URL   = os.getenv("MATRIX_JAVA_URL",   "http://localhost:8890")

BACKENDS = [
    pytest.param(GO_URL,     id="go"),
    pytest.param(PYTHON_URL, id="python"),
    pytest.param(JAVA_URL,   id="java"),
]


@pytest.fixture(params=BACKENDS)
def client(request) -> SuperagentClient:
    """Parametrised fixture: yields one client per backend."""
    return SuperagentClient(base_url=request.param, timeout=30.0)


@pytest.fixture
def go_client() -> SuperagentClient:
    return SuperagentClient(base_url=GO_URL, timeout=30.0)


@pytest.fixture
def python_client() -> SuperagentClient:
    return SuperagentClient(base_url=PYTHON_URL, timeout=30.0)


@pytest.fixture
def java_client() -> SuperagentClient:
    return SuperagentClient(base_url=JAVA_URL, timeout=30.0)
