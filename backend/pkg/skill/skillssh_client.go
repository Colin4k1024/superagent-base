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
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// SkillsShConfig holds configuration for the skills.sh marketplace client.
type SkillsShConfig struct {
	// BaseURL is the skills.sh endpoint (default: https://skills.sh).
	BaseURL string

	// APIKey is an optional Bearer token for REST API access.
	// If empty, the client falls back to CLI-based search (npx skills find).
	APIKey string

	// Timeout for HTTP requests. Defaults to 30s.
	Timeout time.Duration
}

// SkillsShClient implements HubClient for the skills.sh public marketplace.
// It uses the REST API when an API key is configured, otherwise falls back
// to the `npx skills find` CLI for search operations.
type SkillsShClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	useAPI     bool
	cache      *Cache
}

// NewSkillsShClient creates a marketplace client for skills.sh.
func NewSkillsShClient(cfg SkillsShConfig) *SkillsShClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://skills.sh"
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &SkillsShClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     cfg.APIKey,
		httpClient: &http.Client{Timeout: timeout},
		useAPI:     cfg.APIKey != "",
		cache:      NewCache(),
	}
}

// Search finds skills matching the query from skills.sh marketplace.
func (c *SkillsShClient) Search(ctx context.Context, query string, opts SearchOpts) ([]SkillMeta, error) {
	if c.useAPI {
		return c.searchAPI(ctx, query, opts)
	}
	return c.searchCLI(ctx, query, opts)
}

