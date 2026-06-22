package io.superagent.server;

import org.springframework.http.MediaType;
import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Flux;
import reactor.core.publisher.Mono;

import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Workflow execution endpoints.
 *
 * <p>Provides synchronous, streaming, resume, and chat-mode workflow
 * execution, plus workflow metadata retrieval.</p>
 */
@RestController
@RequestMapping("/api/v2/workflows")
public class WorkflowController {

    /**
     * Run a workflow synchronously.
     */
    @PostMapping("/run")
    public Mono<Map<String, Object>> run(@RequestBody Map<String, Object> request) {
        String workflowId = (String) request.getOrDefault("workflow_id", "");
        return Mono.just(ApiResponse.ok(Map.of(
            "workflow_id", workflowId,
            "status", "completed",
            "output", Map.of()
        )).toMap());
    }

    /**
     * Run a workflow with SSE streaming.
     */
    @PostMapping(value = "/stream_run", produces = MediaType.TEXT_EVENT_STREAM_VALUE)
    public Flux<Map<String, Object>> streamRun(@RequestBody Map<String, Object> request) {
        String workflowId = (String) request.getOrDefault("workflow_id", "");
        return Flux.just(
            Map.<String, Object>of("event", "start", "workflow_id", workflowId),
            Map.<String, Object>of("event", "done", "workflow_id", workflowId)
        );
    }

    /**
     * Resume a streaming workflow.
     */
    @PostMapping(value = "/stream_resume", produces = MediaType.TEXT_EVENT_STREAM_VALUE)
    public Flux<Map<String, Object>> streamResume(@RequestBody Map<String, Object> request) {
        return Flux.just(
            Map.<String, Object>of("event", "resumed"),
            Map.<String, Object>of("event", "done")
        );
    }

    /**
     * Chat-mode workflow execution.
     */
    @PostMapping("/chat")
    public Mono<Map<String, Object>> chat(@RequestBody Map<String, Object> request) {
        return Mono.just(ApiResponse.ok(Map.of(
            "workflow_id", request.getOrDefault("workflow_id", ""),
            "status", "completed",
            "response", ""
        )).toMap());
    }

    /**
     * Get workflow metadata.
     */
    @GetMapping("/{workflowId}")
    public Mono<Map<String, Object>> getWorkflow(@PathVariable String workflowId) {
        Map<String, Object> workflow = new LinkedHashMap<>();
        workflow.put("id", workflowId);
        workflow.put("name", workflowId);
        workflow.put("status", "active");
        workflow.put("nodes", java.util.List.of());
        workflow.put("edges", java.util.List.of());
        return Mono.just(ApiResponse.ok(workflow).toMap());
    }
}
