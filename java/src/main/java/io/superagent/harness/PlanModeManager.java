package io.superagent.harness;

import io.superagent.agents.BaseAgent;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Plan mode manager — read-only thinking phase with HITL exit gate.
 *
 * <p>When plan mode is enabled for an agent, the agent can only read
 * and analyze — no writes, no tool executions, no side effects.
 * The agent must produce a plan and request human approval before
 * proceeding to execution.</p>
 */
@Component
public class PlanModeManager {

    private static final Logger log = LoggerFactory.getLogger(PlanModeManager.class);

    private final ConcurrentHashMap<String, Boolean> planModeStates = new ConcurrentHashMap<>();

    /**
     * Enable plan mode for an agent, wrapping it with read-only constraints.
     *
     * @param agent the agent to wrap
     * @return wrapped agent with plan mode enabled
     */
    public BaseAgent enablePlanMode(BaseAgent agent) {
        planModeStates.put(agent.getName(), true);
        log.info("Plan mode enabled for agent: {}", agent.getName());
        return new PlanModeAgent(agent);
    }

    /**
     * Disable plan mode for an agent.
     */
    public void disablePlanMode(String agentName) {
        planModeStates.remove(agentName);
        log.info("Plan mode disabled for agent: {}", agentName);
    }

    /**
     * Check if an agent is in plan mode.
     */
    public boolean isReadOnly(String agentName) {
        return Boolean.TRUE.equals(planModeStates.get(agentName));
    }

    /**
     * Check if plan mode is active (for any agent).
     */
    public boolean hasActivePlanMode() {
        return planModeStates.containsValue(true);
    }

    /**
     * Get all agents currently in plan mode.
     */
    public List<String> getPlanModeAgents() {
        return planModeStates.entrySet().stream()
                .filter(Map.Entry::getValue)
                .map(Map.Entry::getKey)
                .toList();
    }

    /**
     * Wrapped agent that enforces read-only plan mode.
     */
    private static class PlanModeAgent extends BaseAgent {

        private final BaseAgent delegate;

        PlanModeAgent(BaseAgent delegate) {
            super(delegate.getName(), delegate.getDescription(), delegate.getAgentType());
            this.delegate = delegate;
        }

        @Override
        public Map<String, Object> run(Map<String, Object> input) {
            // In plan mode, we add a system instruction to only plan
            Map<String, Object> planInput = new java.util.LinkedHashMap<>(input);
            planInput.put("_plan_mode", true);
            planInput.put("_read_only", true);

            Map<String, Object> result = delegate.run(planInput);
            result.put("plan_mode", true);
            result.put("requires_approval", true);
            return result;
        }

        @Override
        public List<String> getTools() {
            // No tools available in plan mode
            return List.of();
        }
    }
}
