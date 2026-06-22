package io.superagent.skills;

import java.util.Map;

/**
 * Interface for invoking skills by name.
 *
 * <p>Different invoker implementations handle different skill sources:
 * local functions, HTTP endpoints, or composite chains.</p>
 *
 * <p>Maps to Go {@code skill.SkillInvoker} interface.</p>
 */
public interface SkillInvoker {

    /**
     * Invoke a skill with the given input.
     *
     * @param name  skill name
     * @param input input parameters
     * @return output result
     * @throws SkillException if invocation fails
     */
    Map<String, Object> invoke(String name, Map<String, Object> input) throws SkillException;

    /**
     * Check if this invoker can handle the given skill name.
     *
     * @param name skill name
     * @return true if this invoker supports the skill
     */
    boolean canInvoke(String name);

    /**
     * Skill-specific exception.
     */
    class SkillException extends Exception {
        public SkillException(String message) {
            super(message);
        }

        public SkillException(String message, Throwable cause) {
            super(message, cause);
        }
    }
}
