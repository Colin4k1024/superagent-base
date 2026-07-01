package io.superagent.tools.builtin;

import io.superagent.tools.Tool;
import org.springframework.stereotype.Component;

import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.*;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Web search tool via DuckDuckGo Lite (no API key required).
 *
 * <p>URI: {@code builtin/web_search}</p>
 */
@Component
public class WebSearchTool implements Tool {

    private static final String DDGO_URL = "https://lite.duckduckgo.com/lite/?q=";
    private static final Pattern TITLE_PATTERN =
        Pattern.compile("<a[^>]+class=\"result-link\"[^>]*>([^<]+)</a>", Pattern.CASE_INSENSITIVE);
    private static final Pattern SNIPPET_PATTERN =
        Pattern.compile("<td[^>]+class=\"result-snippet\"[^>]*>([^<]+)</td>", Pattern.CASE_INSENSITIVE);

    private final HttpClient client = HttpClient.newBuilder()
        .connectTimeout(Duration.ofSeconds(10))
        .followRedirects(HttpClient.Redirect.NORMAL)
        .build();

    @Override
    public String getName() {
        return "web_search";
    }

    @Override
    public String getDescription() {
        return "Search the web for information on a given query. "
             + "Returns a list of search results with titles and snippets.";
    }

    @Override
    public Map<String, Object> getParameterSchema() {
        return Map.of(
            "type", "object",
            "properties", Map.of(
                "query", Map.of(
                    "type", "string",
                    "description", "The search query"
                ),
                "num_results", Map.of(
                    "type", "integer",
                    "description", "Number of results to return (default: 5)"
                )
            ),
            "required", List.of("query")
        );
    }

    @Override
    public Map<String, Object> execute(Map<String, Object> parameters) {
        String query = (String) parameters.getOrDefault("query", "");
        int numResults = parameters.get("num_results") instanceof Number n ? n.intValue() : 5;

        if (query.isBlank()) {
            return Map.of("tool", getName(), "status", "error", "message", "query is required");
        }

        String encodedQuery = URLEncoder.encode(query, StandardCharsets.UTF_8);
        HttpRequest request = HttpRequest.newBuilder()
            .uri(URI.create(DDGO_URL + encodedQuery))
            .header("User-Agent", "Mozilla/5.0 (compatible; SuperagentBot/1.0)")
            .timeout(Duration.ofSeconds(15))
            .GET()
            .build();

        try {
            HttpResponse<String> resp = client.send(request, HttpResponse.BodyHandlers.ofString());
            List<Map<String, String>> results = parseResults(resp.body(), numResults);
            return Map.of(
                "tool", getName(), "status", "success",
                "query", query, "results", results
            );
        } catch (Exception e) {
            return Map.of("tool", getName(), "status", "error",
                          "query", query, "message", e.getMessage());
        }
    }

    private List<Map<String, String>> parseResults(String html, int limit) {
        List<Map<String, String>> results = new ArrayList<>();
        Matcher titleMatcher   = TITLE_PATTERN.matcher(html);
        Matcher snippetMatcher = SNIPPET_PATTERN.matcher(html);

        while (titleMatcher.find() && results.size() < limit) {
            Map<String, String> r = new LinkedHashMap<>();
            r.put("title", titleMatcher.group(1).strip());
            if (snippetMatcher.find()) {
                r.put("snippet", snippetMatcher.group(1).strip());
            }
            results.add(r);
        }
        return results;
    }
}
