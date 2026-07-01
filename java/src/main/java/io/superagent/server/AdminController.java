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
import java.util.LinkedHashMap;
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

    private static final long START_TIME = System.currentTimeMillis();

    @GetMapping("/status")
    public Mono<Map<String, Object>> status() {
        Runtime rt = Runtime.getRuntime();
        Map<String, Object> data = new LinkedHashMap<>();
        data.put("status", "running");
        data.put("version", "0.1.0-java");
        data.put("runtime", "Spring Boot 3 + WebFlux");
        data.put("agents_loaded", agentFactory.getBuiltAgents().size());
        data.put("agent_names", agentFactory.getBuiltAgents().keySet().stream().toList());
        data.put("mcp_servers", mcpRegistry.listServers().size());
        data.put("uptime_seconds", (System.currentTimeMillis() - START_TIME) / 1000);
        data.put("memory_used_mb", (rt.totalMemory() - rt.freeMemory()) / (1024 * 1024));
        data.put("memory_max_mb", rt.maxMemory() / (1024 * 1024));
        data.put("processors", rt.availableProcessors());
        return Mono.just(data);
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

    @PostMapping("/agents/validate")
    public Mono<Map<String, Object>> validateAgent(@RequestBody Map<String, Object> body) {
        // Accept both "yaml_definition" (legacy) and "yaml" (SDK default)
        String yamlDef = (String) body.getOrDefault("yaml_definition",
                         body.getOrDefault("yaml", ""));
        if (yamlDef.isBlank()) {
            return Mono.just(Map.of("valid", false, "error", "yaml_definition is required"));
        }
        try {
            List<AgentDefinition> defs = agentLoader.loadFromString(yamlDef);
            if (defs.isEmpty()) {
                return Mono.just(Map.of("valid", false, "error", "No agent definition found in YAML"));
            }
            AgentDefinition def = defs.get(0);
            return Mono.just(Map.of(
                "valid", true,
                "name", def.metadata() != null && def.metadata().name() != null ? def.metadata().name() : "",
                "type", def.spec() != null && def.spec().type() != null ? def.spec().type() : ""
            ));
        } catch (Exception e) {
            return Mono.just(Map.of("valid", false, "error", e.getMessage()));
        }
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

    @PutMapping("/agents/{name}")
    public Mono<Map<String, Object>> updateAgent(@PathVariable String name,
                                                  @RequestBody Map<String, Object> body) {
        return Mono.just(Map.of(
            "status", "stub",
            "message", "Agent update not yet implemented"
        ));
    }

    @PostMapping("/mcp/servers")
    public Mono<Map<String, Object>> connectMcpServer(@RequestBody Map<String, Object> body) {
        String name = (String) body.getOrDefault("name", "");
        String url = (String) body.getOrDefault("url", "");
        return Mono.just(Map.of(
            "status", "ok",
            "name", name,
            "connected", false,
            "message", "MCP connect stub"
        ));
    }

    @DeleteMapping("/mcp/servers/{name}")
    public Mono<Map<String, Object>> disconnectMcpServer(@PathVariable String name) {
        return Mono.just(Map.of(
            "status", "ok",
            "name", name,
            "disconnected", true
        ));
    }

    @GetMapping("/mcp/servers/{name}/tools")
    public Mono<Map<String, Object>> getMcpServerTools(@PathVariable String name) {
        return Mono.just(Map.of(
            "server", name,
            "tools", List.of(),
            "count", 0
        ));
    }
}
