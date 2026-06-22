package io.superagent.tools.builtin;

import io.superagent.tools.Tool;
import org.springframework.stereotype.Component;

import java.util.List;
import java.util.Map;

/**
 * HTTP request tool — makes HTTP calls to external APIs.
 *
 * <p>URI: {@code builtin/http_request}</p>
 */
@Component
public class HttpRequestTool implements Tool {

    @Override
    public String getName() {
        return "http_request";
    }

    @Override
    public String getDescription() {
        return "Make an HTTP request to a given URL. "
             + "Supports GET, POST, PUT, DELETE methods with headers and body.";
    }

    @Override
    public Map<String, Object> getParameterSchema() {
        return Map.of(
            "type", "object",
            "properties", Map.of(
                "url", Map.of("type", "string", "description", "Request URL"),
                "method", Map.of("type", "string", "description", "HTTP method (GET, POST, PUT, DELETE)"),
                "headers", Map.of("type", "object", "description", "Request headers"),
                "body", Map.of("type", "string", "description", "Request body (for POST/PUT)")
            ),
            "required", List.of("url", "method")
        );
    }

    @Override
    public Map<String, Object> execute(Map<String, Object> parameters) {
        String url = (String) parameters.getOrDefault("url", "");
        String method = (String) parameters.getOrDefault("method", "GET");
        return Map.of(
            "tool", getName(),
            "status", "stub",
            "url", url,
            "method", method,
            "message", "HttpRequestTool.execute() not yet implemented"
        );
    }
}
