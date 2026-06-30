package io.superagent.mcp;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.*;
import java.net.HttpURLConnection;
import java.net.URI;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.*;
import java.util.concurrent.atomic.AtomicLong;

/**
 * MCP (Model Context Protocol) client using JSON-RPC 2.0.
 *
 * <p>Connects to MCP servers via HTTP/SSE transport and provides
 * tool discovery and invocation capabilities.</p>
 *
 * <p>Maps to Go {@code mcp.Client} with Initialize, ListTools, CallTool.</p>
 */
public class MCPClient {

    private static final Logger log = LoggerFactory.getLogger(MCPClient.class);
    private static final ObjectMapper mapper = new ObjectMapper();

    private final String serverName;
    private final String endpoint;
    private final Map<String, String> headers;
    private final AtomicLong idCounter = new AtomicLong(1);

    private ServerInfo serverInfo;
    private List<ToolDefinition> tools;
    private boolean initialized = false;

    public MCPClient(String serverName, String endpoint) {
        this(serverName, endpoint, Map.of());
    }

    public MCPClient(String serverName, String endpoint, Map<String, String> headers) {
        this.serverName = Objects.requireNonNull(serverName, "serverName");
        this.endpoint = Objects.requireNonNull(endpoint, "endpoint");
        this.headers = headers != null ? Map.copyOf(headers) : Map.of();
    }

    /**
     * Initialize the MCP session with the server.
     *
     * @return server information (name, version, capabilities)
     * @throws MCPException if initialization fails
     */
    public ServerInfo initialize() throws MCPException {
        Map<String, Object> params = Map.of(
            "protocolVersion", "2024-11-05",
            "capabilities", Map.of(
                "tools", Map.of()
            ),
            "clientInfo", Map.of(
                "name", "superagent-java",
                "version", "0.1.0"
            )
        );

        Map<String, Object> result = sendRequest("initialize", params);
        @SuppressWarnings("unchecked")
        Map<String, Object> serverInfoMap = (Map<String, Object>) result.get("serverInfo");

        String name = serverInfoMap != null ? (String) serverInfoMap.getOrDefault("name", serverName) : serverName;
        String version = serverInfoMap != null ? (String) serverInfoMap.getOrDefault("version", "unknown") : "unknown";
        @SuppressWarnings("unchecked")
        Map<String, Object> caps = (Map<String, Object>) result.get("capabilities");

        this.serverInfo = new ServerInfo(name, version, caps != null ? caps : Map.of());
        this.initialized = true;

        // Send initialized notification
        sendNotification("notifications/initialized", Map.of());

        log.info("MCP server '{}' initialized: {} v{}", serverName, name, version);
        return this.serverInfo;
    }

    /**
     * List available tools from the server.
     *
     * @return list of tool definitions
     * @throws MCPException if the request fails
     */
    @SuppressWarnings("unchecked") // JSON-RPC result is Map<String, Object> from raw Jackson deserialization
    public List<ToolDefinition> listTools() throws MCPException {
        ensureInitialized();

        Map<String, Object> result = sendRequest("tools/list", Map.of());
        List<Map<String, Object>> toolsList = (List<Map<String, Object>>) result.get("tools");

        if (toolsList == null) {
            this.tools = List.of();
            return this.tools;
        }

        this.tools = toolsList.stream()
            .map(t -> new ToolDefinition(
                (String) t.get("name"),
                (String) t.getOrDefault("description", ""),
                (Map<String, Object>) t.getOrDefault("inputSchema", Map.of())
            ))
            .toList();

        log.info("MCP server '{}' exposes {} tools", serverName, this.tools.size());
        return this.tools;
    }

    /**
     * Call a tool on the server.
     *
     * @param name      tool name
     * @param arguments tool arguments
     * @return tool call result
     * @throws MCPException if the call fails
     */
    public ToolCallResult callTool(String name, Map<String, Object> arguments) throws MCPException {
        ensureInitialized();

        Map<String, Object> params = Map.of(
            "name", name,
            "arguments", arguments != null ? arguments : Map.of()
        );

        Map<String, Object> result = sendRequest("tools/call", params);

        @SuppressWarnings("unchecked")
        List<Map<String, Object>> content = (List<Map<String, Object>>) result.get("content");
        boolean isError = Boolean.TRUE.equals(result.get("isError"));

        String textContent = "";
        if (content != null && !content.isEmpty()) {
            StringBuilder sb = new StringBuilder();
            for (Map<String, Object> item : content) {
                if ("text".equals(item.get("type"))) {
                    sb.append(item.get("text"));
                }
            }
            textContent = sb.toString();
        }

        return new ToolCallResult(textContent, content, isError);
    }

