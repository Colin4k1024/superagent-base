package io.superagent.harness;

import io.superagent.agents.BaseAgent;
import io.superagent.agents.ChatModelAgent;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * Builder for composing all harness components into a single agent.
 *
 * <p>Follows the AgentScope Java API builder pattern:</p>
 * <pre>
 * BaseAgent agent = HarnessAgentBuilder.create()
 *     .workspace(Path.of("my-workspace"))
 *     .stateStore(new FileAgentStateStore())
 *     .memory(workspace, config)
 *     .compaction(128000)
 *     .toolResultEviction(80000)
 *     .subagentManager(config)
 *     .skillRepository(config)
 *     .enablePlanMode()
 *     .channel()
 *     .build();
 * </pre>
 */
public class HarnessAgentBuilder {

    private static final Logger log = LoggerFactory.getLogger(HarnessAgentBuilder.class);

    private Path workspacePath;
    private AgentStateStore stateStore;
    private HarnessMemory memory;
    private CompactionManager compaction;
    private ToolResultEviction toolResultEviction;
    private SubagentManager subagentManager;
    private SandboxManager sandboxManager;
    private SkillRepository skillRepository;
    private PlanModeManager planModeManager;
    private Channel channel;
    private SessionManager sessionManager;

    // Agent configuration
    private String agentName = "harness-agent";
    private String agentDescription = "AgentScope harness agent";
    private String modelName = "default";
    private List<String> tools = new ArrayList<>();
    private int maxSteps = 10;
    private String systemPrompt = "";
    private boolean planModeEnabled = false;

    private HarnessAgentBuilder() {
    }

    /**
     * Create a new builder instance.
     */
    public static HarnessAgentBuilder create() {
        return new HarnessAgentBuilder();
    }

    /**
     * Set the workspace path.
     */
    public HarnessAgentBuilder workspace(Path path) {
        this.workspacePath = path;
        return this;
    }

    /**
     * Set the state store.
     */
    public HarnessAgentBuilder stateStore(AgentStateStore store) {
        this.stateStore = store;
        return this;
    }

    /**
     * Configure memory with workspace and config.
     */
    public HarnessAgentBuilder memory(Workspace workspace, WorkspaceConfig config) {
        this.memory = new HarnessMemory(config, workspace);
        return this;
    }

    /**
     * Set the memory instance directly.
     */
    public HarnessAgentBuilder memory(HarnessMemory memory) {
        this.memory = memory;
        return this;
    }

    /**
     * Configure compaction with max tokens.
     */
    public HarnessAgentBuilder compaction(int maxTokens) {
        this.compaction = new CompactionManager(maxTokens);
        return this;
    }

    /**
     * Set the compaction manager directly.
     */
    public HarnessAgentBuilder compaction(CompactionManager compaction) {
        this.compaction = compaction;
        return this;
    }

    /**
     * Configure tool result eviction with threshold.
     */
    public HarnessAgentBuilder toolResultEviction(Path evictionDir, int threshold) {
        this.toolResultEviction = new ToolResultEviction(evictionDir, threshold);
        return this;
    }

    /**
     * Set the tool result eviction directly.
     */
    public HarnessAgentBuilder toolResultEviction(ToolResultEviction eviction) {
        this.toolResultEviction = eviction;
        return this;
    }

    /**
     * Configure the sub-agent manager.
     */
    public HarnessAgentBuilder subagentManager(SubagentManager manager) {
        this.subagentManager = manager;
        return this;
    }

    /**
     * Configure the sandbox manager.
     */
    public HarnessAgentBuilder sandboxManager(SandboxManager manager) {
        this.sandboxManager = manager;
        return this;
    }

    /**
     * Configure the skill repository.
     */
    public HarnessAgentBuilder skillRepository(SkillRepository repository) {
        this.skillRepository = repository;
        return this;
    }

    /**
     * Enable plan mode.
     */
    public HarnessAgentBuilder enablePlanMode() {
        this.planModeEnabled = true;
        return this;
    }

    /**
     * Configure the channel.
     */
    public HarnessAgentBuilder channel(Channel channel) {
        this.channel = channel;
        return this;
    }

    /**
     * Configure the session manager.
     */
    public HarnessAgentBuilder sessionManager(SessionManager sessionManager) {
        this.sessionManager = sessionManager;
        return this;
    }

    /**
     * Set the agent name.
     */
    public HarnessAgentBuilder name(String name) {
        this.agentName = name;
        return this;
    }

    /**
     * Set the agent description.
     */
    public HarnessAgentBuilder description(String description) {
        this.agentDescription = description;
        return this;
    }

    /**
     * Set the model name.
     */
    public HarnessAgentBuilder model(String modelName) {
        this.modelName = modelName;
        return this;
    }

    /**
     * Set the tools list.
     */
    public HarnessAgentBuilder tools(List<String> tools) {
        this.tools = new ArrayList<>(tools);
        return this;
    }

    /**
     * Set max steps.
     */
    public HarnessAgentBuilder maxSteps(int maxSteps) {
        this.maxSteps = maxSteps;
        return this;
    }

    /**
     * Set the system prompt.
     */
    public HarnessAgentBuilder systemPrompt(String systemPrompt) {
        this.systemPrompt = systemPrompt;
        return this;
    }

    /**
     * Build the harness agent with all configured components.
     *
     * @return configured BaseAgent
     */
    public BaseAgent build() {
        // Build workspace if path is set
        Workspace workspace = null;
        if (workspacePath != null) {
            workspace = new Workspace(workspacePath);
            workspace.loadWorkspace();
        }

        // Compose system prompt from workspace
        String composedPrompt = systemPrompt;
        if (workspace != null) {
            String workspacePrompt = workspace.buildSystemPrompt();
            if (!workspacePrompt.isBlank()) {
                composedPrompt = workspacePrompt + "\n\n" + systemPrompt;
            }
        }

        // Build the core agent
        ChatModelAgent agent = new ChatModelAgent(
                agentName,
                agentDescription,
                modelName,
                tools,
                maxSteps,
                composedPrompt
        );

        // Wrap with plan mode if enabled
        BaseAgent result = agent;
        if (planModeEnabled && planModeManager != null) {
            result = planModeManager.enablePlanMode(agent);
        }

        log.info("HarnessAgent built: name={}, model={}, tools={}, planMode={}",
                agentName, modelName, tools.size(), planModeEnabled);

        return result;
    }

    /**
     * Get all configured harness components as a summary map.
     */
    public Map<String, Object> getComponentSummary() {
        return Map.of(
                "workspace", workspacePath != null ? workspacePath.toString() : "none",
                "stateStore", stateStore != null ? stateStore.getClass().getSimpleName() : "none",
                "memory", memory != null ? "configured" : "none",
                "compaction", compaction != null ? compaction.getMaxTokens() : "none",
                "toolResultEviction", toolResultEviction != null ? toolResultEviction.getThreshold() : "none",
                "subagentManager", subagentManager != null ? "configured" : "none",
                "sandboxManager", sandboxManager != null ? "configured" : "none",
                "skillRepository", skillRepository != null ? "configured" : "none",
                "planMode", planModeEnabled,
                "channel", channel != null ? "configured" : "none"
        );
    }
}
