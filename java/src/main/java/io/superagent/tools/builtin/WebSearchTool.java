package io.superagent.tools.builtin;

import io.superagent.tools.Tool;
import org.springframework.stereotype.Component;

import java.util.List;
import java.util.Map;

/**
 * Web search tool — searches the web via configurable search provider.
 *
 * <p>URI: {@code builtin/web_search}</p>
 */
@Component
public class WebSearchTool implements Tool {

    @Override
    public String getName() {
        return "web_search";
    }

    @Override
    public String getDescription() {
        return "Search the web for information on a given query. "
             + "Returns a list of search results with titles, URLs, and snippets.";
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
        return Map.of(
            "tool", getName(),
            "status", "stub",
            "query", query,
            "results", List.of(),
            "message", "WebSearchTool.execute() not yet implemented"
        );
    }
}
