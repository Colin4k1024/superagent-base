package io.superagent.skills;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * Chains multiple {@link SkillInvoker} instances.
 *
 * <p>Tries each invoker in order until one claims it can handle the skill.
 * This enables fallback from local → HTTP → other invoker types.</p>
 *
 * <p>Maps to Go {@code skill.CompositeInvoker}.</p>
 */
public class CompositeInvoker implements SkillInvoker {

    private static final Logger log = LoggerFactory.getLogger(CompositeInvoker.class);

    private final List<SkillInvoker> invokers;

    public CompositeInvoker(List<SkillInvoker> invokers) {
        this.invokers = invokers != null ? List.copyOf(invokers) : List.of();
    }

    public CompositeInvoker(SkillInvoker... invokers) {
        this.invokers = List.of(invokers);
    }

    @Override
    public Map<String, Object> invoke(String name, Map<String, Object> input) throws SkillException {
        for (SkillInvoker invoker : invokers) {
            if (invoker.canInvoke(name)) {
                log.debug("Delegating skill '{}' to {}", name, invoker.getClass().getSimpleName());
                return invoker.invoke(name, input);
            }
        }
        throw new SkillException("No invoker found for skill: " + name);
    }

    @Override
    public boolean canInvoke(String name) {
        return invokers.stream().anyMatch(inv -> inv.canInvoke(name));
    }

    /**
     * Get the list of child invokers.
     */
    public List<SkillInvoker> getInvokers() {
        return invokers;
    }

    /**
     * Create a builder for composing invokers.
     */
    public static Builder builder() {
        return new Builder();
    }

    /**
     * Builder for CompositeInvoker.
     */
    public static class Builder {
        private final List<SkillInvoker> invokers = new ArrayList<>();

        public Builder add(SkillInvoker invoker) {
            invokers.add(invoker);
            return this;
        }

        public CompositeInvoker build() {
            return new CompositeInvoker(invokers);
        }
    }
}
