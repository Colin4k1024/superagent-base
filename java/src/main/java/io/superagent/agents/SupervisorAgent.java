package io.superagent.agents;

import io.agentscope.core.ReActAgent;
import io.agentscope.core.agent.Event;
import io.agentscope.core.agent.EventType;
import io.agentscope.core.agent.RuntimeContext;
import io.agentscope.core.message.Msg;
import io.agentscope.core.message.MsgRole;
import io.agentscope.core.model.DashScopeChatModel;
import io.agentscope.core.model.Model;
import io.agentscope.core.model.OpenAIChatModel;
import reactor.core.publisher.Flux;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * Supervisor agent that routes tasks to child agents.
 *
 * <p>The supervisor uses an LLM to decide which child agent should handle
 * each incoming request, then delegates execution to that agent.</p>
 *
 * <p>Maps to Go {@code SupervisorAgent} and Python {@code SupervisorAgent}.</p>
 */
public class SupervisorAgent extends BaseAgent {

    private final List<BaseAgent> children;
    private final String routingModel;
    private final ReActAgent router;

    public SupervisorAgent(String name, String description,
                           List<BaseAgent> children, String routingModel) {
        super(name, description, "supervisor");
        this.children = children != null ? List.copyOf(children) : List.of();
        this.routingModel = routingModel;

        // Build routing agent with child descriptions as system prompt
        String childDescriptions = this.children.stream()
            .map(c -> "- " + c.getName() + ": " + c.getDescription())
            .reduce("", (a, b) -> a + "\n" + b);

        this.router = ReActAgent.builder()
            .name(name + "-router")
            .sysPrompt("You are a task router. Given a user request, decide which child agent should handle it.\n"
                + "Available agents:" + childDescriptions + "\n"
                + "Respond with JSON: {\"agent\": \"agent-name\", \"reason\": \"...\"}")
            .model(resolveModel(routingModel))
            .maxIters(3)
            .build();
    }

    @Override
    public Map<String, Object> run(Map<String, Object> input) {
        if (children.isEmpty()) {
            return Map.of(
                "agent", getName(),
                "type", getAgentType(),
                "routing_model", routingModel,
                "children_count", 0,
                "status", "no_children",
                "message", "No child agents configured"
            );
        }

        // Use router to select child agent
        RuntimeContext ctx = RuntimeContext.empty();
        String userMessage = input.getOrDefault("message", "").toString();
        Msg userMsg = Msg.builder()
            .role(MsgRole.USER)
            .textContent(userMessage)
            .build();

        Msg routingResult = router.call(List.of(userMsg), ctx).block();
        String routingResponse = routingResult != null ? routingResult.getTextContent() : "";

        // Parse routing decision (simplified — real impl would parse JSON)
        BaseAgent selectedChild = selectChild(routingResponse);

        // Delegate to selected child
        Map<String, Object> childResult = selectedChild.run(input);
        Map<String, Object> result = new LinkedHashMap<>(childResult);
        result.put("supervisor", getName());
        result.put("delegated_to", selectedChild.getName());
        result.put("routing_reason", routingResponse);
        return result;
    }

    @Override
    public Flux<Event> callStream(Map<String, Object> input, RuntimeContext context) {
        if (children.isEmpty()) {
            Msg msg = Msg.builder()
                .role(MsgRole.ASSISTANT)
                .textContent("No child agents configured")
                .build();
            return Flux.just(new Event(EventType.AGENT_RESULT, msg, true));
        }

        // Select child (simplified routing)
        BaseAgent selectedChild = children.get(0);

        // Stream from child
        return selectedChild.callStream(input, context);
    }

    @Override
    public List<String> getTools() {
        List<String> allTools = new ArrayList<>();
        for (BaseAgent child : children) {
            allTools.addAll(child.getTools());
        }
        return allTools;
    }

    public List<BaseAgent> getChildren() {
        return children;
    }

    public String getRoutingModel() {
        return routingModel;
    }

    /**
     * Select the best child agent based on routing response text.
     */
    private BaseAgent selectChild(String routingResponse) {
        String lower = routingResponse.toLowerCase();
        for (BaseAgent child : children) {
            if (lower.contains(child.getName().toLowerCase())) {
                return child;
            }
        }
        // Default to first child
        return children.get(0);
    }

    private static Model resolveModel(String modelStr) {
        if (modelStr == null || modelStr.isEmpty() || "default".equals(modelStr)) {
            return OpenAIChatModel.builder().modelName("gpt-4o").build();
        }
        String[] parts = modelStr.split(":", 2);
        String provider = parts[0];
        String modelName = parts.length > 1 ? parts[1] : parts[0];
        return switch (provider) {
            case "dashscope" -> DashScopeChatModel.builder().modelName(modelName).build();
            default -> OpenAIChatModel.builder().modelName(modelName).build();
        };
    }
}
