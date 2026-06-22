package io.superagent.harness;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Optional;

/**
 * File-based agent state store.
 *
 * <p>Stores state as JSON files in {@code ~/.agentscope/state/<agentId>/<userId>/<sessionId>.json}.
 * Each state file is a flat JSON object.</p>
 */
@Component
public class FileAgentStateStore implements AgentStateStore {

    private static final Logger log = LoggerFactory.getLogger(FileAgentStateStore.class);

    private final Path stateRoot;
    private final ObjectMapper objectMapper;

    public FileAgentStateStore() {
        this(Path.of(System.getProperty("user.home"), ".agentscope", "state"));
    }

    public FileAgentStateStore(Path stateRoot) {
        this.stateRoot = stateRoot;
        this.objectMapper = new ObjectMapper();
        this.objectMapper.registerModule(new JavaTimeModule());
    }

    @Override
    public Optional<Map<String, Object>> getState(String userId, String agentId, String sessionId) {
        Path file = statePath(agentId, userId, sessionId);
        if (!Files.exists(file)) {
            return Optional.empty();
        }
        try {
            String content = Files.readString(file);
            Map<String, Object> state = objectMapper.readValue(content, new TypeReference<>() {});
            return Optional.of(state);
        } catch (IOException e) {
            log.error("Failed to read state from {}: {}", file, e.getMessage());
            return Optional.empty();
        }
    }

    @Override
    public void saveState(String userId, String agentId, String sessionId, Map<String, Object> state) {
        Path file = statePath(agentId, userId, sessionId);
        try {
            Files.createDirectories(file.getParent());
            String json = objectMapper.writerWithDefaultPrettyPrinter().writeValueAsString(state);
            Files.writeString(file, json);
            log.debug("State saved to {}", file);
        } catch (IOException e) {
            log.error("Failed to save state to {}: {}", file, e.getMessage());
        }
    }

    @Override
    public void deleteState(String userId, String agentId, String sessionId) {
        Path file = statePath(agentId, userId, sessionId);
        try {
            Files.deleteIfExists(file);
            log.debug("State deleted: {}", file);
        } catch (IOException e) {
            log.error("Failed to delete state at {}: {}", file, e.getMessage());
        }
    }

    private Path statePath(String agentId, String userId, String sessionId) {
        return stateRoot.resolve(agentId).resolve(userId).resolve(sessionId + ".json");
    }
}
