package io.superagent.server;

import io.superagent.agents.AgentState;
import io.superagent.agents.BaseAgent;
import io.superagent.config.AgentBuilderFactory;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Mono;

import java.util.Map;

/**
 * Agent state CRUD endpoints.
 *
 * <p>Provides read/write access to per-agent key-value state,
 * used for interrupt/resume and persistent agent configuration.</p>
 */
@RestController
@RequestMapping("/api/v2/agents")
public class AgentStateController {

    private static final Logger log = LoggerFactory.getLogger(AgentStateController.class);

    private final AgentBuilderFactory agentFactory;

    public AgentStateController(AgentBuilderFactory agentFactory) {
        this.agentFactory = agentFactory;
    }

    /**
     * Get all state keys for an agent.
     */
    @GetMapping("/{agentId}/state")
    public Mono<Map<String, Object>> getState(@PathVariable String agentId) {
        BaseAgent agent = agentFactory.getBuiltAgents().get(agentId);
        if (agent == null) {
            return Mono.just(ApiResponse.error(404, "Agent not found: " + agentId).toMap());
        }
        AgentState state = agent.getState();
        return Mono.just(ApiResponse.ok(state.toMap()).toMap());
    }

    /**
     * Get a single state value by key.
     */
    @GetMapping("/{agentId}/state/{key}")
    public Mono<Map<String, Object>> getStateKey(
            @PathVariable String agentId,
            @PathVariable String key) {
        BaseAgent agent = agentFactory.getBuiltAgents().get(agentId);
        if (agent == null) {
            return Mono.just(ApiResponse.error(404, "Agent not found: " + agentId).toMap());
        }
        AgentState state = agent.getState();
        if (!state.has(key)) {
            return Mono.just(ApiResponse.error(404, "State key not found: " + key).toMap());
        }
        return Mono.just(ApiResponse.ok(Map.of(key, state.get(key))).toMap());
    }

    /**
     * Set a state key-value pair.
     */
    @PostMapping("/{agentId}/state")
    public Mono<Map<String, Object>> setState(
            @PathVariable String agentId,
            @RequestBody Map<String, Object> request) {
        BaseAgent agent = agentFactory.getBuiltAgents().get(agentId);
        if (agent == null) {
            return Mono.just(ApiResponse.error(404, "Agent not found: " + agentId).toMap());
        }
        String key = (String) request.get("key");
        Object value = request.get("value");
        if (key == null) {
            return Mono.just(ApiResponse.error(400, "Missing required field: key").toMap());
        }
        agent.getState().put(key, value);
        log.info("Set state for agent {}: {}={}", agentId, key, value);
        return Mono.just(ApiResponse.ok().toMap());
    }

    /**
     * Delete a state key.
     */
    @DeleteMapping("/{agentId}/state/{key}")
    public Mono<Map<String, Object>> deleteState(
            @PathVariable String agentId,
            @PathVariable String key) {
        BaseAgent agent = agentFactory.getBuiltAgents().get(agentId);
        if (agent == null) {
            return Mono.just(ApiResponse.error(404, "Agent not found: " + agentId).toMap());
        }
        agent.getState().remove(key);
        log.info("Deleted state key {} for agent {}", key, agentId);
        return Mono.just(ApiResponse.ok().toMap());
    }
}
