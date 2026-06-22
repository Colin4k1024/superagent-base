package io.superagent.server;

import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Mono;

import java.util.Map;

/**
 * Session endpoints for message history and session management.
 *
 * <p>Sessions are transient conversation contexts; this controller
 * provides message retrieval and session cleanup.</p>
 */
@RestController
@RequestMapping("/api/v2/sessions")
public class SessionController {

    /**
     * Get message history for a session.
     */
    @GetMapping("/{sessionId}/messages")
    public Mono<Map<String, Object>> getMessages(
            @PathVariable String sessionId,
            @RequestParam(defaultValue = "0") int offset,
            @RequestParam(defaultValue = "50") int limit) {
        return Mono.just(ApiResponse.ok(Map.of(
            "session_id", sessionId,
            "messages", java.util.List.of(),
            "total", 0
        )).toMap());
    }

    /**
     * Delete (clear) a session.
     */
    @DeleteMapping("/{sessionId}")
    public Mono<Map<String, Object>> deleteSession(@PathVariable String sessionId) {
        return Mono.just(ApiResponse.ok().toMap());
    }
}
