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
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// SkillHubClientConfig holds configuration for the internal SkillHub gateway client.
type SkillHubClientConfig struct {
	// BaseURL is the SkillHub gateway endpoint (e.g., http://127.0.0.1:18092/skillhub).
	BaseURL string

	// AccessToken is the user IAM access token (X-Access-Token header).
	AccessToken string

	// AppAccessToken is the application access token (App-Access-Token header).
	AppAccessToken string

	// KCode is the caller application K-Code.
	KCode string

	// VisitKCode is the target application K-Code.
	VisitKCode string

	// Timeout for HTTP requests. Defaults to 30s.
	Timeout time.Duration
}

// SkillHubClient implements HubClient for the internal SkillHub service
// accessed through internal-gateway.
type SkillHubClient struct {
	baseURL        string
	accessToken    string
	appAccessToken string
	kCode          string
	visitKCode     string
	httpClient     *http.Client
	cache          *Cache
}

// NewSkillHubClient creates a client configured for the internal SkillHub API.
func NewSkillHubClient(cfg SkillHubClientConfig) *SkillHubClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &SkillHubClient{
		baseURL:        cfg.BaseURL,
		accessToken:    cfg.AccessToken,
		appAccessToken: cfg.AppAccessToken,
		kCode:          cfg.KCode,
		visitKCode:     cfg.VisitKCode,
		httpClient:     &http.Client{Timeout: timeout},
		cache:          NewCache(),
	}
}

// skillHubResponse wraps the standard response envelope from SkillHub.
type skillHubResponse struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
	Msg  string          `json:"msg,omitempty"`
}

// skillHubSkillItem represents a skill in search/detail results.
type skillHubSkillItem struct {
	Namespace      string `json:"namespace"`
	Slug           string `json:"slug"`
	Coordinate     string `json:"coordinate"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	LatestVersion  string `json:"latestVersion"`
	InstallTarget  string `json:"installTarget"`
	InstallVersion string `json:"installVersion"`
	InstallCommand string `json:"installCommand"`
}

// skillHubSearchResult represents the paginated search response.
type skillHubSearchResult struct {
	Content []skillHubSkillItem `json:"content"`
	Total   int                 `json:"total"`
}

// Search calls GET /api/v1/portal/skills/available?q=&size= on the internal SkillHub.
func (c *SkillHubClient) Search(ctx context.Context, query string, opts SearchOpts) ([]SkillMeta, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v1/portal/skills/available", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("skillhub: parse search URL: %w", err)
	}

	q := u.Query()
	q.Set("q", query)
	size := opts.Limit
	if size <= 0 {
		size = 20
	}
	q.Set("size", strconv.Itoa(size))
	for _, tag := range opts.Tags {
		q.Add("tag", tag)
	}
	u.RawQuery = q.Encode()

	body, err := c.doGet(ctx, u.String())
	if err != nil {
		return nil, err
	}

	var resp skillHubResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		// Try direct array parse (some endpoints return data directly)
		var items []skillHubSkillItem
		if err2 := json.Unmarshal(body, &items); err2 == nil {
			return c.convertItems(items), nil
		}
		return nil, fmt.Errorf("skillhub: decode search response: %w", err)
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("skillhub: search returned code %d: %s", resp.Code, resp.Msg)
	}

	// Try paginated response
	var searchResult skillHubSearchResult
	if err := json.Unmarshal(resp.Data, &searchResult); err != nil {
		// Try direct array
		var items []skillHubSkillItem
		if err2 := json.Unmarshal(resp.Data, &items); err2 != nil {
			return nil, fmt.Errorf("skillhub: decode search data: %w", err)
		}
		return c.convertItems(items), nil
	}
	return c.convertItems(searchResult.Content), nil
}

// Get calls GET /api/v1/portal/skills/{namespace}/{slug} for skill details.
func (c *SkillHubClient) Get(ctx context.Context, name string, version string) (*SkillMeta, error) {
	// name may be "namespace/slug" or just "slug" (defaults to "global")
	namespace, slug := parseCoordinate(name)

	u := fmt.Sprintf("%s/api/v1/portal/skills/%s/%s", c.baseURL,
		url.PathEscape(namespace), url.PathEscape(slug))

	body, err := c.doGet(ctx, u)
	if err != nil {
		return nil, err
	}

	var resp skillHubResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("skillhub: decode detail response: %w", err)
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("skillhub: get %s returned code %d: %s", name, resp.Code, resp.Msg)
	}

	var item skillHubSkillItem
	if err := json.Unmarshal(resp.Data, &item); err != nil {
		return nil, fmt.Errorf("skillhub: decode detail data: %w", err)
	}

	meta := c.convertItem(item)
	if version != "" && version != "latest" {
		meta.Version = version
	}
	return &meta, nil
}

// Install fetches skill metadata and registers it locally.
// Actual ZIP download is handled by the CLI; for the runtime we only need metadata.
func (c *SkillHubClient) Install(ctx context.Context, name string, version string) (*SkillInstance, error) {
	meta, err := c.Get(ctx, name, version)
	if err != nil {
		return nil, fmt.Errorf("skillhub: install %s: %w", name, err)
	}

	instance := &SkillInstance{
		Meta:   *meta,
		Status: "installed",
	}
	c.cache.Set(meta.Name, instance)
	return instance, nil
}

// Uninstall removes the skill from the local cache.
func (c *SkillHubClient) Uninstall(_ context.Context, name string) error {
	c.cache.Delete(name)
	return nil
}

// List returns all locally installed skills.
func (c *SkillHubClient) List(_ context.Context) ([]SkillInstance, error) {
	return c.cache.All(), nil
}

// CheckHealth always returns true for SkillHub-managed skills (health is managed by the hub).
func (c *SkillHubClient) CheckHealth(_ context.Context, _ string) (bool, error) {
	return true, nil
}

// doGet performs an authenticated GET request and returns the raw response body.
func (c *SkillHubClient) doGet(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("skillhub: create request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("skillhub: GET %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("skillhub: read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("skillhub: HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// setHeaders applies authentication headers based on configured credentials.
func (c *SkillHubClient) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")

	if c.accessToken != "" {
		req.Header.Set("X-Access-Token", c.accessToken)
	}
	if c.appAccessToken != "" {
		req.Header.Set("App-Access-Token", c.appAccessToken)
		if c.kCode != "" {
			req.Header.Set("K-Code", c.kCode)
		}
		if c.visitKCode != "" {
			req.Header.Set("VISIT-K-Code", c.visitKCode)
		}
	}
}

// convertItems converts SkillHub API items to SkillMeta slice.
func (c *SkillHubClient) convertItems(items []skillHubSkillItem) []SkillMeta {
	result := make([]SkillMeta, 0, len(items))
	for _, item := range items {
		result = append(result, c.convertItem(item))
	}
	return result
}

// convertItem converts a single SkillHub API item to SkillMeta.
func (c *SkillHubClient) convertItem(item skillHubSkillItem) SkillMeta {
	name := item.Slug
	if item.Coordinate != "" {
		name = item.Coordinate
	}
	version := item.LatestVersion
	if item.InstallVersion != "" {
		version = item.InstallVersion
	}
	return SkillMeta{
		Name:        name,
		Version:     version,
		Description: item.Description,
	}
}

// parseCoordinate splits "namespace/slug" into parts. If no slash, defaults to "global".
func parseCoordinate(name string) (namespace, slug string) {
	for i, ch := range name {
		if ch == '/' {
			return name[:i], name[i+1:]
		}
	}
	return "global", name
}
