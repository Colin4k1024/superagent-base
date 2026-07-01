package io.superagent.server;

import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Mono;

import java.util.Map;

/**
 * Skills and tools listing endpoints.
 *
 * <p>Provides discovery of installed skills, skill search,
 * and registered tool schemas.</p>
 */
@RestController
@RequestMapping("/api/v2")
public class SkillToolController {

    /**
     * List installed skills.
     */
    @GetMapping("/skills")
    public Mono<Map<String, Object>> listSkills() {
        return Mono.just(ApiResponse.ok(Map.of(
            "skills", java.util.List.of(),
            "total", 0
        )).toMap());
    }

    /**
     * Search available skills.
     */
    @GetMapping("/skills/search")
    public Mono<Map<String, Object>> searchSkills(
            @RequestParam("q") String query,
            @RequestParam(defaultValue = "10") int limit) {
        return Mono.just(ApiResponse.ok(Map.of(
            "results", java.util.List.of(),
            "query", query,
            "total", 0
        )).toMap());
    }

    /**
     * List all registered tools and their schemas.
     */
    @GetMapping("/tools")
    public Mono<Map<String, Object>> listTools() {
        return Mono.just(ApiResponse.ok(Map.of(
            "tools", java.util.List.of(),
            "total", 0
        )).toMap());
    }

    /**
     * Get current user info.
     */
    @GetMapping("/me")
    public Mono<Map<String, Object>> me() {
        return Mono.just(ApiResponse.ok(Map.of(
            "id", "local",
            "name", "Local User",
            "email", "user@localhost",
            "role", "admin"
        )).toMap());
    }

    /**
     * Install a skill.
     */
    @PostMapping("/skills/install")
    public Mono<Map<String, Object>> installSkill(@RequestBody Map<String, Object> body) {
        String name = (String) body.getOrDefault("name", "");
        String source = (String) body.getOrDefault("source", "local");
        return Mono.just(Map.of(
            "status", "ok",
            "name", name,
            "source", source,
            "installed", true
        ));
    }

    /**
     * Uninstall a skill.
     */
    @DeleteMapping("/skills/{name}")
    public Mono<Map<String, Object>> uninstallSkill(@PathVariable String name) {
        return Mono.just(Map.of(
            "status", "ok",
            "name", name,
            "uninstalled", true
        ));
    }
}
