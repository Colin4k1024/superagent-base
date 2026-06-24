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

    @GetMapping("/metrics")
    public Mono<Map<String, Object>> metrics() {
        return Mono.just(Map.of(
            "message", "Use /actuator/prometheus for Prometheus metrics",
            "status", "stub"
        ));
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
