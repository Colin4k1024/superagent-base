"""Tests for built-in tools."""

import pytest

from superagent.tools.builtin.web_search import web_search
from superagent.tools.builtin.http_request import http_request
from superagent.tools.builtin.code_execute import code_execute


@pytest.mark.asyncio
async def test_code_execute_python():
    result = await code_execute("print('hello')", language="python")
    assert result["exit_code"] == 0
    assert "hello" in result["stdout"]


@pytest.mark.asyncio
async def test_code_execute_invalid_language():
    result = await code_execute("x=1", language="brainfuck")
    assert result["exit_code"] == -1
    assert "Unsupported language" in result["stderr"]


@pytest.mark.asyncio
async def test_code_execute_syntax_error():
    result = await code_execute("def f(:", language="python")
    assert result["exit_code"] != 0
    assert result["stderr"] != ""


@pytest.mark.asyncio
async def test_http_request_returns_dict():
    """Verify http_request returns the expected dict structure."""
    result = await http_request("https://httpbin.org/get", timeout=5.0)
    assert "status_code" in result
    assert "headers" in result
    assert "body" in result


@pytest.mark.asyncio
async def test_http_request_timeout():
    """Verify timeout handling."""
    result = await http_request("https://10.255.255.1/no-route", timeout=0.1)
    assert result["status_code"] == 0
    assert "timeout" in result["body"].lower() or "error" in result["body"].lower()


@pytest.mark.asyncio
async def test_web_search_returns_list():
    """Verify web_search returns a list (may be empty on network failure)."""
    results = await web_search("python programming", max_results=2)
    assert isinstance(results, list)
    if results:
        assert "title" in results[0]
        assert "url" in results[0]
