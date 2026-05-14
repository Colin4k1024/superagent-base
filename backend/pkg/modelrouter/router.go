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

// Package modelrouter provides a rule-based model routing layer that sits on
// top of the modelbuilder system. Callers supply a RouteRequest describing
// their workload; the router returns the best-matching model ID plus an ordered
// fallback chain derived from the routing config.
package modelrouter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/superagent-ai/superagent-base/backend/pkg/observe"
)

// Router selects a model for a given RouteRequest.
type Router interface {
	// Route selects the primary model and its fallback chain.
	Route(ctx context.Context, req *RouteRequest) (*RouteResult, error)
	// Reload replaces the running configuration atomically.
	Reload(config *RouterConfig) error
}

// RouteRequest carries the workload descriptor used to select a model.
type RouteRequest struct {
	// TaskType describes the kind of work (e.g. "reasoning", "coding", "fast-chat").
	TaskType string
	// Complexity is the estimated request complexity ("low", "medium", "high").
	Complexity string
	// AgentID is an optional caller identifier for future per-agent overrides.
	AgentID string
	// Metadata holds arbitrary key-value pairs for custom strategy extensions.
	Metadata map[string]string
}

// RouteResult is the outcome of a routing decision.
type RouteResult struct {
	// ModelID is the primary selected model.
	ModelID string
	// ProviderName is the provider type string from the routing config.
	ProviderName string
	// Fallbacks is an ordered list of fallback model IDs to try on failure.
	Fallbacks []string
}

// DefaultRouter is the production Router implementation. It holds an ordered
// slice of strategies; Route() tries each strategy in order and returns on the
// first successful match.
type DefaultRouter struct {
	mu         sync.RWMutex
	strategies []Strategy
	providers  map[string]ProviderConfig
}

// NewDefaultRouter constructs a DefaultRouter from a RouterConfig.
func NewDefaultRouter(cfg *RouterConfig) (*DefaultRouter, error) {
	r := &DefaultRouter{}
	if err := r.Reload(cfg); err != nil {
		return nil, err
	}
	return r, nil
}

// Reload replaces the running configuration atomically. Safe to call concurrently.
func (r *DefaultRouter) Reload(cfg *RouterConfig) error {
	if cfg == nil {
		return fmt.Errorf("modelrouter: config must not be nil")
	}

	strategies := make([]Strategy, 0, len(cfg.Strategies))
	for _, sc := range cfg.Strategies {
		s := newStrategyFromConfig(sc)
		strategies = append(strategies, s)
	}

	providers := make(map[string]ProviderConfig, len(cfg.Providers))
	for k, v := range cfg.Providers {
		providers[k] = v
	}

	r.mu.Lock()
	r.strategies = strategies
	r.providers = providers
	r.mu.Unlock()
	return nil
}

// Route iterates over strategies in order and returns on the first match.
// It collects all strategy fallbacks into RouteResult.Fallbacks.
func (r *DefaultRouter) Route(ctx context.Context, req *RouteRequest) (*RouteResult, error) {
	r.mu.RLock()
	strategies := r.strategies
	providers := r.providers
	r.mu.RUnlock()

	var fallbacks []string
	for _, s := range strategies {
		strategyName := s.Name()
		start := time.Now()
		modelID, err := s.Select(ctx, req)
		elapsed := time.Since(start)

		observe.ModelRouteLatency.WithLabelValues(strategyName).Observe(elapsed.Seconds())

		if err != nil {
			continue
		}

		observe.ModelRouteDecisions.WithLabelValues(strategyName, modelID).Inc()

		// Collect fallbacks from remaining strategies.
		fallbacks = buildFallbacks(ctx, req, strategies, strategyName)

		providerName := ""
		if pc, ok := providers[modelID]; ok {
			providerName = pc.Type
		}

		return &RouteResult{
			ModelID:      modelID,
			ProviderName: providerName,
			Fallbacks:    fallbacks,
		}, nil
	}

	return nil, fmt.Errorf("modelrouter: no strategy matched the request")
}

// buildFallbacks collects one fallback model ID from each strategy that was not
// the primary selector. Strategies that return an error are silently skipped.
func buildFallbacks(ctx context.Context, req *RouteRequest, strategies []Strategy, primaryName string) []string {
	var fallbacks []string
	for _, s := range strategies {
		if s.Name() == primaryName {
			continue
		}
		modelID, err := s.Select(ctx, req)
		if err != nil {
			continue
		}
		fallbacks = append(fallbacks, modelID)
	}
	return fallbacks
}

// newStrategyFromConfig creates the appropriate Strategy concrete type from a
// StrategyConfig. The name convention drives the type selection; unknown names
// default to a generic rule-based strategy.
func newStrategyFromConfig(sc StrategyConfig) Strategy {
	switch sc.Name {
	case "capability-based":
		return NewCapabilityStrategy(sc)
	case "cost-optimized":
		return NewCostStrategy(sc)
	case "latency-optimized":
		return NewLatencyStrategy(sc)
	default:
		// Generic rule-based strategy for any other named strategy.
		return &ruleBasedStrategy{
			name:     sc.Name,
			rules:    sc.Rules,
			fallback: sc.Fallback,
		}
	}
}
