package io.superagent.harness;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Docker-based sandbox isolation for agent code execution.
 *
 * <p>Provides isolated execution environments with cross-invocation
 * state persistence.</p>
 */
@Component
public class SandboxManager {

    private static final Logger log = LoggerFactory.getLogger(SandboxManager.class);

    private final Map<String, SandboxHandle> activeSandboxes = new ConcurrentHashMap<>();
    private final Path sandboxStateDir;

    public SandboxManager(WorkspaceConfig config) {
        this(config.getWorkspaceRoot().resolve("sandboxes"));
    }

    public SandboxManager(Path sandboxStateDir) {
        this.sandboxStateDir = sandboxStateDir;
    }

    /**
     * Create a new sandbox environment.
     *
     * @param config sandbox configuration
     * @return handle to the created sandbox
     */
    public SandboxHandle createSandbox(SandboxConfig config) {
        String id = UUID.randomUUID().toString().substring(0, 8);
        String containerName = "sandbox-" + id;

        SandboxHandle handle = new SandboxHandle(id, containerName, config);
        activeSandboxes.put(id, handle);

        // Create state directory for cross-invocation persistence
        try {
            Path statePath = sandboxStateDir.resolve(id);
            Files.createDirectories(statePath);
            handle.setStatePath(statePath);
            log.info("Sandbox created: {} (container: {})", id, containerName);
        } catch (IOException e) {
            log.error("Failed to create sandbox state dir: {}", e.getMessage());
        }

        return handle;
    }

    /**
     * Execute a command in a sandbox.
     *
     * @param handle  sandbox handle
     * @param command command to execute
     * @return execution output
     */
    public SandboxOutput execute(SandboxHandle handle, String command) {
        log.info("Executing in sandbox {}: {}", handle.getId(), command);
        // In production, this would use Docker exec:
        // docker exec <container> sh -c "<command>"
        String output = String.format("[sandbox:%s] Executed: %s", handle.getId(), command);
        return new SandboxOutput(output, 0, true);
    }

    /**
     * Destroy a sandbox and clean up resources.
     *
     * @param handle sandbox handle to destroy
     */
    public void destroySandbox(SandboxHandle handle) {
        activeSandboxes.remove(handle.getId());
        log.info("Sandbox destroyed: {}", handle.getId());
        // In production: docker rm -f <container>
    }

    /**
     * List all active sandbox handles.
     */
    public List<SandboxHandle> listActive() {
        return List.copyOf(activeSandboxes.values());
    }

    // ── Inner types ──────────────────────────────────────────────────────

    /**
     * Handle to a running sandbox.
     */
    public static class SandboxHandle {
        private final String id;
        private final String containerName;
        private final SandboxConfig config;
        private Path statePath;

        SandboxHandle(String id, String containerName, SandboxConfig config) {
            this.id = id;
            this.containerName = containerName;
            this.config = config;
        }

        public String getId() { return id; }
        public String getContainerName() { return containerName; }
        public SandboxConfig getConfig() { return config; }
        public Path getStatePath() { return statePath; }
        void setStatePath(Path statePath) { this.statePath = statePath; }
    }

    /**
     * Sandbox configuration.
     */
    public record SandboxConfig(
            String image,
            int memoryMb,
            int cpuCores,
            int timeoutSeconds,
            Map<String, String> envVars
    ) {
        public SandboxConfig {
            if (image == null) image = "ubuntu:22.04";
            if (memoryMb <= 0) memoryMb = 512;
            if (cpuCores <= 0) cpuCores = 1;
            if (timeoutSeconds <= 0) timeoutSeconds = 300;
            if (envVars == null) envVars = Map.of();
        }
    }

    /**
     * Sandbox execution output.
     */
    public record SandboxOutput(String stdout, int exitCode, boolean success) {
    }
}
