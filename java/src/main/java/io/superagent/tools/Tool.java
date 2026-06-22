package io.superagent.tools;

import java.util.List;
import java.util.Map;

/**
 * Interface for all Superagent tools.
 *
 * <p>Tools are callable functions that agents can invoke during execution.
 * Each tool has a name, description, parameter schema, and an execute method.</p>
 *
 * <p>Maps to Go {@code Tool} interface and Python {@code BaseTool}.</p>
 */
public interface Tool {

    /**
     * Unique tool name (e.g., "web_search", "http_request").
     */
    String getName();

    /**
     * Human-readable description for LLM tool selection.
     */
    String getDescription();

    /**
     * JSON Schema describing the tool's parameters.
     */
    Map<String, Object> getParameterSchema();

    /**
     * Execute the tool with the given parameters.
     *
     * @param parameters tool input parameters
     * @return tool output as a map
     */
    Map<String, Object> execute(Map<String, Object> parameters);

    /**
     * Return the tool's URI (e.g., "builtin/web_search").
     */
    default String getUri() {
        return "builtin/" + getName();
    }
}
