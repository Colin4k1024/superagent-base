package io.superagent.harness;

import java.util.Map;
import java.util.Optional;

/**
 * Interface for agent state persistence.
 *
 * <p>Stores per-user, per-agent, per-session state that persists across
 * conversation turns. Used for interrupt/resume and checkpoint scenarios.</p>
 *
 * <p>Implementations: {@link FileAgentStateStore}, {@link RedisAgentStateStore}.</p>
 */
public interface AgentStateStore {

    /**
     * Get state for a specific user/agent/session combination.
     *
     * @param userId    user identifier
     * @param agentId   agent identifier
     * @param sessionId session identifier
     * @return state map, or empty if no state exists
     */
    Optional<Map<String, Object>> getState(String userId, String agentId, String sessionId);

    /**
     * Save state for a specific user/agent/session combination.
     *
     * @param userId    user identifier
     * @param agentId   agent identifier
     * @param sessionId session identifier
     * @param state     state map to persist
     */
    void saveState(String userId, String agentId, String sessionId, Map<String, Object> state);

    /**
     * Delete state for a specific user/agent/session combination.
     *
     * @param userId    user identifier
     * @param agentId   agent identifier
     * @param sessionId session identifier
     */
    void deleteState(String userId, String agentId, String sessionId);
}
