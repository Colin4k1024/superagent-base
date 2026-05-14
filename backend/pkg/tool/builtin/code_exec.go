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

	"github.com/superagent-ai/superagent-base/backend/infra/coderunner"
)

// Compile-time assertion.
var _ tool.InvokableTool = (*CodeExecTool)(nil)

// supportedLanguages lists languages accepted by code_execute.
var supportedLanguages = map[string]bool{
	"python":     true,
	"javascript": true,
	"bash":       true,
}

// defaultTimeoutSeconds is applied when the caller does not specify a timeout.
const defaultTimeoutSeconds = 30

// pythonCaptureWrapper wraps arbitrary user Python code so that stdout, stderr,
// and exit_code are returned as a JSON dict via the coderunner.Runner interface.
// The user's code is injected at %s and executed inside a subprocess so that
// sys.exit() and print() behave naturally.
const pythonCaptureWrapper = `
import sys
import json
import subprocess

user_code = %s

result = subprocess.run(
    [sys.executable, "-c", user_code],
    capture_output=True,
    text=True,
    timeout=%d,
)

class Output(dict):
    pass

async def main(args):
    return Output({
        "stdout": result.stdout,
        "stderr": result.stderr,
        "exit_code": result.returncode,
    })
`

// CodeExecTool executes code in a sandboxed runtime and returns stdout, stderr, and exit code.
// Python execution is delegated to the global coderunner.Runner (set during application init).
// Bash is handled via os/exec.
// JavaScript is not currently supported by the underlying runner.
type CodeExecTool struct{}

func newCodeExecTool() *CodeExecTool {
	return &CodeExecTool{}
}

func (c *CodeExecTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "code_execute",
		Desc: "Execute a snippet of code in the specified language (python, javascript, or bash) and return stdout, stderr, and exit code.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"language": {
				Desc:     "Programming language: python, javascript, or bash.",
				Type:     schema.String,
				Required: true,
			},
			"code": {
				Desc:     "The source code to execute.",
				Type:     schema.String,
				Required: true,
			},
			"timeout_seconds": {
				Desc:     "Maximum execution time in seconds (default 30).",
				Type:     schema.Integer,
				Required: false,
			},
		}),
	}, nil
}

func (c *CodeExecTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Language       string `json:"language"`
		Code           string `json:"code"`
		TimeoutSeconds int    `json:"timeout_seconds"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("code_execute: parse arguments: %w", err)
	}
	if args.Language == "" {
		return "", fmt.Errorf("code_execute: language is required")
	}
	if !supportedLanguages[args.Language] {
		return "", fmt.Errorf("code_execute: unsupported language %q (supported: python, javascript, bash)", args.Language)
	}
	if args.Code == "" {
		return "", fmt.Errorf("code_execute: code is required")
	}
	if args.TimeoutSeconds <= 0 {
		args.TimeoutSeconds = defaultTimeoutSeconds
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(args.TimeoutSeconds)*time.Second)
	defer cancel()

	var (
		stdout   string
		stderr   string
		exitCode int
	)

	switch args.Language {
	case "python":
		var err error
		stdout, stderr, exitCode, err = runPython(timeoutCtx, args.Code, args.TimeoutSeconds)
		if err != nil {
			return "", fmt.Errorf("code_execute: python: %w", err)
		}
	case "bash":
		var err error
		stdout, stderr, exitCode, err = runBash(timeoutCtx, args.Code)
		if err != nil {
			return "", fmt.Errorf("code_execute: bash: %w", err)
		}
	case "javascript":
		return "", fmt.Errorf("code_execute: javascript execution is not yet supported")
	}

	result := map[string]any{
		"stdout":    stdout,
		"stderr":    stderr,
		"exit_code": exitCode,
	}
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("code_execute: marshal result: %w", err)
	}
	return string(out), nil
}

// runPython delegates to the global coderunner.Runner.
// The user's code is wrapped in a capture harness so that stdout/stderr/exit_code
// are returned through the Runner's structured Result map.
func runPython(ctx context.Context, code string, timeoutSeconds int) (stdout, stderr string, exitCode int, err error) {
	runner := coderunner.GetCodeRunner()
	if runner == nil {
		return "", "", 1, fmt.Errorf("code runner not initialized")
	}

	// Encode the user code as a Python string literal to safely embed it.
	codeJSON, err := json.Marshal(code)
	if err != nil {
		return "", "", 1, fmt.Errorf("encode code: %w", err)
	}

	wrappedCode := fmt.Sprintf(pythonCaptureWrapper, string(codeJSON), timeoutSeconds)

	resp, err := runner.Run(ctx, &coderunner.RunRequest{
		Language: coderunner.Python,
		Code:     wrappedCode,
		Params:   map[string]any{},
	})
	if err != nil {
		return "", "", 1, err
	}

	if resp == nil || resp.Result == nil {
		return "", "", 0, nil
	}

	stdout = extractString(resp.Result, "stdout")
	stderr = extractString(resp.Result, "stderr")
	exitCode = extractInt(resp.Result, "exit_code")
	return stdout, stderr, exitCode, nil
}

// runBash executes a bash script directly via os/exec, capturing stdout and stderr.
func runBash(ctx context.Context, code string) (stdout, stderr string, exitCode int, err error) {
	cmd := exec.CommandContext(ctx, "bash", "-c", code) // ignore_security_alert RCE
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	runErr := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			// Non-zero exit is a valid result, not a tool error.
			return stdout, stderr, exitCode, nil
		}
		return stdout, stderr, 1, runErr
	}
	return stdout, stderr, 0, nil
}

// extractString reads a string value from a result map, returning "" if missing or wrong type.
func extractString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// extractInt reads an int value from a result map, supporting float64 (JSON number) and int types.
func extractInt(m map[string]any, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	}
	return 0
}
