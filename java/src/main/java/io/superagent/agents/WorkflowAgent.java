package io.superagent.agents;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

public class WorkflowAgent extends BaseAgent {

    private static final Logger log = LoggerFactory.getLogger(WorkflowAgent.class);

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
        List<String> executionOrder = topologicalSort();
        Map<String, Map<String, Object>> nodeOutputs = new LinkedHashMap<>();
        List<Map<String, Object>> executionTrace = new ArrayList<>();
        Map<String, Object> context = new LinkedHashMap<>(input);

        for (String nodeId : executionOrder) {
            WorkflowNode node = findNode(nodeId);
            if (node == null) continue;

            if (!evaluatePredecessors(nodeId, nodeOutputs)) {
                log.debug("Skipping node '{}' — predecessor condition not met", nodeId);
                executionTrace.add(Map.of(
                    "node_id", nodeId, "status", "skipped", "reason", "predecessor_condition_false"
                ));
                continue;
            }

            Map<String, Object> nodeInput = buildNodeInput(nodeId, context, nodeOutputs);
            Map<String, Object> nodeResult = executeNode(node, nodeInput);
            nodeOutputs.put(nodeId, nodeResult);

            executionTrace.add(Map.of(
                "node_id", nodeId,
                "type", node.type(),
                "status", nodeResult.getOrDefault("status", "completed")
            ));

            context.put("node_" + nodeId, nodeResult);
        }

        String lastContent = "";
        if (!executionOrder.isEmpty()) {
            Map<String, Object> lastOutput = nodeOutputs.get(executionOrder.get(executionOrder.size() - 1));
            if (lastOutput != null) {
                lastContent = lastOutput.getOrDefault("content", "").toString();
            }
        }

        Map<String, Object> result = new LinkedHashMap<>();
        result.put("agent", getName());
        result.put("type", getAgentType());
        result.put("nodes_count", nodes.size());
        result.put("edges_count", edges.size());
        result.put("execution_order", executionOrder);
        result.put("execution_trace", executionTrace);
        result.put("content", lastContent);
        result.put("status", "completed");
        return result;
    }

    private Map<String, Object> executeNode(WorkflowNode node, Map<String, Object> input) {
        Map<String, Object> result = new LinkedHashMap<>();
        result.put("node_id", node.id());
        result.put("type", node.type());
        result.put("status", "completed");

        switch (node.type()) {
            case "llm_call" -> {
                result.put("content", input.getOrDefault("message", "").toString());
                result.put("model", node.config() != null ? node.config().getOrDefault("model", "default") : "default");
            }
            case "tool_call" -> {
                result.put("content", "Tool executed: " + node.agentRef());
                result.put("tool", node.agentRef());
            }
            case "agent_call" -> {
                result.put("content", "Agent called: " + node.agentRef());
                result.put("agent_ref", node.agentRef());
            }
            case "code" -> {
                result.put("content", "Code block executed");
                result.put("language", node.config() != null ? node.config().getOrDefault("language", "python") : "python");
            }
            case "condition" -> {
                boolean conditionResult = evaluateCondition(node, input);
                result.put("content", String.valueOf(conditionResult));
                result.put("condition_result", conditionResult);
            }
            default -> result.put("content", "Unknown node type: " + node.type());
        }
        return result;
    }

    private boolean evaluateCondition(WorkflowNode node, Map<String, Object> input) {
        if (node.config() == null) return true;
        String expression = (String) node.config().get("expression");
        if (expression == null || expression.isEmpty()) return true;

        Object value = input.get(expression);
        if (value instanceof Boolean b) return b;
        if (value instanceof String s) return !s.isEmpty() && !"false".equalsIgnoreCase(s);
        return value != null;
    }

    private boolean evaluatePredecessors(String nodeId, Map<String, Map<String, Object>> nodeOutputs) {
        for (WorkflowEdge edge : edges) {
            if (!edge.to().equals(nodeId)) continue;
            if (edge.condition() == null || edge.condition().isEmpty()) continue;

            Map<String, Object> predOutput = nodeOutputs.get(edge.from());
            if (predOutput == null) return false;

            Object condResult = predOutput.get("condition_result");
            if (condResult instanceof Boolean b && !b) return false;
        }
        return true;
    }

    private Map<String, Object> buildNodeInput(String nodeId, Map<String, Object> context,
                                                 Map<String, Map<String, Object>> nodeOutputs) {
        Map<String, Object> nodeInput = new LinkedHashMap<>(context);
        for (WorkflowEdge edge : edges) {
            if (edge.to().equals(nodeId)) {
                Map<String, Object> predOutput = nodeOutputs.get(edge.from());
                if (predOutput != null) {
                    String predContent = predOutput.getOrDefault("content", "").toString();
                    nodeInput.put("message", predContent);
                    nodeInput.put("predecessor_" + edge.from(), predOutput);
                }
            }
        }
        return nodeInput;
    }

    private WorkflowNode findNode(String nodeId) {
        for (WorkflowNode node : nodes) {
            if (node.id().equals(nodeId)) return node;
        }
        return null;
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
