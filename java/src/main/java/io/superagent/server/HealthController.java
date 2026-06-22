package io.superagent.server;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;
import reactor.core.publisher.Mono;

import java.time.Instant;
import java.util.Map;

/**
 * Health and readiness endpoints.
 */
@RestController
public class HealthController {

    private final Instant startTime = Instant.now();

    /**
     * Liveness check — always returns 200 if the process is running.
     */
    @GetMapping("/health")
    public Mono<Map<String, Object>> health() {
        return Mono.just(Map.of(
            "status", "UP",
            "service", "superagent-base-java",
            "timestamp", Instant.now().toString()
        ));
    }

    /**
     * Readiness check — returns 200 when the service can accept traffic.
     */
    @GetMapping("/ready")
    public Mono<Map<String, Object>> ready() {
        // TODO: Check agent registry, model connections, Redis connectivity
        long uptimeSeconds = java.time.Duration.between(startTime, Instant.now()).getSeconds();
        return Mono.just(Map.of(
            "status", "READY",
            "uptime_seconds", uptimeSeconds,
            "checks", Map.of(
                "agents", "stub",
                "models", "stub",
                "redis", "stub"
            )
        ));
    }

    /**
     * Prometheus metrics endpoint (also available via Actuator).
     */
    @GetMapping("/metrics")
    public Mono<Map<String, Object>> metrics() {
        // Note: Real metrics served by Actuator /actuator/prometheus
        return Mono.just(Map.of(
            "message", "Use /actuator/prometheus for Prometheus metrics",
            "status", "stub"
        ));
    }
}
