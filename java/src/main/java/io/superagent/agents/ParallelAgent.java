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
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * Parallel fan-out agent — executes all children concurrently.
 *
 * <p>All child agents receive the same input and run in parallel via
 * {@link CompletableFuture}. Results are collected and merged.</p>
 *
 * <p>Maps to Go {@code ParallelAgent} and Python {@code ParallelAgent}.</p>
 */
public class ParallelAgent extends BaseAgent {

    private final List<BaseAgent> children;
    private final ExecutorService executor;

    public ParallelAgent(String name, String description, List<BaseAgent> children) {
        super(name, description, "parallel");
        this.children = children != null ? List.copyOf(children) : List.of();
        this.executor = Executors.newFixedThreadPool(
            Math.max(1, Math.min(Runtime.getRuntime().availableProcessors(), this.children.size())));
    }

    @Override
    public Map<String, Object> run(Map<String, Object> input) {
        if (children.isEmpty()) {
            Map<String, Object> result = new LinkedHashMap<>();
            result.put("agent", getName());
            result.put("type", getAgentType());
            result.put("children_count", 0);
            result.put("status", "empty");
            return result;
        }

        // Fan out to all children
        List<CompletableFuture<Map<String, Object>>> futures = new ArrayList<>();
        for (BaseAgent child : children) {
            futures.add(CompletableFuture.supplyAsync(() -> child.run(input), executor));
        }

        // Wait for all and collect results
        CompletableFuture.allOf(futures.toArray(new CompletableFuture[0])).join();

        List<Map<String, Object>> childResults = new ArrayList<>();
        List<String> childNames = new ArrayList<>();
        for (int i = 0; i < futures.size(); i++) {
            try {
                childResults.add(futures.get(i).join());
                childNames.add(children.get(i).getName());
            } catch (Exception e) {
                childResults.add(Map.of("error", e.getMessage()));
                childNames.add(children.get(i).getName());
            }
        }

        // Merge results
        Map<String, Object> result = new LinkedHashMap<>();
        result.put("agent", getName());
        result.put("type", getAgentType());
        result.put("children_count", children.size());
        result.put("children_results", childNames);
        result.put("status", "completed");

        // Combine content from all children
        StringBuilder combined = new StringBuilder();
        for (Map<String, Object> cr : childResults) {
            String content = cr.getOrDefault("content", "").toString();
            if (!content.isEmpty()) {
                if (combined.length() > 0) combined.append("\n\n");
                combined.append(content);
            }
        }
        result.put("content", combined.toString());

        return result;
    }

    @Override
    public Flux<Event> callStream(Map<String, Object> input, RuntimeContext context) {
        return Flux.create(sink -> {
            // Execute all children in parallel
            List<CompletableFuture<Map<String, Object>>> futures = new ArrayList<>();
            for (BaseAgent child : children) {
                futures.add(CompletableFuture.supplyAsync(() -> child.run(input), executor));
            }

            // Wait and emit
            CompletableFuture.allOf(futures.toArray(new CompletableFuture[0])).join();

            StringBuilder combined = new StringBuilder();
            for (int i = 0; i < children.size(); i++) {
                try {
                    Map<String, Object> childResult = futures.get(i).join();
                    String content = childResult.getOrDefault("content", "").toString();
                    if (!content.isEmpty()) {
                        if (combined.length() > 0) combined.append("\n\n");
                        combined.append(content);
                    }
                } catch (Exception e) {
                    Msg errMsg = Msg.builder()
                        .role(MsgRole.ASSISTANT)
                        .textContent("Child agent error: " + e.getMessage())
                        .build();
                    sink.next(new Event(EventType.HINT, errMsg, false));
                }
            }

            Msg finalMsg = Msg.builder()
                .role(MsgRole.ASSISTANT)
                .textContent(combined.toString())
                .build();
            sink.next(new Event(EventType.AGENT_RESULT, finalMsg, true));
            sink.complete();
        });
    }

    public List<BaseAgent> getChildren() {
        return children;
    }

    public void shutdown() {
        executor.shutdown();
    }
}
