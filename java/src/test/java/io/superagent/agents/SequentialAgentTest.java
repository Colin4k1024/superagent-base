package io.superagent.agents;

import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for {@link SequentialAgent}.
 */
class SequentialAgentTest {

    @Test
    void emptyStepsReturnsStubResult() {
        SequentialAgent agent = new SequentialAgent("pipeline", "test", List.of());
        Map<String, Object> result = agent.run(Map.of());
        assertEquals("sequential", result.get("type"));
        assertEquals(0, result.get("steps_count"));
    }

    @Test
    void preservesStepOrder() {
        BaseAgent step1 = new StubStep("step-1");
        BaseAgent step2 = new StubStep("step-2");
        BaseAgent step3 = new StubStep("step-3");

        SequentialAgent agent = new SequentialAgent(
            "pipeline", "three step pipeline", List.of(step1, step2, step3));

        assertEquals(3, agent.getSteps().size());
        assertEquals("step-1", agent.getSteps().get(0).getName());
        assertEquals("step-3", agent.getSteps().get(2).getName());
    }

    @Test
    void describeIncludesType() {
        SequentialAgent agent = new SequentialAgent("pipe", "desc", List.of());
        assertTrue(agent.describe().contains("sequential"));
    }

    private static class StubStep extends BaseAgent {
        StubStep(String name) {
            super(name, "stub step", "stub");
        }

        @Override
        public Map<String, Object> run(Map<String, Object> input) {
            return Map.of("step", getName());
        }
    }
}
