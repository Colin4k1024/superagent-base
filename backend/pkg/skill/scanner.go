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

// Package skill provides the SKILL.md scanner following the Eino adk/middlewares/skill pattern.
// See: https://www.cloudwego.cn/zh/docs/eino/quick_start/chapter_09_skill_console/
package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const claudeSkillFileName = "SKILL.md"

// ClaudeFrontMatter is the YAML frontmatter of a Claude Code SKILL.md file.
// Field layout mirrors github.com/cloudwego/eino/adk/middlewares/skill.FrontMatter
// for future compatibility when upgrading eino.
type ClaudeFrontMatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	Origin      string   `yaml:"origin"`
}

// ClaudeSkill represents a skill loaded from a SKILL.md file.
type ClaudeSkill struct {
	ClaudeFrontMatter
	// Content is the markdown body after the frontmatter — the skill's instructions.
	Content string
	// BaseDirectory is the absolute directory of the SKILL.md file.
	BaseDirectory string
}

// ScanClaudeSkills scans baseDir for first-level subdirectories that contain a SKILL.md
// and returns all valid skills. Malformed files are skipped with a warning.
func ScanClaudeSkills(baseDir string) ([]ClaudeSkill, error) {
	pattern := filepath.Join(baseDir, "*", claudeSkillFileName)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("skill scanner: glob %q: %w", pattern, err)
	}

	skills := make([]ClaudeSkill, 0, len(matches))
	for _, path := range matches {
		s, loadErr := loadClaudeSkill(path)
		if loadErr != nil {
			// Warn but continue — a single bad file should not prevent others from loading.
			fmt.Fprintf(os.Stderr, "skill scanner: skip %s: %v\n", path, loadErr)
			continue
		}
		skills = append(skills, s)
	}
	return skills, nil
}

func loadClaudeSkill(path string) (ClaudeSkill, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ClaudeSkill{}, fmt.Errorf("read: %w", err)
	}

	fmStr, content, err := parseClaudeFrontmatter(string(raw))
	if err != nil {
		return ClaudeSkill{}, fmt.Errorf("parse frontmatter: %w", err)
	}

	var front ClaudeFrontMatter
	if err = yaml.Unmarshal([]byte(fmStr), &front); err != nil {
		return ClaudeSkill{}, fmt.Errorf("unmarshal frontmatter: %w", err)
	}
	if front.Name == "" {
		return ClaudeSkill{}, fmt.Errorf("missing name in frontmatter")
	}

	return ClaudeSkill{
		ClaudeFrontMatter: front,
		Content:           strings.TrimSpace(content),
		BaseDirectory:     filepath.Dir(path),
	}, nil
}

// parseClaudeFrontmatter splits a SKILL.md string into its YAML frontmatter block
// and the remaining markdown content. The file must start with "---".
func parseClaudeFrontmatter(data string) (frontmatter, content string, err error) {
	const delim = "---"
	data = strings.TrimSpace(data)
	if !strings.HasPrefix(data, delim) {
		return "", "", fmt.Errorf("file does not start with frontmatter delimiter '---'")
	}
	rest := data[len(delim):]
	fm, after, found := strings.Cut(rest, "\n"+delim)
	if !found {
		return "", "", fmt.Errorf("frontmatter closing delimiter '---' not found")
	}
	frontmatter = strings.TrimSpace(fm)
	content = strings.TrimPrefix(after, "\n")
	return frontmatter, content, nil
}

// RegisterClaudeSkills scans baseDir for SKILL.md files and registers each as a local skill.
//
// Each skill runs in inline mode: when invoked, the skill's markdown content is returned
// as the tool result so the calling agent can use it as contextual instructions.
// This matches Eino's default inline context mode described in the skill console guide.
//
// Returns the number of skills successfully registered.
func RegisterClaudeSkills(mgr *Manager, invoker *LocalInvoker, baseDir string) (int, error) {
	skills, err := ScanClaudeSkills(baseDir)
	if err != nil {
		return 0, fmt.Errorf("register claude skills: %w", err)
	}

	for _, s := range skills {
		content := s.Content
		name := s.Name

		// Inline mode: return skill instructions as tool result.
		invoker.Register(name, func(_ context.Context, _ map[string]any) (map[string]any, error) {
			return map[string]any{"content": content}, nil
		})

		mgr.RegisterLocal(SkillMeta{
			Name:        name,
			Version:     "1.0.0",
			Description: s.Description,
			Tags:        s.Tags,
			Input: &JSONSchema{
				Type: "object",
				Properties: map[string]*JSONSchema{
					"query": {
						Type:        "string",
						Description: "Optional context or question about this skill",
					},
				},
			},
			Output: &JSONSchema{
				Type: "object",
				Properties: map[string]*JSONSchema{
					"content": {
						Type:        "string",
						Description: "Skill instructions and reference content",
					},
				},
				Required: []string{"content"},
			},
			Runtime: RuntimeConfig{Type: "embedded"},
			Source:  "claude-skills",
		})
	}

	return len(skills), nil
}
