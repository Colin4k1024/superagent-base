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

// code_search.go implements three code-search built-in tools inspired by
// opencode's grep / find / repo_overview tools:
//
//   - grep_search  — regex content search (uses rg if available, else stdlib)
//   - find_files   — find files by name or extension
//   - repo_overview — directory tree + file-count summary
package builtin

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// Compile-time interface assertions.
var _ tool.InvokableTool = (*GrepSearchTool)(nil)
var _ tool.InvokableTool = (*FindFilesTool)(nil)
var _ tool.InvokableTool = (*RepoOverviewTool)(nil)

const (
	grepDefaultMaxResults     = 50
	findFilesDefaultMaxResult = 200
	repoOverviewDefaultDepth  = 4
	repoOverviewMaxFiles      = 2000
)

// ─────────────────────────────────────────────────────────────────────────────
// GrepSearchTool — builtin/grep_search
// ─────────────────────────────────────────────────────────────────────────────

// GrepSearchTool searches file contents using a regular expression.
// It tries ripgrep (rg) first and falls back to a pure-Go implementation.
type GrepSearchTool struct{}

func newGrepSearchTool() *GrepSearchTool { return &GrepSearchTool{} }

type grepSearchParams struct {
	Pattern    string `json:"pattern"`
	Path       string `json:"path,omitempty"`
	Include    string `json:"include,omitempty"`     // glob filter, e.g. "*.go"
	MaxResults int    `json:"max_results,omitempty"` // default 50
}

type grepMatch struct {
	File          string `json:"file"`
	Line          int    `json:"line"`
	Content       string `json:"content"`
	ContextBefore string `json:"context_before,omitempty"`
	ContextAfter  string `json:"context_after,omitempty"`
}

type grepSearchResult struct {
	Matches  []grepMatch `json:"matches"`
	Count    int         `json:"count"`
	Truncated bool       `json:"truncated"`
}

func (t *GrepSearchTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "grep_search",
		Desc: "Search file contents using a regular expression. Returns matching lines with surrounding context. Uses ripgrep (rg) when available, otherwise falls back to a built-in Go implementation.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"pattern":     {Type: schema.String, Desc: "Regular expression pattern to search for.", Required: true},
			"path":        {Type: schema.String, Desc: "File or directory to search. Defaults to current directory."},
			"include":     {Type: schema.String, Desc: "Glob pattern to filter files, e.g. '*.go' or '*.{ts,tsx}'."},
			"max_results": {Type: schema.Integer, Desc: fmt.Sprintf("Maximum number of matches to return. Defaults to %d.", grepDefaultMaxResults)},
		}),
	}, nil
}

func (t *GrepSearchTool) InvokableRun(_ context.Context, args string, _ ...tool.Option) (string, error) {
	var p grepSearchParams
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", fmt.Errorf("grep_search: invalid params: %w", err)
	}
	if p.Pattern == "" {
		return "", fmt.Errorf("grep_search: pattern is required")
	}
	if p.MaxResults <= 0 {
		p.MaxResults = grepDefaultMaxResults
	}

	searchPath := p.Path
	if searchPath == "" {
		var err error
		searchPath, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("grep_search: get cwd: %w", err)
		}
	}

	var matches []grepMatch
	var err error

	// Prefer ripgrep for speed.
	if _, rerr := exec.LookPath("rg"); rerr == nil {
		matches, err = grepWithRipgrep(p.Pattern, searchPath, p.Include, p.MaxResults+1)
	} else {
		matches, err = grepWithStdlib(p.Pattern, searchPath, p.Include, p.MaxResults+1)
	}
	if err != nil {
		return "", fmt.Errorf("grep_search: %w", err)
	}

	truncated := false
	if len(matches) > p.MaxResults {
		matches = matches[:p.MaxResults]
		truncated = true
	}

	out, _ := json.Marshal(grepSearchResult{Matches: matches, Count: len(matches), Truncated: truncated})
	return string(out), nil
}

// grepWithRipgrep uses `rg --json` for fast search.
func grepWithRipgrep(pattern, path, include string, limit int) ([]grepMatch, error) {
	args := []string{"--json", "-n", "-C", "2", pattern, path}
	if include != "" {
		args = append(args, "--glob", include)
	}
	cmd := exec.Command("rg", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	// rg exits 1 when no matches — not an error we care about.
	_ = cmd.Run()

	var matches []grepMatch
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		if len(matches) >= limit {
			break
		}
		line := scanner.Bytes()
		var msg map[string]json.RawMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}
		typeRaw, ok := msg["type"]
		if !ok {
			continue
		}
		var msgType string
		_ = json.Unmarshal(typeRaw, &msgType)
		if msgType != "match" {
			continue
		}
		dataRaw, ok := msg["data"]
		if !ok {
			continue
		}
		var data struct {
			Path struct {
				Text string `json:"text"`
			} `json:"path"`
			Lines struct {
				Text string `json:"text"`
			} `json:"lines"`
			LineNumber uint64 `json:"line_number"`
		}
		if err := json.Unmarshal(dataRaw, &data); err != nil {
			continue
		}
		matches = append(matches, grepMatch{
			File:    data.Path.Text,
			Line:    int(data.LineNumber),
			Content: strings.TrimRight(data.Lines.Text, "\n"),
		})
	}
	return matches, nil
}

