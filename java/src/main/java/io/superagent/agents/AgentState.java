package io.superagent.agents;

import java.io.Serializable;
import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Set;

/**
 * Serializable state container for an agent session.
 *
 * <p>Stores key-value state that persists across conversation turns.
 * Used for interrupt/resume and checkpoint scenarios.</p>
 *
 * <p>Note: This is a Superagent-specific state class. The real AgentScope
 * {@code io.agentscope.core.state.AgentState} has a different API focused
 * on internal agent lifecycle management.</p>
 */
public class AgentState implements Serializable {

    private static final long serialVersionUID = 1L;

    private final Map<String, Object> data;
    private Instant lastModified;

    public AgentState() {
        this.data = new LinkedHashMap<>();
        this.lastModified = Instant.now();
    }

    public AgentState(Map<String, Object> initialData) {
        this.data = new LinkedHashMap<>(initialData != null ? initialData : Map.of());
        this.lastModified = Instant.now();
    }

    @SuppressWarnings("unchecked")
    public <T> T get(String key) {
        return (T) data.get(key);
    }

    @SuppressWarnings("unchecked")
    public <T> T getOrDefault(String key, T defaultValue) {
        return (T) data.getOrDefault(key, defaultValue);
    }

    public void put(String key, Object value) {
        data.put(key, value);
        lastModified = Instant.now();
    }

    public boolean has(String key) {
        return data.containsKey(key);
    }

    public void remove(String key) {
        data.remove(key);
        lastModified = Instant.now();
    }

    public Set<String> keys() {
        return Set.copyOf(data.keySet());
    }

    public Map<String, Object> toMap() {
        return Map.copyOf(data);
    }

    public boolean isEmpty() {
        return data.isEmpty();
    }

    public Instant lastModified() {
        return lastModified;
    }
}
