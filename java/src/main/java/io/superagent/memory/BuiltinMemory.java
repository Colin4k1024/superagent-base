package io.superagent.memory;

import io.agentscope.core.memory.InMemoryMemory;
import io.agentscope.core.memory.Memory;
import io.agentscope.core.message.Msg;
import io.agentscope.core.message.MsgRole;
import org.springframework.stereotype.Component;

import java.util.List;
import java.util.Map;

/**
 * In-memory memory store using AgentScope's {@link InMemoryMemory}.
 *
 * <p>Wraps the AgentScope memory implementation to provide the legacy
 * {@link MemoryStore} interface. Suitable for development and single-instance
 * deployments. Not persistent across restarts.</p>
 *
 * <p>The entire {@code io.agentscope.core.memory} package is deprecated for
 * removal in AgentScope 2.0.0 with no replacement yet. This suppresses the
 * removal warning until a successor API is available.</p>
 */
@SuppressWarnings("removal") // entire io.agentscope.core.memory package deprecated for removal in 2.0.0 with no replacement
@Component
public class BuiltinMemory implements MemoryStore {

    private final Memory delegate;

    public BuiltinMemory() {
        this.delegate = new InMemoryMemory();
    }

    @Override
    public void store(String sessionId, String role, String content,
                      Map<String, Object> metadata) {
        MsgRole msgRole = switch (role) {
            case "system" -> MsgRole.SYSTEM;
            case "assistant" -> MsgRole.ASSISTANT;
            case "tool" -> MsgRole.TOOL;
            default -> MsgRole.USER;
        };
        Msg msg = Msg.builder()
            .role(msgRole)
            .textContent(content)
            .build();
        delegate.addMessage(msg);
    }

    @Override
    public List<MemoryMessage> retrieve(String sessionId, int limit) {
        List<Msg> messages = delegate.getMessages();
        int from = Math.max(0, messages.size() - limit);
        return messages.subList(from, messages.size()).stream()
            .map(m -> new MemoryMessage(
                java.util.UUID.randomUUID().toString(),
                sessionId,
                m.getRole().name().toLowerCase(),
                m.getTextContent(),
                Map.of(),
                System.currentTimeMillis()
            ))
            .toList();
    }

    @Override
    public List<MemoryMessage> search(String sessionId, String query, int limit) {
        String lowerQuery = query.toLowerCase();
        return delegate.getMessages().stream()
            .filter(m -> m.getTextContent() != null
                && m.getTextContent().toLowerCase().contains(lowerQuery))
            .limit(limit)
            .map(m -> new MemoryMessage(
                java.util.UUID.randomUUID().toString(),
                sessionId,
                m.getRole().name().toLowerCase(),
                m.getTextContent(),
                Map.of(),
                System.currentTimeMillis()
            ))
            .toList();
    }

    @Override
    public void clear(String sessionId) {
        delegate.clear();
    }

    /**
     * Get the underlying AgentScope memory instance.
     */
    public Memory getAgentScopeMemory() {
        return delegate;
    }
}
