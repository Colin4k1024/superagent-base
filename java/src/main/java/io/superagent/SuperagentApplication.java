package io.superagent;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableAsync;

/**
 * Main entry point for Superagent Base Java.
 *
 * <p>Starts a Spring Boot WebFlux server exposing REST + SSE endpoints
 * for agent management, chat, and administration.</p>
 *
 * <p>Default port: 8890 (configurable via {@code server.port}).</p>
 */
@SpringBootApplication
@EnableAsync
public class SuperagentApplication {

    public static void main(String[] args) {
        SpringApplication.run(SuperagentApplication.class, args);
    }
}
