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
	"os"
	"os/exec"
	"time"
)

// ProcessBackend executes tool invocations in isolated OS processes
// with resource constraints. This is the lightweight fallback when
// Docker is unavailable.
type ProcessBackend struct {
	defaultPolicy *Policy
}

// NewProcessBackend creates a process-based sandbox backend.
func NewProcessBackend(defaultPolicy *Policy) *ProcessBackend {
	return &ProcessBackend{defaultPolicy: defaultPolicy}
}

func (p *ProcessBackend) Init(_ context.Context) error {
	return nil
}

func (p *ProcessBackend) Execute(ctx context.Context, req *ExecRequest) (*ExecResult, error) {
	policy := p.mergePolicy(req.Policy)
	timeout := time.Duration(policy.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Parse tool args to extract code for execution.
	var toolArgs struct {
		Language string `json:"language"`
		Code     string `json:"code"`
	}
	if err := json.Unmarshal([]byte(req.Args), &toolArgs); err != nil {
		return &ExecResult{Error: fmt.Sprintf("failed to parse tool args: %s", err.Error())}, nil
	}

	if toolArgs.Code == "" {
		return &ExecResult{Error: "no code provided for sandbox execution"}, nil
	}

	// Execute code in a subprocess with restricted environment.
	var cmd *exec.Cmd
	switch toolArgs.Language {
	case "python", "":
		cmd = exec.CommandContext(execCtx, "python3", "-c", toolArgs.Code)
	case "bash":
		cmd = exec.CommandContext(execCtx, "bash", "-c", toolArgs.Code) // ignore_security_alert RCE
	default:
		return &ExecResult{Error: fmt.Sprintf("unsupported language: %s", toolArgs.Language)}, nil
	}

	// Restrict environment variables.
	cmd.Env = buildRestrictedEnv(policy.AllowEnv)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	if execCtx.Err() != nil {
		return &ExecResult{Error: fmt.Sprintf("sandbox timeout after %s", timeout)}, nil
	}

	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return &ExecResult{Error: fmt.Sprintf("process error: %s", runErr.Error())}, nil
		}
	}

	result := map[string]any{
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"exit_code": exitCode,
	}
	output, _ := json.Marshal(result)
	return &ExecResult{Output: string(output)}, nil
}

func (p *ProcessBackend) Cleanup(_ context.Context) error {
	return nil
}

func (p *ProcessBackend) mergePolicy(override *Policy) *Policy {
	if override != nil {
		return override
	}
	if p.defaultPolicy != nil {
		return p.defaultPolicy
	}
	return &Policy{TimeoutSeconds: 30, MemoryLimitMB: 256}
}

// buildRestrictedEnv constructs an environment slice containing only
// the allowed variable names from the current process environment.
func buildRestrictedEnv(allowEnv []string) []string {
	env := []string{"PATH=/usr/bin:/bin:/usr/local/bin"}
	for _, name := range allowEnv {
		if val, ok := os.LookupEnv(name); ok {
			env = append(env, fmt.Sprintf("%s=%s", name, val))
		}
	}
	return env
}
