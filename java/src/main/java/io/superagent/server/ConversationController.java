package io.superagent.server;

import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Mono;

import java.time.Instant;
import java.util.*;

/**
 * Conversation CRUD + message endpoints.
 *
 * <p>Stub implementation returning empty data structures with correct
 * response format matching the Go base v2 API.</p>
 */
@RestController
@RequestMapping("/api/v2/conversations")
public class ConversationController {

    /**
     * List conversations.
     */
    @GetMapping
    public Mono<Map<String, Object>> list(
            @RequestParam(defaultValue = "0") int offset,
            @RequestParam(defaultValue = "20") int limit) {
        return Mono.just(ApiResponse.ok(Map.of(
            "conversations", List.of(),
            "total", 0
        )).toMap());
    }

    /**
     * Create a new conversation.
     */
    @PostMapping
    public Mono<Map<String, Object>> create(@RequestBody Map<String, Object> request) {
        String id = UUID.randomUUID().toString();
        String agentId = (String) request.getOrDefault("agent_id", "default");
        Map<String, Object> conversation = new LinkedHashMap<>();
        conversation.put("id", id);
        conversation.put("agent_id", agentId);
        conversation.put("title", request.getOrDefault("title", ""));
        conversation.put("created_at", Instant.now().toString());
        conversation.put("updated_at", Instant.now().toString());
        return Mono.just(ApiResponse.ok(conversation).toMap());
    }

    /**
     * Get a conversation by ID.
     */
    @GetMapping("/{id}")
    public Mono<Map<String, Object>> get(@PathVariable String id) {
        Map<String, Object> conversation = new LinkedHashMap<>();
        conversation.put("id", id);
        conversation.put("agent_id", "default");
        conversation.put("title", "");
        conversation.put("created_at", Instant.now().toString());
        conversation.put("updated_at", Instant.now().toString());
        return Mono.just(ApiResponse.ok(conversation).toMap());
    }

    /**
     * Update a conversation.
     */
    @PutMapping("/{id}")
    public Mono<Map<String, Object>> update(
            @PathVariable String id,
            @RequestBody Map<String, Object> request) {
        Map<String, Object> conversation = new LinkedHashMap<>();
        conversation.put("id", id);
        conversation.put("agent_id", request.getOrDefault("agent_id", "default"));
        conversation.put("title", request.getOrDefault("title", ""));
        conversation.put("updated_at", Instant.now().toString());
        return Mono.just(ApiResponse.ok(conversation).toMap());
    }

    /**
     * Delete a conversation.
     */
    @DeleteMapping("/{id}")
    public Mono<Map<String, Object>> delete(@PathVariable String id) {
        return Mono.just(ApiResponse.ok().toMap());
    }

    /**
     * Get messages for a conversation.
     */
    @GetMapping("/{conversationId}/messages")
    public Mono<Map<String, Object>> getMessages(
            @PathVariable String conversationId,
            @RequestParam(defaultValue = "0") int offset,
            @RequestParam(defaultValue = "50") int limit) {
        return Mono.just(ApiResponse.ok(Map.of(
            "messages", List.of(),
            "total", 0
        )).toMap());
    }

    /**
     * Delete a specific message from a conversation.
     */
    @DeleteMapping("/{conversationId}/messages/{messageId}")
    public Mono<Map<String, Object>> deleteMessage(
            @PathVariable String conversationId,
            @PathVariable String messageId) {
        return Mono.just(ApiResponse.ok().toMap());
    }
}
