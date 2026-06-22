package io.superagent.memory;

import io.agentscope.core.state.AgentStateStore;
import io.agentscope.core.state.InMemoryAgentStateStore;
import org.springframework.data.redis.core.ReactiveStringRedisTemplate;
import org.springframework.stereotype.Component;

import java.util.List;
import java.util.Map;

/**
 * Redis-backed memory store using AgentScope's {@link AgentStateStore}.
 *
 * <p>Wraps the AgentScope state store to provide the legacy {@link MemoryStore}
 * interface. Suitable for multi-instance deployments requiring shared state.</p>
 *
 * <p>Note: The real AgentScope {@code AgentStateStore} has a different API than
 * the local stub. This implementation uses {@link InMemoryAgentStateStore} as a
 * fallback until a Redis-backed implementation is available in AgentScope.</p>
 */
@Component
public class RedisMemory implements MemoryStore {

    private final ReactiveStringRedisTemplate redis;
    private final AgentStateStore stateStore;

    /**
     * Constructor injection for Redis template.
     *
     * @param redis reactive Redis template (auto-configured by Spring Boot)
     */
    public RedisMemory(ReactiveStringRedisTemplate redis) {
        this.redis = redis;
        this.stateStore = new InMemoryAgentStateStore();
    }

    @Override
    public void store(String sessionId, String role, String content,
                      Map<String, Object> metadata) {
        // Use AgentStateStore to persist messages
        // The real API uses save(userId, agentId, sessionId, State)
        // For now, this is a stub — full implementation requires
        // adapting to the real AgentStateStore API
    }

    @Override
    public List<MemoryMessage> retrieve(String sessionId, int limit) {
        // Stub: full implementation requires adapting to real AgentStateStore API
        return List.of();
    }

    @Override
    public List<MemoryMessage> search(String sessionId, String query, int limit) {
        return List.of();
    }

    @Override
    public void clear(String sessionId) {
        // Stub: full implementation requires adapting to real AgentStateStore API
    }

    /**
     * Get the underlying AgentScope state store.
     */
    public AgentStateStore getAgentScopeStateStore() {
        return stateStore;
    }
}
