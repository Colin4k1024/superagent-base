package io.superagent.harness;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.LocalDate;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.List;

/**
 * Two-layer memory system: short-term (conversation) + long-term (MEMORY.md).
 *
 * <h3>Storage Layout:</h3>
 * <pre>
 * workspace/
 *   MEMORY.md               — consolidated long-term memory
 *   memory/
 *     2025-06-21.md          — daily memory file
 *     2025-06-22.md
 * </pre>
 *
 * <p>Short-term memory is the current conversation (managed by SessionManager).
 * Long-term memory persists across sessions in MEMORY.md.</p>
 */
@Component
public class HarnessMemory {

    private static final Logger log = LoggerFactory.getLogger(HarnessMemory.class);
    private static final DateTimeFormatter DATE_FMT = DateTimeFormatter.ofPattern("yyyy-MM-dd");

    private final Path workspacePath;
    private final Workspace workspace;

    public HarnessMemory(WorkspaceConfig config, Workspace workspace) {
        this(config.getWorkspaceRoot(), workspace);
    }

    public HarnessMemory(Path workspacePath, Workspace workspace) {
        this.workspacePath = workspacePath;
        this.workspace = workspace;
    }

    /**
     * Get the long-term memory content from MEMORY.md.
     */
    public String getLongTermMemory() {
        return workspace.getMemoryMd();
    }

    /**
     * Add content to today's daily memory file.
     *
     * @param content memory content to append
     */
    public void addDailyMemory(String content) {
        addDailyMemory(LocalDate.now(), content);
    }

    /**
     * Add content to a specific date's daily memory file.
     *
     * @param date    the date
     * @param content memory content to append
     */
    public void addDailyMemory(LocalDate date, String content) {
        Path memoryDir = workspacePath.resolve("memory");
        Path dailyFile = memoryDir.resolve(date.format(DATE_FMT) + ".md");
        try {
            Files.createDirectories(memoryDir);
            String entry = "\n\n## " + date.format(DATE_FMT) + "\n\n" + content + "\n";
            Files.writeString(dailyFile, entry,
                    java.nio.file.StandardOpenOption.CREATE,
                    java.nio.file.StandardOpenOption.APPEND);
            log.info("Daily memory added for {}", date);
        } catch (IOException e) {
            log.error("Failed to write daily memory for {}: {}", date, e.getMessage());
        }
    }

    /**
     * Consolidate daily memory files into MEMORY.md.
     *
     * <p>Reads all daily files from memory/ and merges them into MEMORY.md.
     * Daily files are preserved (not deleted).</p>
     */
    public void consolidate() {
        Path memoryDir = workspacePath.resolve("memory");
        if (!Files.isDirectory(memoryDir)) {
            log.info("No memory directory found, skipping consolidation");
            return;
        }

        List<String> consolidated = new ArrayList<>();

        // Read existing MEMORY.md
        String existing = workspace.getMemoryMd();
        if (existing != null && !existing.isBlank()) {
            consolidated.add(existing);
        }

        // Read daily files in chronological order
        try (var stream = Files.list(memoryDir)) {
            stream.filter(p -> p.toString().endsWith(".md"))
                    .sorted()
                    .forEach(p -> {
                        try {
                            String content = Files.readString(p);
                            if (!content.isBlank()) {
                                consolidated.add(content.trim());
                            }
                        } catch (IOException e) {
                            log.warn("Failed to read daily memory {}: {}", p, e.getMessage());
                        }
                    });
        } catch (IOException e) {
            log.error("Failed to scan memory directory: {}", e.getMessage());
        }

        if (!consolidated.isEmpty()) {
            String merged = String.join("\n\n---\n\n", consolidated);
            workspace.updateMemoryMd(merged);
            log.info("Memory consolidated from {} sources", consolidated.size());
        }
    }

    /**
     * Build memory content formatted for injection into the system prompt.
     */
    public String buildMemoryPrompt() {
        String longTerm = getLongTermMemory();
        if (longTerm == null || longTerm.isBlank()) {
            return "";
        }
        return "# Long-Term Memory\n\n" + longTerm;
    }

    /**
     * Get the list of daily memory file dates.
     */
    public List<LocalDate> listDailyMemoryDates() {
        List<LocalDate> dates = new ArrayList<>();
        Path memoryDir = workspacePath.resolve("memory");
        if (!Files.isDirectory(memoryDir)) {
            return dates;
        }
        try (var stream = Files.list(memoryDir)) {
            stream.filter(p -> p.toString().endsWith(".md"))
                    .forEach(p -> {
                        String name = p.getFileName().toString().replace(".md", "");
                        try {
                            dates.add(LocalDate.parse(name, DATE_FMT));
                        } catch (Exception ignored) {
                            // Not a date-formatted file
                        }
                    });
        } catch (IOException e) {
            log.error("Failed to list daily memory dates: {}", e.getMessage());
        }
        return dates;
    }
}
