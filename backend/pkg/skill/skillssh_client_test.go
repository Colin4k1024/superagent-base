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

package skill

import (
	"context"
	"os/exec"
	"testing"
)

func TestParseSkillsFindOutput(t *testing.T) {
	// Simulated output of `npx skills find testing` (with ANSI stripped).
	output := `Install with npx skills add <owner/repo@skill>

anthropics/skills@webapp-testing  71.4K installs
└ https://skills.sh/anthropics/skills/webapp-testing

wshobson/agents@python-testing-patterns  20.4K installs
└ https://skills.sh/wshobson/agents/python-testing-patterns

supercent-io/skills-template@testing-strategies  11.2K installs
└ https://skills.sh/supercent-io/skills-template/testing-strategies

vercel-labs/json-render@react  2K installs
└ https://skills.sh/vercel-labs/json-render/react
`

	results := ParseSkillsFindOutput(output)

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	// Verify first result
	r := results[0]
	if r.Name != "webapp-testing" {
		t.Errorf("expected name 'webapp-testing', got %q", r.Name)
	}
	if r.Author != "anthropics/skills" {
		t.Errorf("expected author 'anthropics/skills', got %q", r.Author)
	}
	if r.Installs != 71400 {
		t.Errorf("expected installs 71400, got %d", r.Installs)
	}
	if r.URL != "https://skills.sh/anthropics/skills/webapp-testing" {
		t.Errorf("expected URL 'https://skills.sh/anthropics/skills/webapp-testing', got %q", r.URL)
	}
	if r.Source != "skills.sh" {
		t.Errorf("expected source 'skills.sh', got %q", r.Source)
	}
	if r.InstallCmd != "npx skills add anthropics/skills@webapp-testing -g -y" {
		t.Errorf("unexpected install cmd: %q", r.InstallCmd)
	}

	// Verify second result
	r2 := results[1]
	if r2.Name != "python-testing-patterns" {
		t.Errorf("expected name 'python-testing-patterns', got %q", r2.Name)
	}
	if r2.Installs != 20400 {
		t.Errorf("expected installs 20400, got %d", r2.Installs)
	}

	// Verify fourth result (plain number, no K/M suffix)
	r4 := results[3]
	if r4.Name != "react" {
		t.Errorf("expected name 'react', got %q", r4.Name)
	}
	if r4.Installs != 2000 {
		t.Errorf("expected installs 2000, got %d", r4.Installs)
	}
}

func TestParseSkillsFindOutputWithANSI(t *testing.T) {
	// Real output with ANSI escape codes.
	output := "\x1b[38;5;102mInstall with\x1b[0m npx skills add <owner/repo@skill>\n" +
		"\n" +
		"\x1b[38;5;145manthropics/skills@frontend-design\x1b[0m \x1b[36m423.1K installs\x1b[0m\n" +
		"\x1b[38;5;102m└ https://skills.sh/anthropics/skills/frontend-design\x1b[0m\n" +
		"\n" +
		"\x1b[38;5;145mmicrosoft/azure-skills@microsoft-foundry\x1b[0m \x1b[36m1.6M installs\x1b[0m\n" +
		"\x1b[38;5;102m└ https://skills.sh/microsoft/azure-skills/microsoft-foundry\x1b[0m\n"

	results := ParseSkillsFindOutput(output)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Name != "frontend-design" {
		t.Errorf("expected 'frontend-design', got %q", results[0].Name)
	}
	if results[0].Installs != 423100 {
		t.Errorf("expected 423100, got %d", results[0].Installs)
	}

	if results[1].Name != "microsoft-foundry" {
		t.Errorf("expected 'microsoft-foundry', got %q", results[1].Name)
	}
	if results[1].Installs != 1600000 {
		t.Errorf("expected 1600000, got %d", results[1].Installs)
	}
}

func TestParseInstallCount(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"71.4K", 71400},
		{"1.6M", 1600000},
		{"957", 957},
		{"2K", 2000},
		{"0.5K", 500},
		{"", 0},
		{"abc", 0},
	}

	for _, tt := range tests {
		result := parseInstallCount(tt.input)
		if result != tt.expected {
			t.Errorf("parseInstallCount(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestMultiHubClientSearch(t *testing.T) {
	// Create two mock clients with overlapping results.
	client1 := &mockHubClient{
		searchResults: []SkillMeta{
			{Name: "skill-a", Author: "source1", Description: "A from hub1"},
			{Name: "skill-b", Author: "source1", Description: "B from hub1"},
		},
	}
	client2 := &mockHubClient{
		searchResults: []SkillMeta{
			{Name: "skill-b", Author: "source1", Description: "B from hub2"}, // duplicate
			{Name: "skill-c", Author: "source2", Description: "C from hub2"},
		},
	}

	multi := NewMultiHubClient(client1, client2)
	results, err := multi.Search(context.Background(), "test", SearchOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should deduplicate: skill-b from source1 appears only once (first wins).
	if len(results) != 3 {
		t.Fatalf("expected 3 results after dedup, got %d", len(results))
	}

	names := make(map[string]bool)
	for _, r := range results {
		names[r.Author+"/"+r.Name] = true
	}
	if !names["source1/skill-a"] || !names["source1/skill-b"] || !names["source2/skill-c"] {
		t.Errorf("unexpected result set: %v", results)
	}
}

func TestMultiHubClientNilClients(t *testing.T) {
	multi := NewMultiHubClient(nil, nil)
	results, err := multi.Search(context.Background(), "test", SearchOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results for empty client list, got %v", results)
	}
}

// TestSkillsShClientSearchCLI is an integration test that calls the real CLI.
// Skipped when npx is not available.
func TestSkillsShClientSearchCLI(t *testing.T) {
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not available, skipping CLI integration test")
	}

	client := NewSkillsShClient(SkillsShConfig{})
	results, err := client.Search(context.Background(), "react", SearchOpts{Limit: 5})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least 1 result from skills.sh CLI search")
	}

	// Verify basic fields are populated.
	r := results[0]
	if r.Name == "" {
		t.Error("expected non-empty Name")
	}
	if r.Author == "" {
		t.Error("expected non-empty Author")
	}
	if r.Source != "skills.sh" {
		t.Errorf("expected Source 'skills.sh', got %q", r.Source)
	}
	if r.InstallCmd == "" {
		t.Error("expected non-empty InstallCmd")
	}

	t.Logf("Found %d skills, first: %s by %s (%d installs)", len(results), r.Name, r.Author, r.Installs)
}

// mockHubClient is a test helper that returns pre-configured results.
type mockHubClient struct {
	searchResults []SkillMeta
}

func (m *mockHubClient) Search(_ context.Context, _ string, _ SearchOpts) ([]SkillMeta, error) {
	return m.searchResults, nil
}
func (m *mockHubClient) Get(_ context.Context, _ string, _ string) (*SkillMeta, error) {
	return nil, nil
}
func (m *mockHubClient) Install(_ context.Context, _ string, _ string) (*SkillInstance, error) {
	return nil, nil
}
func (m *mockHubClient) Uninstall(_ context.Context, _ string) error { return nil }
func (m *mockHubClient) List(_ context.Context) ([]SkillInstance, error) {
	return nil, nil
}
func (m *mockHubClient) CheckHealth(_ context.Context, _ string) (bool, error) {
	return true, nil
}
