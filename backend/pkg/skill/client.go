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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// HubClient interfaces with the external SkillsHub service.
type HubClient interface {
	Search(ctx context.Context, query string, opts SearchOpts) ([]SkillMeta, error)
	Get(ctx context.Context, name string, version string) (*SkillMeta, error)
	Install(ctx context.Context, name string, version string) (*SkillInstance, error)
	Uninstall(ctx context.Context, name string) error
	List(ctx context.Context) ([]SkillInstance, error)
	CheckHealth(ctx context.Context, name string) (bool, error)
}

// SearchOpts carries optional filters for a skill search request.
type SearchOpts struct {
	Tags   []string
	Limit  int
	Offset int
}

// HTTPHubClientConfig holds the configuration for HTTPHubClient.
type HTTPHubClientConfig struct {
	BaseURL   string
	AuthToken string
	Timeout   time.Duration
}

// HTTPHubClient is the default HubClient implementation backed by the SkillsHub REST API.
type HTTPHubClient struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// NewHTTPHubClient creates an HTTPHubClient with the provided configuration.
// If cfg.Timeout is zero, a 30-second default is used.
func NewHTTPHubClient(cfg HTTPHubClientConfig) *HTTPHubClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &HTTPHubClient{
		baseURL:   cfg.BaseURL,
		authToken: cfg.AuthToken,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Search calls GET /api/skills/search?q={query} and returns matching skill metadata.
func (c *HTTPHubClient) Search(ctx context.Context, query string, opts SearchOpts) ([]SkillMeta, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/skills/search", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("skill: parse search URL: %w", err)
	}

	q := u.Query()
	q.Set("q", query)
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		q.Set("offset", strconv.Itoa(opts.Offset))
	}
	for _, tag := range opts.Tags {
		q.Add("tag", tag)
	}
	u.RawQuery = q.Encode()

	var result []SkillMeta
	if err := c.get(ctx, u.String(), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Get calls GET /api/skills/{name}/{version} and returns the skill's metadata.
func (c *HTTPHubClient) Get(ctx context.Context, name string, version string) (*SkillMeta, error) {
	path := fmt.Sprintf("%s/api/skills/%s/%s", c.baseURL, url.PathEscape(name), url.PathEscape(version))
	var meta SkillMeta
	if err := c.get(ctx, path, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// Install calls POST /api/skills/install and returns the created SkillInstance.
func (c *HTTPHubClient) Install(ctx context.Context, name string, version string) (*SkillInstance, error) {
	body := map[string]string{
		"name":    name,
		"version": version,
	}
	var instance SkillInstance
	if err := c.post(ctx, fmt.Sprintf("%s/api/skills/install", c.baseURL), body, &instance); err != nil {
		return nil, err
	}
	return &instance, nil
}

// Uninstall calls DELETE /api/skills/{name}.
func (c *HTTPHubClient) Uninstall(ctx context.Context, name string) error {
	path := fmt.Sprintf("%s/api/skills/%s", c.baseURL, url.PathEscape(name))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("skill: create delete request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("skill: uninstall %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("skill: uninstall %s returned HTTP %d: %s", name, resp.StatusCode, string(body))
	}
	return nil
}

// List calls GET /api/skills and returns all installed skill instances.
func (c *HTTPHubClient) List(ctx context.Context) ([]SkillInstance, error) {
	path := fmt.Sprintf("%s/api/skills", c.baseURL)
	var instances []SkillInstance
	if err := c.get(ctx, path, &instances); err != nil {
		return nil, err
	}
	return instances, nil
}

// CheckHealth calls GET /api/skills/{name}/health and reports whether the skill is healthy.
func (c *HTTPHubClient) CheckHealth(ctx context.Context, name string) (bool, error) {
	path := fmt.Sprintf("%s/api/skills/%s/health", c.baseURL, url.PathEscape(name))
	var result struct {
		Healthy bool `json:"healthy"`
	}
	if err := c.get(ctx, path, &result); err != nil {
		return false, err
	}
	return result.Healthy, nil
}

// get performs an authenticated GET request and JSON-decodes the response body into dst.
func (c *HTTPHubClient) get(ctx context.Context, rawURL string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("skill: create GET request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("skill: GET %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	return c.decodeResponse(resp, dst)
}

// post performs an authenticated POST request with a JSON body and decodes the response into dst.
func (c *HTTPHubClient) post(ctx context.Context, rawURL string, body any, dst any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("skill: marshal POST body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("skill: create POST request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("skill: POST %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	return c.decodeResponse(resp, dst)
}

func (c *HTTPHubClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
}

func (c *HTTPHubClient) decodeResponse(resp *http.Response, dst any) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("skill: read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("skill: hub returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return fmt.Errorf("skill: decode response: %w", err)
	}
	return nil
}
