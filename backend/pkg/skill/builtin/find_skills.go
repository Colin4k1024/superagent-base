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
	"fmt"

	"github.com/superagent-ai/superagent-base/backend/pkg/skill"
)

// skillsShClient is the package-level client used by FindSkillsSkill.
// It's initialized via InitFindSkills and defaults to CLI-based search.
var skillsShClient *skill.SkillsShClient

// InitFindSkills sets the SkillsShClient used by the find-skills builtin.
// Must be called before agents invoke skill://find-skills.
func InitFindSkills(client *skill.SkillsShClient) {
	skillsShClient = client
}

// FindSkillsSkill searches the skills.sh marketplace for agent skills.
//
// Input fields:
//   - query (required) – search query, e.g. "react testing", "deployment"
//   - limit (optional) – max results to return (default 10)
//
// Output:
//   - skills – array of {name, source, description, installs, url, install_cmd}
//   - count  – number of results returned
//   - tip    – usage suggestion for the agent
func FindSkillsSkill(ctx context.Context, input map[string]any) (map[string]any, error) {
	query, _ := input["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("find-skills: 'query' field is required")
	}

	limit := 10
	if l, ok := input["limit"].(float64); ok && l > 0 {
		limit = int(l)
	} else if l, ok := input["limit"].(int); ok && l > 0 {
		limit = l
	}

	// Use the configured client, or create a default CLI-based one.
	client := skillsShClient
	if client == nil {
		client = skill.NewSkillsShClient(skill.SkillsShConfig{})
	}

	results, err := client.Search(ctx, query, skill.SearchOpts{Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("find-skills: search failed: %w", err)
	}

	// Convert to simplified output format for agents.
	skills := make([]map[string]any, 0, len(results))
	for _, r := range results {
		skills = append(skills, map[string]any{
			"name":        r.Name,
			"source":      r.Author,
			"description": r.Description,
			"installs":    r.Installs,
			"url":         r.URL,
			"install_cmd": r.InstallCmd,
		})
	}

	return map[string]any{
		"skills": skills,
		"count":  len(skills),
		"tip":    "Present these skills to the user with install commands. They can install with: npx skills add <source>@<skill> -g -y",
	}, nil
}
