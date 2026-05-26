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

// Package builtin provides built-in tools for superagent agents.
// file_ops.go implements file system operations inspired by opencode's
// read/write/edit/glob tools, with path-traversal protection.
package builtin

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// Compile-time interface assertions.
var _ tool.InvokableTool = (*FileReadTool)(nil)
var _ tool.InvokableTool = (*FileWriteTool)(nil)
var _ tool.InvokableTool = (*FileEditTool)(nil)
var _ tool.InvokableTool = (*FileGlobTool)(nil)

const (
	fileReadMaxBytes  = 1 * 1024 * 1024 // 1 MB read cap
	fileGlobMaxResult = 500
)

// FileOpsConfig is shared configuration for file operation tools.
type FileOpsConfig struct {
	// WorkspaceDir is the sandbox root. Empty means os.Getwd().
	WorkspaceDir string
	// AllowWrite controls whether write tools (file_write, file_edit) actually
	// persist changes. When false they return a permission error.
	AllowWrite bool
}

// resolveWorkspace returns the absolute workspace root.
func resolveWorkspace(cfg FileOpsConfig) (string, error) {
	root := cfg.WorkspaceDir
	if root == "" {
		var err error
		root, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("abs workspace: %w", err)
	}
	return abs, nil
}

// safeJoinPath resolves userPath relative to root and ensures the result stays
// within root (prevents path-traversal attacks).
func safeJoinPath(root, userPath string) (string, error) {
	var candidate string
	if filepath.IsAbs(userPath) {
		candidate = filepath.Clean(userPath)
	} else {
		candidate = filepath.Clean(filepath.Join(root, userPath))
	}
	if !strings.HasPrefix(candidate, root+string(filepath.Separator)) && candidate != root {
		return "", fmt.Errorf("path %q is outside workspace root %q", userPath, root)
	}
	return candidate, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// FileReadTool — builtin/file_read
// ─────────────────────────────────────────────────────────────────────────────

// FileReadTool reads a file's contents, optionally restricted to a line range.
type FileReadTool struct{ cfg FileOpsConfig }

func newFileReadTool(cfg FileOpsConfig) *FileReadTool { return &FileReadTool{cfg: cfg} }

type fileReadParams struct {
	Path   string `json:"path"`
	Offset int    `json:"offset,omitempty"` // 1-based start line; 0 = beginning
	Limit  int    `json:"limit,omitempty"`  // max lines to return; 0 = all
}

type fileReadResult struct {
	Content   string `json:"content"`
	Lines     int    `json:"lines"`
	Truncated bool   `json:"truncated"`
}

func (t *FileReadTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "file_read",
		Desc: "Read the contents of a file. Supports optional line-range selection via offset and limit. Returns the content as a string along with the total line count.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path":   {Type: schema.String, Desc: "Path to the file (absolute or relative to workspace root).", Required: true},
			"offset": {Type: schema.Integer, Desc: "1-based line number to start reading from. Defaults to 1 (beginning of file)."},
			"limit":  {Type: schema.Integer, Desc: "Maximum number of lines to return. Defaults to 0 (all lines)."},
		}),
	}, nil
}

func (t *FileReadTool) InvokableRun(_ context.Context, args string, _ ...tool.Option) (string, error) {
	var p fileReadParams
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", fmt.Errorf("file_read: invalid params: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("file_read: path is required")
	}

	root, err := resolveWorkspace(t.cfg)
	if err != nil {
		return "", err
	}
	abs, err := safeJoinPath(root, p.Path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("file_read: %w", err)
	}

	truncated := false
	if len(data) > fileReadMaxBytes {
		data = data[:fileReadMaxBytes]
		truncated = true
	}

	// Split into lines for offset/limit handling.
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var allLines []string
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	total := len(allLines)

	start := 0
	if p.Offset > 1 {
		start = p.Offset - 1
	}
	if start >= total {
		start = total
	}
	end := total
	if p.Limit > 0 && start+p.Limit < end {
		end = start + p.Limit
	}
	selected := allLines[start:end]

	res := fileReadResult{
		Content:   strings.Join(selected, "\n"),
		Lines:     total,
		Truncated: truncated,
	}
	out, _ := json.Marshal(res)
	return string(out), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// FileWriteTool — builtin/file_write
// ─────────────────────────────────────────────────────────────────────────────

// FileWriteTool creates or overwrites a file with the provided content.
type FileWriteTool struct{ cfg FileOpsConfig }

func newFileWriteTool(cfg FileOpsConfig) *FileWriteTool { return &FileWriteTool{cfg: cfg} }

type fileWriteParams struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type fileWriteResult struct {
	Success bool   `json:"success"`
	Path    string `json:"path"`
}

func (t *FileWriteTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "file_write",
		Desc: "Create or overwrite a file with the given content. Parent directories are created automatically.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path":    {Type: schema.String, Desc: "Destination path (absolute or relative to workspace root).", Required: true},
			"content": {Type: schema.String, Desc: "Text content to write to the file.", Required: true},
		}),
	}, nil
}

