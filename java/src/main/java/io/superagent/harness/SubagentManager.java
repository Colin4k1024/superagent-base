package io.superagent.harness;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.nio.file.DirectoryStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.*;
import java.util.concurrent.*;

/**
 * Sub-agent declaration loading and orchestration.
 *
 * <p>Loads sub-agent definitions from {@code workspace/subagents/<name>.md}
 * and provides synchronous and asynchronous spawning capabilities.</p>
 */
@Component
public class SubagentManager {

    private static final Logger log = LoggerFactory.getLogger(SubagentManager.class);

    private final Path subagentsDir;
    private final Map<String, String> declarations;
    private final ExecutorService executor;

    public SubagentManager(WorkspaceConfig config) {
        this(config.getWorkspaceRoot().resolve("subagents"));
    }

    public SubagentManager(Path subagentsDir) {
        this.subagentsDir = subagentsDir;
        this.declarations = new LinkedHashMap<>();
        this.executor = Executors.newCachedThreadPool(r -> {
            Thread t = new Thread(r, "subagent-worker");
            t.setDaemon(true);
            return t;
        });
    }

    /**
     * Load all sub-agent declarations from the workspace.
     */
    public void loadDeclarations() {
        declarations.clear();
        if (!Files.isDirectory(subagentsDir)) {
            log.info("No subagents directory found at {}", subagentsDir);
            return;
        }
        try (DirectoryStream<Path> stream = Files.newDirectoryStream(subagentsDir, "*.md")) {
            for (Path file : stream) {
                String name = file.getFileName().toString().replace(".md", "");
                String content = Files.readString(file);
                declarations.put(name, content);
                log.info("Loaded sub-agent declaration: {}", name);
            }
        } catch (IOException e) {
            log.error("Failed to load sub-agent declarations: {}", e.getMessage());
        }
    }

    /**
     * List all available sub-agent names.
     */
    public List<String> listSubagents() {
        return List.copyOf(declarations.keySet());
    }

    /**
     * Get the declaration content for a sub-agent.
     */
    public String getDeclaration(String name) {
        return declarations.get(name);
    }

    /**
     * Spawn a sub-agent synchronously (blocking).
     *
     * @param name sub-agent name
     * @param task task description
     * @return execution result
     */
    public SubagentResult spawn(String name, String task) {
        String declaration = declarations.get(name);
        if (declaration == null) {
            return new SubagentResult(name, task, null, false, "Sub-agent not found: " + name);
        }

        log.info("Spawning sub-agent '{}' with task: {}", name, task);
        // In a full implementation, this would create a new agent instance
        // and execute the task. For now, return a structured result.
        String result = String.format("Sub-agent '%s' executed task: %s", name, task);
        return new SubagentResult(name, task, result, true, null);
    }

    /**
     * Spawn a sub-agent asynchronously with notification on completion.
     *
     * @param name sub-agent name
     * @param task task description
     * @return future that completes with the result
     */
    public CompletableFuture<SubagentResult> spawnBackground(String name, String task) {
        return CompletableFuture.supplyAsync(() -> spawn(name, task), executor);
    }

    /**
     * Shutdown the executor service.
     */
    public void shutdown() {
        executor.shutdown();
        try {
            if (!executor.awaitTermination(5, TimeUnit.SECONDS)) {
                executor.shutdownNow();
            }
        } catch (InterruptedException e) {
            executor.shutdownNow();
            Thread.currentThread().interrupt();
        }
    }

    /**
     * Result of a sub-agent execution.
     */
    public record SubagentResult(
            String name,
            String task,
            String result,
            boolean success,
            String error
    ) {
    }
}
