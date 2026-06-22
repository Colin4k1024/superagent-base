package io.superagent.memory;

import java.util.List;
import java.util.Map;

/**
 * Interface for agent memory stores.
 *
 * <p>Memory stores provide conversation history, context, and long-term
 * knowledge storage for agents. Implementations may be in-memory,
 * Redis-backed, or connected to external memory services.</p>
 *
 * <p>Maps to Go {@code MemoryStore} and Python {@code MemoryStore}.</p>
 */
public interface MemoryStore {

    /**
     * Store a message in the memory.
     *
     * @param sessionId conversation/session identifier
     * @param role      message role (user, assistant, system, tool)
     * @param content   message content
     * @param metadata  optional metadata
     */
    void store(String sessionId, String role, String content,
               Map<String, Object> metadata);

    /**
     * Retrieve recent messages for a session.
     *
     * @param sessionId conversation/session identifier
     * @param limit     maximum number of messages to return
     * @return list of messages in chronological order
     */
    List<MemoryMessage> retrieve(String sessionId, int limit);

    /**
     * Search memory by semantic similarity.
     *
     * @param sessionId conversation/session identifier
     * @param query     search query
     * @param limit     maximum results
     * @return matching messages ranked by relevance
     */
    List<MemoryMessage> search(String sessionId, String query, int limit);

    /**
     * Clear all memory for a session.
     */
    void clear(String sessionId);

    /**
     * A stored memory message.
     */
    record MemoryMessage(String id, String sessionId, String role,
                         String content, Map<String, Object> metadata,
                         long timestamp) {
    }
}
