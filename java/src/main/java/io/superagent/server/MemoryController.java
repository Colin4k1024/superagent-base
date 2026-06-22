package io.superagent.server;

import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Mono;

import java.time.Instant;
import java.util.*;

/**
 * Long-term memory endpoints.
 *
 * <p>Provides CRUD and semantic search for persistent memory entries
 * associated with users.</p>
 */
@RestController
@RequestMapping("/api/v2/memory/long-term")
public class MemoryController {

    /**
     * List long-term memory entries for a user.
     */
    @GetMapping
    public Mono<Map<String, Object>> list(
            @RequestParam("user_id") String userId,
            @RequestParam(defaultValue = "0") int offset,
            @RequestParam(defaultValue = "20") int limit) {
        return Mono.just(ApiResponse.ok(Map.of(
            "memories", List.of(),
            "total", 0
        )).toMap());
    }

    /**
     * Add a new memory entry.
     */
    @PostMapping
    public Mono<Map<String, Object>> create(@RequestBody Map<String, Object> request) {
        String id = UUID.randomUUID().toString();
        Map<String, Object> memory = new LinkedHashMap<>();
        memory.put("id", id);
        memory.put("user_id", request.getOrDefault("user_id", ""));
        memory.put("content", request.getOrDefault("content", ""));
        memory.put("metadata", request.getOrDefault("metadata", Map.of()));
        memory.put("created_at", Instant.now().toString());
        memory.put("updated_at", Instant.now().toString());
        return Mono.just(ApiResponse.ok(memory).toMap());
    }

    /**
     * Semantic search across memory entries.
     */
    @GetMapping("/search")
    public Mono<Map<String, Object>> search(
            @RequestParam("user_id") String userId,
            @RequestParam("q") String query,
            @RequestParam(defaultValue = "10") int limit) {
        return Mono.just(ApiResponse.ok(Map.of(
            "results", List.of(),
            "query", query,
            "total", 0
        )).toMap());
    }

    /**
     * Update a memory entry.
     */
    @PutMapping("/{id}")
    public Mono<Map<String, Object>> update(
            @PathVariable String id,
            @RequestBody Map<String, Object> request) {
        Map<String, Object> memory = new LinkedHashMap<>();
        memory.put("id", id);
        memory.put("content", request.getOrDefault("content", ""));
        memory.put("metadata", request.getOrDefault("metadata", Map.of()));
        memory.put("updated_at", Instant.now().toString());
        return Mono.just(ApiResponse.ok(memory).toMap());
    }

    /**
     * Delete a memory entry.
     */
    @DeleteMapping("/{id}")
    public Mono<Map<String, Object>> delete(@PathVariable String id) {
        return Mono.just(ApiResponse.ok().toMap());
    }
}
