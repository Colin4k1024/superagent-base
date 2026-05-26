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

// shell_exec.go implements the shell_execute built-in tool, inspired by
// opencode's shell tool. It is more general than code_execute because it
// supports an arbitrary working directory, custom environment variables, and
// any shell command — not only scripting-language code blocks.
package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// Compile-time interface assertion.
var _ tool.InvokableTool = (*ShellExecuteTool)(nil)

const shellDefaultTimeoutSeconds = 30

// ShellExecuteTool runs an arbitrary shell command via `sh -c`.
type ShellExecuteTool struct{}

func newShellExecuteTool() *ShellExecuteTool { return &ShellExecuteTool{} }

type shellExecuteParams struct {
	Command        string            `json:"command"`
	Cwd            string            `json:"cwd,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
}

type shellExecuteResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	TimedOut bool   `json:"timed_out"`
}

func (t *ShellExecuteTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "shell_execute",
		Desc: "Execute a shell command via `sh -c`. Supports custom working directory, environment variables, and timeout. Returns stdout, stderr, and exit code.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command":         {Type: schema.String, Desc: "The shell command to execute.", Required: true},
			"cwd":             {Type: schema.String, Desc: "Working directory for the command. Defaults to the current process directory."},
			"env":             {Type: schema.Object, Desc: "Additional environment variables as key-value pairs, e.g. {\"KEY\": \"VALUE\"}."},
			"timeout_seconds": {Type: schema.Integer, Desc: fmt.Sprintf("Maximum execution time in seconds. Defaults to %d.", shellDefaultTimeoutSeconds)},
		}),
	}, nil
}

func (t *ShellExecuteTool) InvokableRun(_ context.Context, args string, _ ...tool.Option) (string, error) {
	var p shellExecuteParams
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", fmt.Errorf("shell_execute: invalid params: %w", err)
	}
	if p.Command == "" {
		return "", fmt.Errorf("shell_execute: command is required")
	}

	timeout := time.Duration(p.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = shellDefaultTimeoutSeconds * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", p.Command)

	if p.Cwd != "" {
		cmd.Dir = p.Cwd
	}

	// Inherit current process environment, then overlay user-supplied vars.
	if len(p.Env) > 0 {
		envPairs := cmd.Environ()
		for k, v := range p.Env {
			envPairs = append(envPairs, k+"="+v)
		}
		cmd.Env = envPairs
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := shellExecuteResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
		TimedOut: false,
	}

	if ctx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		result.ExitCode = -1
	} else if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			return "", fmt.Errorf("shell_execute: run command: %w", err)
		}
	}

	out, _ := json.Marshal(result)
	return string(out), nil
}
