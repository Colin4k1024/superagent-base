package io.superagent.config;

import io.superagent.agents.AgentLoopAgent;
import io.superagent.agents.BaseAgent;
import io.superagent.agents.ChatModelAgent;
import io.superagent.agents.ParallelAgent;
import io.superagent.agents.SequentialAgent;
import io.superagent.agents.SupervisorAgent;
import io.superagent.agents.WorkflowAgent;
import io.superagent.models.ModelRegistry;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Factory that builds {@link BaseAgent} instances from {@link AgentDefinition} records.
 *
 * <p>Handles two-pass build: leaf agents first, then orchestration agents
 * that reference sub-agents by name.</p>
 */
@Component
public class AgentBuilderFactory {

    private static final Logger log = LoggerFactory.getLogger(AgentBuilderFactory.class);

    private final ModelRegistry modelRegistry;
    private final ConcurrentHashMap<String, BaseAgent> builtAgents = new ConcurrentHashMap<>();

    public AgentBuilderFactory(ModelRegistry modelRegistry) {
        this.modelRegistry = modelRegistry;
    }

    /**
     * Build an agent from its definition.
     *
     * @param def parsed agent definition
     * @return built agent instance
     */
    public BaseAgent build(AgentDefinition def) {
        String type = def.spec().type();
        String name = def.metadata().name();

        return switch (type) {
            case "chat_model_agent", "deep_agent" -> buildChatModelAgent(def);
            case "supervisor" -> buildSupervisorAgent(def);
            case "sequential" -> buildSequentialAgent(def);
            case "parallel" -> buildParallelAgent(def);
            case "workflow" -> buildWorkflowAgent(def);
            case "agentloop" -> buildAgentLoopAgent(def);
            default -> {
                log.warn("Unknown agent type '{}', defaulting to ChatModelAgent", type);
                yield buildChatModelAgent(def);
            }
        };
    }

    private ChatModelAgent buildChatModelAgent(AgentDefinition def) {
        String modelName = def.spec().model() != null ? def.spec().model().primary() : "default";
        List<String> tools = def.spec().tools().stream()
            .map(AgentDefinition.ToolRef::ref)
            .toList();

        ChatModelAgent agent = new ChatModelAgent(
            def.metadata().name(),
            def.spec().systemPrompt(),
            modelName,
            tools,
            10,  // maxSteps — TODO: from config
            def.spec().systemPrompt()
        );
        builtAgents.put(agent.getName(), agent);
        return agent;
    }

    private SupervisorAgent buildSupervisorAgent(AgentDefinition def) {
        List<BaseAgent> children = resolveSubAgents(def.spec().subAgents());
        String routingModel = def.spec().model() != null ? def.spec().model().primary() : "default";

        SupervisorAgent agent = new SupervisorAgent(
            def.metadata().name(),
            def.spec().systemPrompt(),
            children,
            routingModel
        );
        builtAgents.put(agent.getName(), agent);
        return agent;
    }

    private SequentialAgent buildSequentialAgent(AgentDefinition def) {
        List<BaseAgent> steps = resolveSubAgents(def.spec().subAgents());
        SequentialAgent agent = new SequentialAgent(
            def.metadata().name(),
            def.spec().systemPrompt(),
            steps
        );
        builtAgents.put(agent.getName(), agent);
        return agent;
    }

    private ParallelAgent buildParallelAgent(AgentDefinition def) {
        List<BaseAgent> children = resolveSubAgents(def.spec().subAgents());
        ParallelAgent agent = new ParallelAgent(
            def.metadata().name(),
            def.spec().systemPrompt(),
            children
        );
        builtAgents.put(agent.getName(), agent);
        return agent;
    }

    private WorkflowAgent buildWorkflowAgent(AgentDefinition def) {
        List<WorkflowAgent.WorkflowNode> nodes = List.of();  // TODO: parse from def
        List<WorkflowAgent.WorkflowEdge> edges = List.of();  // TODO: parse from def
        WorkflowAgent agent = new WorkflowAgent(
            def.metadata().name(),
            def.spec().systemPrompt(),
            nodes,
            edges
        );
        builtAgents.put(agent.getName(), agent);
        return agent;
    }

    private AgentLoopAgent buildAgentLoopAgent(AgentDefinition def) {
        // For agentloop, the first sub-agent is the main agent
        List<BaseAgent> children = resolveSubAgents(def.spec().subAgents());
        BaseAgent mainAgent = children.isEmpty()
            ? buildChatModelAgent(def)
            : children.get(0);

        AgentLoopAgent agent = new AgentLoopAgent(
            def.metadata().name(),
            def.spec().systemPrompt(),
            mainAgent
        );
        builtAgents.put(agent.getName(), agent);
        return agent;
    }

    /**
     * Resolve sub-agent references to built agent instances.
     */
    private List<BaseAgent> resolveSubAgents(List<AgentDefinition.SubAgentRef> refs) {
        List<BaseAgent> agents = new ArrayList<>();
        for (AgentDefinition.SubAgentRef ref : refs) {
            BaseAgent agent = builtAgents.get(ref.ref());
            if (agent != null) {
                agents.add(agent);
            } else {
                log.warn("Sub-agent '{}' not found in built agents", ref.ref());
            }
        }
        return agents;
    }

    /**
     * Return all built agents.
     */
    public Map<String, BaseAgent> getBuiltAgents() {
        return Map.copyOf(builtAgents);
    }
}
