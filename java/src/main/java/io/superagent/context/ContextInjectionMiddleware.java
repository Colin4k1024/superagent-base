package io.superagent.context;

import io.superagent.agents.BaseAgent;
import io.agentscope.core.agent.Event;
import io.agentscope.core.agent.RuntimeContext;
import reactor.core.publisher.Flux;

import java.time.Instant;
import java.time.ZoneId;
import java.time.format.DateTimeFormatter;
import java.util.*;

/**
 * Middleware that injects contextual information into agent messages.
 *
 * <p>Prepends system messages with timestamp, session metadata, and static
 * context before the agent processes user input.</p>
 *
 * <p>Maps to Go {@code contextInjectionMiddleware} in mw_context_injection.go.</p>
 */
public class ContextInjectionMiddleware {

    private static final DateTimeFormatter TIMESTAMP_FMT =
        DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ss'Z'").withZone(ZoneId.of("UTC"));

    private final boolean injectTimestamp;
    private final boolean injectSessionMetadata;
    private final String staticContext;

    /**
     * Create context injection middleware with all options.
     *
     * @param injectTimestamp      whether to inject current timestamp
     * @param injectSessionMetadata whether to inject session metadata
     * @param staticContext         static context string to inject (nullable)
     */
    public ContextInjectionMiddleware(boolean injectTimestamp,
                                       boolean injectSessionMetadata,
                                       String staticContext) {
        this.injectTimestamp = injectTimestamp;
        this.injectSessionMetadata = injectSessionMetadata;
        this.staticContext = staticContext;
    }

    /**
     * Create middleware with timestamp injection only.
     */
    public static ContextInjectionMiddleware withTimestamp() {
        return new ContextInjectionMiddleware(true, false, null);
    }

    /**
     * Create middleware with session metadata injection only.
     */
    public static ContextInjectionMiddleware withSessionMetadata() {
        return new ContextInjectionMiddleware(false, true, null);
    }

    /**
     * Create middleware with static context injection only.
     */
    public static ContextInjectionMiddleware withStaticContext(String context) {
        return new ContextInjectionMiddleware(false, false, context);
    }

    /**
     * Create middleware with all injections enabled.
     */
    public static ContextInjectionMiddleware all(String staticContext) {
        return new ContextInjectionMiddleware(true, true, staticContext);
    }

    /**
     * Build context messages to prepend.
     *
     * @param sessionId session identifier
     * @param metadata  optional session metadata
     * @return list of context strings to inject as system messages
     */
    public List<String> buildContextMessages(String sessionId, Map<String, Object> metadata) {
        List<String> messages = new ArrayList<>();

        if (injectTimestamp) {
            messages.add("[Context] Current time: " + TIMESTAMP_FMT.format(Instant.now()));
        }

        if (injectSessionMetadata) {
            messages.add("[Context] Session: " + sessionId);
            if (metadata != null && !metadata.isEmpty()) {
                StringBuilder metaStr = new StringBuilder("[Context] Metadata: ");
                metadata.forEach((k, v) -> metaStr.append(k).append("=").append(v).append(", "));
                // Remove trailing comma
                if (metaStr.length() > 2) {
                    metaStr.setLength(metaStr.length() - 2);
                }
                messages.add(metaStr.toString());
            }
        }

        if (staticContext != null && !staticContext.isBlank()) {
            messages.add("[Context] " + staticContext);
        }

        return messages;
    }

    /**
     * Inject context messages into a message list (prepend as system messages).
     *
     * @param messages  original message list (modified in-place)
     * @param sessionId session identifier
     * @param metadata  optional session metadata
     */
    public void inject(List<String> messages, String sessionId, Map<String, Object> metadata) {
        List<String> contextMsgs = buildContextMessages(sessionId, metadata);
        if (!contextMsgs.isEmpty()) {
            messages.addAll(0, contextMsgs);
        }
    }

    /**
     * Inject context into a map-based input (for agent.run()).
     *
     * @param input     input map (modified in-place)
     * @param sessionId session identifier
     */
    public void injectIntoInput(Map<String, Object> input, String sessionId) {
        List<String> contextMsgs = buildContextMessages(sessionId, null);
        if (!contextMsgs.isEmpty()) {
            String originalMessage = input.getOrDefault("message", "").toString();
            StringBuilder enriched = new StringBuilder();
            for (String ctx : contextMsgs) {
                enriched.append(ctx).append("\n");
            }
            enriched.append(originalMessage);
            input.put("message", enriched.toString());
        }
    }

    /**
     * Create a new agent that wraps the given agent with context injection.
     *
     * @param agent     agent to wrap
     * @param sessionId session identifier
     * @return wrapped agent
     */
    public BaseAgent wrap(BaseAgent agent, String sessionId) {
        return new ContextInjectedAgent(agent, this, sessionId);
    }

    public boolean isInjectTimestamp() {
        return injectTimestamp;
    }

    public boolean isInjectSessionMetadata() {
        return injectSessionMetadata;
    }

    public String getStaticContext() {
        return staticContext;
    }

    /**
     * Wrapper agent that injects context before delegating to the wrapped agent.
     */
    private static class ContextInjectedAgent extends BaseAgent {
        private final BaseAgent delegate;
        private final ContextInjectionMiddleware middleware;
        private final String sessionId;

        ContextInjectedAgent(BaseAgent delegate, ContextInjectionMiddleware middleware, String sessionId) {
            super(delegate.getName(), delegate.getDescription(), delegate.getAgentType());
            this.delegate = delegate;
            this.middleware = middleware;
            this.sessionId = sessionId;
        }

        @Override
        public Map<String, Object> run(Map<String, Object> input) {
            Map<String, Object> enriched = new LinkedHashMap<>(input);
            middleware.injectIntoInput(enriched, sessionId);
            return delegate.run(enriched);
        }

        @Override
        public Flux<Event> callStream(Map<String, Object> input, RuntimeContext context) {
            Map<String, Object> enriched = new LinkedHashMap<>(input);
            middleware.injectIntoInput(enriched, sessionId);
            return delegate.callStream(enriched, context);
        }

        @Override
        public List<String> getTools() {
            return delegate.getTools();
        }
    }
}
