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

package hiagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultTimeout = 60 * time.Second

// Client is a thread-safe HTTP client for HiAgent v2.0.0 API.
type Client struct {
	cfg    Config
	http   *http.Client
}

func NewClient(cfg Config) *Client {
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: defaultTimeout},
	}
}

// CreateConversation creates a new conversation and returns the AppConversationID.
func (c *Client) CreateConversation(ctx context.Context, userID string) (string, error) {
	reqBody := CreateConversationRequest{
		UserID: userID,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("hiagent: marshal create_conversation request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/create_conversation", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("hiagent: build create_conversation request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("hiagent: create_conversation request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("hiagent: read create_conversation response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("hiagent: create_conversation returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result CreateConversationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("hiagent: decode create_conversation response: %w", err)
	}

	if result.BaseResp != nil && result.BaseResp.StatusCode != 0 {
		return "", fmt.Errorf("hiagent: create_conversation error: %s (code=%d)", result.BaseResp.StatusMessage, result.BaseResp.StatusCode)
	}

	if result.Conversation.AppConversationID == "" {
		return "", fmt.Errorf("hiagent: create_conversation returned empty AppConversationID")
	}

	return result.Conversation.AppConversationID, nil
}

// ChatQueryBlocking sends a blocking query and returns the complete response.
func (c *Client) ChatQueryBlocking(ctx context.Context, chatReq *ChatQueryV2Request) (*ChatQueryV2Response, error) {
	chatReq.ResponseMode = "blocking"

	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("hiagent: marshal chat_query_v2 request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/chat_query_v2", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("hiagent: build chat_query_v2 request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hiagent: chat_query_v2 request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("hiagent: read chat_query_v2 response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
	}

	var result ChatQueryV2Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("hiagent: decode chat_query_v2 response: %w", err)
	}

	return &result, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Apikey", c.cfg.APIKey)
}

// APIError represents a non-200 response from HiAgent.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("hiagent: API returned HTTP %d: %s", e.StatusCode, e.Body)
}

// IsSessionExpired returns true if the error indicates the conversation no longer exists.
func IsSessionExpired(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound || apiErr.StatusCode == http.StatusGone
	}
	return false
}
