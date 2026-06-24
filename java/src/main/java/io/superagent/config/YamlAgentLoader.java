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

        List<AgentDefinition.ToolRef> tools = parseTools(spec);
        AgentDefinition.MemoryConfig memory = parseMemory(spec);
        AgentDefinition.InterruptConfig interrupt = parseInterrupt(spec);
        List<AgentDefinition.SubAgentRef> subAgents = parseSubAgents(spec);
        AgentDefinition.WorkflowConfig workflow = parseWorkflow(spec);
        Map<String, Object> evolution = (Map<String, Object>) spec.getOrDefault("evolution", Map.of());
        Map<String, Object> observability = (Map<String, Object>) spec.getOrDefault("observability", Map.of());

        return new AgentDefinition.Spec(
            type, model, systemPrompt,
            tools, memory, interrupt, subAgents, workflow,
            evolution, observability
        );
    }

    @SuppressWarnings("unchecked")
    private List<AgentDefinition.ToolRef> parseTools(Map<String, Object> spec) {
        Object toolsRaw = spec.get("tools");
        if (!(toolsRaw instanceof List<?> toolsList)) return List.of();

        return toolsList.stream()
            .map(item -> {
                if (item instanceof Map) {
                    Map<String, Object> m = (Map<String, Object>) item;
                    return new AgentDefinition.ToolRef((String) m.getOrDefault("ref", ""));
                }
                return new AgentDefinition.ToolRef(item.toString());
            })
            .filter(t -> !t.ref().isEmpty())
            .toList();
    }

    @SuppressWarnings("unchecked")
    private AgentDefinition.MemoryConfig parseMemory(Map<String, Object> spec) {
        Map<String, Object> memMap = (Map<String, Object>) spec.get("memory");
        if (memMap == null) return null;

        String backend = (String) memMap.getOrDefault("backend", "builtin");
        Map<String, Object> options = (Map<String, Object>) memMap.getOrDefault("options", Map.of());
        return new AgentDefinition.MemoryConfig(backend, options);
    }

    @SuppressWarnings("unchecked")
    private AgentDefinition.InterruptConfig parseInterrupt(Map<String, Object> spec) {
        Map<String, Object> intMap = (Map<String, Object>) spec.get("interrupt");
        if (intMap == null) return null;

        boolean enabled = Boolean.TRUE.equals(intMap.get("enabled"));
        String checkpointBackend = (String) intMap.getOrDefault("checkpoint_backend", "redis");
        Object timeoutRaw = intMap.get("timeout_seconds");
        int timeoutSeconds = timeoutRaw instanceof Number n ? n.intValue() : 300;
        return new AgentDefinition.InterruptConfig(enabled, checkpointBackend, timeoutSeconds);
    }

    @SuppressWarnings("unchecked")
    private List<AgentDefinition.SubAgentRef> parseSubAgents(Map<String, Object> spec) {
        Object subRaw = spec.get("sub_agents");
        if (subRaw == null) subRaw = spec.get("subAgents");
        if (!(subRaw instanceof List<?> subList)) return List.of();

        return subList.stream()
            .map(item -> {
                if (item instanceof Map) {
                    Map<String, Object> m = (Map<String, Object>) item;
                    return new AgentDefinition.SubAgentRef((String) m.getOrDefault("ref", ""));
                }
                return new AgentDefinition.SubAgentRef(item.toString());
            })
            .filter(s -> !s.ref().isEmpty())
            .toList();
    }

    @SuppressWarnings("unchecked")
    private AgentDefinition.WorkflowConfig parseWorkflow(Map<String, Object> spec) {
        Map<String, Object> wfMap = (Map<String, Object>) spec.get("workflow");
        if (wfMap == null) return null;

        List<AgentDefinition.WorkflowNode> nodes = List.of();
        Object nodesRaw = wfMap.get("nodes");
        if (nodesRaw instanceof List<?> nodesList) {
            nodes = nodesList.stream()
                .map(item -> {
                    if (item instanceof Map) {
                        Map<String, Object> m = (Map<String, Object>) item;
                        Map<String, Object> cfg = m.get("config") instanceof Map
                            ? (Map<String, Object>) m.get("config") : Map.of();
                        return new AgentDefinition.WorkflowNode(
                            (String) m.get("id"),
                            (String) m.get("type"),
                            (String) m.get("agent_ref"),
                            cfg
                        );
                    }
                    return null;
                })
                .filter(java.util.Objects::nonNull)
                .toList();
        }

        List<AgentDefinition.WorkflowEdge> edges = List.of();
        Object edgesRaw = wfMap.get("edges");
        if (edgesRaw instanceof List<?> edgesList) {
            edges = edgesList.stream()
                .map(item -> {
                    if (item instanceof Map) {
                        Map<String, Object> m = (Map<String, Object>) item;
                        return new AgentDefinition.WorkflowEdge(
                            (String) m.get("from"),
                            (String) m.get("to"),
                            (String) m.get("condition")
                        );
                    }
                    return null;
                })
                .filter(java.util.Objects::nonNull)
                .toList();
        }

        return new AgentDefinition.WorkflowConfig(nodes, edges);
    }
}
