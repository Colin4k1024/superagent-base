package io.superagent.tools;

import io.superagent.tools.builtin.CodeExecuteTool;
import io.superagent.tools.builtin.HttpRequestTool;
import io.superagent.tools.builtin.WebSearchTool;
import org.junit.jupiter.api.Test;

import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for builtin tool stubs.
 */
class CodeExecuteToolTest {

    @Test
    void webSearchReturnsStubResult() {
        WebSearchTool tool = new WebSearchTool();
        assertEquals("web_search", tool.getName());
        assertEquals("builtin/web_search", tool.getUri());

        Map<String, Object> result = tool.execute(Map.of("query", "test"));
        assertEquals("stub", result.get("status"));
        assertEquals("test", result.get("query"));
    }

    @Test
    void httpRequestReturnsStubResult() {
        HttpRequestTool tool = new HttpRequestTool();
        assertEquals("http_request", tool.getName());

        Map<String, Object> result = tool.execute(Map.of(
            "url", "https://example.com",
            "method", "GET"
        ));
        assertEquals("stub", result.get("status"));
        assertEquals("https://example.com", result.get("url"));
    }

    @Test
    void codeExecuteReturnsStubResult() {
        CodeExecuteTool tool = new CodeExecuteTool();
        assertEquals("code_execute", tool.getName());

        Map<String, Object> result = tool.execute(Map.of(
            "language", "python",
            "code", "print('hello')"
        ));
        assertEquals("stub", result.get("status"));
        assertEquals("python", result.get("language"));
        assertEquals(0, result.get("exit_code"));
    }

    @Test
    void allToolsHaveParameterSchemas() {
        Tool[] tools = {
            new WebSearchTool(),
            new HttpRequestTool(),
            new CodeExecuteTool()
        };
        for (Tool tool : tools) {
            Map<String, Object> schema = tool.getParameterSchema();
            assertNotNull(schema, tool.getName() + " should have a parameter schema");
            assertEquals("object", schema.get("type"));
        }
    }
}
