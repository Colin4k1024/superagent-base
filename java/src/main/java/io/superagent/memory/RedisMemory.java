package io.superagent.memory;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.data.redis.core.ReactiveStringRedisTemplate;
import org.springframework.stereotype.Component;

import java.time.Duration;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.UUID;

/**
 * Redis-backed memory store.
 *
 * <p>Key pattern: {@code superagent:memory:<sessionId>}
 * Each value is a JSON-serialised {@link MemoryStore.MemoryMessage} stored as a Redis LIST.
 * Messages are appended with RPUSH and trimmed to {@link #MAX_MESSAGES}.</p>
 */
@Component
public class RedisMemory implements MemoryStore {

    private static final Logger log = LoggerFactory.getLogger(RedisMemory.class);
    private static final String KEY_PREFIX = "superagent:memory:";
    private static final Duration TTL = Duration.ofHours(24);
    private static final int MAX_MESSAGES = 500;

    private final ReactiveStringRedisTemplate redis;
    private final ObjectMapper mapper = new ObjectMapper();

    public RedisMemory(ReactiveStringRedisTemplate redis) {
        this.redis = redis;
    }

    @Override
    public void store(String sessionId, String role, String content,
                      Map<String, Object> metadata) {
        String key = KEY_PREFIX + sessionId;
        MemoryMessage msg = new MemoryMessage(
                UUID.randomUUID().toString(),
                sessionId,
                role,
                content,
                metadata != null ? metadata : Map.of(),
                System.currentTimeMillis()
        );
        try {
            String json = mapper.writeValueAsString(msg);
            redis.opsForList().rightPush(key, json)
                .then(redis.expire(key, TTL))
                .then(trimIfNeeded(key))
                .subscribe(
                    v -> {},
                    e -> log.error("RedisMemory.store failed for session {}: {}", sessionId, e.getMessage())
                );
        } catch (JsonProcessingException e) {
            log.error("Failed to serialise message for session {}: {}", sessionId, e.getMessage());
        }
    }

    @Override
    public List<MemoryMessage> retrieve(String sessionId, int limit) {
        String key = KEY_PREFIX + sessionId;
        try {
            List<String> raw = redis.opsForList()
                .range(key, -Math.max(limit, 1), -1)
                .collectList()
                .block(Duration.ofSeconds(5));
            if (raw == null) return List.of();
            List<MemoryMessage> result = new ArrayList<>();
            for (String json : raw) {
                try {
                    result.add(mapper.readValue(json, MemoryMessage.class));
                } catch (JsonProcessingException e) {
                    log.warn("Skipping malformed message in session {}", sessionId);
                }
            }
            return result;
        } catch (Exception e) {
            log.error("RedisMemory.retrieve failed for session {}: {}", sessionId, e.getMessage());
            return List.of();
        }
    }

    @Override
    public List<MemoryMessage> search(String sessionId, String query, int limit) {
        // Simple substring search over recent messages
        return retrieve(sessionId, MAX_MESSAGES).stream()
            .filter(m -> m.content().toLowerCase().contains(query.toLowerCase()))
            .limit(limit)
            .toList();
    }

    @Override
    public void clear(String sessionId) {
        redis.delete(KEY_PREFIX + sessionId)
            .subscribe(
                v -> log.debug("Cleared memory for session {}", sessionId),
                e -> log.error("RedisMemory.clear failed: {}", e.getMessage())
            );
    }

    private reactor.core.publisher.Mono<Void> trimIfNeeded(String key) {
        return redis.opsForList().size(key)
            .flatMap(size -> {
                if (size > MAX_MESSAGES) {
                    return redis.opsForList().trim(key, size - MAX_MESSAGES, -1).then();
                }
                return reactor.core.publisher.Mono.<Void>empty();
            });
    }
}
