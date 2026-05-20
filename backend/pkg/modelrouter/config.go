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

// RouterConfig is the top-level routing configuration loaded from YAML.
type RouterConfig struct {
	Strategies []StrategyConfig          `yaml:"strategies"`
	Providers  map[string]ProviderConfig `yaml:"providers"`
	Feedback   *FeedbackConfig           `yaml:"feedback,omitempty"`
}

// StrategyConfig defines a named routing strategy with ordered rules.
type StrategyConfig struct {
	Name     string       `yaml:"name"`
	Rules    []RuleConfig `yaml:"rules"`
	Fallback string       `yaml:"fallback"`
}

// RuleConfig is a single routing rule: if all match fields satisfy the request,
// route to the specified model ID.
type RuleConfig struct {
	// Match maps RouteRequest field names to a list of accepted values.
	// Supported fields: "task" (TaskType), "complexity" (Complexity).
	Match map[string][]string `yaml:"match"`
	// Route is the model ID to select when this rule matches.
	Route string `yaml:"route"`
}

// ProviderConfig holds provider-level metadata for a model ID.
type ProviderConfig struct {
	Type     string      `yaml:"type"`
	Endpoint string      `yaml:"endpoint,omitempty"`
	Pricing  PricingInfo `yaml:"pricing,omitempty"`
}

// PricingInfo holds the per-1k-token cost for a provider (USD).
type PricingInfo struct {
	InputPer1K  float64 `yaml:"input_per_1k"`
	OutputPer1K float64 `yaml:"output_per_1k"`
}

// FeedbackConfig controls the real-time feedback subsystem.
// When Enabled is false (default), AdaptiveStrategy is not registered and the
// feedback path is entirely bypassed with zero overhead.
type FeedbackConfig struct {
	Enabled       bool                 `yaml:"enabled"`
	EMAAlpha      float64              `yaml:"ema_alpha,omitempty"`
	MinSamples    int                  `yaml:"min_samples,omitempty"`
	StaleDuration string               `yaml:"stale_duration,omitempty"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker,omitempty"`
}
