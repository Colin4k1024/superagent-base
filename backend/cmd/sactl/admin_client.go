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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// AdminClient sends requests to the Superagent admin API.
type AdminClient struct {
	BaseURL    string
	AdminKey   string
	HTTPClient *http.Client
}

// NewAdminClient builds an AdminClient from environment variables.
// SUPERAGENT_URL defaults to http://localhost:8888.
// SUPERAGENT_ADMIN_KEY is the bearer token for admin endpoints.
func NewAdminClient() *AdminClient {
	baseURL := os.Getenv("SUPERAGENT_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8888"
	}
	adminKey := os.Getenv("SUPERAGENT_ADMIN_KEY")
	return &AdminClient{
		BaseURL:  baseURL,
		AdminKey: adminKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ApplyResult is the server response after applying an agent definition.
type ApplyResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"`  // "created" or "updated"
	Message string `json:"message"`
}

// ValidateResult is the server response after validating an agent definition.
type ValidateResult struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors"`
}

// ApplyAgent sends the YAML definition to the admin API.
// It first extracts the agent name from the YAML, then attempts
// PUT /api/v2/admin/agents/:name.  On a 404 it falls back to
// POST /api/v2/admin/agents.
func (c *AdminClient) ApplyAgent(name string, yamlContent []byte) (*ApplyResult, error) {
	// Try PUT (update) first.
	body, status, err := c.doRequest(http.MethodPut, "/api/v2/admin/agents/"+name, yamlContent)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		// Fall back to POST (create).
		body, status, err = c.doRequest(http.MethodPost, "/api/v2/admin/agents", yamlContent)
		if err != nil {
			return nil, err
		}
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("server returned HTTP %d: %s", status, string(body))
	}
	var result ApplyResult
	if err := json.Unmarshal(body, &result); err != nil {
		// Server response not JSON — synthesise a result so the caller can still
		// display something meaningful.
		result = ApplyResult{
			Name:    name,
			Status:  "applied",
			Message: string(body),
		}
	}
	return &result, nil
}

// ValidateAgent sends the YAML definition to the remote validate endpoint.
// POST /api/v2/admin/agents/validate
func (c *AdminClient) ValidateAgent(yamlContent []byte) (*ValidateResult, error) {
	body, status, err := c.doRequest(http.MethodPost, "/api/v2/admin/agents/validate", yamlContent)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("server returned HTTP %d: %s", status, string(body))
	}
	var result ValidateResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("invalid response from server: %w", err)
	}
	return &result, nil
}

// doRequest executes a single HTTP request with the YAML body and the admin
// bearer token, returning the response body, status code, and any transport
// error.
func (c *AdminClient) doRequest(method, path string, body []byte) ([]byte, int, error) {
	req, err := http.NewRequest(method, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-yaml")
	if c.AdminKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.AdminKey)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("http %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}
	return respBody, resp.StatusCode, nil
}
