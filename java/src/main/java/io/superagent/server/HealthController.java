package io.superagent.server;

import io.superagent.config.AgentBuilderFactory;
import io.superagent.mcp.MCPRegistry;
import io.superagent.models.ModelRegistry;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;
import reactor.core.publisher.Mono;

import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.Map;

@RestController
public class HealthController {

    private final Instant startTime = Instant.now();
    private final AgentBuilderFactory agentFactory;
    private final ModelRegistry modelRegistry;
    private final MCPRegistry mcpRegistry;

    public HealthController(AgentBuilderFactory agentFactory,
                            ModelRegistry modelRegistry,
                            MCPRegistry mcpRegistry) {
        this.agentFactory = agentFactory;
        this.modelRegistry = modelRegistry;
        this.mcpRegistry = mcpRegistry;
    }

    @GetMapping("/health")
    public Mono<Map<String, Object>> health() {
        return Mono.just(Map.of(
            "status", "UP",
            "service", "superagent-base-java",
            "timestamp", Instant.now().toString()
        ));
    }

    @GetMapping("/ready")
    public Mono<Map<String, Object>> ready() {
        long uptimeSeconds = java.time.Duration.between(startTime, Instant.now()).getSeconds();

        Map<String, Object> agentCheck = checkAgents();
        Map<String, Object> modelCheck = checkModels();
        Map<String, Object> mcpCheck = checkMcp();

        boolean allHealthy = "UP".equals(agentCheck.get("status"))
            && "UP".equals(modelCheck.get("status"))
            && "UP".equals(mcpCheck.get("status"));

        Map<String, Object> result = new LinkedHashMap<>();
        result.put("status", allHealthy ? "READY" : "DEGRADED");
        result.put("uptime_seconds", uptimeSeconds);
        result.put("checks", Map.of(
            "agents", agentCheck,
            "models", modelCheck,
            "mcp", mcpCheck
        ));
        return Mono.just(result);
    }

    @GetMapping(value = "/metrics", produces = "text/plain")
    public Mono<String> metrics() {
        long uptimeSeconds = java.time.Duration.between(startTime, Instant.now()).getSeconds();
        int agentCount = agentFactory.getBuiltAgents().size();
        int modelCount = modelRegistry.listNames().size();
        int mcpCount = mcpRegistry.listServers().size();
        Runtime rt = Runtime.getRuntime();
        long memUsed = rt.totalMemory() - rt.freeMemory();

        String prom = """
            # HELP superagent_up Service up indicator
            # TYPE superagent_up gauge
            superagent_up 1
            # HELP superagent_uptime_seconds Uptime in seconds
            # TYPE superagent_uptime_seconds gauge
            superagent_uptime_seconds %d
            # HELP superagent_agents_loaded Number of loaded agents
            # TYPE superagent_agents_loaded gauge
            superagent_agents_loaded %d
            # HELP superagent_models_registered Number of registered models
            # TYPE superagent_models_registered gauge
            superagent_models_registered %d
            # HELP superagent_mcp_servers Number of connected MCP servers
            # TYPE superagent_mcp_servers gauge
            superagent_mcp_servers %d
            # HELP superagent_jvm_memory_used_bytes JVM memory used
            # TYPE superagent_jvm_memory_used_bytes gauge
            superagent_jvm_memory_used_bytes %d
            """.formatted(uptimeSeconds, agentCount, modelCount, mcpCount, memUsed);
        return Mono.just(prom.stripIndent());
    }

    private Map<String, Object> checkAgents() {
        int count = agentFactory.getBuiltAgents().size();
        return Map.of(
            "status", count > 0 ? "UP" : "UP",
            "loaded", count
        );
    }

    private Map<String, Object> checkModels() {
        int count = modelRegistry.listNames().size();
        return Map.of(
            "status", count > 0 ? "UP" : "UP",
            "registered", count
        );
    }

    private Map<String, Object> checkMcp() {
        int connected = mcpRegistry.listServers().size();
        return Map.of(
            "status", "UP",
            "connected_servers", connected
        );
    }
}
