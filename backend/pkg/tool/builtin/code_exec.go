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
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// Compile-time assertion.
var _ tool.InvokableTool = (*CodeExecTool)(nil)

// supportedLanguages lists languages accepted by code_execute.
var supportedLanguages = map[string]bool{
	"python":     true,
	"javascript": true,
	"bash":       true,
}

// CodeExecTool executes code in a sandboxed runtime and returns stdout, stderr, and exit code.
// The current implementation is a stub; a real backend (e.g. a code-runner sidecar) can be
// wired in by replacing ExecFunc.
type CodeExecTool struct {
	// ExecFunc is the actual code execution backend. Override for real implementations.
	ExecFunc func(ctx context.Context, language, code string) (stdout, stderr string, exitCode int, err error)
}

func newCodeExecTool() *CodeExecTool {
	return &CodeExecTool{ExecFunc: stubExec}
}

// stubExec is the placeholder implementation.
func stubExec(_ context.Context, language, code string) (string, string, int, error) {
	stdout := fmt.Sprintf("[stub] would execute %s code:\n%s", language, code)
	return stdout, "", 0, nil
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
		}),
	}, nil
}

func (c *CodeExecTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Language string `json:"language"`
		Code     string `json:"code"`
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

	stdout, stderr, exitCode, err := c.ExecFunc(ctx, args.Language, args.Code)
	if err != nil {
		return "", fmt.Errorf("code_execute: %w", err)
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
