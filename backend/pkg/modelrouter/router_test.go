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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig is a minimal RouterConfig used across multiple test cases.
var testConfig = &RouterConfig{
	Strategies: []StrategyConfig{
		{
			Name: "capability-based",
			Rules: []RuleConfig{
				{Match: map[string][]string{"task": {"reasoning", "planning"}}, Route: "deepseek-r1"},
				{Match: map[string][]string{"task": {"coding", "review"}}, Route: "claude-sonnet"},
				{Match: map[string][]string{"task": {"vision", "multimodal"}}, Route: "gpt-4o"},
				{Match: map[string][]string{"task": {"fast-chat", "simple"}}, Route: "doubao-pro"},
			},
			Fallback: "gpt-4o",
		},
		{
			Name: "cost-optimized",
			Rules: []RuleConfig{
				{Match: map[string][]string{"complexity": {"low"}}, Route: "doubao-lite"},
				{Match: map[string][]string{"complexity": {"medium"}}, Route: "deepseek-v3"},
				{Match: map[string][]string{"complexity": {"high"}}, Route: "claude-sonnet"},
			},
			Fallback: "deepseek-v3",
		},
	},
	Providers: map[string]ProviderConfig{
		"deepseek-r1":   {Type: "openai-compatible", Endpoint: "https://api.deepseek.com/v1"},
		"claude-sonnet": {Type: "openai-compatible", Endpoint: "https://api.anthropic.com/v1"},
		"gpt-4o":        {Type: "openai"},
		"doubao-pro":    {Type: "ark"},
		"doubao-lite":   {Type: "ark"},
		"deepseek-v3":   {Type: "openai-compatible", Endpoint: "https://api.deepseek.com/v1"},
	},
}

// TestCapabilityRouting verifies task-type-based routing.
func TestCapabilityRouting(t *testing.T) {
	cases := []struct {
		task     string
		expected string
	}{
		{"reasoning", "deepseek-r1"},
		{"planning", "deepseek-r1"},
		{"coding", "claude-sonnet"},
		{"review", "claude-sonnet"},
		{"vision", "gpt-4o"},
		{"multimodal", "gpt-4o"},
		{"fast-chat", "doubao-pro"},
		{"simple", "doubao-pro"},
	}

	router, err := NewDefaultRouter(testConfig)
	require.NoError(t, err)

	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.task, func(t *testing.T) {
			result, err := router.Route(ctx, &RouteRequest{TaskType: tc.task})
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.ModelID)
		})
	}
}

// TestCostRouting verifies complexity-based routing via the cost strategy.
func TestCostRouting(t *testing.T) {
	// Build a router with only the cost-optimized strategy so we can test it
	// in isolation without capability-based strategy winning first.
	cfg := &RouterConfig{
		Strategies: []StrategyConfig{testConfig.Strategies[1]},
		Providers:  testConfig.Providers,
	}
	router, err := NewDefaultRouter(cfg)
	require.NoError(t, err)

	ctx := context.Background()
	cases := []struct {
		complexity string
		expected   string
	}{
		{"low", "doubao-lite"},
		{"medium", "deepseek-v3"},
		{"high", "claude-sonnet"},
	}
	for _, tc := range cases {
		t.Run(tc.complexity, func(t *testing.T) {
			result, err := router.Route(ctx, &RouteRequest{Complexity: tc.complexity})
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.ModelID)
		})
	}
}

// TestFallbackWhenNoPrimaryRuleMatches verifies that the fallback model is
// returned when no rule matches the TaskType.
func TestFallbackWhenNoPrimaryRuleMatches(t *testing.T) {
	router, err := NewDefaultRouter(testConfig)
	require.NoError(t, err)

	// "unknown-task" does not match any capability rule; fallback is gpt-4o.
	result, err := router.Route(context.Background(), &RouteRequest{TaskType: "unknown-task"})
	require.NoError(t, err)
	assert.Equal(t, "gpt-4o", result.ModelID)
}

