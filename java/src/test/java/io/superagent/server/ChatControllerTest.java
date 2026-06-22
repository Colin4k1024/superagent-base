package io.superagent.server;

import io.superagent.config.AgentBuilderFactory;
import io.superagent.config.YamlAgentLoader;
import io.superagent.mcp.MCPRegistry;
import io.superagent.models.ModelRegistry;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.reactive.WebFluxTest;
import org.springframework.test.context.ContextConfiguration;
import org.springframework.test.web.reactive.server.WebTestClient;

import java.util.Map;

/**
 * Tests for {@link ChatController} endpoints.
 */
@WebFluxTest(ChatController.class)
@ContextConfiguration(classes = {
    ChatController.class,
    ChatControllerTest.TestBeans.class
})
class ChatControllerTest {

    @Autowired
    private WebTestClient webClient;

    @org.springframework.context.annotation.Configuration
    static class TestBeans {
        @org.springframework.context.annotation.Bean
        ModelRegistry modelRegistry() {
            return new ModelRegistry();
        }

        @org.springframework.context.annotation.Bean
        AgentBuilderFactory agentBuilderFactory(ModelRegistry reg) {
            return new AgentBuilderFactory(reg);
        }

        @org.springframework.context.annotation.Bean
        YamlAgentLoader yamlAgentLoader() {
            return new YamlAgentLoader("configs/agents");
        }

        @org.springframework.context.annotation.Bean
        MCPRegistry mcpRegistry() {
            return new MCPRegistry();
        }
    }

    @Test
    void chatReturnsErrorResponseForMissingAgent() {
        webClient.post()
            .uri("/api/v2/chat")
            .bodyValue(Map.of(
                "agent_id", "test-agent",
                "session_id", "s1",
                "message", "hello"
            ))
            .exchange()
            .expectStatus().isOk()
            .expectBody()
            .jsonPath("$.agent_id").isEqualTo("test-agent")
            .jsonPath("$.status").isEqualTo("error");
    }

    @Test
    void listAgentsReturnsCount() {
        webClient.get()
            .uri("/api/v2/agents")
            .exchange()
            .expectStatus().isOk()
            .expectBody()
            .jsonPath("$.count").isEqualTo(0);
    }

    @Test
    void interruptStateReturnsFalse() {
        webClient.get()
            .uri("/api/v2/chat/interrupt_state?session_id=s1")
            .exchange()
            .expectStatus().isOk()
            .expectBody()
            .jsonPath("$.interrupted").isEqualTo(false);
    }

    @Test
    void resumeReturnsStub() {
        webClient.post()
            .uri("/api/v2/chat/resume")
            .bodyValue(Map.of("session_id", "s1"))
            .exchange()
            .expectStatus().isOk()
            .expectBody()
            .jsonPath("$.status").isEqualTo("stub");
    }
}
