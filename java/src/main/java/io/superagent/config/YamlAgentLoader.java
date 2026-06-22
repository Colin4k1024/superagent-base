package io.superagent.config;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import org.yaml.snakeyaml.Yaml;

import java.io.IOException;
import java.nio.file.DirectoryStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * Loads agent definitions from YAML files in the configured agents directory.
 *
 * <p>Scans {@code superagent.agents-dir} for {@code *.yaml} files and parses
 * each into an {@link AgentDefinition} record.</p>
 */
@Component
public class YamlAgentLoader {

    private static final Logger log = LoggerFactory.getLogger(YamlAgentLoader.class);

    private final String agentsDir;

    public YamlAgentLoader(
            @Value("${superagent.agents-dir:configs/agents}") String agentsDir) {
        this.agentsDir = agentsDir;
    }

    /**
     * Load all agent definitions from the configured directory.
     *
     * @return list of parsed agent definitions
     */
    public List<AgentDefinition> loadAll() {
        List<AgentDefinition> definitions = new ArrayList<>();
        Path dir = Path.of(agentsDir);

        if (!Files.isDirectory(dir)) {
            log.warn("Agents directory does not exist: {}", agentsDir);
            return definitions;
        }

        try (DirectoryStream<Path> stream = Files.newDirectoryStream(dir, "*.yaml")) {
            for (Path file : stream) {
                try {
                    AgentDefinition def = loadFile(file);
                    definitions.add(def);
                    log.info("Loaded agent: {} from {}", def.metadata().name(), file);
                } catch (Exception e) {
                    log.error("Failed to load agent from {}: {}", file, e.getMessage());
                }
            }
        } catch (IOException e) {
            log.error("Failed to scan agents directory: {}", e.getMessage());
        }

        return definitions;
    }

    /**
     * Load a single agent definition from a YAML file.
     */
    @SuppressWarnings("unchecked")
    public AgentDefinition loadFile(Path file) throws IOException {
        Yaml yaml = new Yaml();
        Map<String, Object> data;
        try (var reader = Files.newBufferedReader(file)) {
            data = yaml.load(reader);
        }

        // Parse into record (stub: full mapping logic TBD)
        String apiVersion = (String) data.getOrDefault("apiVersion", "superagent/v1");
        String kind = (String) data.getOrDefault("kind", "Agent");

        Map<String, Object> meta = (Map<String, Object>) data.getOrDefault("metadata", Map.of());
        AgentDefinition.Metadata metadata = new AgentDefinition.Metadata(
            (String) meta.get("name"),
            (String) meta.get("version"),
            (Map<String, String>) meta.getOrDefault("labels", Map.of())
        );

        Map<String, Object> spec = (Map<String, Object>) data.getOrDefault("spec", Map.of());
        AgentDefinition.Spec specRecord = parseSpec(spec);

        return new AgentDefinition(apiVersion, kind, metadata, specRecord);
    }

    @SuppressWarnings("unchecked")
    private AgentDefinition.Spec parseSpec(Map<String, Object> spec) {
        String type = (String) spec.getOrDefault("type", "chat_model_agent");
        String systemPrompt = (String) spec.getOrDefault("system_prompt", "");

        Map<String, Object> modelMap = (Map<String, Object>) spec.getOrDefault("model", Map.of());
        AgentDefinition.ModelConfig model = new AgentDefinition.ModelConfig(
            (String) modelMap.get("primary"),
            (String) modelMap.get("fallback"),
            (String) modelMap.get("router")
        );

        // Stub: parse tools, memory, interrupt, etc.
        return new AgentDefinition.Spec(
            type, model, systemPrompt,
            List.of(),   // tools — TODO: parse
            null,        // memory — TODO: parse
            null,        // interrupt — TODO: parse
            List.of(),   // subAgents — TODO: parse
            null,        // workflow — TODO: parse
            Map.of(),    // evolution — TODO: parse
            Map.of()     // observability — TODO: parse
        );
    }
}
