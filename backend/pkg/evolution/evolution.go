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

// Package evolution provides local experience self-evolution capability.
//
// Signal flow:
//
//	Eino callback (Tool/Model events)
//	  → SignalCollector.Collect()  (async goroutine)
//	    → LocalGeneStore.SaveGene()  → MySQL
//
// Query flow:
//
//	AgentBuilder → EvolutionAdvisor.Recommend()
//	  → LocalGeneStore.Search()  → MySQL
//	    → Recommendation slice injected into system prompt
package evolution

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Signal describes a single execution event observed during agent operation.
type Signal struct {
	// Type classifies the event: tool_success, tool_error, model_invoke, agent_done, node_done, etc.
	Type      string
	AgentName string
	SessionID string
	// Component is the tool name, model id, or node id that was involved.
	Component string
	// Input is a hash or short summary of the input.
	Input string
	// Output is a summary of the result (empty on error).
	Output string
	// Error holds the error message when the event represents a failure.
	Error     string
	Duration  time.Duration
	Metadata  map[string]any
	Timestamp time.Time
}

// Recommendation is a Gene suggestion returned by EvolutionAdvisor.
type Recommendation struct {
	GeneID      string
	Strategy    any
	Confidence  float64
	UseCount    int
	SuccessRate float64
}

// Engine is the top-level facade for the evolution capability.
// Obtain one via Init(); a nil *Engine is safe (all methods are no-ops).
type Engine struct {
	store     *LocalGeneStore
	collector *SignalCollector
	advisor   *EvolutionAdvisor
	cfg       Config
}

// Init creates and warms up an Engine backed by the provided *gorm.DB.
// Returns nil, nil when cfg.Enabled is false so callers can treat nil as "disabled".
func Init(_ context.Context, cfg Config, db *gorm.DB) (*Engine, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if db == nil {
		return nil, fmt.Errorf("evolution: *gorm.DB is required when EVOLUTION_ENABLED=true")
	}

	store, err := NewLocalGeneStore(db)
	if err != nil {
		return nil, fmt.Errorf("evolution: init store: %w", err)
	}

	e := &Engine{
		store: store,
		cfg:   cfg,
	}
	e.collector = newSignalCollector(store)
	e.advisor = newEvolutionAdvisor(store, cfg.MinConfidence, cfg.MaxSuggestions)

	return e, nil
}

// Shutdown is a no-op in local mode. Kept for API compatibility.
func (e *Engine) Shutdown() {}

// Collector returns the SignalCollector for direct signal submission.
func (e *Engine) Collector() *SignalCollector {
	if e == nil {
		return nil
	}
	return e.collector
}

// Advisor returns the EvolutionAdvisor for querying Gene recommendations.
func (e *Engine) Advisor() *EvolutionAdvisor {
	if e == nil {
		return nil
	}
	return e.advisor
}

// Store returns the underlying LocalGeneStore for direct queries (admin API).
func (e *Engine) Store() *LocalGeneStore {
	if e == nil {
		return nil
	}
	return e.store
}

// Config returns the effective configuration.
func (e *Engine) Config() Config {
	if e == nil {
		return Config{}
	}
	return e.cfg
}
