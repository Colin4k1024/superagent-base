package io.superagent.models;

import io.agentscope.core.model.DashScopeChatModel;
import io.agentscope.core.model.Model;
import io.agentscope.core.model.OpenAIChatModel;
import org.springframework.stereotype.Component;

import java.util.Map;
import java.util.Optional;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Registry of available model configurations.
 *
 * <p>Models are registered by name and resolved at runtime by agents.
 * Supports multiple providers (OpenAI, Anthropic, local, etc.).</p>
 *
 * <h3>AgentScope 2.0 Integration</h3>
 * <p>Can create {@link Model} instances from registered configurations
 * for use with {@code ReActAgent}.</p>
 */
@Component
public class ModelRegistry {

    private final ConcurrentHashMap<String, ModelConfig> models =
        new ConcurrentHashMap<>();

    /**
     * Register a model configuration.
     */
    public void register(String name, ModelConfig config) {
        models.put(name, config);
    }

    /**
     * Resolve a model by name.
     */
    public Optional<ModelConfig> resolve(String name) {
        return Optional.ofNullable(models.get(name));
    }

    /**
     * Check if a model is registered.
     */
    public boolean has(String name) {
        return models.containsKey(name);
    }

    /**
     * Return all registered model names.
     */
    public java.util.Set<String> listNames() {
        return models.keySet();
    }

    /**
     * Create a {@link Model} instance from a registered configuration.
     *
     * @param name model name (e.g., "gpt-4o", "dashscope:qwen-max")
     * @return Model instance, or a default OpenAI model if not found
     */
    public Model createChatModel(String name) {
        if (name == null || name.isEmpty() || "default".equals(name)) {
            return OpenAIChatModel.builder().modelName("gpt-4o").build();
        }

        // Parse "provider:model" format
        String[] parts = name.split(":", 2);
        String provider = parts[0];
        String modelName = parts.length > 1 ? parts[1] : parts[0];

        return switch (provider) {
            case "dashscope" -> DashScopeChatModel.builder()
                .modelName(modelName)
                .apiKey(resolveApiKey(name))
                .build();
            case "openai" -> OpenAIChatModel.builder()
                .modelName(modelName)
                .baseUrl(resolveBaseUrl(name))
                .apiKey(resolveApiKey(name))
                .build();
            default -> {
                // Try to resolve from registry
                Optional<ModelConfig> config = resolve(name);
                if (config.isPresent()) {
                    ModelConfig cfg = config.get();
                    yield switch (cfg.provider()) {
                        case "dashscope" -> DashScopeChatModel.builder()
                            .modelName(cfg.modelName())
                            .apiKey(resolveApiKey(cfg.apiKeyEnv()))
                            .build();
                        default -> OpenAIChatModel.builder()
                            .modelName(cfg.modelName())
                            .baseUrl(cfg.baseUrl())
                            .apiKey(resolveApiKey(cfg.apiKeyEnv()))
                            .build();
                    };
                }
                yield OpenAIChatModel.builder().modelName(name).build();
            }
        };
    }

    private String resolveApiKey(String envVar) {
        if (envVar == null || envVar.isEmpty()) return "";
        // Try environment variable first
        String envValue = System.getenv(envVar);
        if (envValue != null) return envValue;
        // Try as direct value
        return envVar;
    }

    private String resolveBaseUrl(String name) {
        Optional<ModelConfig> config = resolve(name);
        return config.map(ModelConfig::baseUrl).orElse("https://api.openai.com/v1");
    }

    /**
     * Model configuration record.
     */
    public record ModelConfig(
        String name,
        String provider,
        String modelName,
        String apiKeyEnv,
        String baseUrl,
        Map<String, Object> parameters
    ) {
    }
}
