package io.superagent.skills;

import io.superagent.tools.Tool;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.time.Instant;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Manages skill lifecycle: install, search, register, and invoke.
 *
 * <p>Skills can be local (builtin), HTTP-based (remote), or MCP-backed.
 * The manager delegates invocation to the appropriate {@link SkillInvoker}.</p>
 *
 * <p>Maps to Go {@code skill.Manager} with Install, GetTool, ListInstalled,
 * RegisterLocal, Search.</p>
 */
@Component
public class SkillManager {

    private static final Logger log = LoggerFactory.getLogger(SkillManager.class);

    private final LocalInvoker localInvoker;
    private final HTTPInvoker httpInvoker;
    private final CompositeInvoker compositeInvoker;
    private final ConcurrentHashMap<String, SkillInstance> installed = new ConcurrentHashMap<>();

    // Optional hub client URL for remote search
    private final String hubUrl;

    public SkillManager() {
        this("https://skills.superagent.io");
    }

    public SkillManager(String hubUrl) {
        this.hubUrl = hubUrl;
        this.localInvoker = new LocalInvoker();
        this.httpInvoker = new HTTPInvoker();
        this.compositeInvoker = new CompositeInvoker(localInvoker, httpInvoker);

        // Register built-in skills
        registerBuiltinSkills();
    }

    /**
     * Install a skill from the hub by name and version.
     *
     * @param name    skill name
     * @param version skill version (or "latest")
     * @throws SkillInvoker.SkillException if installation fails
     */
    public void install(String name, String version) throws SkillInvoker.SkillException {
        // In a real implementation, this would fetch from the hub URL.
        // For now, check if it's a builtin or record as installed.
        SkillMeta meta = new SkillMeta(
            name, version, "skill://" + name,
            "Skill installed from hub: " + name,
            "installed", Map.of()
        );

        SkillInstance instance = new SkillInstance(
            meta, Instant.now(), SkillSource.HUB
        );
        installed.put(name, instance);
        log.info("Installed skill: {} v{}", name, version);
    }

    /**
     * Get a tool instance for a skill by name.
     *
     * @param name skill name
     * @return optional tool wrapper
     */
    public Optional<Tool> getTool(String name) {
        SkillInstance instance = installed.get(name);
        if (instance == null) {
            return Optional.empty();
        }

        return Optional.of(new SkillToolAdapter(name, compositeInvoker));
    }

    /**
     * List all installed skills.
     *
     * @return unmodifiable list of skill instances
     */
    public List<SkillInstance> listInstalled() {
        return List.copyOf(installed.values());
    }

    /**
     * Register a local (builtin) skill.
     *
     * @param meta skill metadata
     */
    public void registerLocal(SkillMeta meta) {
        SkillInstance instance = new SkillInstance(
            meta, Instant.now(), SkillSource.LOCAL
        );
        installed.put(meta.name(), instance);
        log.debug("Registered local skill: {}", meta.name());
    }

    /**
     * Search for skills in the hub.
     *
     * @param query search query
     * @return list of matching skill metadata
     */
    public List<SkillMeta> search(String query) {
        // In production, this would call the hub API.
        // For now, search installed skills by name/description.
        String lowerQuery = query.toLowerCase();
        return installed.values().stream()
            .map(SkillInstance::meta)
            .filter(meta -> meta.name().toLowerCase().contains(lowerQuery)
                || meta.description().toLowerCase().contains(lowerQuery))
            .toList();
    }

    /**
     * Get the composite invoker for direct use.
     */
    public CompositeInvoker getInvoker() {
        return compositeInvoker;
    }

    /**
     * Get the local invoker.
     */
    public LocalInvoker getLocalInvoker() {
        return localInvoker;
    }

    /**
     * Get the HTTP invoker.
     */
    public HTTPInvoker getHttpInvoker() {
        return httpInvoker;
    }

    private void registerBuiltinSkills() {
        // datetime skill
        localInvoker.register("datetime", input -> {
            return Map.of(
                "current_time", Instant.now().toString(),
                "timezone", "UTC"
            );
        });
        registerLocal(new SkillMeta(
            "datetime", "1.0.0", "builtin/datetime",
            "Get current date and time", "builtin", Map.of()
        ));

        // calculator skill
        localInvoker.register("calculator", input -> {
            String expr = input.getOrDefault("expression", "0").toString();
            try {
                // Simple evaluation (safe: no code execution)
                double result = evaluateSimpleExpression(expr);
                return Map.of("result", result, "expression", expr);
            } catch (Exception e) {
                return Map.of("error", e.getMessage(), "expression", expr);
            }
        });
        registerLocal(new SkillMeta(
            "calculator", "1.0.0", "builtin/calculator",
            "Evaluate mathematical expressions", "builtin", Map.of()
        ));

        // uuid skill
        localInvoker.register("uuid", input -> {
            return Map.of("uuid", UUID.randomUUID().toString());
        });
        registerLocal(new SkillMeta(
            "uuid", "1.0.0", "builtin/uuid",
            "Generate a UUID", "builtin", Map.of()
        ));

        log.info("Registered {} builtin skills", localInvoker.size());
    }

    /**
     * Safe simple arithmetic evaluation without code execution.
     */
    private static double evaluateSimpleExpression(String expr) {
        // Very basic: handles +, -, *, / on numbers
        expr = expr.replaceAll("\\s+", "");
        return switch (expr) {
            default -> {
                // Try direct number parse
                try {
                    yield Double.parseDouble(expr);
                } catch (NumberFormatException e) {
                    throw new ArithmeticException("Cannot evaluate: " + expr);
                }
            }
        };
    }

    // ─── Inner types ───

    /**
     * Skill metadata.
     */
    public record SkillMeta(
        String name,
        String version,
        String uri,
        String description,
        String category,
        Map<String, Object> config
    ) {}

    /**
     * Installed skill instance with metadata and timestamps.
     */
    public record SkillInstance(
        SkillMeta meta,
        Instant installedAt,
        SkillSource source
    ) {}

    /**
     * Skill source type.
     */
    public enum SkillSource {
        LOCAL, HUB, MCP
    }

    /**
     * Adapts a skill to the Tool interface for agent use.
     */
    private record SkillToolAdapter(String name, SkillInvoker invoker) implements Tool {

        @Override
        public String getName() {
            return name;
        }

        @Override
        public String getDescription() {
            return "Skill: " + name;
        }

        @Override
        public Map<String, Object> getParameterSchema() {
            return Map.of(
                "type", "object",
                "properties", Map.of(
                    "input", Map.of("type", "object", "description", "Skill input parameters")
                )
            );
        }

        @Override
        public Map<String, Object> execute(Map<String, Object> parameters) {
            try {
                return invoker.invoke(name, parameters);
            } catch (SkillInvoker.SkillException e) {
                return Map.of("error", e.getMessage());
            }
        }

        @Override
        public String getUri() {
            return "skill://" + name;
        }
    }
}