// Get retrieves skill details by name. For skills.sh, name format is "source/skill".
func (c *SkillsShClient) Get(ctx context.Context, name string, version string) (*SkillMeta, error) {
	if c.useAPI {
		return c.getAPI(ctx, name)
	}
	// Without API, search for the skill name and return the first match.
	results, err := c.searchCLI(ctx, name, SearchOpts{Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("skills.sh: skill %q not found", name)
	}
	return &results[0], nil
}

// Install fetches metadata and registers the skill locally.
func (c *SkillsShClient) Install(ctx context.Context, name string, version string) (*SkillInstance, error) {
	meta, err := c.Get(ctx, name, version)
	if err != nil {
		return nil, fmt.Errorf("skills.sh: install %s: %w", name, err)
	}
	instance := &SkillInstance{
		Meta:   *meta,
		Status: "installed",
	}
	c.cache.Set(meta.Name, instance)
	return instance, nil
}

// Uninstall removes the skill from local cache.
func (c *SkillsShClient) Uninstall(_ context.Context, name string) error {
	c.cache.Delete(name)
	return nil
}

// List returns locally installed skills from this marketplace.
func (c *SkillsShClient) List(_ context.Context) ([]SkillInstance, error) {
	return c.cache.All(), nil
}

// CheckHealth always returns true for skills.sh marketplace skills.
func (c *SkillsShClient) CheckHealth(_ context.Context, _ string) (bool, error) {
	return true, nil
}

// --- REST API methods (requires API key) ---

func (c *SkillsShClient) searchAPI(ctx context.Context, query string, opts SearchOpts) ([]SkillMeta, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v1/skills/search", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("skills.sh: parse URL: %w", err)
	}

	q := u.Query()
	q.Set("q", query)
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	q.Set("limit", strconv.Itoa(limit))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("skills.sh: create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("skills.sh: search request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("skills.sh: read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("skills.sh: search returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var items []skillsShItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("skills.sh: decode response: %w", err)
	}

	results := make([]SkillMeta, 0, len(items))
	for _, item := range items {
		results = append(results, item.toSkillMeta(c.baseURL))
	}
	return results, nil
}

func (c *SkillsShClient) getAPI(ctx context.Context, name string) (*SkillMeta, error) {
	// name format: "source/skill" e.g. "anthropics/skills/webapp-testing"
	u := fmt.Sprintf("%s/api/v1/skills/%s", c.baseURL, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("skills.sh: create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("skills.sh: get request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("skills.sh: read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("skills.sh: get returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var item skillsShItem
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, fmt.Errorf("skills.sh: decode response: %w", err)
	}

	meta := item.toSkillMeta(c.baseURL)
	return &meta, nil
}

// --- CLI-based search (no API key required) ---

func (c *SkillsShClient) searchCLI(ctx context.Context, query string, opts SearchOpts) ([]SkillMeta, error) {
	cmd := exec.CommandContext(ctx, "npx", "skills", "find", query)
	cmd.Env = append(cmd.Environ(), "DISABLE_TELEMETRY=1", "NO_COLOR=1")

	output, err := cmd.Output()
	if err != nil {
		// If npx is not available or command fails, return empty rather than error
		// to avoid breaking the search when CLI is not installed.
		return nil, nil
	}

	results := ParseSkillsFindOutput(string(output))
	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}
	return results, nil
}

// --- Output parser ---

// ansiRegex matches ANSI escape codes for stripping from CLI output.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?\x07`)

// stripANSI removes ANSI escape codes from a string.
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// skillLineRegex matches the skill line format: "owner/repo@skill  NK installs"
var skillLineRegex = regexp.MustCompile(`^(\S+?)@(\S+)\s+(.+?)\s+installs?$`)

// urlLineRegex matches the URL line: "└ https://skills.sh/..."
var urlLineRegex = regexp.MustCompile(`^[└└\s]+\s*(https?://\S+)$`)

// ParseSkillsFindOutput parses the output of `npx skills find` into SkillMeta.
func ParseSkillsFindOutput(output string) []SkillMeta {
	lines := strings.Split(output, "\n")
	var results []SkillMeta
	var current *SkillMeta

	for _, line := range lines {
		clean := strings.TrimSpace(stripANSI(line))
		if clean == "" {
			continue
		}

		// Try matching skill line: "owner/repo@skill  NK installs"
		if m := skillLineRegex.FindStringSubmatch(clean); m != nil {
			source := m[1]  // e.g., "vercel-labs/agent-skills"
			skill := m[2]   // e.g., "react-best-practices"
			instStr := m[3] // e.g., "71.4K"

			installs := parseInstallCount(instStr)
			installCmd := fmt.Sprintf("npx skills add %s@%s -g -y", source, skill)

			meta := SkillMeta{
				Name:        skill,
				Version:     "latest",
				Author:      source,
				Description: fmt.Sprintf("%s (from %s)", humanizeName(skill), source),
				Tags:        []string{"skills.sh"},
				Installs:    installs,
				Source:      "skills.sh",
				InstallCmd:  installCmd,
			}
			results = append(results, meta)
			current = &results[len(results)-1]
			continue
		}

		// Try matching URL line
		if current != nil {
			if m := urlLineRegex.FindStringSubmatch(clean); m != nil {
				current.URL = m[1]
				current = nil
				continue
			}
		}
	}

	return results
}

// parseInstallCount converts "71.4K" or "1.6M" or "957" to an integer.
func parseInstallCount(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	multiplier := 1.0
	if strings.HasSuffix(s, "K") {
		multiplier = 1000
		s = strings.TrimSuffix(s, "K")
	} else if strings.HasSuffix(s, "M") {
		multiplier = 1000000
		s = strings.TrimSuffix(s, "M")
	}

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int(math.Round(val * multiplier))
}

// humanizeName converts "webapp-testing" to "Webapp Testing".
func humanizeName(s string) string {
	parts := strings.Split(s, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

// --- API response types ---

type skillsShItem struct {
	ID         string `json:"id"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	Source     string `json:"source"`
	Installs   int    `json:"installs"`
	SourceType string `json:"sourceType"`
	InstallURL string `json:"installUrl"`
	URL        string `json:"url"`
}

func (item skillsShItem) toSkillMeta(baseURL string) SkillMeta {
	name := item.Slug
	if name == "" {
		name = item.Name
	}
	skillURL := item.URL
	if skillURL == "" && item.Source != "" && name != "" {
		skillURL = fmt.Sprintf("%s/%s/%s", baseURL, item.Source, name)
	}
	installCmd := ""
	if item.InstallURL != "" {
		installCmd = fmt.Sprintf("npx skills add %s -g -y", item.InstallURL)
	} else if item.Source != "" && name != "" {
		installCmd = fmt.Sprintf("npx skills add %s@%s -g -y", item.Source, name)
	}

	return SkillMeta{
		Name:        name,
		Version:     "latest",
		Author:      item.Source,
		Description: fmt.Sprintf("%s (from %s)", humanizeName(name), item.Source),
		Tags:        []string{"skills.sh"},
		Installs:    item.Installs,
		Source:      "skills.sh",
		InstallCmd:  installCmd,
		URL:         skillURL,
	}
}
