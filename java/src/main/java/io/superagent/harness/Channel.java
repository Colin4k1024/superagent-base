package io.superagent.harness;

import io.agentscope.core.agent.Event;
import io.agentscope.core.message.Msg;
import io.superagent.agents.BaseAgent;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;
import reactor.core.publisher.Flux;
import reactor.core.publisher.Sinks;

import java.util.LinkedHashMap;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.Semaphore;

/**
 * Session routing with per-session concurrency control and SSE event streaming.
 *
 * <p>Provides a unified entry point for agent execution with:</p>
 * <ul>
 *   <li>Per-session concurrency control (one request per session at a time)</li>
 *   <li>Multi-agent routing</li>
 *   <li>SSE event streaming via Reactor Sinks</li>
 * </ul>
 */
@Component
public class Channel {

    private static final Logger log = LoggerFactory.getLogger(Channel.class);

    private final ConcurrentHashMap<String, Semaphore> sessionLocks = new ConcurrentHashMap<>();
    private final ConcurrentHashMap<String, Sinks.Many<Event>> sessionSinks = new ConcurrentHashMap<>();

    public Channel() {
    }

    /**
     * Handle a chat request through the channel.
     *
     * <p>Acquires a per-session lock, delegates to the agent, and streams
     * events back through a Reactor Sink.</p>
     *
     * @param agent   the agent to execute
     * @param request the request parameters
     * @return SSE event stream
     */
    public Flux<Map<String, Object>> handle(BaseAgent agent, Map<String, Object> request) {
        String sessionId = request.getOrDefault("session_id", "default").toString();

        // Acquire per-session lock
        Semaphore lock = sessionLocks.computeIfAbsent(sessionId, k -> new Semaphore(1));
        if (!lock.tryAcquire()) {
            return Flux.just(
                    Map.<String, Object>of("event", "error", "data", "Session " + sessionId + " is busy"),
                    Map.<String, Object>of("event", "done", "data", "")
            );
        }

        try {
            // Create or get the session sink
            Sinks.Many<Event> sink = sessionSinks.computeIfAbsent(sessionId,
                    k -> Sinks.many().multicast().onBackpressureBuffer());

            // Execute the agent and stream events, mapping Event to Map
            return agent.callStream(request, io.agentscope.core.agent.RuntimeContext.builder()
                            .sessionId(sessionId)
                            .build())
                    .map(this::eventToMap)
                    .doFinally(signal -> {
                        lock.release();
                        log.debug("Session {} released", sessionId);
                    });
        } catch (Exception e) {
            lock.release();
            log.error("Channel error for session {}: {}", sessionId, e.getMessage());
            return Flux.just(
                    Map.<String, Object>of("event", "error", "data", e.getMessage()),
                    Map.<String, Object>of("event", "done", "data", "")
            );
        }
    }

    /**
     * Check if a session is currently busy.
     */
    public boolean isSessionBusy(String sessionId) {
        Semaphore lock = sessionLocks.get(sessionId);
        return lock != null && !lock.tryAcquire();
    }

    /**
     * Get the active session count.
     */
    public int getActiveSessionCount() {
        return (int) sessionLocks.values().stream()
                .filter(s -> s.availablePermits() == 0)
                .count();
    }

    /**
     * Clean up a session's resources.
     */
    public void cleanupSession(String sessionId) {
        sessionLocks.remove(sessionId);
        Sinks.Many<Event> sink = sessionSinks.remove(sessionId);
        if (sink != null) {
            sink.tryEmitComplete();
        }
        log.debug("Session {} cleaned up", sessionId);
    }

    private Map<String, Object> eventToMap(Event event) {
        Map<String, Object> map = new LinkedHashMap<>();
        map.put("event", event.getType().name().toLowerCase());
        map.put("last", event.isLast());
        Msg msg = event.getMessage();
        if (msg != null) {
            map.put("data", msg.getTextContent());
        }
        return map;
    }
}
