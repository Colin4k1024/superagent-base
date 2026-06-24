package io.superagent.harness;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.DirectoryStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.*;

/**
 * Multi-source skill loading with four-layer composition.
 *
 * <p>Skill loading priority (highest to lowest):</p>
 * <ol>
 *   <li>Workspace skills ({@code workspace/skills/*.md})</li>
 *   <li>Classpath skills ({@code classpath:skills/*.md})</li>
 *   <li>Git skills (external git repo)</li>
 *   <li>MySQL skills (database-stored)</li>
 * </ol>
 *
 * <p>Higher-priority sources override lower-priority skills with the same name.</p>
 */
@Component
public class SkillRepository {

    private static final Logger log = LoggerFactory.getLogger(SkillRepository.class);

    private final Path workspacePath;
    private final Map<String, SkillDefinition> skills;

    public SkillRepository() {
        this(Path.of(".agentscope/workspace"));
    }

    public SkillRepository(WorkspaceConfig config) {
        this(config.getWorkspaceRoot());
    }

    public SkillRepository(Path workspacePath) {
        this.workspacePath = workspacePath;
        this.skills = new LinkedHashMap<>();
    }

    /**
     * Load skills from the workspace skills/ directory.
     */
    public List<SkillDefinition> loadFromWorkspace() {
        return loadFromWorkspace(workspacePath.resolve("skills"));
    }

    /**
     * Load skills from a specific path.
     */
    public List<SkillDefinition> loadFromWorkspace(Path skillsDir) {
        List<SkillDefinition> loaded = new ArrayList<>();
        if (!Files.isDirectory(skillsDir)) {
            return loaded;
        }
        try (DirectoryStream<Path> stream = Files.newDirectoryStream(skillsDir, "*.md")) {
            for (Path file : stream) {
                String name = file.getFileName().toString().replace(".md", "");
                String content = Files.readString(file);
                SkillDefinition skill = new SkillDefinition(name, content, "workspace");
                loaded.add(skill);
                skills.put(name, skill);
                log.info("Loaded workspace skill: {}", name);
            }
        } catch (IOException e) {
            log.error("Failed to load workspace skills: {}", e.getMessage());
        }
        return loaded;
    }

    /**
     * Load skills from the classpath (skills/*.md resources).
     */
    public List<SkillDefinition> loadFromClasspath() {
        List<SkillDefinition> loaded = new ArrayList<>();
        // Scan classpath for skill resources
        try {
            InputStream is = getClass().getClassLoader().getResourceAsStream("skills");
            if (is != null) {
                // In a full implementation, this would enumerate classpath resources
                log.info("Classpath skills directory found");
            }
        } catch (Exception e) {
            log.debug("No classpath skills directory: {}", e.getMessage());
        }
        return loaded;
    }

    /**
     * Load all skills with four-layer composition.
     *
     * <p>Priority: workspace → classpath → git → mysql</p>
     */
    public List<SkillDefinition> loadAll() {
        skills.clear();

        // Layer 1: Workspace (highest priority)
        loadFromWorkspace();

        // Layer 2: Classpath
        for (SkillDefinition skill : loadFromClasspath()) {
            skills.putIfAbsent(skill.name(), skill);
        }

        // Layer 3: Git (placeholder)
        for (SkillDefinition skill : loadFromGit()) {
            skills.putIfAbsent(skill.name(), skill);
        }

        // Layer 4: MySQL (lowest priority)
        for (SkillDefinition skill : loadFromDatabase()) {
            skills.putIfAbsent(skill.name(), skill);
        }

        log.info("Total skills loaded: {}", skills.size());
        return List.copyOf(skills.values());
    }

    /**
     * Get a skill by name.
     */
    public SkillDefinition getSkill(String name) {
        return skills.get(name);
    }

    /**
     * Get all loaded skill names.
     */
    public List<String> listSkillNames() {
        return List.copyOf(skills.keySet());
    }

    /**
     * Get all loaded skills.
     */
    public List<SkillDefinition> getAllSkills() {
        return List.copyOf(skills.values());
    }

    // ── Placeholder loaders for git/mysql layers ────────────────────────

    private List<SkillDefinition> loadFromGit() {
        // Placeholder: would clone/pull from configured git repos
        return List.of();
    }

    private List<SkillDefinition> loadFromDatabase() {
        // Placeholder: would query MySQL for stored skill definitions
        return List.of();
    }

    /**
     * Skill definition record.
     */
    public record SkillDefinition(
            String name,
            String content,
            String source
    ) {
    }
}
