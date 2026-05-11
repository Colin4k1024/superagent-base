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
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// Compile-time assertion.
var _ tool.InvokableTool = (*WebSearchTool)(nil)

// WebSearchTool searches the web for a query and returns title/url/snippet results.
// The current implementation is a stub; a real backend search API can be wired in
// by replacing newWebSearchTool or providing an alternate SearchFunc.
type WebSearchTool struct {
	// SearchFunc is the actual search backend. Override for real implementations.
	SearchFunc func(ctx context.Context, query string, maxResults int) ([]WebSearchResult, error)
}

// WebSearchResult holds a single web search result.
type WebSearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

func newWebSearchTool() *WebSearchTool {
	return &WebSearchTool{SearchFunc: stubWebSearch}
}

// stubWebSearch is the placeholder implementation.
func stubWebSearch(_ context.Context, query string, maxResults int) ([]WebSearchResult, error) {
	if maxResults <= 0 {
		maxResults = 5
	}
	results := make([]WebSearchResult, 0, maxResults)
	for i := 1; i <= maxResults; i++ {
		results = append(results, WebSearchResult{
			Title:   fmt.Sprintf("Result %d for: %s", i, query),
			URL:     fmt.Sprintf("https://example.com/search?q=%s&n=%d", query, i),
			Snippet: fmt.Sprintf("Placeholder snippet %d for query %q", i, query),
		})
	}
	return results, nil
}

func (w *WebSearchTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "web_search",
		Desc: "Search the web for a query and return a list of results with title, URL, and snippet.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Desc:     "The search query string.",
				Type:     schema.String,
				Required: true,
			},
			"max_results": {
				Desc:     "Maximum number of results to return (default: 5).",
				Type:     schema.Integer,
				Required: false,
			},
		}),
	}, nil
}

func (w *WebSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Query      string `json:"query"`
		MaxResults int    `json:"max_results"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("web_search: parse arguments: %w", err)
	}
	if args.Query == "" {
		return "", fmt.Errorf("web_search: query is required")
	}
	if args.MaxResults <= 0 {
		args.MaxResults = 5
	}

	results, err := w.SearchFunc(ctx, args.Query, args.MaxResults)
	if err != nil {
		return "", fmt.Errorf("web_search: %w", err)
	}

	out, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("web_search: marshal results: %w", err)
	}
	return string(out), nil
}
