package io.superagent.harness;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.nio.file.DirectoryStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.*;

/**
 * Workspace manager for the AgentScope harness.
 *
 * <p>Loads workspace files (AGENTS.md, MEMORY.md, tools.json, skills/, subagents/)
 * and composes them into a system prompt that is injected each turn.</p>
 *
 * <h3>Workspace Layout:</h3>
 * <pre>
 * workspace/
 *   AGENTS.md       — agent personality / system prompt
 *   MEMORY.md       — long-term memory
 *   tools.json      — MCP tool whitelist
 *   skills/         — skill definitions (*.md files)
 *   subagents/      — sub-agent declarations (*.md files)
 * </pre>
 */
@Component
public class Workspace {

    private static final Logger log = LoggerFactory.getLogger(Workspace.class);

    private final Path workspacePath;
    private final ObjectMapper objectMapper;

    private String agentsMd;
    private String memoryMd;
    private List<Map<String, Object>> toolsConfig;
    private Map<String, String> skills;
    private Map<String, String> subagents;

    public Workspace(WorkspaceConfig config) {
        this(config.getWorkspaceRoot());
    }

    public Workspace(Path workspacePath) {
        this.workspacePath = workspacePath;
        this.objectMapper = new ObjectMapper();
        this.skills = new LinkedHashMap<>();
        this.subagents = new LinkedHashMap<>();
        this.toolsConfig = List.of();
    }

    /**
     * Load all workspace files from disk.
     */
    public void loadWorkspace() {
        loadWorkspace(workspacePath);
    }

    /**
     * Load workspace from a specific path.
     */
    public void loadWorkspace(Path path) {
        this.agentsMd = readFileIfExists(path.resolve("AGENTS.md"));
        this.memoryMd = readFileIfExists(path.resolve("MEMORY.md"));
        this.toolsConfig = loadToolsConfig(path.resolve("tools.json"));
        this.skills = loadMarkdownDir(path.resolve("skills"));
        this.subagents = loadMarkdownDir(path.resolve("subagents"));
        log.info("Workspace loaded from {}: agentsMd={}, memoryMd={}, tools={}, skills={}, subagents={}",
                path,
                agentsMd != null ? agentsMd.length() + " chars" : "null",
                memoryMd != null ? memoryMd.length() + " chars" : "null",
                toolsConfig.size(),
                skills.size(),
                subagents.size());
    }

    /**
     * Build the composed system prompt from workspace files.
     *
     * <p>Injects AGENTS.md personality, MEMORY.md long-term memory,
     * available skills, and sub-agent declarations.</p>
     */
    public String buildSystemPrompt() {
        StringBuilder sb = new StringBuilder();

        // Agent personality from AGENTS.md
        if (agentsMd != null && !agentsMd.isBlank()) {
            sb.append(agentsMd).append("\n\n");
        }

        // Long-term memory from MEMORY.md
        if (memoryMd != null && !memoryMd.isBlank()) {
            sb.append("# Long-Term Memory\n\n").append(memoryMd).append("\n\n");
        }

        // Available skills
        if (!skills.isEmpty()) {
            sb.append("# Available Skills\n\n");
            skills.forEach((name, content) ->
                    sb.append("## ").append(name).append("\n").append(content).append("\n\n"));
        }

        // Sub-agent declarations
        if (!subagents.isEmpty()) {
            sb.append("# Available Sub-Agents\n\n");
            subagents.forEach((name, content) ->
                    sb.append("## ").append(name).append("\n").append(content).append("\n\n"));
        }

        return sb.toString().trim();
    }

    /**
     * Get the tools configuration from tools.json.
     */
    public List<Map<String, Object>> getToolsConfig() {
        return toolsConfig;
    }

    /**
     * Get loaded skill definitions (name → content).
     */
    public Map<String, String> getSkills() {
        return Map.copyOf(skills);
    }

    /**
     * Get loaded sub-agent declarations (name → content).
     */
    public Map<String, String> getSubagents() {
        return Map.copyOf(subagents);
    }

    /**
     * Get the raw AGENTS.md content.
     */
    public String getAgentsMd() {
        return agentsMd;
    }

    /**
     * Get the raw MEMORY.md content.
     */
    public String getMemoryMd() {
        return memoryMd;
    }

    /**
     * Update the MEMORY.md content (in-memory and on disk).
     */
    public void updateMemoryMd(String content) {
        this.memoryMd = content;
        try {
            Files.writeString(workspacePath.resolve("MEMORY.md"), content);
        } catch (IOException e) {
            log.error("Failed to write MEMORY.md: {}", e.getMessage());
        }
    }

    /**
     * Get the workspace root path.
     */
    public Path getWorkspacePath() {
        return workspacePath;
    }

    // ── Private helpers ──────────────────────────────────────────────────

    private String readFileIfExists(Path file) {
        try {
            if (Files.exists(file) && Files.isRegularFile(file)) {
                return Files.readString(file);
            }
        } catch (IOException e) {
            log.warn("Failed to read {}: {}", file, e.getMessage());
        }
        return null;
    }

    private List<Map<String, Object>> loadToolsConfig(Path toolsFile) {
        String content = readFileIfExists(toolsFile);
        if (content == null || content.isBlank()) {
            return List.of();
        }
        try {
            return objectMapper.readValue(content, new TypeReference<>() {});
        } catch (IOException e) {
            log.warn("Failed to parse tools.json: {}", e.getMessage());
            return List.of();
        }
    }

    private Map<String, String> loadMarkdownDir(Path dir) {
        Map<String, String> result = new LinkedHashMap<>();
        if (!Files.isDirectory(dir)) {
            return result;
        }
        try (DirectoryStream<Path> stream = Files.newDirectoryStream(dir, "*.md")) {
            for (Path file : stream) {
                String name = file.getFileName().toString().replace(".md", "");
                String content = Files.readString(file);
                result.put(name, content);
            }
        } catch (IOException e) {
            log.warn("Failed to scan directory {}: {}", dir, e.getMessage());
        }
        return result;
    }
}
