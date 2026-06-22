package io.superagent.tools;

import org.springframework.stereotype.Component;

import java.util.Map;

/**
 * Wraps an MCP (Model Context Protocol) server tool as a local {@link Tool}.
 *
 * <p>URI format: {@code mcp://server-name/tool-name}</p>
 *
 * <p>Delegates execution to a remote MCP server via stdio or SSE transport.</p>
 */
@Component
public class McpToolWrapper implements Tool {

    private final String serverName;
    private final String toolName;
    private final String description;
    private final Map<String, Object> parameterSchema;

    public McpToolWrapper(String serverName, String toolName,
                          String description, Map<String, Object> parameterSchema) {
        this.serverName = serverName;
        this.toolName = toolName;
        this.description = description;
        this.parameterSchema = parameterSchema;
    }

    /**
     * No-arg constructor for Spring component scanning.
     * Actual instances are created by MCP client discovery.
     */
    public McpToolWrapper() {
        this("unknown", "unknown", "MCP tool placeholder", Map.of());
    }

    @Override
    public String getName() {
        return toolName;
    }

    @Override
    public String getDescription() {
        return description;
    }

    @Override
    public Map<String, Object> getParameterSchema() {
        return parameterSchema;
    }

    @Override
    public Map<String, Object> execute(Map<String, Object> parameters) {
        // TODO: Implement MCP tool delegation
        // 1. Resolve MCP server connection from serverName
        // 2. Serialize parameters to MCP request format
        // 3. Send request via stdio/SSE transport
        // 4. Parse MCP response
        return Map.of(
            "tool", getName(),
            "server", serverName,
            "uri", getUri(),
            "status", "stub",
            "message", "McpToolWrapper.execute() not yet implemented"
        );
    }

    @Override
    public String getUri() {
        return "mcp://" + serverName + "/" + toolName;
    }

    public String getServerName() {
        return serverName;
    }
}
