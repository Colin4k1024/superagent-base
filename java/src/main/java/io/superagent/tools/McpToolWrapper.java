package io.superagent.tools;

import io.superagent.mcp.MCPClient;
import io.superagent.mcp.MCPRegistry;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Component;

import java.util.LinkedHashMap;
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

    private static final Logger log = LoggerFactory.getLogger(McpToolWrapper.class);

    private final String serverName;
    private final String toolName;
    private final String description;
    private final Map<String, Object> parameterSchema;

    @Autowired
    private MCPRegistry mcpRegistry;

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
        if (mcpRegistry == null) {
            return errorResult("MCPRegistry not available — McpToolWrapper was not Spring-managed");
        }

        return mcpRegistry.getClient(serverName)
            .map(client -> delegateToMcp(client, parameters))
            .orElseGet(() -> errorResult("MCP server '" + serverName + "' is not connected"));
    }

    private Map<String, Object> delegateToMcp(MCPClient client, Map<String, Object> parameters) {
        try {
            MCPClient.ToolCallResult mcpResult = client.callTool(toolName, parameters);

            Map<String, Object> result = new LinkedHashMap<>();
            result.put("tool", getName());
            result.put("server", serverName);
            result.put("uri", getUri());
            result.put("content", mcpResult.text());
            result.put("is_error", mcpResult.isError());
            result.put("status", mcpResult.isError() ? "error" : "success");
            return result;
        } catch (MCPClient.MCPException e) {
            log.error("MCP tool call failed for {}/{}: {}", serverName, toolName, e.getMessage());
            return errorResult("MCP call failed: " + e.getMessage());
        }
    }

    private Map<String, Object> errorResult(String message) {
        return Map.of(
            "tool", getName(),
            "server", serverName,
            "uri", getUri(),
            "status", "error",
            "message", message
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
