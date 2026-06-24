package io.superagent.models;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.util.Comparator;
import java.util.List;
import java.util.Map;
import java.util.Optional;

/**
 * Model routing with fallback chains.
 *
 * <p>Selects the best model for a given request based on capability,
 * cost, and latency strategies. Falls back to alternative models on failure.</p>
 *
 * <p>Routing strategies (configured via model's {@code parameters} map):</p>
 * <ul>
 *   <li>{@code cost} — prefer lowest-cost model</li>
 *   <li>{@code latency} — prefer lowest-latency model</li>
 *   <li>{@code balanced} (default) — weighted score of cost + latency</li>
 * </ul>
 */
@Component
public class ModelRouter {

    private static final Logger log = LoggerFactory.getLogger(ModelRouter.class);

    private final ModelRegistry registry;

    public ModelRouter(ModelRegistry registry) {
        this.registry = registry;
    }

    /**
     * Route to the best available model for the given capability.
     *
     * <p>Scans all registered models, filters by those whose {@code parameters}
     * map contains a matching {@code capabilities} list, then ranks by the
     * configured strategy (cost/latency/balanced).</p>
     *
     * @param capability required capability (e.g., "chat", "code", "vision")
     * @return the resolved model configuration
     */
    public Optional<ModelRegistry.ModelConfig> route(String capability) {
        return registry.listNames().stream()
            .map(registry::resolve)
            .filter(Optional::isPresent)
            .map(Optional::get)
            .filter(config -> supportsCapability(config, capability))
            .min(buildComparator(getStrategy(null)))
            .or(() -> {
                // Fallback: try any model whose name matches the capability
                return registry.resolve(capability);
            });
    }

    /**
     * Route with a specific strategy override.
     *
     * @param capability required capability
     * @param strategy   routing strategy ("cost", "latency", "balanced")
     * @return the resolved model configuration
     */
    public Optional<ModelRegistry.ModelConfig> route(String capability, String strategy) {
        return registry.listNames().stream()
            .map(registry::resolve)
            .filter(Optional::isPresent)
            .map(Optional::get)
            .filter(config -> supportsCapability(config, capability))
            .min(buildComparator(getStrategy(strategy)))
            .or(() -> registry.resolve(capability));
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
                log.debug("Using fallback model '{}' for capability '{}'", fallback, capability);
                return model;
            }
        }
        return Optional.empty();
    }

    /**
     * Check if a model config supports the requested capability.
     * A model supports a capability if:
     * - its parameters map has a "capabilities" list containing the capability, OR
     * - its name contains the capability string (heuristic fallback)
     */
    @SuppressWarnings("unchecked")
    private boolean supportsCapability(ModelRegistry.ModelConfig config, String capability) {
        if (config.parameters() == null) return false;

        Object caps = config.parameters().get("capabilities");
        if (caps instanceof List<?> capList) {
            return capList.contains(capability);
        }

        // Heuristic: model name contains capability
        return config.name() != null && config.name().contains(capability);
    }

    /**
     * Determine the routing strategy from config or override.
     */
    private String getStrategy(String override) {
        if (override != null && !override.isEmpty()) return override;
        return "balanced";
    }

    /**
     * Build a comparator for ranking models by strategy.
     */
    private Comparator<ModelRegistry.ModelConfig> buildComparator(String strategy) {
        return switch (strategy) {
            case "cost" -> Comparator.comparingDouble(this::extractCost);
            case "latency" -> Comparator.comparingDouble(this::extractLatency);
            default -> // balanced: weighted score
                Comparator.comparingDouble(
                    (ModelRegistry.ModelConfig c) -> extractCost(c) * 0.5 + extractLatency(c) * 0.5);
        };
    }

    /**
     * Extract cost score from model parameters (lower is better).
     * Defaults to 1.0 if not specified.
     */
    private double extractCost(ModelRegistry.ModelConfig config) {
        if (config.parameters() == null) return 1.0;
        Object cost = config.parameters().get("cost");
        if (cost instanceof Number n) return n.doubleValue();
        return 1.0;
    }

    /**
     * Extract latency score from model parameters (lower is better).
     * Defaults to 1.0 if not specified.
     */
    private double extractLatency(ModelRegistry.ModelConfig config) {
        if (config.parameters() == null) return 1.0;
        Object latency = config.parameters().get("latency");
        if (latency instanceof Number n) return n.doubleValue();
        return 1.0;
    }
}
