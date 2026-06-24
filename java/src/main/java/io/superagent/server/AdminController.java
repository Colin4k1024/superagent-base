package io.superagent.server;

import io.superagent.config.AgentBuilderFactory;
import io.superagent.config.AgentDefinition;
import io.superagent.config.YamlAgentLoader;
import io.superagent.mcp.MCPRegistry;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.MediaType;
import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Flux;
import reactor.core.publisher.Mono;
import reactor.core.publisher.Sinks;

import java.time.Instant;
import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/v2/admin")
public class AdminController {

    private static final Logger log = LoggerFactory.getLogger(AdminController.class);

    private final MCPRegistry mcpRegistry;
    private final YamlAgentLoader agentLoader;
    private final AgentBuilderFactory agentFactory;
    private final Sinks.Many<Map<String, Object>> logSink = Sinks.many().multicast().onBackpressureBuffer();

    public AdminController(MCPRegistry mcpRegistry, YamlAgentLoader agentLoader,
                           AgentBuilderFactory agentFactory) {
        this.mcpRegistry = mcpRegistry;
        this.agentLoader = agentLoader;
        this.agentFactory = agentFactory;
    }

    @GetMapping("/status")
    public Mono<Map<String, Object>> status() {
        Runtime rt = Runtime.getRuntime();
        return Mono.just(Map.of(
            "status", "running",
            "version", "0.1.0-java",
            "runtime", "Spring Boot 3 + WebFlux",
            "agents_loaded", agentFactory.getBuiltAgents().size(),
            "mcp_servers", mcpRegistry.listServers().size(),
            "memory_used_mb", (rt.totalMemory() - rt.freeMemory()) / (1024 * 1024),
            "memory_max_mb", rt.maxMemory() / (1024 * 1024),
            "processors", rt.availableProcessors()
        ));
    }

    @PostMapping("/reload")
    public Mono<Map<String, Object>> reload() {
        try {
            List<AgentDefinition> definitions = agentLoader.loadAll();
            int count = 0;
            for (AgentDefinition def : definitions) {
                agentFactory.build(def);
                count++;
            }
            log.info("Agent reload completed: {} agents rebuilt", count);
            logSink.tryEmitNext(Map.of(
                "level", "INFO",
                "timestamp", Instant.now().toString(),
                "message", "Agent reload completed: " + count + " agents rebuilt"
            ));
            return Mono.just(Map.of(
                "status", "success",
                "agents_reloaded", count,
                "total_agents", agentFactory.getBuiltAgents().size()
            ));
        } catch (Exception e) {
            log.error("Agent reload failed: {}", e.getMessage());
            return Mono.just(Map.of(
                "status", "error",
                "message", e.getMessage()
            ));
        }
    }

    @GetMapping("/agents")
    public Mono<Map<String, Object>> listAgents() {
        var agents = agentFactory.getBuiltAgents();
        List<Map<String, Object>> agentList = agents.values().stream()
            .map(a -> Map.<String, Object>of(
                "name", a.getName(),
                "type", a.getAgentType(),
                "description", a.getDescription()
            ))
            .toList();
        return Mono.just(Map.of(
            "agents", agentList,
            "count", agentList.size()
        ));
    }

    @GetMapping("/agents/{name}")
    public Mono<Map<String, Object>> getAgent(@PathVariable String name) {
        var agent = agentFactory.getBuiltAgents().get(name);
        if (agent == null) {
            return Mono.just(Map.of(
                "name", name,
                "status", "not_found",
                "message", "Agent not found: " + name
            ));
        }
        return Mono.just(Map.of(
            "name", agent.getName(),
            "type", agent.getAgentType(),
            "description", agent.getDescription(),
            "tools", agent.getTools()
        ));
    }

    @PostMapping("/agents")
    public Mono<Map<String, Object>> createAgent(@RequestBody Map<String, Object> definition) {
        return Mono.just(Map.of(
            "status", "stub",
            "message", "Agent creation not yet implemented"
        ));
    }

    @DeleteMapping("/agents/{name}")
    public Mono<Map<String, Object>> deleteAgent(@PathVariable String name) {
        return Mono.just(Map.of(
            "name", name,
            "status", "stub",
            "message", "Agent deletion not yet implemented"
        ));
    }

    @GetMapping("/mcp/servers")
    public Mono<Map<String, Object>> listMcpServers() {
        var servers = mcpRegistry.listServers();
        return Mono.just(Map.of(
            "servers", servers,
            "count", servers.size(),
            "connected", mcpRegistry.isConnected(servers.isEmpty() ? "" : servers.get(0))
        ));
    }

    @GetMapping(value = "/logs", produces = MediaType.TEXT_EVENT_STREAM_VALUE)
    public Flux<Map<String, Object>> logStream() {
        return logSink.asFlux()
            .startWith(Map.<String, Object>of(
                "level", "INFO",
                "timestamp", Instant.now().toString(),
                "message", "Connected to log stream"
            ));
    }

    public Sinks.Many<Map<String, Object>> getLogSink() {
        return logSink;
    }
}
