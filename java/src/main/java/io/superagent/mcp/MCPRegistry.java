package io.superagent.mcp;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Registry of MCP server connections.
 *
 * <p>Manages multiple MCP client instances, providing connect/disconnect
 * lifecycle and lookup by server name.</p>
 *
 * <p>Maps to Go {@code mcp.Registry} with Connect, GetClient, ListServers.</p>
 */
@Component
public class MCPRegistry {

    private static final Logger log = LoggerFactory.getLogger(MCPRegistry.class);

    private final ConcurrentHashMap<String, MCPClient> clients = new ConcurrentHashMap<>();

    /**
     * Connect to an MCP server and register the client.
     *
     * @param config server configuration
     * @return the connected client
     * @throws MCPClient.MCPException if connection or initialization fails
     */
    public MCPClient connect(ServerConfig config) throws MCPClient.MCPException {
        MCPClient client = new MCPClient(config.name(), config.endpoint(), config.headers());
        client.initialize();
        clients.put(config.name(), client);
        log.info("Connected to MCP server '{}' at {}", config.name(), config.endpoint());
        return client;
    }

    /**
     * Get a registered client by server name.
     *
     * @param name server name
     * @return optional client
     */
    public Optional<MCPClient> getClient(String name) {
        return Optional.ofNullable(clients.get(name));
    }

    /**
     * List all connected server names.
     *
     * @return unmodifiable list of server names
     */
    public List<String> listServers() {
        return List.copyOf(clients.keySet());
    }

    /**
     * Get all clients as a map.
     *
     * @return unmodifiable map of name → client
     */
    public Map<String, MCPClient> getAllClients() {
        return Map.copyOf(clients);
    }

    /**
     * Disconnect from a specific server.
     *
     * @param name server name
     * @return true if the server was connected and removed
     */
    public boolean disconnect(String name) {
        MCPClient removed = clients.remove(name);
        if (removed != null) {
            log.info("Disconnected from MCP server '{}'", name);
            return true;
        }
        return false;
    }

    /**
     * Disconnect from all servers.
     */
    public void disconnectAll() {
        int count = clients.size();
        clients.clear();
        log.info("Disconnected from all {} MCP servers", count);
    }

    /**
     * Check if a server is connected.
     *
     * @param name server name
     * @return true if connected
     */
    public boolean isConnected(String name) {
        return clients.containsKey(name);
    }

    /**
     * Server connection configuration.
     */
    public record ServerConfig(
        String name,
        String endpoint,
        Map<String, String> headers
    ) {
        public ServerConfig {
            headers = headers != null ? Map.copyOf(headers) : Map.of();
        }

        public ServerConfig(String name, String endpoint) {
            this(name, endpoint, Map.of());
        }
    }
}
