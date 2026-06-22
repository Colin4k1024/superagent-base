package io.superagent.agents;

import io.agentscope.core.ReActAgent;
import io.agentscope.core.agent.Event;
import io.agentscope.core.agent.EventType;
import io.agentscope.core.agent.RuntimeContext;
import io.agentscope.core.memory.InMemoryMemory;
import io.agentscope.core.message.Msg;
import io.agentscope.core.message.MsgRole;
import io.agentscope.core.model.DashScopeChatModel;
import io.agentscope.core.model.Model;
import io.agentscope.core.model.OpenAIChatModel;
import io.agentscope.core.tool.Toolkit;
import reactor.core.publisher.Flux;

import java.util.List;
import java.util.Map;

/**
 * ReAct (Reasoning + Acting) agent backed by a chat model.
 *
 * <p>Wraps AgentScope's {@link ReActAgent} to provide tool-calling LLM interactions.
 * This is the primary agent type for conversational, tool-using agents.</p>
 *
 * <p>Maps to Go {@code ChatModelAgent} and Python {@code ChatModelAgent}.</p>
 */
public class ChatModelAgent extends BaseAgent {

    private final String modelName;
    private final List<String> tools;
    private final int maxSteps;
    private final String systemPrompt;
    private final ReActAgent reactAgent;

    public ChatModelAgent(String name, String description, String modelName,
                          List<String> tools, int maxSteps, String systemPrompt) {
        super(name, description, "chat_model_agent");
        this.modelName = modelName;
        this.tools = tools != null ? List.copyOf(tools) : List.of();
        this.maxSteps = maxSteps;
        this.systemPrompt = systemPrompt;

        // Build the internal ReActAgent with resolved model
        Model model = resolveModel(modelName);
        Toolkit toolkit = buildToolkit(this.tools);

        this.reactAgent = ReActAgent.builder()
            .name(name)
            .sysPrompt(systemPrompt != null ? systemPrompt : "")
            .model(model)
            .toolkit(toolkit)
            .maxIters(maxSteps)
            .build();
    }

    @Override
    public Map<String, Object> run(Map<String, Object> input) {
        RuntimeContext context = RuntimeContext.builder()
            .sessionId(input.getOrDefault("session_id", "default").toString())
            .build();

        String userMessage = input.getOrDefault("message", "").toString();
        Msg userMsg = Msg.builder()
            .role(MsgRole.USER)
            .textContent(userMessage)
            .build();

        // Call the ReActAgent (blocking)
        Msg result = reactAgent.call(List.of(userMsg), context).block();

        String content = result != null ? result.getTextContent() : "";

        // Merge agent metadata into result
        return new java.util.LinkedHashMap<>(Map.of(
            "agent", getName(),
            "type", getAgentType(),
            "model", modelName,
            "content", content
        ));
    }

    @Override
    public Flux<Event> callStream(Map<String, Object> input, RuntimeContext context) {
        String userMessage = input.getOrDefault("message", "").toString();
        Msg userMsg = Msg.builder()
            .role(MsgRole.USER)
            .textContent(userMessage)
            .build();

        return reactAgent.stream(
            List.of(userMsg),
            io.agentscope.core.agent.StreamOptions.defaults(),
            context
        );
    }

    @Override
    public List<String> getTools() {
        return tools;
    }

    public String getModelName() {
        return modelName;
    }

    public int getMaxSteps() {
        return maxSteps;
    }

    public String getSystemPrompt() {
        return systemPrompt;
    }

    /**
     * Resolve a model from "provider:model" or plain model name.
     */
    private static Model resolveModel(String modelStr) {
        if (modelStr == null || modelStr.isEmpty() || "default".equals(modelStr)) {
            return OpenAIChatModel.builder().modelName("gpt-4o").build();
        }
        String[] parts = modelStr.split(":", 2);
        String provider = parts[0];
        String modelName = parts.length > 1 ? parts[1] : parts[0];

        return switch (provider) {
            case "dashscope" -> DashScopeChatModel.builder().modelName(modelName).build();
            case "openai" -> OpenAIChatModel.builder().modelName(modelName).build();
            default -> OpenAIChatModel.builder().modelName(modelStr).build();
        };
    }

    /**
     * Build a Toolkit from tool URI references.
     * Currently returns an empty toolkit (tools will be wired when the tool registry is connected).
     */
    private static Toolkit buildToolkit(List<String> toolRefs) {
        Toolkit toolkit = new Toolkit();
        // Tool resolution from URI refs will be implemented when
        // the tool registry is connected.
        return toolkit;
    }
}