// grepWithStdlib walks files and matches line-by-line using regexp.
func grepWithStdlib(pattern, rootPath, include string, limit int) ([]grepMatch, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("compile pattern: %w", err)
	}

	var matches []grepMatch
	err = filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil || d.IsDir() {
			return nil
		}
		if include != "" {
			matched, _ := filepath.Match(include, d.Name())
			if !matched {
				return nil
			}
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		var lines []string
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			lines = append(lines, sc.Text())
		}

		for i, l := range lines {
			if re.MatchString(l) {
				m := grepMatch{
					File:    path,
					Line:    i + 1,
					Content: l,
				}
				if i > 0 {
					m.ContextBefore = lines[i-1]
				}
				if i < len(lines)-1 {
					m.ContextAfter = lines[i+1]
				}
				matches = append(matches, m)
				if len(matches) >= limit {
					return fs.SkipAll
				}
			}
		}
		return nil
	})
	return matches, err
}

// ─────────────────────────────────────────────────────────────────────────────
// FindFilesTool — builtin/find_files
// ─────────────────────────────────────────────────────────────────────────────

// FindFilesTool searches for files by name or extension.
type FindFilesTool struct{}

func newFindFilesTool() *FindFilesTool { return &FindFilesTool{} }

type findFilesParams struct {
	Name       string `json:"name,omitempty"`
	Extension  string `json:"extension,omitempty"`
	BaseDir    string `json:"base_dir,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
}

type findFilesResult struct {
	Files []string `json:"files"`
	Count int      `json:"count"`
}

func (t *FindFilesTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "find_files",
		Desc: "Find files by name substring or extension. At least one of 'name' or 'extension' is required.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"name":        {Type: schema.String, Desc: "Substring to match against file names (case-insensitive)."},
			"extension":   {Type: schema.String, Desc: "File extension to filter by, e.g. '.go' or 'go' (leading dot optional)."},
			"base_dir":    {Type: schema.String, Desc: "Directory to search from. Defaults to current directory."},
			"max_results": {Type: schema.Integer, Desc: fmt.Sprintf("Maximum number of results. Defaults to %d.", findFilesDefaultMaxResult)},
		}),
	}, nil
}

func (t *FindFilesTool) InvokableRun(_ context.Context, args string, _ ...tool.Option) (string, error) {
	var p findFilesParams
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", fmt.Errorf("find_files: invalid params: %w", err)
	}
	if p.Name == "" && p.Extension == "" {
		return "", fmt.Errorf("find_files: at least one of 'name' or 'extension' is required")
	}
	if p.MaxResults <= 0 {
		p.MaxResults = findFilesDefaultMaxResult
	}

	baseDir := p.BaseDir
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("find_files: get cwd: %w", err)
		}
	}

	ext := p.Extension
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	nameLower := strings.ToLower(p.Name)

	var files []string
	_ = filepath.WalkDir(baseDir, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil || d.IsDir() {
			return nil
		}
		fname := d.Name()
		matchName := nameLower == "" || strings.Contains(strings.ToLower(fname), nameLower)
		matchExt := ext == "" || strings.EqualFold(filepath.Ext(fname), ext)
		if matchName && matchExt {
			rel, _ := filepath.Rel(baseDir, path)
			files = append(files, rel)
		}
		if len(files) >= p.MaxResults {
			return fs.SkipAll
		}
		return nil
	})

	out, _ := json.Marshal(findFilesResult{Files: files, Count: len(files)})
	return string(out), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// RepoOverviewTool — builtin/repo_overview
// ─────────────────────────────────────────────────────────────────────────────

// RepoOverviewTool returns a directory tree and basic statistics.
type RepoOverviewTool struct{}

func newRepoOverviewTool() *RepoOverviewTool { return &RepoOverviewTool{} }

type repoOverviewParams struct {
	BaseDir  string `json:"base_dir,omitempty"`
	MaxDepth int    `json:"max_depth,omitempty"`
}

type repoOverviewResult struct {
	Tree       string `json:"tree"`
	TotalFiles int    `json:"total_files"`
	TotalDirs  int    `json:"total_dirs"`
}

func (t *RepoOverviewTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "repo_overview",
		Desc: "Return a directory tree and file/directory counts for a repository or directory. Useful for understanding project structure before diving into code.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"base_dir":  {Type: schema.String, Desc: "Root directory to inspect. Defaults to current directory."},
			"max_depth": {Type: schema.Integer, Desc: fmt.Sprintf("Maximum directory depth to render. Defaults to %d.", repoOverviewDefaultDepth)},
		}),
	}, nil
}

func (t *RepoOverviewTool) InvokableRun(_ context.Context, args string, _ ...tool.Option) (string, error) {
	var p repoOverviewParams
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return "", fmt.Errorf("repo_overview: invalid params: %w", err)
	}

	baseDir := p.BaseDir
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("repo_overview: get cwd: %w", err)
		}
	}
	maxDepth := p.MaxDepth
	if maxDepth <= 0 {
		maxDepth = repoOverviewDefaultDepth
	}

	var sb strings.Builder
	totalFiles, totalDirs := 0, 0
	fileCount := 0

	var walk func(dir, prefix string, depth int)
	walk = func(dir, prefix string, depth int) {
		if depth > maxDepth || fileCount >= repoOverviewMaxFiles {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for i, entry := range entries {
			if fileCount >= repoOverviewMaxFiles {
				sb.WriteString(prefix + "...\n")
				return
			}
			connector := "├── "
			childPrefix := prefix + "│   "
			if i == len(entries)-1 {
				connector = "└── "
				childPrefix = prefix + "    "
			}
			sb.WriteString(prefix + connector + entry.Name() + "\n")
			fileCount++
			if entry.IsDir() {
				totalDirs++
				walk(filepath.Join(dir, entry.Name()), childPrefix, depth+1)
			} else {
				totalFiles++
			}
		}
	}

	sb.WriteString(filepath.Base(baseDir) + "/\n")
	walk(baseDir, "", 1)

	out, _ := json.Marshal(repoOverviewResult{
		Tree:       sb.String(),
		TotalFiles: totalFiles,
		TotalDirs:  totalDirs,
	})
	return string(out), nil
}
