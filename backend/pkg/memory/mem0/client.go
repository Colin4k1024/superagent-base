/*
 * Copyright 2025 coze-dev Authors
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

package mem0

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

const (
	defaultTimeout = 30 * time.Second
	headerAuth     = "Authorization"
	headerCT       = "Content-Type"
	contentJSON    = "application/json"
)

// APIClient is a thin HTTP client for the Mem0 REST API.
// Base URL should be the root of the Mem0 deployment, e.g. "https://api.mem0.ai".
type APIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewAPIClient creates an APIClient with a 30-second default timeout.
func NewAPIClient(baseURL, apiKey string) *APIClient {
	return NewAPIClientWithHTTP(baseURL, apiKey, &http.Client{Timeout: defaultTimeout})
}

// NewAPIClientWithHTTP creates an APIClient with a custom http.Client, useful
// for tests that inject an httptest transport.
func NewAPIClientWithHTTP(baseURL, apiKey string, hc *http.Client) *APIClient {
	return &APIClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: hc,
	}
}

// ── Request / Response Types ─────────────────────────────────────────────────

// APIMessage is the Mem0 wire format for a single chat message.
type APIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AddMemoryRequest is the body sent to POST /v1/memories.
type AddMemoryRequest struct {
	Messages []APIMessage   `json:"messages"`
	UserID   string         `json:"user_id,omitempty"`
	AgentID  string         `json:"agent_id,omitempty"`
	RunID    string         `json:"run_id,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// AddMemoryResponse is the body returned by POST /v1/memories.
type AddMemoryResponse struct {
	ID      string `json:"id"`
	Message string `json:"message,omitempty"`
}

// SearchRequest is the body sent to POST /v1/memories/search.
type SearchRequest struct {
	Query   string `json:"query"`
	UserID  string `json:"user_id,omitempty"`
	AgentID string `json:"agent_id,omitempty"`
	RunID   string `json:"run_id,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// MemoryResult is a single memory record returned by Mem0.
type MemoryResult struct {
	ID        string         `json:"id"`
	Memory    string         `json:"memory"`
	UserID    string         `json:"user_id"`
	AgentID   string         `json:"agent_id"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
	Score     float64        `json:"score,omitempty"`
}

// UpdateMemoryRequest is the body sent to PUT /v1/memories/{id}.
type UpdateMemoryRequest struct {
	Memory string `json:"memory"`
}

// HistoryEntry represents one change event in a memory's history.
type HistoryEntry struct {
	ID          string `json:"id"`
	MemoryID    string `json:"memory_id"`
	OldMemory   string `json:"old_memory"`
	NewMemory   string `json:"new_memory"`
	Event       string `json:"event"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ── API Methods ──────────────────────────────────────────────────────────────

// AddMemory calls POST /v1/memories.
func (c *APIClient) AddMemory(ctx context.Context, req AddMemoryRequest) (*AddMemoryResponse, error) {
	var resp AddMemoryResponse
	if err := c.doJSON(ctx, http.MethodPost, "/v1/memories", req, &resp); err != nil {
		return nil, fmt.Errorf("mem0 AddMemory: %w", err)
	}
	return &resp, nil
}

// SearchMemory calls POST /v1/memories/search.
func (c *APIClient) SearchMemory(ctx context.Context, req SearchRequest) ([]MemoryResult, error) {
	var results []MemoryResult
	if err := c.doJSON(ctx, http.MethodPost, "/v1/memories/search", req, &results); err != nil {
		return nil, fmt.Errorf("mem0 SearchMemory: %w", err)
	}
	return results, nil
}

// GetAllMemories calls GET /v1/memories with optional user_id, limit, offset.
func (c *APIClient) GetAllMemories(ctx context.Context, userID string, limit, offset int) ([]MemoryResult, error) {
	params := url.Values{}
	if userID != "" {
		params.Set("user_id", userID)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	path := "/v1/memories"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var results []MemoryResult
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &results); err != nil {
		return nil, fmt.Errorf("mem0 GetAllMemories: %w", err)
	}
	return results, nil
}

// UpdateMemory calls PUT /v1/memories/{memoryID}.
func (c *APIClient) UpdateMemory(ctx context.Context, memoryID string, content string) error {
	path := "/v1/memories/" + memoryID
	body := UpdateMemoryRequest{Memory: content}
	if err := c.doJSON(ctx, http.MethodPut, path, body, nil); err != nil {
		return fmt.Errorf("mem0 UpdateMemory: %w", err)
	}
	return nil
}

// DeleteMemory calls DELETE /v1/memories/{memoryID}.
func (c *APIClient) DeleteMemory(ctx context.Context, memoryID string) error {
	path := "/v1/memories/" + memoryID
	if err := c.doJSON(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("mem0 DeleteMemory: %w", err)
	}
	return nil
}

// GetMemoryHistory calls GET /v1/memories/{memoryID}/history.
func (c *APIClient) GetMemoryHistory(ctx context.Context, memoryID string) ([]HistoryEntry, error) {
	path := "/v1/memories/" + memoryID + "/history"
	var history []HistoryEntry
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &history); err != nil {
		return nil, fmt.Errorf("mem0 GetMemoryHistory: %w", err)
	}
	return history, nil
}

// ── HTTP Helpers ─────────────────────────────────────────────────────────────

// doJSON encodes body as JSON, sends the request, and decodes the response
// into out (if out is non-nil). A non-2xx status code is returned as an error.
func (c *APIClient) doJSON(ctx context.Context, method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set(headerAuth, "Token "+c.apiKey)
	if body != nil {
		req.Header.Set(headerCT, contentJSON)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(respBytes))
	}

	if out != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}
