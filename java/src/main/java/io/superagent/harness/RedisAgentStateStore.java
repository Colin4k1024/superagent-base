package io.superagent.harness;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.autoconfigure.condition.ConditionalOnBean;
import org.springframework.data.redis.core.ReactiveRedisTemplate;
import org.springframework.stereotype.Component;
import reactor.core.publisher.Mono;

import java.time.Duration;
import java.util.Map;
import java.util.Optional;

/**
 * Redis-backed agent state store using Spring ReactiveRedisTemplate.
 *
 * <p>Stores state as JSON strings in Redis with a TTL of 7 days.
 * Key format: {@code harness:state:<agentId>:<userId>:<sessionId>}.</p>
 *
 * <p>Only activated when a {@link ReactiveRedisTemplate} bean is available.</p>
 */
@Component
@ConditionalOnBean(ReactiveRedisTemplate.class)
public class RedisAgentStateStore implements AgentStateStore {

    private static final Logger log = LoggerFactory.getLogger(RedisAgentStateStore.class);
    private static final String KEY_PREFIX = "harness:state:";
    private static final Duration TTL = Duration.ofDays(7);

    private final ReactiveRedisTemplate<String, String> redisTemplate;
    private final ObjectMapper objectMapper;

    public RedisAgentStateStore(ReactiveRedisTemplate<String, String> redisTemplate) {
        this.redisTemplate = redisTemplate;
        this.objectMapper = new ObjectMapper();
    }

    @Override
    public Optional<Map<String, Object>> getState(String userId, String agentId, String sessionId) {
        String key = keyFor(agentId, userId, sessionId);
        try {
            String json = redisTemplate.opsForValue().get(key).block();
            if (json == null) {
                return Optional.empty();
            }
            Map<String, Object> state = objectMapper.readValue(json, new TypeReference<>() {});
            return Optional.of(state);
        } catch (Exception e) {
            log.error("Failed to get state from Redis for key {}: {}", key, e.getMessage());
            return Optional.empty();
        }
    }

    @Override
    public void saveState(String userId, String agentId, String sessionId, Map<String, Object> state) {
        String key = keyFor(agentId, userId, sessionId);
        try {
            String json = objectMapper.writeValueAsString(state);
            redisTemplate.opsForValue().set(key, json, TTL).block();
            log.debug("State saved to Redis: {}", key);
        } catch (Exception e) {
            log.error("Failed to save state to Redis for key {}: {}", key, e.getMessage());
        }
    }

    @Override
    public void deleteState(String userId, String agentId, String sessionId) {
        String key = keyFor(agentId, userId, sessionId);
        try {
            redisTemplate.delete(key).block();
            log.debug("State deleted from Redis: {}", key);
        } catch (Exception e) {
            log.error("Failed to delete state from Redis for key {}: {}", key, e.getMessage());
        }
    }

    private String keyFor(String agentId, String userId, String sessionId) {
        return KEY_PREFIX + agentId + ":" + userId + ":" + sessionId;
    }
}
