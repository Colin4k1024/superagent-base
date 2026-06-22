package io.superagent.agents;

import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for {@link BaseAgent} contract.
 */
class BaseAgentTest {

    @Test
    void agentStoresNameAndType() {
        BaseAgent agent = new StubAgent("test-agent", "A test agent");
        assertEquals("test-agent", agent.getName());
        assertEquals("A test agent", agent.getDescription());
        assertEquals("stub", agent.getAgentType());
    }

    @Test
    void describeReturnsFormattedString() {
        BaseAgent agent = new StubAgent("my-agent", "Does things");
        assertEquals("[stub] my-agent: Does things", agent.describe());
    }

    @Test
    void defaultToolsIsEmpty() {
        BaseAgent agent = new StubAgent("test", "desc");
        assertTrue(agent.getTools().isEmpty());
    }

    @Test
    void runReturnsMap() {
        BaseAgent agent = new StubAgent("test", "desc");
        Map<String, Object> result = agent.run(Map.of("input", "hello"));
        assertNotNull(result);
        assertEquals("hello", result.get("echo"));
    }

    /**
     * Minimal concrete implementation for testing the abstract base.
     */
    private static class StubAgent extends BaseAgent {
        StubAgent(String name, String description) {
            super(name, description, "stub");
        }

        @Override
        public Map<String, Object> run(Map<String, Object> input) {
            return Map.of("echo", input.getOrDefault("input", ""));
        }
    }
}
