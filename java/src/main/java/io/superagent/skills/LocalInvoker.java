package io.superagent.skills;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.function.Function;

/**
 * Invokes locally-registered skill functions.
 *
 * <p>Builtin skills (datetime, calculator, uuid) are registered as
 * simple functions that execute in-process.</p>
 *
 * <p>Maps to Go {@code skill.LocalInvoker}.</p>
 */
public class LocalInvoker implements SkillInvoker {

    private static final Logger log = LoggerFactory.getLogger(LocalInvoker.class);

    private final ConcurrentHashMap<String, Function<Map<String, Object>, Map<String, Object>>> skills
        = new ConcurrentHashMap<>();

    /**
     * Register a local skill function.
     *
     * @param name skill name
     * @param fn   function that takes input map and returns output map
     */
    public void register(String name, Function<Map<String, Object>, Map<String, Object>> fn) {
        skills.put(name, fn);
        log.debug("Registered local skill: {}", name);
    }

    /**
     * Unregister a local skill.
     *
     * @param name skill name
     * @return true if the skill was registered
     */
    public boolean unregister(String name) {
        return skills.remove(name) != null;
    }

    @Override
    public Map<String, Object> invoke(String name, Map<String, Object> input) throws SkillException {
        Function<Map<String, Object>, Map<String, Object>> fn = skills.get(name);
        if (fn == null) {
            throw new SkillException("Local skill not found: " + name);
        }

        try {
            Map<String, Object> result = fn.apply(input != null ? input : Map.of());
            log.debug("Local skill '{}' invoked successfully", name);
            return result;
        } catch (Exception e) {
            throw new SkillException("Local skill '" + name + "' failed: " + e.getMessage(), e);
        }
    }

    @Override
    public boolean canInvoke(String name) {
        return skills.containsKey(name);
    }

    /**
     * Get all registered skill names.
     */
    public Set<String> getRegisteredNames() {
        return Set.copyOf(skills.keySet());
    }

    /**
     * Get count of registered skills.
     */
    public int size() {
        return skills.size();
    }
}
