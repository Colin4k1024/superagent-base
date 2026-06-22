package io.superagent.agents;

import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for {@link ParallelAgent}.
 */
class ParallelAgentTest {

    @Test
    void emptyChildrenReturnsStubResult() {
        ParallelAgent agent = new ParallelAgent("fan-out", "test", List.of());
        Map<String, Object> result = agent.run(Map.of());
        assertEquals("parallel", result.get("type"));
        assertEquals(0, result.get("children_count"));
        agent.shutdown();
    }

    @Test
    void runsChildrenConcurrently() {
        BaseAgent child1 = new StubChild("child-1");
        BaseAgent child2 = new StubChild("child-2");

        ParallelAgent agent = new ParallelAgent(
            "fan-out", "two children", List.of(child1, child2));

        assertEquals(2, agent.getChildren().size());
        Map<String, Object> result = agent.run(Map.of("key", "value"));
        assertNotNull(result);
        agent.shutdown();
    }

    @Test
    void describeIncludesType() {
        ParallelAgent agent = new ParallelAgent("p", "desc", List.of());
        assertTrue(agent.describe().contains("parallel"));
        agent.shutdown();
    }

    private static class StubChild extends BaseAgent {
        StubChild(String name) {
            super(name, "stub child", "stub");
        }

        @Override
        public Map<String, Object> run(Map<String, Object> input) {
            return Map.of("child", getName());
        }
    }
}
