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
	"fmt"
)

// Strategy is the core routing abstraction. Each implementation selects a
// model ID given a RouteRequest; it returns ErrNoMatch when no rule applies.
type Strategy interface {
	Select(ctx context.Context, req *RouteRequest) (modelID string, err error)
	Name() string
}

// ErrNoMatch is returned by a Strategy when no rule matches the request.
var ErrNoMatch = fmt.Errorf("modelrouter: no matching rule")

// ruleBasedStrategy evaluates an ordered list of RuleConfigs against a
// RouteRequest. The first rule whose match constraints are all satisfied wins.
// If no rule matches, it falls back to the configured fallback model ID.
type ruleBasedStrategy struct {
	name     string
	rules    []RuleConfig
	fallback string
}

func (s *ruleBasedStrategy) Name() string { return s.name }

func (s *ruleBasedStrategy) Select(_ context.Context, req *RouteRequest) (string, error) {
	for _, rule := range s.rules {
		if ruleMatches(rule, req) {
			return rule.Route, nil
		}
	}
	if s.fallback != "" {
		return s.fallback, nil
	}
	return "", ErrNoMatch
}

// ruleMatches returns true when every field constraint in rule.Match is
// satisfied by the values present in req.
func ruleMatches(rule RuleConfig, req *RouteRequest) bool {
	for field, allowed := range rule.Match {
		value := requestField(req, field)
		if !containsString(allowed, value) {
			return false
		}
	}
	return true
}

// requestField extracts a named field from RouteRequest.
// Supported: "task" → TaskType, "complexity" → Complexity.
// Unknown fields are treated as empty string (never matches a non-empty allowed list).
func requestField(req *RouteRequest, field string) string {
	switch field {
	case "task":
		return req.TaskType
	case "complexity":
		return req.Complexity
	default:
		return ""
	}
}

func containsString(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

// CapabilityStrategy routes by TaskType (e.g. "reasoning" → deepseek-r1).
// It is a named wrapper around ruleBasedStrategy for clarity at the call site.
type CapabilityStrategy struct{ ruleBasedStrategy }

// NewCapabilityStrategy constructs a CapabilityStrategy from a StrategyConfig.
func NewCapabilityStrategy(cfg StrategyConfig) *CapabilityStrategy {
	return &CapabilityStrategy{ruleBasedStrategy{
		name:     cfg.Name,
		rules:    cfg.Rules,
		fallback: cfg.Fallback,
	}}
}

// CostStrategy routes by Complexity (e.g. "low" → cheap model).
type CostStrategy struct{ ruleBasedStrategy }

// NewCostStrategy constructs a CostStrategy from a StrategyConfig.
func NewCostStrategy(cfg StrategyConfig) *CostStrategy {
	return &CostStrategy{ruleBasedStrategy{
		name:     cfg.Name,
		rules:    cfg.Rules,
		fallback: cfg.Fallback,
	}}
}

// LatencyStrategy routes to the lowest-latency provider. This implementation
// delegates to an inner strategy but is a named extension point so callers can
// swap in a latency-aware implementation later.
type LatencyStrategy struct{ ruleBasedStrategy }

// NewLatencyStrategy constructs a LatencyStrategy from a StrategyConfig.
func NewLatencyStrategy(cfg StrategyConfig) *LatencyStrategy {
	return &LatencyStrategy{ruleBasedStrategy{
		name:     cfg.Name,
		rules:    cfg.Rules,
		fallback: cfg.Fallback,
	}}
}

// FallbackChain wraps a primary Strategy and, on ErrNoMatch, tries each
// fallback model ID in order, returning the first non-empty one.
type FallbackChain struct {
	primary   Strategy
	fallbacks []string
}

// NewFallbackChain constructs a FallbackChain.
func NewFallbackChain(primary Strategy, fallbacks ...string) *FallbackChain {
	return &FallbackChain{primary: primary, fallbacks: fallbacks}
}

func (fc *FallbackChain) Name() string { return fc.primary.Name() }

// Select tries the primary strategy first; on ErrNoMatch it returns the first
// fallback model ID, or ErrNoMatch when no fallbacks are configured.
func (fc *FallbackChain) Select(ctx context.Context, req *RouteRequest) (string, error) {
	modelID, err := fc.primary.Select(ctx, req)
	if err == nil {
		return modelID, nil
	}
	if len(fc.fallbacks) > 0 {
		return fc.fallbacks[0], nil
	}
	return "", ErrNoMatch
}