// TestFallbackChainStrategy verifies FallbackChain returns primary on match and
// chain fallback model when the primary strategy has no rule or config fallback.
func TestFallbackChainStrategy(t *testing.T) {
	// Build a capability strategy with no built-in fallback so we can observe
	// the FallbackChain's own fallback kicking in.
	noFallbackCfg := StrategyConfig{
		Name: "capability-based",
		Rules: []RuleConfig{
			{Match: map[string][]string{"task": {"reasoning"}}, Route: "deepseek-r1"},
		},
		// No Fallback field set.
	}
	primary := NewCapabilityStrategy(noFallbackCfg)
	chain := NewFallbackChain(primary, "fallback-model")

	ctx := context.Background()

	// Should return primary match.
	modelID, err := chain.Select(ctx, &RouteRequest{TaskType: "reasoning"})
	require.NoError(t, err)
	assert.Equal(t, "deepseek-r1", modelID)

	// Should return chain fallback when primary strategy has no matching rule
	// and no built-in fallback.
	modelID, err = chain.Select(ctx, &RouteRequest{TaskType: "unknown"})
	require.NoError(t, err)
	assert.Equal(t, "fallback-model", modelID)
}

// TestConfigLoadingFromYAML verifies that LoadConfigFromBytes correctly
// parses the YAML and produces a working router.
func TestConfigLoadingFromYAML(t *testing.T) {
	yaml := `
strategies:
  - name: capability-based
    rules:
      - match: {task: [reasoning]}
        route: deepseek-r1
    fallback: gpt-4o
providers:
  deepseek-r1:
    type: openai-compatible
    endpoint: https://api.deepseek.com/v1
  gpt-4o:
    type: openai
`
	cfg, err := LoadConfigFromBytes([]byte(yaml))
	require.NoError(t, err)
	require.Len(t, cfg.Strategies, 1)
	assert.Equal(t, "capability-based", cfg.Strategies[0].Name)
	assert.Equal(t, "gpt-4o", cfg.Strategies[0].Fallback)
	assert.Equal(t, "openai", cfg.Providers["gpt-4o"].Type)

	router, err := NewDefaultRouter(cfg)
	require.NoError(t, err)

	result, err := router.Route(context.Background(), &RouteRequest{TaskType: "reasoning"})
	require.NoError(t, err)
	assert.Equal(t, "deepseek-r1", result.ModelID)
	assert.Equal(t, "openai-compatible", result.ProviderName)
}

// TestRouterReload verifies that Reload atomically swaps configuration.
func TestRouterReload(t *testing.T) {
	router, err := NewDefaultRouter(testConfig)
	require.NoError(t, err)

	newCfg := &RouterConfig{
		Strategies: []StrategyConfig{
			{
				Name:  "capability-based",
				Rules: []RuleConfig{{Match: map[string][]string{"task": {"reasoning"}}, Route: "new-model"}},
			},
		},
		Providers: map[string]ProviderConfig{"new-model": {Type: "custom"}},
	}

	require.NoError(t, router.Reload(newCfg))

	result, err := router.Route(context.Background(), &RouteRequest{TaskType: "reasoning"})
	require.NoError(t, err)
	assert.Equal(t, "new-model", result.ModelID)
	assert.Equal(t, "custom", result.ProviderName)
}

// TestRoute_RecordsMetrics verifies that Route runs without panicking when
// Prometheus metrics are recorded (label cardinality must match registration).
func TestRoute_RecordsMetrics(t *testing.T) {
	router, err := NewDefaultRouter(testConfig)
	require.NoError(t, err)

	// Any matching request causes ModelRouteDecisions and ModelRouteLatency to be recorded.
	result, err := router.Route(context.Background(), &RouteRequest{TaskType: "reasoning"})
	require.NoError(t, err)
	assert.NotEmpty(t, result.ModelID)
}

// TestRecordModelLatency verifies that RecordModelLatency does not panic.
func TestRecordModelLatency(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RecordModelLatency panicked: %v", r)
		}
	}()
	RecordModelLatency("gpt-4o", "openai", 250*time.Millisecond)
}

// TestNoMatchReturnsError verifies that Route returns an error when all
// strategies fail to match and have no fallback configured.
func TestNoMatchReturnsError(t *testing.T) {
	cfg := &RouterConfig{
		Strategies: []StrategyConfig{
			{
				Name:  "capability-based",
				Rules: []RuleConfig{{Match: map[string][]string{"task": {"reasoning"}}, Route: "deepseek-r1"}},
				// No fallback.
			},
		},
	}
	router, err := NewDefaultRouter(cfg)
	require.NoError(t, err)

	_, err = router.Route(context.Background(), &RouteRequest{TaskType: "unknown"})
	assert.Error(t, err)
}
