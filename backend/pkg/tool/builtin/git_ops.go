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

// git_ops.go implements three read-only Git built-in tools inspired by
// opencode's git integration:
//
//   - git_status  — working tree status
//   - git_diff    — staged or unstaged diff
//   - git_log     — recent commit history
//
// No write operations (commit, push, reset) are exposed to prevent accidents.
package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// Compile-time interface assertions.
var _ tool.InvokableTool = (*GitStatusTool)(nil)
var _ tool.InvokableTool = (*GitDiffTool)(nil)
var _ tool.InvokableTool = (*GitLogTool)(nil)

const gitDefaultLogLimit = 20

// gitResult is the common output envelope for all git tools.
type gitResult struct {
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
}

// runGit executes a git sub-command and returns its combined output.
func runGit(cwd string, gitArgs ...string) (gitResult, error) {
	cmd := exec.Command("git", gitArgs...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return gitResult{}, fmt.Errorf("git %s: %w", gitArgs[0], err)
		}
	}

	output := stdout.String()
	if output == "" && stderr.Len() > 0 {
		output = stderr.String()
	}
	return gitResult{Output: output, ExitCode: exitCode}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// GitStatusTool — builtin/git_status
// ─────────────────────────────────────────────────────────────────────────────

// GitStatusTool reports the working tree status.
type GitStatusTool struct{}

func newGitStatusTool() *GitStatusTool { return &GitStatusTool{} }

type gitStatusParams struct {
	Cwd string `json:"cwd,omitempty"`
}

func (t *GitStatusTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "git_status",
		Desc: "Show the working tree status (`git status --short`). Returns modified, added, deleted, and untracked files.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"cwd": {Type: schema.String, Desc: "Repository root directory. Defaults to current working directory."},
		}),
	}, nil
}

func (t *GitStatusTool) InvokableRun(_ context.Context, args string, _ ...tool.Option) (string, error) {
	var p gitStatusParams
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", fmt.Errorf("git_status: invalid params: %w", err)
	}
	res, err := runGit(p.Cwd, "status", "--short")
	if err != nil {
		return "", err
	}
	out, _ := json.Marshal(res)
	return string(out), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// GitDiffTool — builtin/git_diff
// ─────────────────────────────────────────────────────────────────────────────

// GitDiffTool shows changes between the working tree and the index (or HEAD).
type GitDiffTool struct{}

func newGitDiffTool() *GitDiffTool { return &GitDiffTool{} }

type gitDiffParams struct {
	Cwd    string `json:"cwd,omitempty"`
	Staged bool   `json:"staged,omitempty"` // true = --cached (staged diff)
	File   string `json:"file,omitempty"`   // limit diff to a specific file
}

func (t *GitDiffTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "git_diff",
		Desc: "Show changes between the working tree and the index, or between the index and HEAD when staged=true. Optionally limit to a specific file.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"cwd":    {Type: schema.String, Desc: "Repository root directory. Defaults to current working directory."},
			"staged": {Type: schema.Boolean, Desc: "When true, show staged changes (git diff --cached). Defaults to false (unstaged changes)."},
			"file":   {Type: schema.String, Desc: "Limit diff output to this file path."},
		}),
	}, nil
}

func (t *GitDiffTool) InvokableRun(_ context.Context, args string, _ ...tool.Option) (string, error) {
	var p gitDiffParams
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", fmt.Errorf("git_diff: invalid params: %w", err)
	}

	gitArgs := []string{"diff"}
	if p.Staged {
		gitArgs = append(gitArgs, "--cached")
	}
	if p.File != "" {
		gitArgs = append(gitArgs, "--", p.File)
	}

	res, err := runGit(p.Cwd, gitArgs...)
	if err != nil {
		return "", err
	}
	out, _ := json.Marshal(res)
	return string(out), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// GitLogTool — builtin/git_log
// ─────────────────────────────────────────────────────────────────────────────

// GitLogTool returns the recent commit history.
type GitLogTool struct{}

func newGitLogTool() *GitLogTool { return &GitLogTool{} }

type gitLogParams struct {
	Cwd   string `json:"cwd,omitempty"`
	Limit int    `json:"limit,omitempty"` // default 20
}

func (t *GitLogTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "git_log",
		Desc: fmt.Sprintf("Show recent commit history. Returns a formatted log with hash, author, date, and subject. Defaults to the last %d commits.", gitDefaultLogLimit),
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"cwd":   {Type: schema.String, Desc: "Repository root directory. Defaults to current working directory."},
			"limit": {Type: schema.Integer, Desc: fmt.Sprintf("Number of commits to show. Defaults to %d.", gitDefaultLogLimit)},
		}),
	}, nil
}

func (t *GitLogTool) InvokableRun(_ context.Context, args string, _ ...tool.Option) (string, error) {
	var p gitLogParams
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", fmt.Errorf("git_log: invalid params: %w", err)
	}
	limit := p.Limit
	if limit <= 0 {
		limit = gitDefaultLogLimit
	}

	res, err := runGit(p.Cwd,
		"log",
		fmt.Sprintf("-n%d", limit),
		"--pretty=format:%h %an <%ae> %ad %s",
		"--date=short",
	)
	if err != nil {
		return "", err
	}
	out, _ := json.Marshal(res)
	return string(out), nil
}
