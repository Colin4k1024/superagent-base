package io.superagent.config;

import java.util.List;
import java.util.Map;

/**
 * Record representing a parsed agent YAML definition.
 *
 * <p>Maps 1:1 to the {@code apiVersion: superagent/v1, kind: Agent} YAML schema
 * used by the Go and Python bases.</p>
 */
public record AgentDefinition(
    String apiVersion,
    String kind,
    Metadata metadata,
    Spec spec
) {
    /**
     * Agent metadata (name, version, labels).
     */
    public record Metadata(
        String name,
        String version,
        Map<String, String> labels
    ) {
    }

    /**
     * Agent specification (type, model, tools, etc.).
     */
    public record Spec(
        String type,
        ModelConfig model,
        String systemPrompt,
        List<ToolRef> tools,
        MemoryConfig memory,
        InterruptConfig interrupt,
        List<SubAgentRef> subAgents,
        WorkflowConfig workflow,
        Map<String, Object> evolution,
        Map<String, Object> observability
    ) {
    }

    /**
     * Model configuration reference.
     */
    public record ModelConfig(
        String primary,
        String fallback,
        String router
    ) {
    }

    /**
     * Tool reference (URI-based: builtin/, mcp://, skill://).
     */
    public record ToolRef(String ref) {
    }

    /**
     * Memory backend configuration.
     */
    public record MemoryConfig(
        String backend,
        Map<String, Object> options
    ) {
    }

    /**
     * Interrupt/resume configuration.
     */
    public record InterruptConfig(
        boolean enabled,
        String checkpointBackend,
        int timeoutSeconds
    ) {
    }

    /**
     * Sub-agent reference for orchestration agents.
     */
    public record SubAgentRef(String ref) {
    }

    /**
     * Workflow DAG configuration.
     */
    public record WorkflowConfig(
        List<WorkflowNode> nodes,
        List<WorkflowEdge> edges
    ) {
    }

    /**
     * Workflow node definition.
     */
    public record WorkflowNode(
        String id,
        String type,
        String agentRef,
        Map<String, Object> config
    ) {
    }

    /**
     * Workflow edge definition.
     */
    public record WorkflowEdge(
        String from,
        String to,
        String condition
    ) {
    }
}
