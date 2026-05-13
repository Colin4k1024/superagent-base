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

package agentdef

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/superagent-ai/superagent-base/backend/pkg/logs"
	"github.com/superagent-ai/superagent-base/backend/pkg/observe"
)

// ModelRuntimeConfig holds the LLM endpoint configuration.
type ModelRuntimeConfig struct {
	BaseURL string // LLM API endpoint
	APIKey  string // API key
	ModelID string // Model identifier
	Type    string // "openai" (default) or "claude" for Anthropic protocol
}

// RuntimeConfig is the configuration for AgentRuntime.
type RuntimeConfig struct {
	// ConfigDir is the path to the directory containing agent YAML files.
	ConfigDir string
	// ModelConfig holds the default LLM endpoint settings.
	ModelConfig ModelRuntimeConfig
}

// AgentRuntime manages the lifecycle of all built agents.
// It loads YAML definitions, builds Agent instances, and keeps them in sync
// with the filesystem via the Reloader/Watcher.
type AgentRuntime struct {
	mu           sync.RWMutex
	agents       map[string]Agent
	builder      *AgentBuilder
	reloader     *Reloader
	watcher      *Watcher
	configDir    string
	startTime    time.Time
	lastReloadAt time.Time
}

// NewRuntime creates an AgentRuntime.  Call Start to populate and activate it.
func NewRuntime(cfg RuntimeConfig, builder *AgentBuilder) *AgentRuntime {
	rt := &AgentRuntime{
		agents:    make(map[string]Agent),
		builder:   builder,
		configDir: cfg.ConfigDir,
		startTime: time.Now(),
	}
	// Register a registry resolver on the builder so orchestration agents can
	// look up already-loaded sub-agents by name at build time.
	builder.agentRegistry = func(name string) (Agent, bool) {
		rt.mu.RLock()
		defer rt.mu.RUnlock()
		a, ok := rt.agents[name]
		return a, ok
	}
	return rt
}

// Start loads all agent YAMLs from ConfigDir, builds each agent, then starts
// a filesystem watcher so changes are picked up automatically.
func (rt *AgentRuntime) Start(ctx context.Context) error {
	reloader, err := NewReloader(ctx, rt.configDir)
	if err != nil {
		return fmt.Errorf("agentdef runtime: start reloader: %w", err)
	}
	rt.reloader = reloader

	// Build all initially loaded definitions.
	if err := rt.buildAll(ctx, reloader.List(), reloader); err != nil {
		return err
	}

	// Subscribe to hot-reload events.  On any add/update, do a full two-pass
	// rebuild of all loaded definitions so orchestration agents can resolve
	// sub-agents that may have been added in the same batch.  buildAll is
	// idempotent; the watcher debounces filesystem events so rebuilds are infrequent.
	reloader.OnChange(func(evt ChangeEvent) {
		if evt.Type == ChangeDeleted {
			rt.mu.Lock()
			delete(rt.agents, evt.AgentName)
			rt.mu.Unlock()
			logs.Infof("agentdef runtime: agent %q removed", evt.AgentName)
			return
		}

		names := reloader.List()
		if err := rt.buildAll(ctx, names, reloader); err != nil {
			logs.Warnf("agentdef runtime: hot-reload buildAll: %v", err)
		}
	})

	// Start filesystem watcher in background.
	watcher, err := NewWatcher(rt.configDir, reloader)
	if err != nil {
		// Non-fatal: hot-reload simply won't work without a watcher.
		logs.Warnf("agentdef runtime: could not start watcher: %v", err)
	} else {
		rt.watcher = watcher
		go watcher.Start(ctx)
	}

	return nil
}

// GetAgent returns the built Agent for the given name.
func (rt *AgentRuntime) GetAgent(name string) (Agent, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	a, ok := rt.agents[name]
	return a, ok
}

// ListAgents returns the names of all currently loaded agents.
func (rt *AgentRuntime) ListAgents() []string {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	names := make([]string, 0, len(rt.agents))
	for n := range rt.agents {
		names = append(names, n)
	}
	return names
}

// Stop releases watcher resources.  In-flight Chat calls are not interrupted.
func (rt *AgentRuntime) Stop() {
	if rt.watcher != nil {
		_ = rt.watcher.Close()
	}
}