    /**
     * Check if the client is initialized.
     */
    public boolean isInitialized() {
        return initialized;
    }

    /**
     * Get the server name.
     */
    public String getServerName() {
        return serverName;
    }

    /**
     * Get the endpoint URL.
     */
    public String getEndpoint() {
        return endpoint;
    }

    /**
     * Get cached tools (may be null if listTools not called).
     */
    public List<ToolDefinition> getCachedTools() {
        return tools;
    }

    /**
     * Get server info (may be null if not initialized).
     */
    public ServerInfo getServerInfo() {
        return serverInfo;
    }

    private void ensureInitialized() throws MCPException {
        if (!initialized) {
            throw new MCPException("MCP client '" + serverName + "' not initialized. Call initialize() first.");
        }
    }

    private Map<String, Object> sendRequest(String method, Map<String, Object> params) throws MCPException {
        long id = idCounter.getAndIncrement();
        Map<String, Object> request = new LinkedHashMap<>();
        request.put("jsonrpc", "2.0");
        request.put("id", id);
        request.put("method", method);
        request.put("params", params);

        String jsonBody;
        try {
            jsonBody = mapper.writeValueAsString(request);
        } catch (JsonProcessingException e) {
            throw new MCPException("Failed to serialize request: " + e.getMessage(), e);
        }

        String responseBody = doPost(jsonBody);

        @SuppressWarnings("unchecked")
        Map<String, Object> response = parseJson(responseBody);

        if (response.containsKey("error")) {
            @SuppressWarnings("unchecked")
            Map<String, Object> error = (Map<String, Object>) response.get("error");
            throw new MCPException("MCP error " + error.get("code") + ": " + error.get("message"));
        }

        @SuppressWarnings("unchecked")
        Map<String, Object> result = (Map<String, Object>) response.get("result");
        return result != null ? result : Map.of();
    }

    private void sendNotification(String method, Map<String, Object> params) {
        Map<String, Object> notification = new LinkedHashMap<>();
        notification.put("jsonrpc", "2.0");
        notification.put("method", method);
        notification.put("params", params);

        try {
            String jsonBody = mapper.writeValueAsString(notification);
            doPost(jsonBody);
        } catch (Exception e) {
            log.warn("Failed to send notification '{}': {}", method, e.getMessage());
        }
    }

    @SuppressWarnings("unchecked")
    private Map<String, Object> parseJson(String json) throws MCPException {
        try {
            return mapper.readValue(json, Map.class);
        } catch (JsonProcessingException e) {
            throw new MCPException("Failed to parse response: " + e.getMessage(), e);
        }
    }

    private String doPost(String jsonBody) throws MCPException {
        try {
            URL url = URI.create(endpoint).toURL();
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setRequestMethod("POST");
            conn.setRequestProperty("Content-Type", "application/json");
            conn.setRequestProperty("Accept", "application/json");
            for (var entry : headers.entrySet()) {
                conn.setRequestProperty(entry.getKey(), entry.getValue());
            }
            conn.setDoOutput(true);
            conn.setConnectTimeout(10_000);
            conn.setReadTimeout(30_000);

            try (OutputStream os = conn.getOutputStream()) {
                os.write(jsonBody.getBytes(StandardCharsets.UTF_8));
            }

            int status = conn.getResponseCode();
            InputStream is = status >= 400 ? conn.getErrorStream() : conn.getInputStream();
            String body = new String(is.readAllBytes(), StandardCharsets.UTF_8);

            if (status >= 400) {
                throw new MCPException("HTTP " + status + ": " + body);
            }
            return body;
        } catch (MCPException e) {
            throw e;
        } catch (IOException e) {
            throw new MCPException("Connection failed to " + endpoint + ": " + e.getMessage(), e);
        }
    }

    // ─── Inner types ───

    /**
     * Server information returned during initialization.
     */
    public record ServerInfo(String name, String version, Map<String, Object> capabilities) {}

    /**
     * Tool definition exposed by an MCP server.
     */
    public record ToolDefinition(String name, String description, Map<String, Object> inputSchema) {}

    /**
     * Result of a tool call.
     */
    public record ToolCallResult(String text, List<Map<String, Object>> content, boolean isError) {}

    /**
     * MCP-specific exception.
     */
    public static class MCPException extends Exception {
        public MCPException(String message) {
            super(message);
        }

        public MCPException(String message, Throwable cause) {
            super(message, cause);
        }
    }
}
