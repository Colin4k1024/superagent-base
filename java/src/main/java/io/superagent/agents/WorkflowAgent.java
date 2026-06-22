package io.superagent.agents;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * DAG-based workflow agent with topological sort execution.
 *
 * <p>Nodes are connected by directed edges. Execution follows a topological
 * ordering so that all dependencies are satisfied before a node runs.
 * Supports conditional branching via node-level predicates.</p>
 *
 * <p>Maps to Go {@code WorkflowAgent} and Python {@code WorkflowAgent}.</p>
 */
public class WorkflowAgent extends BaseAgent {

    private final List<WorkflowNode> nodes;
    private final List<WorkflowEdge> edges;

    public WorkflowAgent(String name, String description,
                         List<WorkflowNode> nodes, List<WorkflowEdge> edges) {
        super(name, description, "workflow");
        this.nodes = nodes != null ? List.copyOf(nodes) : List.of();
        this.edges = edges != null ? List.copyOf(edges) : List.of();
    }

    @Override
    public Map<String, Object> run(Map<String, Object> input) {
        // TODO: Implement DAG workflow execution
        // 1. Build adjacency list from edges
        // 2. Topological sort to determine execution order
        // 3. Execute nodes in order, passing context between them
        // 4. Handle conditional branches (skip nodes when predicate is false)
        // 5. Return final node output with full execution trace

        List<String> executionOrder = topologicalSort();

        Map<String, Object> result = new LinkedHashMap<>();
        result.put("agent", getName());
        result.put("type", getAgentType());
        result.put("nodes_count", nodes.size());
        result.put("edges_count", edges.size());
        result.put("execution_order", executionOrder);
        result.put("status", "stub");
        result.put("message", "WorkflowAgent.run() not yet implemented");
        return result;
    }

    /**
     * Compute topological ordering of workflow nodes.
     * Returns node IDs in dependency-respecting order.
     */
    private List<String> topologicalSort() {
        // Kahn's algorithm
        Map<String, Integer> inDegree = new LinkedHashMap<>();
        Map<String, List<String>> adjacency = new LinkedHashMap<>();

        for (WorkflowNode node : nodes) {
            inDegree.putIfAbsent(node.id(), 0);
            adjacency.putIfAbsent(node.id(), new ArrayList<>());
        }
        for (WorkflowEdge edge : edges) {
            adjacency.computeIfAbsent(edge.from(), k -> new ArrayList<>()).add(edge.to());
            inDegree.merge(edge.to(), 1, Integer::sum);
        }

        List<String> queue = new ArrayList<>();
        for (var entry : inDegree.entrySet()) {
            if (entry.getValue() == 0) {
                queue.add(entry.getKey());
            }
        }

        List<String> sorted = new ArrayList<>();
        while (!queue.isEmpty()) {
            String current = queue.remove(0);
            sorted.add(current);
            for (String neighbor : adjacency.getOrDefault(current, List.of())) {
                int newDegree = inDegree.merge(neighbor, -1, Integer::sum);
                if (newDegree == 0) {
                    queue.add(neighbor);
                }
            }
        }
        return sorted;
    }

    public List<WorkflowNode> getNodes() {
        return nodes;
    }

    public List<WorkflowEdge> getEdges() {
        return edges;
    }

    /**
     * A node in the workflow DAG.
     */
    public record WorkflowNode(String id, String type, String agentRef,
                                Map<String, Object> config) {
    }

    /**
     * A directed edge between two workflow nodes.
     */
    public record WorkflowEdge(String from, String to, String condition) {
    }
}
