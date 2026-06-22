package io.superagent.harness;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * Context window overflow detection and message compaction.
 *
 * <p>When the conversation history exceeds the model's context window limit,
 * older messages are summarized to reduce token count while preserving key context.</p>
 *
 * <p>Uses a simple heuristic: estimate ~4 chars per token, then trim the oldest
 * messages when overflow is detected.</p>
 */
@Component
public class CompactionManager {

    private static final Logger log = LoggerFactory.getLogger(CompactionManager.class);

    /** Default maximum context tokens. */
    private static final int DEFAULT_MAX_TOKENS = 128_000;

    /** Estimated characters per token (conservative). */
    private static final double CHARS_PER_TOKEN = 4.0;

    /** Ratio of messages to keep when compacting. */
    private static final double KEEP_RATIO = 0.5;

    private final int maxTokens;

    public CompactionManager() {
        this(DEFAULT_MAX_TOKENS);
    }

    public CompactionManager(int maxTokens) {
        this.maxTokens = maxTokens;
    }

    /**
     * Check if the messages exceed the context window limit.
     *
     * @param messages list of message maps
     * @return true if compaction is needed
     */
    public boolean needsCompaction(List<Map<String, Object>> messages) {
        return needsCompaction(messages, maxTokens);
    }

    /**
     * Check if the messages exceed a specific token limit.
     *
     * @param messages  list of message maps
     * @param maxTokens maximum allowed tokens
     * @return true if estimated tokens exceed the limit
     */
    public boolean needsCompaction(List<Map<String, Object>> messages, int maxTokens) {
        int estimatedTokens = estimateTokens(messages);
        boolean needed = estimatedTokens > maxTokens;
        if (needed) {
            log.info("Compaction needed: estimated {} tokens > {} limit", estimatedTokens, maxTokens);
        }
        return needed;
    }

    /**
     * Compact messages by summarizing older entries.
     *
     * <p>Keeps the most recent messages and replaces older ones with a summary placeholder.</p>
     *
     * @param messages list of message maps
     * @return compacted list of messages
     */
    public List<Map<String, Object>> compact(List<Map<String, Object>> messages) {
        return compact(messages, maxTokens);
    }

    /**
     * Compact messages with a specific token limit.
     *
     * @param messages  list of message maps
     * @param maxTokens target token count
     * @return compacted list of messages
     */
    public List<Map<String, Object>> compact(List<Map<String, Object>> messages, int maxTokens) {
        if (messages.isEmpty() || !needsCompaction(messages, maxTokens)) {
            return messages;
        }

        int keepCount = Math.max(1, (int) (messages.size() * KEEP_RATIO));
        int removeCount = messages.size() - keepCount;

        // Keep the first message (system) and the last keepCount messages
        List<Map<String, Object>> compacted = new ArrayList<>();

        // Always keep the first message if it exists
        if (!messages.isEmpty()) {
            compacted.add(messages.get(0));
        }

        // Add a summary placeholder for removed messages
        Map<String, Object> summary = Map.of(
                "role", "system",
                "content", "[Compacted " + removeCount + " older messages to fit context window]",
                "compacted", true,
                "removed_count", removeCount
        );
        compacted.add(summary);

        // Keep the most recent messages
        int startIdx = Math.max(1, messages.size() - keepCount);
        for (int i = startIdx; i < messages.size(); i++) {
            compacted.add(messages.get(i));
        }

        log.info("Compacted {} messages to {} (removed {})",
                messages.size(), compacted.size(), removeCount);

        return compacted;
    }

    /**
     * Estimate the token count for a list of messages.
     */
    public int estimateTokens(List<Map<String, Object>> messages) {
        int totalChars = 0;
        for (Map<String, Object> msg : messages) {
            Object content = msg.get("content");
            if (content != null) {
                totalChars += content.toString().length();
            }
            // Account for role and metadata overhead
            totalChars += 20;
        }
        return (int) (totalChars / CHARS_PER_TOKEN);
    }

    /**
     * Get the configured maximum token limit.
     */
    public int getMaxTokens() {
        return maxTokens;
    }
}
