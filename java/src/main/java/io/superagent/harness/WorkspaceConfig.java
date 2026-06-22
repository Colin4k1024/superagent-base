package io.superagent.harness;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Configuration;

import java.nio.file.Path;

/**
 * Workspace configuration for the agent harness.
 *
 * <p>Defines paths and runtime settings used by the AgentScope harness
 * to locate agent definitions, tools, and workspace files.</p>
 */
@Configuration
public class WorkspaceConfig {

    private final String agentsDir;
    private final String workspaceRoot;

    public WorkspaceConfig(
            @Value("${superagent.agents-dir:configs/agents}") String agentsDir,
            @Value("${user.dir:.}") String workspaceRoot) {
        this.agentsDir = agentsDir;
        this.workspaceRoot = workspaceRoot;
    }

    /**
     * Get the path to the agents directory.
     */
    public Path getAgentsPath() {
        return Path.of(workspaceRoot, agentsDir);
    }

    /**
     * Get the path to a specific agent YAML file.
     */
    public Path getAgentPath(String agentName) {
        return getAgentsPath().resolve(agentName + ".yaml");
    }

    /**
     * Get the workspace root path.
     */
    public Path getWorkspaceRoot() {
        return Path.of(workspaceRoot);
    }

    public String getAgentsDir() {
        return agentsDir;
    }
}
