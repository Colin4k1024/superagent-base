package io.superagent.agents;

import io.agentscope.core.agent.Event;
import io.agentscope.core.agent.EventType;
import io.agentscope.core.agent.RuntimeContext;
import io.agentscope.core.message.Msg;
import io.agentscope.core.message.MsgRole;
import reactor.core.publisher.Flux;

import java.util.List;
import java.util.Map;

/**
 * Abstract base for all Superagent agents.
 *
 * <p>Every agent type extends this class and implements {@link #run(Map)}
 * for execution and {@link #describe()} for introspection.</p>
 *
 * <p>Maps to the Go {@code Agent} interface and Python {@code BaseAgent}.</p>
 *
 * <h3>AgentScope 2.0 Integration</h3>
 * <p>This class bridges the existing Superagent API with AgentScope's
 * {@code ReActAgent} model. The {@link #callStream} method
 * provides SSE-compatible streaming via {@code Flux<Event>}.</p>
 */
public abstract class BaseAgent {

    private final String name;
    private final String description;
    private final String agentType;
    private final AgentState state = new AgentState();

    protected BaseAgent(String name, String description, String agentType) {
        this.name = name;
        this.description = description;
        this.agentType = agentType;
    }

    /**
     * Execute the agent with the given input.
     *
     * @param input key-value input context
     * @return agent output as a map
     */
    public abstract Map<String, Object> run(Map<String, Object> input);

    /**
     * Execute the agent with streaming events.
     *
     * <p>Default implementation wraps {@link #run} into a single AGENT_RESULT event.
     * Subclasses that support streaming (ChatModelAgent, SupervisorAgent) override
     * this with real SSE event emission.</p>
     *
     * @param input   key-value input context
     * @param context runtime context (session, user)
     * @return stream of typed events
     */
    public Flux<Event> callStream(Map<String, Object> input, RuntimeContext context) {
        Map<String, Object> result = run(input);
        String content = result.getOrDefault("content", result.getOrDefault("message", "")).toString();
        Msg msg = Msg.builder()
            .role(MsgRole.ASSISTANT)
            .textContent(content)
            .build();
        return Flux.just(new Event(EventType.AGENT_RESULT, msg, true));
    }

    /**
     * Return a human-readable description of this agent's capabilities.
     */
    public String describe() {
        return String.format("[%s] %s: %s", agentType, name, description);
    }

    /**
     * Return the list of tool names this agent can use.
     * Subclasses that support tools should override this.
     */
    public List<String> getTools() {
        return List.of();
    }

    /**
     * Get the agent's mutable state for interrupt/resume support.
     */
    public AgentState getState() {
        return state;
    }

    /**
     * Merge new state into the agent's current state.
     */
    public void setState(AgentState newState) {
        for (String key : newState.keys()) {
            state.put(key, newState.get(key));
        }
    }

    public String getName() {
        return name;
    }

    public String getDescription() {
        return description;
    }

    public String getAgentType() {
        return agentType;
    }
}
