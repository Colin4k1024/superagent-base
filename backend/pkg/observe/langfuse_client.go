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

package observe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LangfuseClient provides HTTP access to the Langfuse public REST API.
type LangfuseClient struct {
	baseURL    string
	authHeader string
	httpClient *http.Client
}

// NewLangfuseClient creates a client from the existing LangfuseConfig.
func NewLangfuseClient(cfg LangfuseConfig) *LangfuseClient {
	if !cfg.Enabled || cfg.PublicKey == "" || cfg.SecretKey == "" {
		return nil
	}
	return &LangfuseClient{
		baseURL:    strings.TrimRight(cfg.Host, "/"),
		authHeader: cfg.AuthHeader(),
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// TraceListParams holds query parameters for listing traces.
type TraceListParams struct {
	Page    int    `json:"page"`
	Limit   int    `json:"limit"`
	OrderBy string `json:"orderBy"`
	Name    string `json:"name"`
	FromTS  string `json:"fromTimestamp"`
	ToTS    string `json:"toTimestamp"`
}

// TraceListResponse is the Langfuse /api/public/traces response.
type TraceListResponse struct {
	Data []json.RawMessage `json:"data"`
	Meta map[string]any    `json:"meta"`
}

// ListTraces queries Langfuse for trace list.
func (c *LangfuseClient) ListTraces(ctx context.Context, params TraceListParams) (*TraceListResponse, error) {
	q := url.Values{}
	if params.Page > 0 {
		q.Set("page", fmt.Sprintf("%d", params.Page))
	}
	if params.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", params.Limit))
	}
	if params.OrderBy != "" {
		q.Set("orderBy", params.OrderBy)
	}
	if params.Name != "" {
		q.Set("name", params.Name)
	}
	if params.FromTS != "" {
		q.Set("fromTimestamp", params.FromTS)
	}
	if params.ToTS != "" {
		q.Set("toTimestamp", params.ToTS)
	}

	var resp TraceListResponse
	if err := c.get(ctx, "/api/public/traces", q, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetTrace retrieves a single trace by ID.
func (c *LangfuseClient) GetTrace(ctx context.Context, traceID string) (json.RawMessage, error) {
	var resp json.RawMessage
	if err := c.get(ctx, "/api/public/traces/"+traceID, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// DailyMetricsResponse holds the daily metrics from Langfuse.
type DailyMetricsResponse struct {
	Data []json.RawMessage `json:"data"`
	Meta map[string]any    `json:"meta"`
}

// GetDailyMetrics retrieves daily usage metrics from Langfuse.
func (c *LangfuseClient) GetDailyMetrics(ctx context.Context, fromDate, toDate string) (*DailyMetricsResponse, error) {
	q := url.Values{}
	if fromDate != "" {
		q.Set("fromTimestamp", fromDate)
	}
	if toDate != "" {
		q.Set("toTimestamp", toDate)
	}

	var resp DailyMetricsResponse
	if err := c.get(ctx, "/api/public/metrics/daily", q, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListSessions retrieves sessions from Langfuse.
func (c *LangfuseClient) ListSessions(ctx context.Context, page, limit int) (json.RawMessage, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	var resp json.RawMessage
	if err := c.get(ctx, "/api/public/sessions", q, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// get performs an authenticated GET request.
func (c *LangfuseClient) get(ctx context.Context, path string, query url.Values, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("langfuse: build request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("langfuse: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("langfuse: HTTP %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("langfuse: decode response: %w", err)
	}
	return nil
}
