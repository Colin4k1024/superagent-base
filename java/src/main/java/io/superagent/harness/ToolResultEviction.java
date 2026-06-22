package io.superagent.harness;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Instant;
import java.util.Map;

/**
 * Large tool result handling with file offload.
 *
 * <p>When a tool result exceeds the size threshold (default 80K chars),
 * the full result is written to a file and a placeholder is injected
 * into the context instead.</p>
 */
@Component
public class ToolResultEviction {

    private static final Logger log = LoggerFactory.getLogger(ToolResultEviction.class);

    /** Default eviction threshold in characters. */
    private static final int DEFAULT_THRESHOLD = 80_000;

    private final int threshold;
    private final Path evictionDir;

    public ToolResultEviction(WorkspaceConfig config) {
        this(config.getWorkspaceRoot().resolve("evictions"), DEFAULT_THRESHOLD);
    }

    public ToolResultEviction(Path evictionDir, int threshold) {
        this.evictionDir = evictionDir;
        this.threshold = threshold;
    }

    /**
     * Check if a tool result should be evicted.
     *
     * @param result tool result content
     * @return true if the result exceeds the threshold
     */
    public boolean shouldEvict(String result) {
        return result != null && result.length() > threshold;
    }

    /**
     * Evict a large tool result to a file.
     *
     * <p>Writes the full result to disk and returns a placeholder
     * that references the file path.</p>
     *
     * @param result the large tool result
     * @return eviction result with placeholder and file path
     */
    public EvictionResult evict(String result) {
        if (!shouldEvict(result)) {
            return new EvictionResult(result, null, false);
        }

        try {
            Files.createDirectories(evictionDir);
            String filename = "evicted_" + Instant.now().toEpochMilli() + ".txt";
            Path filePath = evictionDir.resolve(filename);
            Files.writeString(filePath, result);

            String placeholder = "[Tool result evicted — " + result.length()
                    + " chars written to " + filePath.getFileName()
                    + ". Read the file to see full content.]";

            log.info("Evicted tool result ({} chars) to {}", result.length(), filePath);
            return new EvictionResult(placeholder, filePath.toString(), true);
        } catch (IOException e) {
            log.error("Failed to evict tool result: {}", e.getMessage());
            // Fall back to truncation
            String truncated = result.substring(0, Math.min(result.length(), threshold)) + "\n... [truncated]";
            return new EvictionResult(truncated, null, true);
        }
    }

    /**
     * Read an evicted result back from its file path.
     *
     * @param filePath path to the evicted file
     * @return the full result content, or null if not found
     */
    public String readEvicted(String filePath) {
        try {
            return Files.readString(Path.of(filePath));
        } catch (IOException e) {
            log.error("Failed to read evicted file {}: {}", filePath, e.getMessage());
            return null;
        }
    }

    /**
     * Get the eviction threshold.
     */
    public int getThreshold() {
        return threshold;
    }

    /**
     * Result of an eviction operation.
     *
     * @param content   the placeholder (or original content if not evicted)
     * @param filePath  path to the evicted file, or null if not evicted
     * @param evicted   whether eviction occurred
     */
    public record EvictionResult(String content, String filePath, boolean evicted) {
    }
}