func (t *FileWriteTool) InvokableRun(_ context.Context, args string, _ ...tool.Option) (string, error) {
	var p fileWriteParams
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", fmt.Errorf("file_write: invalid params: %w", err)
	}
	if p.Path == "" {
		return "", fmt.Errorf("file_write: path is required")
	}
	if !t.cfg.AllowWrite {
		return "", fmt.Errorf("file_write: write operations are disabled for this agent")
	}

	root, err := resolveWorkspace(t.cfg)
	if err != nil {
		return "", err
	}
	abs, err := safeJoinPath(root, p.Path)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return "", fmt.Errorf("file_write: create parent dirs: %w", err)
	}
	if err := os.WriteFile(abs, []byte(p.Content), 0644); err != nil {
		return "", fmt.Errorf("file_write: %w", err)
	}

	out, _ := json.Marshal(fileWriteResult{Success: true, Path: abs})
	return string(out), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// FileEditTool — builtin/file_edit
// ─────────────────────────────────────────────────────────────────────────────

// FileEditTool performs an exact string replacement within a file.
type FileEditTool struct{ cfg FileOpsConfig }

func newFileEditTool(cfg FileOpsConfig) *FileEditTool { return &FileEditTool{cfg: cfg} }

type fileEditParams struct {
	Path      string `json:"path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

func (t *FileEditTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "file_edit",
		Desc: "Replace an exact string in a file. The old_string must appear exactly once in the file; if it does not exist or appears multiple times the operation fails.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path":       {Type: schema.String, Desc: "Path to the file to edit.", Required: true},
			"old_string": {Type: schema.String, Desc: "The exact text to find and replace.", Required: true},
			"new_string": {Type: schema.String, Desc: "The replacement text.", Required: true},
		}),
	}, nil
}

func (t *FileEditTool) InvokableRun(_ context.Context, args string, _ ...tool.Option) (string, error) {
	var p fileEditParams
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", fmt.Errorf("file_edit: invalid params: %w", err)
	}
	if p.Path == "" || p.OldString == "" {
		return "", fmt.Errorf("file_edit: path and old_string are required")
	}
	if !t.cfg.AllowWrite {
		return "", fmt.Errorf("file_edit: write operations are disabled for this agent")
	}

	root, err := resolveWorkspace(t.cfg)
	if err != nil {
		return "", err
	}
	abs, err := safeJoinPath(root, p.Path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return "", fmt.Errorf("file_edit: read file: %w", err)
	}

	content := string(data)
	count := strings.Count(content, p.OldString)
	if count == 0 {
		return "", fmt.Errorf("file_edit: old_string not found in %q", p.Path)
	}
	if count > 1 {
		return "", fmt.Errorf("file_edit: old_string appears %d times in %q; provide more context to make it unique", count, p.Path)
	}

	updated := strings.Replace(content, p.OldString, p.NewString, 1)
	if err := os.WriteFile(abs, []byte(updated), 0644); err != nil {
		return "", fmt.Errorf("file_edit: write file: %w", err)
	}

	out, _ := json.Marshal(fileWriteResult{Success: true, Path: abs})
	return string(out), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// FileGlobTool — builtin/file_glob
// ─────────────────────────────────────────────────────────────────────────────

// FileGlobTool finds files matching a glob pattern.
type FileGlobTool struct{ cfg FileOpsConfig }

func newFileGlobTool(cfg FileOpsConfig) *FileGlobTool { return &FileGlobTool{cfg: cfg} }

type fileGlobParams struct {
	Pattern string `json:"pattern"`
	BaseDir string `json:"base_dir,omitempty"`
}

type fileGlobResult struct {
	Matches []string `json:"matches"`
	Count   int      `json:"count"`
}

func (t *FileGlobTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "file_glob",
		Desc: "Find files matching a glob pattern. Returns a list of matching paths relative to the workspace root.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"pattern":  {Type: schema.String, Desc: "Glob pattern, e.g. '**/*.go' or 'src/*.ts'.", Required: true},
			"base_dir": {Type: schema.String, Desc: "Base directory to search from; defaults to workspace root."},
		}),
	}, nil
}

func (t *FileGlobTool) InvokableRun(_ context.Context, args string, _ ...tool.Option) (string, error) {
	var p fileGlobParams
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", fmt.Errorf("file_glob: invalid params: %w", err)
	}
	if p.Pattern == "" {
		return "", fmt.Errorf("file_glob: pattern is required")
	}

	root, err := resolveWorkspace(t.cfg)
	if err != nil {
		return "", err
	}

	baseDir := root
	if p.BaseDir != "" {
		baseDir, err = safeJoinPath(root, p.BaseDir)
		if err != nil {
			return "", err
		}
	}

	var matches []string
	err = filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			return nil
		}
		// Match only the path relative to baseDir so patterns like **/*.go work.
		rel, _ := filepath.Rel(baseDir, path)
		matched, merr := filepath.Match(p.Pattern, rel)
		if merr != nil {
			return merr
		}
		// Also try matching against just the filename for simple patterns.
		if !matched {
			matched, _ = filepath.Match(p.Pattern, d.Name())
		}
		if matched {
			matches = append(matches, rel)
		}
		if len(matches) >= fileGlobMaxResult {
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("file_glob: walk: %w", err)
	}

	out, _ := json.Marshal(fileGlobResult{Matches: matches, Count: len(matches)})
	return string(out), nil
}
