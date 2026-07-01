package io.superagent.tools.builtin;

import io.superagent.tools.Tool;
import org.springframework.stereotype.Component;

import java.io.*;
import java.nio.file.*;
import java.util.*;
import java.util.concurrent.TimeUnit;

/**
 * Code execution tool via subprocess isolation.
 *
 * <p>URI: {@code builtin/code_execute}</p>
 *
 * <p>Supports python3 and node. Writes code to a temp file,
 * runs it as a subprocess, captures stdout/stderr, and cleans up.</p>
 *
 * <p><b>Security note:</b> Runs arbitrary code — use only in trusted/sandboxed environments.</p>
 */
@Component
public class CodeExecuteTool implements Tool {

    @Override
    public String getName() {
        return "code_execute";
    }

    @Override
    public String getDescription() {
        return "Execute code in a sandboxed environment. "
             + "Supports Python and JavaScript. Returns stdout, stderr, and exit code.";
    }

    @Override
    public Map<String, Object> getParameterSchema() {
        return Map.of(
            "type", "object",
            "properties", Map.of(
                "language", Map.of(
                    "type", "string",
                    "description", "Programming language (python, javascript)"
                ),
                "code", Map.of(
                    "type", "string",
                    "description", "The code to execute"
                ),
                "timeout", Map.of(
                    "type", "integer",
                    "description", "Execution timeout in seconds (default: 30)"
                )
            ),
            "required", List.of("language", "code")
        );
    }

    @Override
    public Map<String, Object> execute(Map<String, Object> parameters) {
        String language = (String) parameters.getOrDefault("language", "python");
        String code     = (String) parameters.getOrDefault("code", "");
        int timeout     = parameters.get("timeout") instanceof Number n ? n.intValue() : 30;

        String ext = language.startsWith("python") ? ".py" : ".js";
        String cmd = language.startsWith("python") ? "python3" : "node";

        Path tmpFile = null;
        try {
            tmpFile = Files.createTempFile("sa_exec_", ext);
            Files.writeString(tmpFile, code);

            Process proc = new ProcessBuilder(cmd, tmpFile.toString())
                .redirectErrorStream(false)
                .start();

            boolean finished = proc.waitFor(timeout, TimeUnit.SECONDS);
            if (!finished) {
                proc.destroyForcibly();
                return Map.of(
                    "tool",      getName(),
                    "status",    "timeout",
                    "language",  language,
                    "exit_code", -1,
                    "stdout",    "",
                    "stderr",    "Execution timed out after " + timeout + "s"
                );
            }

            String stdout = new String(proc.getInputStream().readAllBytes());
            String stderr = new String(proc.getErrorStream().readAllBytes());

            return Map.of(
                "tool",      getName(),
                "status",    proc.exitValue() == 0 ? "success" : "error",
                "language",  language,
                "exit_code", proc.exitValue(),
                "stdout",    stdout,
                "stderr",    stderr
            );
        } catch (Exception e) {
            return Map.of(
                "tool",      getName(),
                "status",    "error",
                "language",  language,
                "exit_code", -1,
                "stdout",    "",
                "stderr",    e.getMessage() != null ? e.getMessage() : e.getClass().getSimpleName()
            );
        } finally {
            if (tmpFile != null) {
                try { Files.deleteIfExists(tmpFile); } catch (IOException ignored) {}
            }
        }
    }
}
