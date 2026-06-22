package io.superagent.agents;

import io.agentscope.core.agent.Event;
import io.agentscope.core.agent.EventType;
import io.agentscope.core.agent.RuntimeContext;
import io.agentscope.core.message.Msg;
import io.agentscope.core.message.MsgRole;
import reactor.core.publisher.Flux;

import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * Sequential pipeline agent — executes children in order.
 *
 * <p>Each child agent receives the output of the previous one as input.
 * The final agent's output is returned as the pipeline result.</p>
 *
 * <p>Maps to Go {@code SequentialAgent} and Python {@code SequentialAgent}.</p>
 */
public class SequentialAgent extends BaseAgent {

    private final List<BaseAgent> steps;

    public SequentialAgent(String name, String description, List<BaseAgent> steps) {
        super(name, description, "sequential");
        this.steps = steps != null ? List.copyOf(steps) : List.of();
    }

    @Override
    public Map<String, Object> run(Map<String, Object> input) {
        if (steps.isEmpty()) {
            Map<String, Object> result = new LinkedHashMap<>();
            result.put("agent", getName());
            result.put("type", getAgentType());
            result.put("steps_count", 0);
            result.put("status", "empty");
            result.put("pipeline", List.of());
            return result;
        }

        // Execute pipeline
        Map<String, Object> currentInput = new LinkedHashMap<>(input);
        List<Map<String, Object>> stepResults = new ArrayList<>();
        List<String> stepNames = new ArrayList<>();

        for (BaseAgent step : steps) {
            stepNames.add(step.getName());
            Map<String, Object> stepResult = step.run(currentInput);
            stepResults.add(stepResult);
            // Pass output as next input
            currentInput = new LinkedHashMap<>(stepResult);
        }

        Map<String, Object> result = new LinkedHashMap<>();
        result.put("agent", getName());
        result.put("type", getAgentType());
        result.put("steps_count", steps.size());
        result.put("pipeline", stepNames);
        result.put("status", "completed");

        // Merge final step output
        if (!stepResults.isEmpty()) {
            Map<String, Object> finalOutput = stepResults.get(stepResults.size() - 1);
            result.put("content", finalOutput.getOrDefault("content", ""));
            result.put("final_step", steps.get(steps.size() - 1).getName());
        }

        return result;
    }

    @Override
    public Flux<Event> callStream(Map<String, Object> input, RuntimeContext context) {
        return Flux.create(sink -> {
            Map<String, Object> currentInput = new LinkedHashMap<>(input);

            for (int i = 0; i < steps.size(); i++) {
                BaseAgent step = steps.get(i);

                Map<String, Object> stepResult = step.run(currentInput);
                currentInput = new LinkedHashMap<>(stepResult);

                String content = stepResult.getOrDefault("content", "").toString();
                if (!content.isEmpty()) {
                    Msg msg = Msg.builder()
                        .role(MsgRole.ASSISTANT)
                        .textContent(content)
                        .build();
                    sink.next(new Event(EventType.HINT, msg, false));
                }
            }

            String finalContent = currentInput.getOrDefault("content", "").toString();
            Msg finalMsg = Msg.builder()
                .role(MsgRole.ASSISTANT)
                .textContent(finalContent)
                .build();
            sink.next(new Event(EventType.AGENT_RESULT, finalMsg, true));
            sink.complete();
        });
    }

    public List<BaseAgent> getSteps() {
        return steps;
    }
}
