package io.superagent.models;

import org.springframework.stereotype.Component;

import java.util.List;
import java.util.Optional;

/**
 * Model routing with fallback chains.
 *
 * <p>Selects the best model for a given request based on capability,
 * cost, and latency strategies. Falls back to alternative models on failure.</p>
 */
@Component
public class ModelRouter {

    private final ModelRegistry registry;

    public ModelRouter(ModelRegistry registry) {
        this.registry = registry;
    }

    /**
     * Route to the best available model for the given capability.
     *
     * @param capability required capability (e.g., "chat", "code", "vision")
     * @return the resolved model configuration
     */
    public Optional<ModelRegistry.ModelConfig> route(String capability) {
        // TODO: Implement routing strategy
        // 1. Filter models by capability
        // 2. Apply strategy (cost/latency/balanced)
        // 3. Return best match
        return Optional.empty();
    }

    /**
     * Route with fallback chain — tries primary then alternatives.
     *
     * @param capability required capability
     * @param fallbacks  ordered fallback model names
     * @return first available model in the chain
     */
    public Optional<ModelRegistry.ModelConfig> routeWithFallback(
            String capability, List<String> fallbacks) {
        // Try primary route first
        Optional<ModelRegistry.ModelConfig> primary = route(capability);
        if (primary.isPresent()) {
            return primary;
        }

        // Try fallbacks in order
        for (String fallback : fallbacks) {
            Optional<ModelRegistry.ModelConfig> model = registry.resolve(fallback);
            if (model.isPresent()) {
                return model;
            }
        }
        return Optional.empty();
    }
}
