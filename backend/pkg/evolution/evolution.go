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

// Package evolution integrates the Oris Go SDK to give superagent-base
// experience self-evolution capability.
//
// Signal flow:
//
//	Eino callback (Tool/Model events)
//	  → SignalCollector.Collect()  (async goroutine)
//	    → experience.Client.Share()  → Oris Experience Repo
//
// Query flow (Phase 2):
//
//	AgentBuilder → EvolutionAdvisor.Recommend()
//	  → experience.Client.Fetch()  → Oris Experience Repo
//	    → Recommendation slice injected into system prompt
//
// Federation (Phase 3):
//
//	Hub client registers this node and sends periodic heartbeats.
//	FederatedSearch queries genes across all connected nodes.
package evolution

import (
	"context"
	"fmt"
	"time"

	experienceclient "github.com/Colin4k1024/Oris/sdks/go/experience"
	hubclient "github.com/Colin4k1024/Oris/sdks/go/hub"
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
	expClient *experienceclient.Client
	hubClient *hubclient.Client // nil when HubURL is empty
	collector *SignalCollector
	advisor   *EvolutionAdvisor
	cfg       Config
	stopHub   context.CancelFunc // cancels the hub heartbeat goroutine
}

// Init creates and warms up an Engine from cfg.
// Returns nil, nil when cfg.Enabled is false so callers can treat nil as "disabled".
func Init(ctx context.Context, cfg Config) (*Engine, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if cfg.ExperienceURL == "" {
		return nil, fmt.Errorf("evolution: ORIS_EXPERIENCE_URL is required when EVOLUTION_ENABLED=true")
	}

	expClient := experienceclient.NewClient(experienceclient.Config{
		BaseURL:  cfg.ExperienceURL,
		APIKey:   cfg.APIKey,
		Seed:     cfg.Seed,
		SenderID: cfg.SenderID,
	})

	// Best-effort public key registration; failure is non-fatal.
	_ = expClient.RegisterPublicKey(ctx)

	e := &Engine{
		expClient: expClient,
		cfg:       cfg,
	}
	e.collector = newSignalCollector(expClient)
	e.advisor = newEvolutionAdvisor(expClient, cfg.MinConfidence, cfg.MaxSuggestions)

	// Phase 3: initialise Hub client when HubURL is configured.
	if cfg.HubURL != "" {
		e.hubClient = hubclient.NewClient(hubclient.Config{
			BaseURL: cfg.HubURL,
			APIKey:  cfg.APIKey,
			Seed:    cfg.Seed,
			NodeID:  cfg.SenderID,
		})
		if err := e.registerAndStartHeartbeat(ctx, cfg); err != nil {
			// Non-fatal — federation is optional.
			e.hubClient = nil
		}
	}

	return e, nil
}

// registerAndStartHeartbeat registers this node with the Hub and starts the
// background heartbeat goroutine.
func (e *Engine) registerAndStartHeartbeat(ctx context.Context, cfg Config) error {
	resp, err := e.hubClient.Register(ctx, &hubclient.RegisterRequest{
		NodeID:       cfg.SenderID,
		Endpoint:     cfg.NodeEndpoint,
		Capabilities: []string{"evolve", "experience"},
		Version:      "0.1.0",
	})
	if err != nil {
		return fmt.Errorf("hub register: %w", err)
	}

	interval := time.Duration(resp.HeartbeatIntervalSeconds) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}

	hbCtx, cancel := context.WithCancel(context.Background())
	e.stopHub = cancel

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-ticker.C:
				_, _ = e.hubClient.Heartbeat(hbCtx, hubclient.NodeStatusActive)
			}
		}
	}()

	return nil
}

// FederatedSearch queries genes across all Hub-connected nodes.
// Returns nil when Hub is not configured.
func (e *Engine) FederatedSearch(ctx context.Context, query string, minConfidence float64, limit int) ([]hubclient.GeneResult, error) {
	if e == nil || e.hubClient == nil {
		return nil, nil
	}
	result, err := e.hubClient.Search(ctx, &hubclient.FederatedQuery{
		Query:         query,
		MinConfidence: minConfidence,
		Limit:         limit,
	})
	if err != nil {
		return nil, err
	}
	return result.Results, nil
}

// DiscoverNodes returns peer nodes registered with the Hub.
func (e *Engine) DiscoverNodes(ctx context.Context) ([]hubclient.NodeInfo, error) {
	if e == nil || e.hubClient == nil {
		return nil, nil
	}
	result, err := e.hubClient.Discover(ctx, &hubclient.DiscoveryQuery{
		Capabilities: []string{"evolve"},
	})
	if err != nil {
		return nil, err
	}
	return result.Nodes, nil
}

// Shutdown stops the heartbeat goroutine gracefully.
func (e *Engine) Shutdown() {
	if e == nil || e.stopHub == nil {
		return
	}
	e.stopHub()
}

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

// Config returns the effective configuration.
func (e *Engine) Config() Config {
	if e == nil {
		return Config{}
	}
	return e.cfg
}
