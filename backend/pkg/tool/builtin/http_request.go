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

package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// Compile-time assertion.
var _ tool.InvokableTool = (*HTTPRequestTool)(nil)

const defaultHTTPTimeout = 30 * time.Second

// HTTPRequestTool makes an HTTP request with configurable method, headers, and body.
type HTTPRequestTool struct {
	client *http.Client
}

func newHTTPRequestTool() *HTTPRequestTool {
	return &HTTPRequestTool{
		client: &http.Client{Timeout: defaultHTTPTimeout},
	}
}

// NewHTTPRequestToolWithTimeout creates an HTTPRequestTool with a custom timeout.
func NewHTTPRequestToolWithTimeout(timeout time.Duration) *HTTPRequestTool {
	return &HTTPRequestTool{
		client: &http.Client{Timeout: timeout},
	}
}

func (h *HTTPRequestTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "http_request",
		Desc: "Make an HTTP request to a URL and return the response status code, headers, and body.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"url": {
				Desc:     "The target URL.",
				Type:     schema.String,
				Required: true,
			},
			"method": {
				Desc:     "HTTP method: GET, POST, PUT, PATCH, DELETE (default: GET).",
				Type:     schema.String,
				Required: false,
			},
			"headers": {
				Desc:     "Request headers as a JSON object (key-value string pairs).",
				Type:     schema.Object,
				Required: false,
			},
			"body": {
				Desc:     "Request body as a string.",
				Type:     schema.String,
				Required: false,
			},
		}),
	}, nil
}

func (h *HTTPRequestTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		URL     string            `json:"url"`
		Method  string            `json:"method"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("http_request: parse arguments: %w", err)
	}
	if args.URL == "" {
		return "", fmt.Errorf("http_request: url is required")
	}
	if args.Method == "" {
		args.Method = http.MethodGet
	}
	args.Method = strings.ToUpper(args.Method)

	var bodyReader io.Reader
	if args.Body != "" {
		bodyReader = bytes.NewBufferString(args.Body)
	}

	req, err := http.NewRequestWithContext(ctx, args.Method, args.URL, bodyReader)
	if err != nil {
		return "", fmt.Errorf("http_request: create request: %w", err)
	}
	for k, v := range args.Headers {
		req.Header.Set(k, v)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http_request: execute: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("http_request: read response body: %w", err)
	}

	// Collect response headers as a flat map.
	respHeaders := make(map[string]string, len(resp.Header))
	for k, vals := range resp.Header {
		respHeaders[k] = strings.Join(vals, ", ")
	}

	result := map[string]any{
		"status_code": resp.StatusCode,
		"headers":     respHeaders,
		"body":        string(respBody),
	}

	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("http_request: marshal result: %w", err)
	}
	return string(out), nil
}
