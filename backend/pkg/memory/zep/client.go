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

// Package zep provides a Zep (https://www.getzep.com) memory backend.
package zep

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const defaultTimeout = 30 * time.Second

// APIClient is a thin HTTP client for the Zep Cloud / Open-Source REST API.
// Base URL should be the root endpoint, e.g. "https://api.getzep.com".
type APIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewAPIClient returns an APIClient with the default 30-second timeout.
func NewAPIClient(baseURL, apiKey string) *APIClient {
	return NewAPIClientWithHTTP(baseURL, apiKey, &http.Client{Timeout: defaultTimeout})
}

// NewAPIClientWithHTTP returns an APIClient with a caller-supplied http.Client.
func NewAPIClientWithHTTP(baseURL, apiKey string, hc *http.Client) *APIClient {
	return &APIClient{baseURL: baseURL, apiKey: apiKey, httpClient: hc}
}

// ── Wire Types ────────────────────────────────────────────────────────────────

// Message is a single chat message in Zep format.
type Message struct {
	UUID       string         `json:"uuid,omitempty"`
	Role       string         `json:"role"`
	RoleType   string         `json:"role_type,omitempty"` // "user", "assistant", "system"
	Content    string         `json:"content"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  string         `json:"created_at,omitempty"`
	TokenCount int            `json:"token_count,omitempty"`
}

// Memory is the response from GET /api/v2/sessions/{sessionID}/memory.
type Memory struct {
	Context  string    `json:"context,omitempty"` // summarised context string
	Messages []Message `json:"messages"`
	Facts    []string  `json:"facts,omitempty"`
}

// SearchResult is one item from POST /api/v2/sessions/{sessionID}/search.
type SearchResult struct {
	Message *Message `json:"message"`
	Score   float64  `json:"score"`
	Fact    string   `json:"fact,omitempty"`
}

// Session represents a Zep session object.
type Session struct {
	SessionID string         `json:"session_id"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt string         `json:"created_at,omitempty"`
	UpdatedAt string         `json:"updated_at,omitempty"`
}

// addMemoryRequest is the body for POST /api/v2/sessions/{sessionID}/memory.
type addMemoryRequest struct {
	Messages []Message      `json:"messages"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// searchRequest is the body for POST /api/v2/sessions/{sessionID}/search.
type searchRequest struct {
	Text       string `json:"text"`
	Limit      int    `json:"limit,omitempty"`
	SearchType string `json:"search_type,omitempty"` // "similarity" | "mmr"
}

// updateSessionRequest is the body for PATCH /api/v2/sessions/{sessionID}.
type updateSessionRequest struct {
	Metadata map[string]any `json:"metadata"`
}

// ── API Methods ───────────────────────────────────────────────────────────────

// EnsureSession creates a session if it does not already exist (idempotent).
func (c *APIClient) EnsureSession(ctx context.Context, sessionID string) error {
	body := map[string]any{"session_id": sessionID}
	err := c.doJSON(ctx, http.MethodPost, "/api/v2/sessions", body, nil)
	// Zep returns 409 if session already exists — treat as success.
	if isConflict(err) {
		return nil
	}
	return err
}

// AddMemory appends messages to a Zep session.
// POST /api/v2/sessions/{sessionID}/memory
func (c *APIClient) AddMemory(ctx context.Context, sessionID string, messages []Message) error {
	path := "/api/v2/sessions/" + sessionID + "/memory"
	body := addMemoryRequest{Messages: messages}
	if err := c.doJSON(ctx, http.MethodPost, path, body, nil); err != nil {
		return fmt.Errorf("zep AddMemory: %w", err)
	}
	return nil
}

// GetMemory fetches the memory for a session. lastN controls how many messages
// are returned (0 = server default).
// GET /api/v2/sessions/{sessionID}/memory
func (c *APIClient) GetMemory(ctx context.Context, sessionID string, lastN int) (*Memory, error) {
	path := "/api/v2/sessions/" + sessionID + "/memory"
	if lastN > 0 {
		path += "?lastn=" + strconv.Itoa(lastN)
	}
	var mem Memory
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &mem); err != nil {
		return nil, fmt.Errorf("zep GetMemory: %w", err)
	}
	return &mem, nil
}

// SearchMemory performs a semantic search over session messages.
// POST /api/v2/sessions/{sessionID}/search
func (c *APIClient) SearchMemory(ctx context.Context, sessionID string, query string, limit int) ([]SearchResult, error) {
	path := "/api/v2/sessions/" + sessionID + "/search"
	body := searchRequest{Text: query, Limit: limit, SearchType: "similarity"}
	var results []SearchResult
	if err := c.doJSON(ctx, http.MethodPost, path, body, &results); err != nil {
		return nil, fmt.Errorf("zep SearchMemory: %w", err)
	}
	return results, nil
}

// DeleteSession removes a Zep session and all its memory.
// DELETE /api/v2/sessions/{sessionID}
func (c *APIClient) DeleteSession(ctx context.Context, sessionID string) error {
	path := "/api/v2/sessions/" + sessionID
	if err := c.doJSON(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("zep DeleteSession: %w", err)
	}
	return nil
}

// GetSession fetches session metadata.
// GET /api/v2/sessions/{sessionID}
func (c *APIClient) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	path := "/api/v2/sessions/" + sessionID
	var s Session
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &s); err != nil {
		return nil, fmt.Errorf("zep GetSession: %w", err)
	}
	return &s, nil
}

// UpdateSessionMetadata patches arbitrary metadata onto a session.
// PATCH /api/v2/sessions/{sessionID}
func (c *APIClient) UpdateSessionMetadata(ctx context.Context, sessionID string, metadata map[string]any) error {
	path := "/api/v2/sessions/" + sessionID
	body := updateSessionRequest{Metadata: metadata}
	if err := c.doJSON(ctx, http.MethodPatch, path, body, nil); err != nil {
		return fmt.Errorf("zep UpdateSessionMetadata: %w", err)
	}
	return nil
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

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
	req.Header.Set("Authorization", "Api-Key "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
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
		return &apiError{status: resp.StatusCode, body: string(respBytes)}
	}

	if out != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// apiError carries an HTTP status code so callers can inspect it.
type apiError struct {
	status int
	body   string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("status %d: %s", e.status, e.body)
}

func isConflict(err error) bool {
	if err == nil {
		return false
	}
	ae, ok := err.(*apiError)
	return ok && ae.status == http.StatusConflict
}
