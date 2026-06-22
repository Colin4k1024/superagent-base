package io.superagent.agents;

import io.agentscope.core.agent.Event;
import io.agentscope.core.agent.EventType;
import io.agentscope.core.agent.RuntimeContext;
import io.agentscope.core.message.Msg;
import io.agentscope.core.message.MsgRole;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import reactor.core.publisher.Flux;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * Agent that loops by calling a child agent repeatedly until a done marker is detected
 * or max turns are reached.
 *
 * <p>Each turn: call child → check for done marker → accumulate context → repeat.
 * The child agent receives the full conversation history as context.</p>
 *
 * <p>Maps to Go {@code AgentLoopAgent} in agentloop.go.</p>
 */
public class AgentLoopAgent extends BaseAgent {

    private static final Logger log = LoggerFactory.getLogger(AgentLoopAgent.class);

    /** Default done marker that signals loop completion. */
    public static final String DEFAULT_DONE_MARKER = "[DONE]";

    /** Default maximum number of turns. */
    public static final int DEFAULT_MAX_TURNS = 25;

    private final BaseAgent mainAgent;
    private final int maxTurns;
    private final String doneMarker;

    /**
     * Create an agent loop with default settings.
     *
     * @param name      agent name
     * @param description agent description
     * @param mainAgent the child agent to call in each turn
     */
    public AgentLoopAgent(String name, String description, BaseAgent mainAgent) {
        this(name, description, mainAgent, DEFAULT_MAX_TURNS, DEFAULT_DONE_MARKER);
    }

    /**
     * Create an agent loop with custom settings.
     *
     * @param name       agent name
     * @param description agent description
     * @param mainAgent  the child agent to call in each turn
     * @param maxTurns   maximum number of loop turns
     * @param doneMarker string that signals loop completion
     */
    public AgentLoopAgent(String name, String description, BaseAgent mainAgent,
                          int maxTurns, String doneMarker) {
        super(name, description, "agentloop");
        this.mainAgent = mainAgent;
        this.maxTurns = maxTurns > 0 ? maxTurns : DEFAULT_MAX_TURNS;
        this.doneMarker = doneMarker != null ? doneMarker : DEFAULT_DONE_MARKER;
    }

    @Override
    public Map<String, Object> run(Map<String, Object> input) {
        List<String> accumulatedMessages = new ArrayList<>();
        String userMessage = input.getOrDefault("message", "").toString();
        accumulatedMessages.add(userMessage);

        for (int turn = 0; turn < maxTurns; turn++) {
            log.debug("AgentLoop '{}' turn {}/{}", getName(), turn + 1, maxTurns);

            // Build context with accumulated history
            Map<String, Object> turnInput = new LinkedHashMap<>(input);
            turnInput.put("message", String.join("\n", accumulatedMessages));
            turnInput.put("turn", turn);
            turnInput.put("max_turns", maxTurns);

            try {
                Map<String, Object> result = mainAgent.run(turnInput);
                String content = result.getOrDefault("content", "").toString();

                // Check for done marker
                if (content.contains(doneMarker)) {
                    log.info("AgentLoop '{}' completed at turn {} (done marker detected)",
                        getName(), turn + 1);
                    // Strip the done marker from final content
                    String cleanContent = content.replace(doneMarker, "").trim();
                    return buildResult(cleanContent, turn + 1, true);
                }

                accumulatedMessages.add(content);

                // Check if the result explicitly signals completion
                if (Boolean.TRUE.equals(result.get("done"))) {
                    log.info("AgentLoop '{}' completed at turn {} (done flag)", getName(), turn + 1);
                    return buildResult(content, turn + 1, true);
                }
            } catch (Exception e) {
                log.error("AgentLoop '{}' failed at turn {}: {}", getName(), turn + 1, e.getMessage());
                return buildResult("Error at turn " + (turn + 1) + ": " + e.getMessage(), turn + 1, false);
            }
        }

        log.warn("AgentLoop '{}' reached max turns ({})", getName(), maxTurns);
        String lastMessage = accumulatedMessages.isEmpty() ? "" :
            accumulatedMessages.get(accumulatedMessages.size() - 1);
        return buildResult(lastMessage, maxTurns, false);
    }

    @Override
    public Flux<Event> callStream(Map<String, Object> input, RuntimeContext context) {
        return Flux.create(sink -> {
            List<String> accumulatedMessages = new ArrayList<>();
            String userMessage = input.getOrDefault("message", "").toString();
            accumulatedMessages.add(userMessage);

            for (int turn = 0; turn < maxTurns; turn++) {
                Map<String, Object> turnInput = new LinkedHashMap<>(input);
                turnInput.put("message", String.join("\n", accumulatedMessages));
                turnInput.put("turn", turn);

                // Emit progress event
                Msg progressMsg = Msg.builder()
                    .role(MsgRole.ASSISTANT)
                    .textContent("Turn " + (turn + 1) + "/" + maxTurns)
                    .build();
                sink.next(new Event(EventType.HINT, progressMsg, false));

                try {
                    Map<String, Object> result = mainAgent.run(turnInput);
                    String content = result.getOrDefault("content", "").toString();

                    if (content.contains(doneMarker)) {
                        String cleanContent = content.replace(doneMarker, "").trim();
                        Msg doneMsg = Msg.builder()
                            .role(MsgRole.ASSISTANT)
                            .textContent(cleanContent)
                            .build();
                        sink.next(new Event(EventType.AGENT_RESULT, doneMsg, true));
                        sink.complete();
                        return;
                    }

                    accumulatedMessages.add(content);
                } catch (Exception e) {
                    Msg errMsg = Msg.builder()
                        .role(MsgRole.ASSISTANT)
                        .textContent("Error at turn " + (turn + 1) + ": " + e.getMessage())
                        .build();
                    sink.next(new Event(EventType.AGENT_RESULT, errMsg, true));
                    sink.complete();
                    return;
                }
            }

            // Max turns reached
            String lastMsg = accumulatedMessages.isEmpty() ? "" :
                accumulatedMessages.get(accumulatedMessages.size() - 1);
            Msg finalMsg = Msg.builder()
                .role(MsgRole.ASSISTANT)
                .textContent(lastMsg)
                .build();
            sink.next(new Event(EventType.AGENT_RESULT, finalMsg, true));
            sink.complete();
        });
    }

    @Override
    public List<String> getTools() {
        return mainAgent.getTools();
    }

    public BaseAgent getMainAgent() {
        return mainAgent;
    }

    public int getMaxTurns() {
        return maxTurns;
    }

    public String getDoneMarker() {
        return doneMarker;
    }

    private Map<String, Object> buildResult(String content, int turnsUsed, boolean completed) {
        Map<String, Object> result = new LinkedHashMap<>();
        result.put("agent", getName());
        result.put("type", getAgentType());
        result.put("content", content);
        result.put("turns_used", turnsUsed);
        result.put("max_turns", maxTurns);
        result.put("completed", completed);
        result.put("done_marker", doneMarker);
        return result;
    }
}
