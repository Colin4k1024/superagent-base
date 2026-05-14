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
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"golang.org/x/net/html"
)

// Compile-time assertion.
var _ tool.InvokableTool = (*WebSearchTool)(nil)

const webSearchTimeout = 10 * time.Second

// WebSearchTool searches the web for a query and returns title/url/snippet results.
// Backend selection is controlled by environment variables:
//   - SEARCH_PROVIDER=serper + SEARCH_API_KEY=<key>: uses Serper.dev Google Search API
//   - Otherwise (default, no API key required): uses DuckDuckGo HTML scraping
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
	searchFunc := selectSearchBackend()
	return &WebSearchTool{SearchFunc: searchFunc}
}

// selectSearchBackend picks the search implementation based on environment variables.
func selectSearchBackend() func(ctx context.Context, query string, maxResults int) ([]WebSearchResult, error) {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("SEARCH_PROVIDER")))
	apiKey := strings.TrimSpace(os.Getenv("SEARCH_API_KEY"))
	if provider == "serper" && apiKey != "" {
		return serperSearch(apiKey)
	}
	return duckduckgoSearch
}

// serperSearch returns a search function backed by the Serper.dev Google Search API.
func serperSearch(apiKey string) func(ctx context.Context, query string, maxResults int) ([]WebSearchResult, error) {
	client := &http.Client{Timeout: webSearchTimeout}
	return func(ctx context.Context, query string, maxResults int) ([]WebSearchResult, error) {
		if maxResults <= 0 {
			maxResults = 5
		}

		payload, err := json.Marshal(map[string]any{
			"q":   query,
			"num": maxResults,
		})
		if err != nil {
			return nil, fmt.Errorf("serper: marshal request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			"https://google.serper.dev/search", bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("serper: create request: %w", err)
		}
		req.Header.Set("X-API-KEY", apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("serper: http request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			return nil, fmt.Errorf("serper: unexpected status %d: %s", resp.StatusCode, string(body))
		}

		var serperResp struct {
			Organic []struct {
				Title   string `json:"title"`
				Link    string `json:"link"`
				Snippet string `json:"snippet"`
			} `json:"organic"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&serperResp); err != nil {
			return nil, fmt.Errorf("serper: decode response: %w", err)
		}

		results := make([]WebSearchResult, 0, len(serperResp.Organic))
		for _, item := range serperResp.Organic {
			results = append(results, WebSearchResult{
				Title:   item.Title,
				URL:     item.Link,
				Snippet: item.Snippet,
			})
		}
		return results, nil
	}
}

// duckduckgoSearch searches using the DuckDuckGo HTML endpoint (no API key required).
func duckduckgoSearch(ctx context.Context, query string, maxResults int) ([]WebSearchResult, error) {
	if maxResults <= 0 {
		maxResults = 5
	}

	searchURL := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)
	client := &http.Client{Timeout: webSearchTimeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("duckduckgo: create request: %w", err)
	}
	// Set a browser-like User-Agent to avoid bot blocking.
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; superagent-bot/1.0)")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("duckduckgo: http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("duckduckgo: unexpected status %d", resp.StatusCode)
	}

	results, err := parseDDGHTML(resp.Body, maxResults)
	if err != nil {
		return nil, fmt.Errorf("duckduckgo: parse results: %w", err)
	}
	return results, nil
}

// parseDDGHTML parses the DuckDuckGo HTML response and extracts search results.
// DuckDuckGo HTML result structure:
//
//	<div class="result results_links...">
//	  <h2 class="result__title"><a class="result__a" href="...">Title</a></h2>
//	  <a class="result__snippet">Snippet text</a>
//	</div>
func parseDDGHTML(r io.Reader, maxResults int) ([]WebSearchResult, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	var results []WebSearchResult
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if len(results) >= maxResults {
			return
		}
		if isElement(n, "div") && hasClass(n, "result") && !hasClass(n, "result--more") {
			result := extractDDGResult(n)
			if result.Title != "" && result.URL != "" {
				results = append(results, result)
			}
		}
		for c := n.FirstChild; c != nil && len(results) < maxResults; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return results, nil
}

// extractDDGResult extracts a single search result from a DDG result div node.
func extractDDGResult(n *html.Node) WebSearchResult {
	var result WebSearchResult
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch {
			case n.Data == "a" && hasClass(n, "result__a"):
				result.Title = strings.TrimSpace(textContent(n))
				result.URL = getAttr(n, "href")
				// DDG sometimes wraps links as redirect URLs — unwrap if needed.
				if strings.HasPrefix(result.URL, "//duckduckgo.com/l/") {
					if parsed, err := url.ParseRequestURI("https:" + result.URL); err == nil {
						if redirect := parsed.Query().Get("uddg"); redirect != "" {
							if decoded, err := url.QueryUnescape(redirect); err == nil {
								result.URL = decoded
							}
						}
					}
				}
			case n.Data == "a" && hasClass(n, "result__snippet"):
				result.Snippet = strings.TrimSpace(textContent(n))
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return result
}

// isElement reports whether n is an HTML element with tag name tag.
func isElement(n *html.Node, tag string) bool {
	return n.Type == html.ElementNode && n.Data == tag
}

// hasClass reports whether an HTML node has a given CSS class.
func hasClass(n *html.Node, class string) bool {
	for _, attr := range n.Attr {
		if attr.Key == "class" {
			for _, c := range strings.Fields(attr.Val) {
				if c == class {
					return true
				}
			}
		}
	}
	return false
}

// getAttr returns the value of the named attribute on n, or empty string.
func getAttr(n *html.Node, name string) string {
	for _, attr := range n.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

// textContent returns the concatenated text content of n and its descendants.
func textContent(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
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
