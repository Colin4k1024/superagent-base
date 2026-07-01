package io.superagent.tools.builtin;

import io.superagent.tools.Tool;
import org.springframework.stereotype.Component;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * HTTP request tool — makes HTTP calls to external APIs.
 *
 * <p>URI: {@code builtin/http_request}</p>
 */
@Component
public class HttpRequestTool implements Tool {

    private final HttpClient client = HttpClient.newBuilder()
        .connectTimeout(Duration.ofSeconds(10))
        .followRedirects(HttpClient.Redirect.NORMAL)
        .build();

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
                "url",     Map.of("type", "string",  "description", "Request URL"),
                "method",  Map.of("type", "string",  "description", "HTTP method (GET, POST, PUT, DELETE)"),
                "headers", Map.of("type", "object",  "description", "Request headers"),
                "body",    Map.of("type", "string",  "description", "Request body (for POST/PUT)")
            ),
            "required", List.of("url", "method")
        );
    }

    @Override
    @SuppressWarnings("unchecked")
    public Map<String, Object> execute(Map<String, Object> parameters) {
        String url    = (String) parameters.getOrDefault("url", "");
        String method = ((String) parameters.getOrDefault("method", "GET")).toUpperCase();
        String body   = (String) parameters.getOrDefault("body", "");

        HttpRequest.Builder builder = HttpRequest.newBuilder()
            .uri(URI.create(url))
            .timeout(Duration.ofSeconds(30));

        // Set custom headers
        Object headersRaw = parameters.get("headers");
        if (headersRaw instanceof Map<?, ?> hdrs) {
            ((Map<String, String>) hdrs).forEach(builder::header);
        }
        if (!builder.build().headers().map().containsKey("content-type")) {
            builder.header("Content-Type", "application/json");
        }

        builder = switch (method) {
            case "POST"   -> builder.POST(HttpRequest.BodyPublishers.ofString(body));
            case "PUT"    -> builder.PUT(HttpRequest.BodyPublishers.ofString(body));
            case "DELETE" -> builder.DELETE();
            default       -> builder.GET();
        };

        try {
            HttpResponse<String> resp = client.send(builder.build(),
                HttpResponse.BodyHandlers.ofString());
            Map<String, Object> result = new LinkedHashMap<>();
            result.put("tool",        getName());
            result.put("status",      "success");
            result.put("status_code", resp.statusCode());
            result.put("url",         url);
            result.put("method",      method);
            result.put("body",        resp.body());
            return result;
        } catch (Exception e) {
            return Map.of("tool", getName(), "status", "error",
                          "url", url, "message", e.getMessage());
        }
    }
}
