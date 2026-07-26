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

package modelrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

// ComplexityAnalyzer evaluates the complexity of a user request.
type ComplexityAnalyzer interface {
	// Analyze returns the complexity level ("low", "medium", "high") for the given messages.
	Analyze(ctx context.Context, messages []*schema.Message) (string, error)
}

// ComplexityConfig configures the LLM-based complexity analyzer.
type ComplexityConfig struct {
	Enabled           bool   `yaml:"enabled"`
	Model             string `yaml:"model"`
	BaseURL           string `yaml:"base_url,omitempty"`
	APIKey            string `yaml:"api_key,omitempty"`
	FallbackComplexity string `yaml:"fallback_complexity,omitempty"`
	CacheTTL          string `yaml:"cache_ttl,omitempty"`
}

// LLMComplexityAnalyzer uses a lightweight LLM to classify task complexity.
type LLMComplexityAnalyzer struct {
	modelID   string
	baseURL   string
	apiKey    string
	fallback  string
	cacheTTL  time.Duration
	cache     sync.Map // conversationID -> cacheEntry
}

type cacheEntry struct {
	complexity string
	expiresAt  time.Time
}

// NewLLMComplexityAnalyzer creates a new analyzer from config.
func NewLLMComplexityAnalyzer(cfg ComplexityConfig) *LLMComplexityAnalyzer {
	fallback := cfg.FallbackComplexity
	if fallback == "" {
		fallback = "medium"
	}
	cacheTTL := 5 * time.Minute
	if cfg.CacheTTL != "" {
		if d, err := time.ParseDuration(cfg.CacheTTL); err == nil {
			cacheTTL = d
		}
	}
	return &LLMComplexityAnalyzer{
		modelID:  cfg.Model,
		baseURL:  cfg.BaseURL,
		apiKey:   cfg.APIKey,
		fallback: fallback,
		cacheTTL: cacheTTL,
	}
}

const complexityPrompt = `You are a task complexity classifier. Analyze the user's request and classify its complexity.

Rules:
- "low": simple greetings, short Q&A, translation, simple lookup, yes/no questions
- "medium": general reasoning, code generation, document writing, multi-step tasks, summarization
- "high": complex reasoning, architecture design, debugging complex systems, long-context analysis, multi-tool coordination, mathematical proofs

Respond with ONLY a JSON object, no other text:
{"complexity": "low"} or {"complexity": "medium"} or {"complexity": "high"}`

// Analyze classifies the complexity of the given messages.
func (a *LLMComplexityAnalyzer) Analyze(ctx context.Context, messages []*schema.Message) (string, error) {
	if len(messages) == 0 {
		return a.fallback, nil
	}

	// Check cache using conversationID from context if available.
	if convID := getConversationID(ctx); convID != "" {
		if cached, ok := a.cache.Load(convID); ok {
			entry := cached.(cacheEntry)
			if time.Now().Before(entry.expiresAt) {
				return entry.complexity, nil
			}
			a.cache.Delete(convID)
		}
	}

	// Extract recent user messages for analysis (last 3 turns).
	analysisMessages := extractRecentUserMessages(messages, 3)
	if len(analysisMessages) == 0 {
		return a.fallback, nil
	}

	complexity, err := a.callLLM(ctx, analysisMessages)
	if err != nil {
		return a.fallback, nil // degrade gracefully
	}

	// Cache the result.
	if convID := getConversationID(ctx); convID != "" {
		a.cache.Store(convID, cacheEntry{
			complexity: complexity,
			expiresAt:  time.Now().Add(a.cacheTTL),
		})
	}

	return complexity, nil
}

func (a *LLMComplexityAnalyzer) callLLM(ctx context.Context, userMessages []string) (string, error) {
	// Create a short-lived context with timeout for the analysis call.
	analyzeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	chatModel, err := einoopenai.NewChatModel(analyzeCtx, &einoopenai.ChatModelConfig{
		BaseURL: a.baseURL,
		APIKey:  a.apiKey,
		Model:   a.modelID,
	})
	if err != nil {
		return "", fmt.Errorf("complexity analyzer: create model: %w", err)
	}

	// Build the analysis prompt.
	userContent := strings.Join(userMessages, "\n---\n")

	msgs := []*schema.Message{
		{Role: schema.System, Content: complexityPrompt},
		{Role: schema.User, Content: userContent},
	}

	resp, err := chatModel.Generate(analyzeCtx, msgs)
	if err != nil {
		return "", fmt.Errorf("complexity analyzer: generate: %w", err)
	}

	return parseComplexityResponse(resp.Content)
}

func parseComplexityResponse(content string) (string, error) {
	// Try to parse as JSON first.
	var result struct {
		Complexity string `json:"complexity"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &result); err == nil {
		switch result.Complexity {
		case "low", "medium", "high":
			return result.Complexity, nil
		}
	}

	// Fallback: look for keywords.
	content = strings.ToLower(content)
	if strings.Contains(content, `"high"`) || strings.Contains(content, "high") {
		return "high", nil
	}
	if strings.Contains(content, `"low"`) || strings.Contains(content, "low") {
		return "low", nil
	}
	if strings.Contains(content, `"medium"`) || strings.Contains(content, "medium") {
		return "medium", nil
	}

	return "", fmt.Errorf("complexity analyzer: unrecognized response: %s", content)
}

// extractRecentUserMessages extracts the last n user message contents.
func extractRecentUserMessages(messages []*schema.Message, n int) []string {
	var userMsgs []string
	for i := len(messages) - 1; i >= 0 && len(userMsgs) < n; i-- {
		if messages[i].Role == schema.User && messages[i].Content != "" {
			userMsgs = append([]string{messages[i].Content}, userMsgs...)
		}
	}
	return userMsgs
}

// getConversationID extracts conversation ID from context (if set by the runtime).
func getConversationID(ctx context.Context) string {
	if v := ctx.Value(conversationIDKey{}); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

type conversationIDKey struct{}

// WithConversationID returns a context with the conversation ID for cache lookup.
func WithConversationID(ctx context.Context, convID string) context.Context {
	return context.WithValue(ctx, conversationIDKey{}, convID)
}
