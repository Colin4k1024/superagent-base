package io.superagent.harness;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Instant;
import java.util.*;

/**
 * Session log management with append-only JSONL files.
 *
 * <p>Each session is stored as {@code sessions/<sessionId>.log.jsonl} —
 * one JSON object per line, appended after each message exchange.</p>
 */
@Component
public class SessionManager {

    private static final Logger log = LoggerFactory.getLogger(SessionManager.class);

    private final Path sessionsDir;
    private final ObjectMapper objectMapper;

    public SessionManager(WorkspaceConfig config) {
        this(config.getWorkspaceRoot().resolve("sessions"));
    }

    public SessionManager(Path sessionsDir) {
        this.sessionsDir = sessionsDir;
        this.objectMapper = new ObjectMapper();
        this.objectMapper.registerModule(new JavaTimeModule());
    }

    /**
     * Get all messages for a session.
     *
     * @param sessionId session identifier
     * @return list of messages in chronological order
     */
    public List<Map<String, Object>> getMessages(String sessionId) {
        Path file = sessionFile(sessionId);
        if (!Files.exists(file)) {
            return List.of();
        }
        List<Map<String, Object>> messages = new ArrayList<>();
        try {
            List<String> lines = Files.readAllLines(file);
            for (String line : lines) {
                if (line.isBlank()) continue;
                @SuppressWarnings("unchecked")
                Map<String, Object> msg = objectMapper.readValue(line, Map.class);
                messages.add(msg);
            }
        } catch (IOException e) {
            log.error("Failed to read session log {}: {}", file, e.getMessage());
        }
        return messages;
    }

    /**
     * Append a message to the session log.
     *
     * @param sessionId session identifier
     * @param role      message role (user, assistant, system, tool)
     * @param content   message content
     * @param metadata  optional metadata
     */
    public void appendMessage(String sessionId, String role, String content,
                              Map<String, Object> metadata) {
        Path file = sessionFile(sessionId);
        try {
            Files.createDirectories(file.getParent());
            Map<String, Object> entry = new LinkedHashMap<>();
            entry.put("role", role);
            entry.put("content", content);
            entry.put("timestamp", Instant.now().toString());
            if (metadata != null && !metadata.isEmpty()) {
                entry.put("metadata", metadata);
            }
            String json = objectMapper.writeValueAsString(entry);
            Files.writeString(file, json + System.lineSeparator(),
                    java.nio.file.StandardOpenOption.CREATE,
                    java.nio.file.StandardOpenOption.APPEND);
            log.debug("Appended message to session {}: {}", sessionId, role);
        } catch (IOException e) {
            log.error("Failed to append to session {}: {}", sessionId, e.getMessage());
        }
    }

    /**
     * Append a pre-built message map to the session log.
     */
    public void appendMessage(String sessionId, Map<String, Object> message) {
        Path file = sessionFile(sessionId);
        try {
            Files.createDirectories(file.getParent());
            String json = objectMapper.writeValueAsString(message);
            Files.writeString(file, json + System.lineSeparator(),
                    java.nio.file.StandardOpenOption.CREATE,
                    java.nio.file.StandardOpenOption.APPEND);
        } catch (IOException e) {
            log.error("Failed to append to session {}: {}", sessionId, e.getMessage());
        }
    }

    /**
     * Get the count of messages in a session.
     */
    public int getMessageCount(String sessionId) {
        return getMessages(sessionId).size();
    }

    /**
     * Delete a session log file.
     */
    public void deleteSession(String sessionId) {
        try {
            Files.deleteIfExists(sessionFile(sessionId));
        } catch (IOException e) {
            log.error("Failed to delete session {}: {}", sessionId, e.getMessage());
        }
    }

    /**
     * List all session IDs.
     */
    public List<String> listSessions() {
        List<String> sessions = new ArrayList<>();
        if (!Files.isDirectory(sessionsDir)) {
            return sessions;
        }
        try (var stream = Files.list(sessionsDir)) {
            stream.filter(p -> p.toString().endsWith(".log.jsonl"))
                    .forEach(p -> {
                        String name = p.getFileName().toString();
                        sessions.add(name.replace(".log.jsonl", ""));
                    });
        } catch (IOException e) {
            log.error("Failed to list sessions: {}", e.getMessage());
        }
        return sessions;
    }

    private Path sessionFile(String sessionId) {
        return sessionsDir.resolve(sessionId + ".log.jsonl");
    }
}
