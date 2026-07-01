package io.superagent.server;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import io.agentscope.core.agent.Event;
import io.agentscope.core.agent.EventType;
import io.agentscope.core.agent.RuntimeContext;
import io.agentscope.core.message.Msg;
import io.superagent.agents.BaseAgent;
import io.superagent.config.AgentBuilderFactory;
import io.superagent.config.YamlAgentLoader;
import io.superagent.mcp.MCPRegistry;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.MediaType;
import org.springframework.http.codec.ServerSentEvent;
import org.springframework.web.bind.annotation.*;
import reactor.core.publisher.Flux;
import reactor.core.publisher.Mono;

import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * REST + SSE endpoints for chat interactions.
 *
 * <p>Provides both synchronous and streaming (SSE) chat APIs,
 * compatible with the Go/Python base API contract.</p>
 *
 * <h3>AgentScope 2.0 Integration</h3>
 * <p>Streaming endpoint delegates to {@link BaseAgent#callStream} which
 * returns {@code Flux<Event>} for typed SSE events.</p>
 */
@RestController
@RequestMapping("/api/v2")
public class ChatController {

    private static final Logger log = LoggerFactory.getLogger(ChatController.class);

    private final AgentBuilderFactory agentFactory;
    private final YamlAgentLoader agentLoader;
    private final MCPRegistry mcpRegistry;
    private final ObjectMapper objectMapper = new ObjectMapper();

    public ChatController(AgentBuilderFactory agentFactory, YamlAgentLoader agentLoader,
                          MCPRegistry mcpRegistry) {
        this.agentFactory = agentFactory;
        this.agentLoader = agentLoader;
        this.mcpRegistry = mcpRegistry;
    }

    /**
     * Synchronous chat endpoint.
     *
     * @param request chat request body
     * @return agent response
     */
    @PostMapping("/chat")
    public Mono<Map<String, Object>> chat(@RequestBody Map<String, Object> request) {
        String agentId = (String) request.getOrDefault("agent_id", "default");
        String message = (String) request.getOrDefault("message", "");
        String sessionId = (String) request.getOrDefault("session_id", "default");

        BaseAgent agent = resolveAgent(agentId);
        if (agent == null) {
            return Mono.just(Map.of(
                "agent_id", agentId,
                "session_id", sessionId,
                "status", "error",
                "message", "Agent not found: " + agentId
            ));
        }

        try {
            Map<String, Object> result = agent.run(request);
            Map<String, Object> response = new LinkedHashMap<>(result);
            response.put("agent_id", agentId);
            response.put("session_id", sessionId);
            return Mono.just(response);
        } catch (Exception e) {
            log.error("Chat execution failed for agent {}: {}", agentId, e.getMessage());
            return Mono.just(Map.of(
                "agent_id", agentId,
                "session_id", sessionId,
                "status", "error",
                "message", e.getMessage()
            ));
        }
    }

    /**
     * Streaming chat endpoint (SSE) — A2UI protocol.
     *
     * <p>Each event is: {@code data: {"type":"<t>","data":"<content>"}\n\n}
     * matching the Go/Python A2UI contract expected by the frontend.</p>
     */
    @PostMapping(value = "/chat/stream", produces = MediaType.TEXT_EVENT_STREAM_VALUE)
    public Flux<ServerSentEvent<String>> chatStream(@RequestBody Map<String, Object> request) {
        String agentId = (String) request.getOrDefault("agent_id", "default");
        String sessionId = (String) request.getOrDefault("session_id", "default");

        BaseAgent agent = resolveAgent(agentId);
        if (agent == null) {
            return Flux.just(
                sseEvent("error", "Agent not found: " + agentId),
                sseEvent("done", "")
            );
        }

        RuntimeContext context = RuntimeContext.builder()
            .sessionId(sessionId)
            .build();

        return agent.callStream(request, context)
            .map(this::eventToSse)
            .onErrorResume(e -> {
                log.error("Stream error for agent {}: {}", agentId, e.getMessage());
                return Flux.just(
                    sseEvent("error", e.getMessage()),
                    sseEvent("done", "")
                );
            });
    }

    /**
     * Resume an interrupted conversation.
     */
    @PostMapping("/chat/resume")
    public Mono<Map<String, Object>> resume(@RequestBody Map<String, Object> request) {
        return Mono.just(Map.of(
            "status", "stub",
            "message", "ChatController.resume() not yet implemented"
        ));
    }

    /**
     * Abort a running chat session.
     */
    @PostMapping("/chat/abort")
    public Mono<Map<String, Object>> abort(@RequestBody Map<String, Object> request) {
        String sessionId = (String) request.getOrDefault("session_id", "");
        return Mono.just(Map.of(
            "code", 0,
            "msg", "ok",
            "data", Map.of("session_id", sessionId, "aborted", true)
        ));
    }

    /**
     * Get interrupt state for a session.
     */
    @GetMapping("/chat/interrupt_state")
    public Mono<Map<String, Object>> interruptState(
            @RequestParam("session_id") String sessionId) {
        return Mono.just(Map.of(
            "session_id", sessionId,
            "interrupted", false,
            "status", "stub"
        ));
    }

    /**
     * List all loaded agents.
     */
    @GetMapping("/agents")
    public Mono<Map<String, Object>> listAgents() {
        Map<String, BaseAgent> agents = agentFactory.getBuiltAgents();
        List<Map<String, Object>> agentList = agents.values().stream()
            .map(a -> Map.<String, Object>of(
                "name", a.getName(),
                "type", a.getAgentType(),
                "description", a.getDescription(),
                "tools", a.getTools()
            ))
            .toList();
        return Mono.just(Map.of(
            "agents", agentList,
            "count", agentList.size()
        ));
    }

    /**
     * List connected MCP servers.
     */
    @GetMapping("/mcp/servers")
    public Mono<Map<String, Object>> listMcpServers() {
        List<String> servers = mcpRegistry.listServers();
        List<Map<String, Object>> serverList = servers.stream()
            .map(name -> {
                var client = mcpRegistry.getClient(name);
                Map<String, Object> info = new LinkedHashMap<>();
                info.put("name", name);
                info.put("connected", true);
                client.ifPresent(c -> {
                    info.put("endpoint", c.getEndpoint());
                    if (c.getServerInfo() != null) {
                        info.put("server_version", c.getServerInfo().version());
                    }
                    if (c.getCachedTools() != null) {
                        info.put("tools_count", c.getCachedTools().size());
                    }
                });
                return info;
            })
            .toList();
        return Mono.just(Map.of(
            "servers", serverList,
            "count", serverList.size()
        ));
    }

    /**
     * Resolve an agent by ID, loading definitions if needed.
     */
    private BaseAgent resolveAgent(String agentId) {
        // First check built agents
        BaseAgent agent = agentFactory.getBuiltAgents().get(agentId);
        if (agent != null) return agent;

        // Try loading from YAML
        try {
            var definitions = agentLoader.loadAll();
            for (var def : definitions) {
                agentFactory.build(def);
            }
            return agentFactory.getBuiltAgents().get(agentId);
        } catch (Exception e) {
            log.warn("Failed to load agent definitions: {}", e.getMessage());
            return null;
        }
    }

    /**
     * Convert an AgentScope Event to an A2UI ServerSentEvent.
     * Output format: data: {"type":"<t>","data":"<content>"}\n\n
     */
    private ServerSentEvent<String> eventToSse(Event event) {
        String eventType = switch (event.getType()) {
            case REASONING -> "thinking";
            case TOOL_RESULT -> "tool_result";
            case HINT -> "progress";
            case AGENT_RESULT -> "done";
            case SUMMARY -> "text";
            case ALL -> "text";
        };

        Msg msg = event.getMessage();
        String content = (msg != null) ? msg.getTextContent() : "";
        return sseEvent(eventType, content);
    }

    /** Build a properly-formatted A2UI SSE data frame. */
    private ServerSentEvent<String> sseEvent(String type, String data) {
        try {
            String json = objectMapper.writeValueAsString(
                Map.of("type", type, "data", data == null ? "" : data)
            );
            return ServerSentEvent.<String>builder().data(json).build();
        } catch (JsonProcessingException e) {
            return ServerSentEvent.<String>builder()
                .data("{\"type\":\"error\",\"data\":\"json serialization failed\"}")
                .build();
        }
    }
}