// StartTime returns when the runtime was created.
func (rt *AgentRuntime) StartTime() time.Time {
	return rt.startTime
}

// LastReloadAt returns the timestamp of the most recent successful reload.
func (rt *AgentRuntime) LastReloadAt() time.Time {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.lastReloadAt
}

// Reload triggers a full reload of all agent definitions from the config directory.
func (rt *AgentRuntime) Reload(ctx context.Context) error {
	if rt.reloader == nil {
		return fmt.Errorf("reloader not initialized")
	}
	if err := rt.reloader.ReloadDir(ctx, rt.configDir); err != nil {
		return err
	}
	rt.mu.Lock()
	rt.lastReloadAt = time.Now()
	rt.mu.Unlock()
	return nil
}

// AgentInfo holds basic information about a loaded agent for status reporting.
type AgentInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
}

// AgentInfoList returns metadata about all loaded agents.
func (rt *AgentRuntime) AgentInfoList() []AgentInfo {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	infos := make([]AgentInfo, 0, len(rt.agents))
	for name, agent := range rt.agents {
		agentType := "unknown"
		if def := agent.GetDefinition(); def != nil {
			agentType = def.Spec.Type
		}
		infos = append(infos, AgentInfo{
			Name:        name,
			Type:        agentType,
			Status:      "ok",
			Description: agent.Description(),
		})
	}
	return infos
}

// buildAll constructs Agent instances for every name in names from loader.
//
// It performs a two-pass build so orchestration agents can reference leaf
// agents that were built in the first pass:
//
//  1. Build all non-orchestration (leaf) agents.
//  2. Build orchestration agents; sub-agent lookups resolve via the registry
//     that was wired to the builder in NewRuntime.
func (rt *AgentRuntime) buildAll(ctx context.Context, names []string, loader interface {
	Get(string) (*AgentDefinition, bool)
}) error {
	built := make(map[string]Agent, len(names))

	// Collect all definitions first so we can separate leaf from orchestration.
	defs := make(map[string]*AgentDefinition, len(names))
	for _, name := range names {
		def, ok := loader.Get(name)
		if !ok {
			continue
		}
		defs[name] = def
	}

	// Pass 1: build leaf agents (non-orchestration types).
	var buildErrors []string
	for name, def := range defs {
		if _, isOrch := orchestrationTypes[def.Spec.Type]; isOrch {
			continue
		}
		agent, err := rt.builder.Build(ctx, def)
		if err != nil {
			// Log warning and record metric; keep existing agent intact.
			logs.Warnf("agentdef runtime: build %q failed (keeping old): %v", name, err)
			observe.AgentReloadFailures.WithLabelValues(name).Inc()
			buildErrors = append(buildErrors, name)
			continue
		}
		built[name] = agent
		logs.Infof("agentdef runtime: built agent %q (model=%s)", name, def.Spec.Model.Primary)
	}

	// Incremental merge: only update successfully built agents; preserve
	// existing agents that failed to rebuild (H8 fix).
	rt.mu.Lock()
	for name, agent := range built {
		rt.agents[name] = agent
	}
	// Remove agents that were deleted from YAML (present in old map, absent from defs).
	for name := range rt.agents {
		if _, inDefs := defs[name]; !inDefs {
			delete(rt.agents, name)
		}
	}
	rt.mu.Unlock()

	// Pass 2: build orchestration agents (can now resolve sub-agents via registry).
	for name, def := range defs {
		if _, isOrch := orchestrationTypes[def.Spec.Type]; !isOrch {
			continue
		}
		agent, err := rt.builder.Build(ctx, def)
		if err != nil {
			logs.Warnf("agentdef runtime: build orchestration %q failed (keeping old): %v", name, err)
			observe.AgentReloadFailures.WithLabelValues(name).Inc()
			buildErrors = append(buildErrors, name)
			continue
		}
		rt.mu.Lock()
		rt.agents[name] = agent
		rt.mu.Unlock()
		logs.Infof("agentdef runtime: built orchestration agent %q (type=%s)", name, def.Spec.Type)
	}

	if len(buildErrors) > 0 {
		logs.Warnf("agentdef runtime: %d agent(s) failed to build: %v", len(buildErrors), buildErrors)
	}
	return nil
}
