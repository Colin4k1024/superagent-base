package io.superagent.agents;

import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for {@link WorkflowAgent} including topological sort.
 */
class WorkflowAgentTest {

    @Test
    void emptyWorkflowReturnsStubResult() {
        WorkflowAgent agent = new WorkflowAgent("wf", "test", List.of(), List.of());
        Map<String, Object> result = agent.run(Map.of());
        assertEquals("workflow", result.get("type"));
        assertEquals(0, result.get("nodes_count"));
    }

    @Test
    void topologicalSortRespectsOrder() {
        var nodeA = new WorkflowAgent.WorkflowNode("a", "llm_call", null, Map.of());
        var nodeB = new WorkflowAgent.WorkflowNode("b", "tool_call", null, Map.of());
        var nodeC = new WorkflowAgent.WorkflowNode("c", "llm_call", null, Map.of());

        var edge1 = new WorkflowAgent.WorkflowEdge("a", "b", null);
        var edge2 = new WorkflowAgent.WorkflowEdge("b", "c", null);

        WorkflowAgent agent = new WorkflowAgent(
            "pipeline", "A→B→C", List.of(nodeA, nodeB, nodeC), List.of(edge1, edge2));

        Map<String, Object> result = agent.run(Map.of());
        @SuppressWarnings("unchecked")
        List<String> order = (List<String>) result.get("execution_order");

        assertNotNull(order);
        assertEquals(3, order.size());
        assertEquals("a", order.get(0));
        assertEquals("b", order.get(1));
        assertEquals("c", order.get(2));
    }

    @Test
    void parallelNodesHaveSameDepth() {
        var nodeA = new WorkflowAgent.WorkflowNode("a", "llm_call", null, Map.of());
        var nodeB = new WorkflowAgent.WorkflowNode("b", "llm_call", null, Map.of());
        var nodeC = new WorkflowAgent.WorkflowNode("c", "llm_call", null, Map.of());

        // A→B, A→C (B and C are parallel)
        var edge1 = new WorkflowAgent.WorkflowEdge("a", "b", null);
        var edge2 = new WorkflowAgent.WorkflowEdge("a", "c", null);

        WorkflowAgent agent = new WorkflowAgent(
            "fan-out", "A→{B,C}", List.of(nodeA, nodeB, nodeC), List.of(edge1, edge2));

        Map<String, Object> result = agent.run(Map.of());
        @SuppressWarnings("unchecked")
        List<String> order = (List<String>) result.get("execution_order");

        assertEquals(3, order.size());
        assertEquals("a", order.get(0));
        // B and C should both come after A (order between them is not guaranteed)
        assertTrue(order.indexOf("b") > 0);
        assertTrue(order.indexOf("c") > 0);
    }

    @Test
    void describeIncludesType() {
        WorkflowAgent agent = new WorkflowAgent("wf", "desc", List.of(), List.of());
        assertTrue(agent.describe().contains("workflow"));
    }
}
