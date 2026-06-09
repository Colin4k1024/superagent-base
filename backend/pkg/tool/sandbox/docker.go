/*
 * Copyright 2025 superagent-ai Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// DockerBackend executes tool invocations inside Docker containers.
// It provides container-level isolation with network, filesystem, and memory constraints.
type DockerBackend struct {
	defaultPolicy *Policy
	image         string
	available     bool
}

// NewDockerBackend creates a Docker-based sandbox backend.
func NewDockerBackend(defaultPolicy *Policy) *DockerBackend {
	image := "python:3.11-slim"
	return &DockerBackend{
		defaultPolicy: defaultPolicy,
		image:         image,
	}
}

func (d *DockerBackend) Init(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker sandbox: docker not available: %w", err)
	}
	d.available = true
	return nil
}

func (d *DockerBackend) Execute(ctx context.Context, req *ExecRequest) (*ExecResult, error) {
	if !d.available {
		return nil, fmt.Errorf("docker sandbox: not initialized")
	}

	policy := d.mergePolicy(req.Policy)
	timeout := time.Duration(policy.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build docker run arguments with security constraints.
	args := d.buildDockerArgs(policy)

	// The container receives a JSON payload on stdin and writes its result to stdout.
	// The entrypoint script reads stdin, executes the tool, and writes JSON output.
	entrypoint := `
import sys, json
payload = json.loads(sys.stdin.read())
tool_name = payload.get("tool", "")
tool_args = payload.get("args", "")
# For code_execute, extract and run the code from args
try:
    args_obj = json.loads(tool_args)
    code = args_obj.get("code", "")
    if code:
        import subprocess
        result = subprocess.run(
            [sys.executable, "-c", code],
            capture_output=True, text=True,
            timeout=` + fmt.Sprintf("%d", policy.TimeoutSeconds) + `
        )
        output = json.dumps({
            "stdout": result.stdout,
            "stderr": result.stderr,
            "exit_code": result.returncode
        })
        print(output)
    else:
        print(json.dumps({"error": "no code provided"}))
except Exception as e:
    print(json.dumps({"error": str(e)}))
`
	args = append(args, d.image, "python3", "-c", entrypoint)

	payload, _ := json.Marshal(map[string]string{
		"tool": req.ToolName,
		"args": req.Args,
	})

	cmd := exec.CommandContext(execCtx, "docker", args...)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if execCtx.Err() != nil {
			return &ExecResult{Error: fmt.Sprintf("sandbox timeout after %s", timeout)}, nil
		}
		return &ExecResult{
			Output: stdout.String(),
			Error:  fmt.Sprintf("container error: %s, stderr: %s", err.Error(), stderr.String()),
		}, nil
	}

	return &ExecResult{Output: stdout.String()}, nil
}

func (d *DockerBackend) Cleanup(_ context.Context) error {
	return nil
}

// buildDockerArgs constructs docker run flags from the policy.
func (d *DockerBackend) buildDockerArgs(policy *Policy) []string {
	args := []string{
		"run", "--rm", "-i",
		"--network=none",
		"--read-only",
		"--pids-limit=64",
		"--cpu-period=100000",
		"--cpu-quota=50000",
	}

	if policy.MemoryLimitMB > 0 {
		args = append(args, fmt.Sprintf("--memory=%dm", policy.MemoryLimitMB))
		args = append(args, fmt.Sprintf("--memory-swap=%dm", policy.MemoryLimitMB))
	}

	if len(policy.AllowNet) > 0 {
		// Replace --network=none with bridge when network access is allowed.
		args[3] = "--network=bridge"
	}

	for _, path := range policy.AllowWrite {
		args = append(args, fmt.Sprintf("--tmpfs=%s:rw,size=64m", path))
	}
	for _, path := range policy.AllowRead {
		args = append(args, fmt.Sprintf("-v=%s:%s:ro", path, path))
	}
	for _, env := range policy.AllowEnv {
		args = append(args, fmt.Sprintf("-e=%s", env))
	}

	return args
}

func (d *DockerBackend) mergePolicy(override *Policy) *Policy {
	if override != nil {
		return override
	}
	if d.defaultPolicy != nil {
		return d.defaultPolicy
	}
	return &Policy{TimeoutSeconds: 30, MemoryLimitMB: 256}
}
