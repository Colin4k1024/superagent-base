package io.superagent.server;

import io.superagent.mcp.MCPRegistry;
import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Flux;
import reactor.core.publisher.Mono;

import java.util.Map;

/**
 * Admin endpoints for system management.
 *
 * <p>Provides agent reload, status, log streaming, and CRUD operations
 * for agents, users, MCP servers, and webhooks.</p>
 */
@RestController
@RequestMapping("/api/v2/admin")
public class AdminController {

    private final MCPRegistry mcpRegistry;

    public AdminController(MCPRegistry mcpRegistry) {
        this.mcpRegistry = mcpRegistry;
    }

    /**
     * Get system status.
     */
    @GetMapping("/status")
    public Mono<Map<String, Object>> status() {
        Runtime rt = Runtime.getRuntime();
        return Mono.just(Map.of(
            "status", "running",
            "version", "0.1.0-java",
            "runtime", "Spring Boot 3 + WebFlux",
            "memory_used_mb", (rt.totalMemory() - rt.freeMemory()) / (1024 * 1024),
            "memory_max_mb", rt.maxMemory() / (1024 * 1024),
            "processors", rt.availableProcessors()
        ));
    }

    /**
     * Trigger agent hot-reload.
     */
    @PostMapping("/reload")
    public Mono<Map<String, Object>> reload() {
        // TODO: Trigger YamlAgentLoader rescan + AgentBuilderFactory rebuild
        return Mono.just(Map.of(
            "status", "stub",
            "message", "Agent reload not yet implemented"
        ));
    }

    /**
     * List all agents (admin view with full config).
     */
    @GetMapping("/agents")
    public Mono<Map<String, Object>> listAgents() {
        return Mono.just(Map.of(
            "agents", java.util.List.of(),
            "status", "stub"
        ));
    }

    /**
     * Get a specific agent definition.
     */
    @GetMapping("/agents/{name}")
    public Mono<Map<String, Object>> getAgent(@PathVariable String name) {
        return Mono.just(Map.of(
            "name", name,
            "status", "stub",
            "message", "Agent detail not yet implemented"
        ));
    }

    /**
     * Create or update an agent definition.
     */
    @PostMapping("/agents")
    public Mono<Map<String, Object>> createAgent(@RequestBody Map<String, Object> definition) {
        return Mono.just(Map.of(
            "status", "stub",
            "message", "Agent creation not yet implemented"
        ));
    }

    /**
     * Delete an agent definition.
     */
    @DeleteMapping("/agents/{name}")
    public Mono<Map<String, Object>> deleteAgent(@PathVariable String name) {
        return Mono.just(Map.of(
            "name", name,
            "status", "stub",
            "message", "Agent deletion not yet implemented"
        ));
    }

    /**
     * List MCP servers.
     */
    @GetMapping("/mcp/servers")
    public Mono<Map<String, Object>> listMcpServers() {
        var servers = mcpRegistry.listServers();
        return Mono.just(Map.of(
            "servers", servers,
            "count", servers.size(),
            "connected", mcpRegistry.isConnected(servers.isEmpty() ? "" : servers.get(0))
        ));
    }

    /**
     * SSE log stream endpoint.
     */
    @GetMapping(value = "/logs", produces = org.springframework.http.MediaType.TEXT_EVENT_STREAM_VALUE)
    public Flux<Map<String, Object>> logStream() {
        // TODO: Wire to log appender for real-time log streaming
        return Flux.just(
            Map.<String, Object>of("level", "INFO", "message", "Log streaming not yet implemented")
        );
    }
}
