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

// Package letta provides a Letta (https://letta.com, formerly MemGPT) memory backend.
package letta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultTimeout = 30 * time.Second

// APIClient is a thin HTTP client for the Letta REST API.
// Base URL should be the root endpoint, e.g. "https://api.letta.com".
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

// UserMessage is the request body for POST /v1/agents/{agentID}/messages.
type UserMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// SendMessageRequest is the request body for sending a message to a Letta agent.
type SendMessageRequest struct {
	Messages []UserMessage `json:"messages"`
	Stream   bool          `json:"stream_steps,omitempty"`
}

// MessageContent is one item in a Letta agent response.
type MessageContent struct {
	Type    string `json:"message_type"`
	Content string `json:"content,omitempty"`
}

// Response is the response from POST /v1/agents/{agentID}/messages.
type Response struct {
	Messages []MessageContent `json:"messages"`
	Usage    map[string]any   `json:"usage,omitempty"`
}

// CoreMemory holds the Letta agent's in-context memory sections.
type CoreMemory struct {
	Memory map[string]BlockMemory `json:"memory"`
}

// BlockMemory is one named memory block (e.g. "human", "persona").
type BlockMemory struct {
	Value       string `json:"value"`
	Limit       int    `json:"limit"`
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// UpdateCoreMemoryRequest is the body for PATCH /v1/agents/{agentID}/core-memory/blocks/{blockLabel}.
type UpdateCoreMemoryRequest struct {
	Value string `json:"value"`
}

// ArchivalEntry is one archival memory passage returned by Letta.
type ArchivalEntry struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp,omitempty"`
}

// InsertArchivalRequest is the body for POST /v1/agents/{agentID}/archival.
type InsertArchivalRequest struct {
	Text string `json:"text"`
}

// ArchivalSearchRequest is the body for GET /v1/agents/{agentID}/archival with query params.
// Letta uses query params, not a body, for archival search.
type archivalSearchParams struct {
	Query string
	Limit int
}

// ── API Methods ───────────────────────────────────────────────────────────────

// SendMessage sends one or more messages to a Letta agent and returns the response.
// POST /v1/agents/{agentID}/messages
func (c *APIClient) SendMessage(ctx context.Context, agentID string, message string) (*Response, error) {
	path := "/v1/agents/" + agentID + "/messages"
	body := SendMessageRequest{
		Messages: []UserMessage{{Role: "user", Content: message}},
	}
	var resp Response
	if err := c.doJSON(ctx, http.MethodPost, path, body, &resp); err != nil {
		return nil, fmt.Errorf("letta SendMessage: %w", err)
	}
	return &resp, nil
}

// GetCoreMemory fetches the agent's core (in-context) memory.
// GET /v1/agents/{agentID}/core-memory
func (c *APIClient) GetCoreMemory(ctx context.Context, agentID string) (*CoreMemory, error) {
	path := "/v1/agents/" + agentID + "/core-memory"
	var cm CoreMemory
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &cm); err != nil {
		return nil, fmt.Errorf("letta GetCoreMemory: %w", err)
	}
	return &cm, nil
}

// UpdateCoreMemory updates a named block in the agent's core memory.
// PATCH /v1/agents/{agentID}/core-memory/blocks/{blockLabel}
func (c *APIClient) UpdateCoreMemory(ctx context.Context, agentID string, section string, value string) error {
	path := "/v1/agents/" + agentID + "/core-memory/blocks/" + section
	body := UpdateCoreMemoryRequest{Value: value}
	if err := c.doJSON(ctx, http.MethodPatch, path, body, nil); err != nil {
		return fmt.Errorf("letta UpdateCoreMemory: %w", err)
	}
	return nil
}

// SearchArchival searches the agent's archival (long-term) memory.
// GET /v1/agents/{agentID}/archival?query=...&limit=...
func (c *APIClient) SearchArchival(ctx context.Context, agentID string, query string, limit int) ([]ArchivalEntry, error) {
	path := fmt.Sprintf("/v1/agents/%s/archival?query=%s", agentID, urlEncode(query))
	if limit > 0 {
		path += fmt.Sprintf("&limit=%d", limit)
	}
	var entries []ArchivalEntry
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &entries); err != nil {
		return nil, fmt.Errorf("letta SearchArchival: %w", err)
	}
	return entries, nil
}

// InsertArchival adds a passage to the agent's archival memory.
// POST /v1/agents/{agentID}/archival
func (c *APIClient) InsertArchival(ctx context.Context, agentID string, content string) (*ArchivalEntry, error) {
	path := "/v1/agents/" + agentID + "/archival"
	body := InsertArchivalRequest{Text: content}
	var entry ArchivalEntry
	if err := c.doJSON(ctx, http.MethodPost, path, body, &entry); err != nil {
		return nil, fmt.Errorf("letta InsertArchival: %w", err)
	}
	return &entry, nil
}

// DeleteArchival removes an archival passage by ID.
// DELETE /v1/agents/{agentID}/archival/{passageID}
func (c *APIClient) DeleteArchival(ctx context.Context, agentID string, passageID string) error {
	path := "/v1/agents/" + agentID + "/archival/" + passageID
	if err := c.doJSON(ctx, http.MethodDelete, path, nil, nil); err != nil {
		return fmt.Errorf("letta DeleteArchival: %w", err)
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
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
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
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(respBytes))
	}

	if out != nil && len(respBytes) > 0 {
		if err := json.Unmarshal(respBytes, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// urlEncode performs minimal percent-encoding for a query parameter value.
func urlEncode(s string) string {
	const hex = "0123456789ABCDEF"
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUnreserved(c) {
			b = append(b, c)
		} else {
			b = append(b, '%', hex[c>>4], hex[c&15])
		}
	}
	return string(b)
}

func isUnreserved(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~'
}
