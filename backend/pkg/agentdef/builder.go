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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	einoark "github.com/cloudwego/eino-ext/components/model/ark"
	einoclaude "github.com/cloudwego/eino-ext/components/model/claude"
	einodeepseek "github.com/cloudwego/eino-ext/components/model/deepseek"
	einoollama "github.com/cloudwego/eino-ext/components/model/ollama"
	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"github.com/superagent-ai/superagent-base/backend/infra/cache"
	"github.com/superagent-ai/superagent-base/backend/infra/checkpoint"
	"github.com/superagent-ai/superagent-base/backend/pkg/evolution"
	"github.com/superagent-ai/superagent-base/backend/pkg/graphs"
	"github.com/superagent-ai/superagent-base/backend/pkg/mcp"
	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
	"github.com/superagent-ai/superagent-base/backend/pkg/modelrouter"
	"github.com/superagent-ai/superagent-base/backend/pkg/observe"
	"github.com/superagent-ai/superagent-base/backend/pkg/skill"
	"github.com/superagent-ai/superagent-base/backend/pkg/tool"
)

// Agent is the runtime interface for a built agent.
type Agent interface {
	// Name returns the agent's unique name from its YAML definition.
	Name() string
	// Description returns a human-readable description.
	Description() string
	// Chat sends a message and streams response chunks over the returned channel.
	// The channel is closed when the response is complete or an error occurs.
	Chat(ctx context.Context, sessionID string, message string) (<-chan string, error)
	// GetDefinition returns the AgentDefinition that produced this agent.
	GetDefinition() *AgentDefinition
}

// BuilderOption configures an AgentBuilder.
type BuilderOption func(*AgentBuilder)

// WithModelRouter sets the model router used to resolve model IDs.
func WithModelRouter(r modelrouter.Router) BuilderOption {
	return func(b *AgentBuilder) { b.modelRouter = r }
}

// WithToolManager sets the tool manager that resolves builtin tool references.
func WithToolManager(m *tool.Manager) BuilderOption {
	return func(b *AgentBuilder) { b.toolManager = m }
}

// WithMemoryFactory sets the factory used to create memory backends.
func WithMemoryFactory(f func(config memory.BackendConfig) (memory.Backend, error)) BuilderOption {
	return func(b *AgentBuilder) { b.memoryFactory = f }
}

// WithMCPRegistry sets the MCP registry used to resolve mcp:// tool references.
func WithMCPRegistry(r *mcp.Registry) BuilderOption {
	return func(b *AgentBuilder) { b.mcpRegistry = r }
}

// WithModelConfig sets the default LLM endpoint for all built agents.
func WithModelConfig(cfg ModelRuntimeConfig) BuilderOption {
	return func(b *AgentBuilder) { b.modelConfig = cfg }
}

// WithAgentRegistry sets the resolver used to look up sub-agents by name when
// building orchestration types (supervisor, sequential, parallel).
func WithAgentRegistry(fn func(name string) (Agent, bool)) BuilderOption {
	return func(b *AgentBuilder) { b.agentRegistry = fn }
}

// WithSkillManager sets the skill manager used to resolve skill:// tool references.
func WithSkillManager(mgr *skill.Manager) BuilderOption {
	return func(b *AgentBuilder) { b.skillManager = mgr }
}

// WithEvolutionAdvisor injects an EvolutionAdvisor so the builder can prepend
// Gene-based experience recommendations into agent system prompts.
func WithEvolutionAdvisor(a *evolution.EvolutionAdvisor) BuilderOption {
	return func(b *AgentBuilder) { b.evolutionAdvisor = a }
}

// WithEvolutionCollector injects a SignalCollector used for workflow node-level signals.
func WithEvolutionCollector(c *evolution.SignalCollector) BuilderOption {
	return func(b *AgentBuilder) { b.evolutionCollector = c }
}

// WithRedisClient sets the Redis client used for the redis checkpoint backend.
func WithRedisClient(client cache.Cmdable) BuilderOption {
	return func(b *AgentBuilder) { b.redisClient = client }
}

// AgentBuilder converts AgentDefinitions into running Agent instances.
type AgentBuilder struct {
	modelRouter         modelrouter.Router
	toolManager         *tool.Manager
	memoryFactory       func(config memory.BackendConfig) (memory.Backend, error)
	mcpRegistry         *mcp.Registry
	skillManager        *skill.Manager
	modelConfig         ModelRuntimeConfig
	agentRegistry       func(name string) (Agent, bool)
	redisClient         cache.Cmdable
	evolutionAdvisor    *evolution.EvolutionAdvisor
	evolutionCollector  *evolution.SignalCollector
}

// NewAgentBuilder creates an AgentBuilder with optional configuration.
func NewAgentBuilder(opts ...BuilderOption) *AgentBuilder {
	b := &AgentBuilder{}
	for _, o := range opts {
		o(b)
	}
	return b
}

// Build creates an Agent from a validated AgentDefinition.
//
// When a ModelRuntimeConfig is provided via WithModelConfig, Build creates a
// real Eino ChatModel pointing at the configured LLM endpoint.  If tools are
// present it wraps the model in a ReAct agent; otherwise it calls the model
// directly.  Without a ModelRuntimeConfig the builder falls back to the stub
// implementation so existing tests continue to pass.
//
// For orchestration types (supervisor, sequential, parallel, plan_execute) the
// builder resolves sub-agent references via the registered agent registry.
//
// When spec.interrupt.enabled is true, the resulting agent is wrapped with
// NewInterruptableAgent so that confirmation-seeking model outputs are
// transparently captured and callers can Resume later.
func (b *AgentBuilder) Build(ctx context.Context, def *AgentDefinition) (Agent, error) {
	if def == nil {
		return nil, fmt.Errorf("agentdef: Build: def is nil")
	}

	// Dispatch orchestration types before resolving any model config.
	switch def.Spec.Type {
	case "supervisor":
		return b.buildSupervisor(ctx, def)
	case "sequential":
		return b.buildSequential(ctx, def)
	case "parallel":
		return b.buildParallel(ctx, def)
	case "plan_execute":
		return b.buildPlanExecute(ctx, def)
	case "workflow":
		return b.buildWorkflow(def)
	case "eino_graph":
		return b.buildEinoGraph(ctx, def)
	case "agentloop":
		return b.buildAgentLoop(ctx, def)
	}

	// deep_agent: prepend a step-by-step reasoning prefix to the system prompt.
	if def.Spec.Type == "deep_agent" {
		const deepReasoningPrefix = "You are in deep reasoning mode. Think step by step before answering. "
		def = shallowCopyDefWithSystemPrompt(def, deepReasoningPrefix+def.Spec.SystemPrompt)
	}

	// Evolution advisor: query Gene recommendations and prepend to system prompt.
	if b.evolutionAdvisor != nil && def.Spec.Evolution != nil && def.Spec.Evolution.Enabled {
		recs := b.evolutionAdvisor.Recommend(ctx, def.Metadata.Name+" "+def.Spec.SystemPrompt)
		if len(recs) > 0 {
			def = shallowCopyDefWithSystemPrompt(def, buildEvolutionPromptPrefix(recs)+def.Spec.SystemPrompt)
		}
	}

	// Resolve primary model ID (may be overridden by router).
	modelID := def.Spec.Model.Primary
	if b.modelRouter != nil && def.Spec.Model.Router != "" {
		result, err := b.modelRouter.Route(ctx, &modelrouter.RouteRequest{
			Metadata: map[string]string{"strategy": def.Spec.Model.Router},
		})
		if err == nil {
			modelID = result.ModelID
		}
	}

	// Resolve tools.
	toolRefs := make([]resolvedTool, 0, len(def.Spec.Tools))
	for _, ref := range def.Spec.Tools {
		rt, err := b.resolveToolRef(ref)
		if err != nil {
			return nil, fmt.Errorf("agentdef: Build: resolve tool %q: %w", ref.Ref, err)
		}
		toolRefs = append(toolRefs, rt)
	}

	// Resolve memory backend.
	var memBackend memory.Backend
	if b.memoryFactory != nil && def.Spec.Memory.Backend != "" {
		cfg := memory.BackendConfig{
			Type:    def.Spec.Memory.Backend,
			Options: def.Spec.Memory.Config,
		}
		var err error
		memBackend, err = b.memoryFactory(cfg)
		if err != nil {
			return nil, fmt.Errorf("agentdef: Build: init memory backend %q: %w", def.Spec.Memory.Backend, err)
		}
	}

	var built Agent

	// If no real model config is provided fall back to the stub agent so
	// existing unit tests that don't set up a model endpoint keep passing.
	if b.modelConfig.BaseURL == "" {
		built = &chatAgent{
			def:        def,
			modelID:    modelID,
			tools:      toolRefs,
			memBackend: memBackend,
		}
		return b.maybeWrapInterruptable(built, def), nil
	}

	// Build a real Eino ChatModel dispatched by provider protocol.
	effectiveModelID := modelID
	if effectiveModelID == "" {
		effectiveModelID = b.modelConfig.ModelID
	}

	// Resolve per-agent model config overrides.
	baseURL := b.modelConfig.BaseURL
	apiKey := b.modelConfig.APIKey
	protocol := b.modelConfig.Type // global default

	if def.Spec.Model.BaseURL != "" {
		baseURL = def.Spec.Model.BaseURL
	}
	if def.Spec.Model.APIKeyEnv != "" {
		if v := os.Getenv(def.Spec.Model.APIKeyEnv); v != "" {
			apiKey = v
		}
	}
	if def.Spec.Model.Protocol != "" {
		protocol = def.Spec.Model.Protocol
	}

	chatModel, err := b.createChatModel(ctx, protocol, baseURL, apiKey, effectiveModelID)
	if err != nil {
		return nil, fmt.Errorf("agentdef: Build %q: create model (protocol=%s): %w", def.Metadata.Name, protocol, err)
	}

	// Gather Eino-compatible tools from resolved refs.
	einoTools := b.resolveEinoTools(ctx, toolRefs)

	if len(einoTools) > 0 {
		// ReAct agent with tool calling.
		reactAgent, err := react.NewAgent(ctx, &react.AgentConfig{
			ToolCallingModel: chatModel,
			ToolsConfig: compose.ToolsNodeConfig{
				Tools: einoTools,
			},
			MaxStep:                10,
			StreamToolCallChecker: fullStreamToolCallChecker,
		})
		if err != nil {
			return nil, fmt.Errorf("agentdef: Build: create react agent: %w", err)
		}
		built = &einoReactAgent{
			def:          def,
			modelID:      effectiveModelID,
			provider:     protocol,
			memBackend:   memBackend,
			agent:        reactAgent,
			systemPrompt: def.Spec.SystemPrompt,
		}
	} else {
		// Simple chat agent without tools.
		built = &einoChatAgent{
			def:          def,
			modelID:      effectiveModelID,
			provider:     protocol,
			memBackend:   memBackend,
			chatModel:    chatModel,
			systemPrompt: def.Spec.SystemPrompt,
		}
	}

	// Apply middleware wrapping (timeout, retry, rate_limit, cache).
	built = b.applyMiddleware(built, def)

	// Apply fallback wrapping after middleware but before interrupt.
	if def.Spec.Model.Fallback != "" {
		fallbackModelID := def.Spec.Model.Fallback
		fallbackModel, fbErr := b.createChatModel(ctx, protocol, baseURL, apiKey, fallbackModelID)
		if fbErr != nil {
			return nil, fmt.Errorf("agentdef: Build %q: create fallback model: %w", def.Metadata.Name, fbErr)
		}
		var fbAgent Agent
		if len(einoTools) > 0 {
			fbReactAgent, fbReactErr := react.NewAgent(ctx, &react.AgentConfig{
				ToolCallingModel: fallbackModel,
				ToolsConfig: compose.ToolsNodeConfig{
					Tools: einoTools,
				},
				MaxStep:                10,
				StreamToolCallChecker: fullStreamToolCallChecker,
			})
			if fbReactErr != nil {
				return nil, fmt.Errorf("agentdef: Build: create fallback react agent: %w", fbReactErr)
			}
			fbAgent = &einoReactAgent{
				def:          def,
				modelID:      fallbackModelID,
				provider:     protocol,
				memBackend:   memBackend,
				agent:        fbReactAgent,
				systemPrompt: def.Spec.SystemPrompt,
			}
		} else {
			fbAgent = &einoChatAgent{
				def:          def,
				modelID:      fallbackModelID,
				provider:     protocol,
				memBackend:   memBackend,
				chatModel:    fallbackModel,
				systemPrompt: def.Spec.SystemPrompt,
			}
		}
		// Apply the same middleware to the fallback agent.
		fbAgent = b.applyMiddleware(fbAgent, def)
		built = &fallbackAgent{primary: built, fallback: fbAgent}
	}

	return b.maybeWrapInterruptable(built, def), nil
}

// maybeWrapInterruptable wraps agent with InterruptableAgent when the
// definition's interrupt config is present and enabled.
func (b *AgentBuilder) maybeWrapInterruptable(agent Agent, def *AgentDefinition) Agent {
	ic := def.Spec.Interrupt
	if ic == nil || !ic.Enabled {
		return agent
	}

	var store CheckpointStore
	switch ic.CheckpointBackend {
	case "redis":
		if b.redisClient != nil {
			store = checkpoint.NewRedisStore(b.redisClient)
		} else {
			store = checkpoint.NewInMemoryStore()
		}
	default:
		store = checkpoint.NewInMemoryStore()
	}

	timeout := time.Duration(ic.TimeoutSeconds) * time.Second
	return NewInterruptableAgent(agent, store, timeout)
}

// applyMiddleware wraps the agent with middleware components from spec.middleware[]
// and spec.observability settings.
// Supported middleware: timeout, retry. Others are silently ignored (reserved for future).
func (b *AgentBuilder) applyMiddleware(agent Agent, def *AgentDefinition) Agent {
	result := agent

	// Apply observability wrapper first (outermost = measures total latency).
	obs := def.Spec.Observability
	if obs.Metrics || obs.Tracing {
		result = &observedAgent{inner: result, enableMetrics: obs.Metrics}
	}

	// Apply middleware pipeline.
	for _, mw := range def.Spec.Middleware {
		switch mw.Name {
		case "timeout":
			seconds := 30 // default
			if v, ok := mw.Config["seconds"]; ok {
				switch t := v.(type) {
				case int:
					seconds = t
				case float64:
					seconds = int(t)
				}
			}
			result = &timeoutAgent{inner: result, timeout: time.Duration(seconds) * time.Second}

		case "retry":
			maxAttempts := 3
			if v, ok := mw.Config["max_attempts"]; ok {
				switch t := v.(type) {
				case int:
					maxAttempts = t
				case float64:
					maxAttempts = int(t)
				}
			}
			result = &retryAgent{inner: result, maxAttempts: maxAttempts}

		case "rate_limit":
			rpm := 60 // default requests per minute
			if v, ok := mw.Config["requests_per_minute"]; ok {
				switch t := v.(type) {
				case int:
					rpm = t
				case float64:
					rpm = int(t)
				}
			}
			result = &rateLimitAgent{
				inner: result,
				rpm:   rpm,
				times: make([]time.Time, 0, rpm),
			}

		case "cache":
			ttlSeconds := 300 // default 5 minutes
			if v, ok := mw.Config["ttl_seconds"]; ok {
				switch t := v.(type) {
				case int:
					ttlSeconds = t
				case float64:
					ttlSeconds = int(t)
				}
			}
			result = &cacheAgent{
				inner: result,
				ttl:   time.Duration(ttlSeconds) * time.Second,
				cache: make(map[string]cacheEntry),
			}
		}
	}
	return result
}

// observedAgent wraps an Agent with Prometheus metrics recording.
type observedAgent struct {
	inner         Agent
	enableMetrics bool
}

func (a *observedAgent) Name() string                    { return a.inner.Name() }
func (a *observedAgent) Description() string             { return a.inner.Description() }
func (a *observedAgent) GetDefinition() *AgentDefinition { return a.inner.GetDefinition() }

func (a *observedAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	start := time.Now()
	ch, err := a.inner.Chat(ctx, sessionID, message)
	if err != nil {
		if a.enableMetrics {
			observe.AgentRequestsTotal.WithLabelValues(a.inner.Name(), "error").Inc()
			observe.AgentRequestDuration.WithLabelValues(a.inner.Name()).Observe(time.Since(start).Seconds())
		}
		return nil, err
	}

	// Wrap channel to record completion metrics.
	out := make(chan string, 64)
	go func() {
		defer close(out)
		hadError := false
		for token := range ch {
			if strings.HasPrefix(token, "[error]") {
				hadError = true
			}
			select {
			case out <- token:
			case <-ctx.Done():
				if a.enableMetrics {
					observe.AgentRequestsTotal.WithLabelValues(a.inner.Name(), "cancelled").Inc()
					observe.AgentRequestDuration.WithLabelValues(a.inner.Name()).Observe(time.Since(start).Seconds())
				}
				return
			}
		}
		if a.enableMetrics {
			status := "success"
			if hadError {
				status = "error"
			}
			observe.AgentRequestsTotal.WithLabelValues(a.inner.Name(), status).Inc()
			observe.AgentRequestDuration.WithLabelValues(a.inner.Name()).Observe(time.Since(start).Seconds())
		}
	}()
	return out, nil
}

// timeoutAgent wraps an Agent with a per-request timeout.
type timeoutAgent struct {
	inner   Agent
	timeout time.Duration
}

func (a *timeoutAgent) Name() string                    { return a.inner.Name() }
func (a *timeoutAgent) Description() string             { return a.inner.Description() }
func (a *timeoutAgent) GetDefinition() *AgentDefinition { return a.inner.GetDefinition() }

func (a *timeoutAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	ch, err := a.inner.Chat(ctx, sessionID, message)
	if err != nil {
		cancel()
		return nil, err
	}
	// Wrap channel to cancel context after draining.
	out := make(chan string, 64)
	go func() {
		defer close(out)
		defer cancel()
		for token := range ch {
			select {
			case out <- token:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

// retryAgent wraps an Agent with retry-on-failure logic.
type retryAgent struct {
	inner       Agent
	maxAttempts int
}

func (a *retryAgent) Name() string                    { return a.inner.Name() }
func (a *retryAgent) Description() string             { return a.inner.Description() }
func (a *retryAgent) GetDefinition() *AgentDefinition { return a.inner.GetDefinition() }

func (a *retryAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	var lastErr error
	for attempt := 0; attempt < a.maxAttempts; attempt++ {
		ch, err := a.inner.Chat(ctx, sessionID, message)
		if err == nil {
			return ch, nil
		}
		lastErr = err
		// Exponential backoff: 100ms, 200ms, 400ms... — cancellable via ctx.
		backoff := time.Duration(100<<uint(attempt)) * time.Millisecond
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return nil, fmt.Errorf("agentdef: retry exhausted after %d attempts: %w", a.maxAttempts, lastErr)
}

// ─── fallback agent ─────────────────────────────────────────────────────────

// fallbackAgent wraps a primary Agent with a fallback Agent. If primary.Chat()
// returns an error, the fallback agent is tried instead.
type fallbackAgent struct {
	primary  Agent
	fallback Agent
}

func (a *fallbackAgent) Name() string                    { return a.primary.Name() }
func (a *fallbackAgent) Description() string             { return a.primary.Description() }
func (a *fallbackAgent) GetDefinition() *AgentDefinition { return a.primary.GetDefinition() }

func (a *fallbackAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	ch, err := a.primary.Chat(ctx, sessionID, message)
	if err != nil {
		// Primary failed; try fallback.
		return a.fallback.Chat(ctx, sessionID, message)
	}
	return ch, nil
}

// ─── rate limit agent ───────────────────────────────────────────────────────

// rateLimitAgent wraps an Agent with a simple token-bucket rate limiter based
// on request timestamps within a sliding window.
type rateLimitAgent struct {
	inner Agent
	rpm   int
	mu    sync.Mutex
	times []time.Time
}

func (a *rateLimitAgent) Name() string                    { return a.inner.Name() }
func (a *rateLimitAgent) Description() string             { return a.inner.Description() }
func (a *rateLimitAgent) GetDefinition() *AgentDefinition { return a.inner.GetDefinition() }

func (a *rateLimitAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	a.mu.Lock()
	now := time.Now()
	windowStart := now.Add(-time.Minute)

	// Evict timestamps outside the sliding window.
	valid := a.times[:0]
	for _, t := range a.times {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}
	a.times = valid

	if len(a.times) >= a.rpm {
		a.mu.Unlock()
		return nil, fmt.Errorf("agentdef: rate limit exceeded (%d requests/minute)", a.rpm)
	}
	a.times = append(a.times, now)
	a.mu.Unlock()

	return a.inner.Chat(ctx, sessionID, message)
}

// ─── cache agent ────────────────────────────────────────────────────────────

// cacheEntry holds a cached response with its expiration time.
type cacheEntry struct {
	tokens    []string
	expiresAt time.Time
}

// maxCacheEntries is the maximum number of cached responses retained in memory.
const maxCacheEntries = 1000

// cacheAgent wraps an Agent with an in-memory response cache keyed by
// SHA256(agentName + sessionID + message).
type cacheAgent struct {
	inner Agent
	ttl   time.Duration
	mu    sync.RWMutex
	cache map[string]cacheEntry
}

func (a *cacheAgent) Name() string                    { return a.inner.Name() }
func (a *cacheAgent) Description() string             { return a.inner.Description() }
func (a *cacheAgent) GetDefinition() *AgentDefinition { return a.inner.GetDefinition() }

func (a *cacheAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	key := cacheKey(a.inner.Name(), sessionID, message)

	// Check cache (read lock).
	a.mu.RLock()
	if entry, ok := a.cache[key]; ok && time.Now().Before(entry.expiresAt) {
		a.mu.RUnlock()
		ch := make(chan string, len(entry.tokens))
		for _, tok := range entry.tokens {
			ch <- tok
		}
		close(ch)
		return ch, nil
	}
	a.mu.RUnlock()

	// Cache miss: call inner agent.
	innerCh, err := a.inner.Chat(ctx, sessionID, message)
	if err != nil {
		return nil, err
	}

	// Wrap channel to collect tokens for caching.
	out := make(chan string, 64)
	go func() {
		defer close(out)
		var tokens []string
		for tok := range innerCh {
			select {
			case out <- tok:
				tokens = append(tokens, tok)
			case <-ctx.Done():
				return
			}
		}
		// Only cache successful responses (no error tokens).
		hasError := false
		for _, tok := range tokens {
			if strings.HasPrefix(tok, "[error]") {
				hasError = true
				break
			}
		}
		if !hasError {
			// Store in cache with eviction of expired/oldest entries.
			a.mu.Lock()
			now := time.Now()
			// Evict expired entries first.
			for k, e := range a.cache {
				if now.After(e.expiresAt) {
					delete(a.cache, k)
				}
			}
			// If still at capacity, evict the oldest entry.
			if len(a.cache) >= maxCacheEntries {
				var oldestKey string
				var oldestTime time.Time
				for k, e := range a.cache {
					if oldestKey == "" || e.expiresAt.Before(oldestTime) {
						oldestKey = k
						oldestTime = e.expiresAt
					}
				}
				if oldestKey != "" {
					delete(a.cache, oldestKey)
				}
			}
			a.cache[key] = cacheEntry{
				tokens:    tokens,
				expiresAt: time.Now().Add(a.ttl),
			}
			a.mu.Unlock()
		}
	}()
	return out, nil
}

// cacheKey produces a SHA256 hex digest from the agent name, session ID, and message.
// Fields are separated by a null byte to prevent collisions where field boundaries
// shift (e.g. ("a","bc","d") vs ("ab","c","d")).
func cacheKey(agentName, sessionID, message string) string {
	h := sha256.New()
	h.Write([]byte(agentName))
	h.Write([]byte{0})
	h.Write([]byte(sessionID))
	h.Write([]byte{0})
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// resolveEinoTools converts resolved tool refs to Eino einotool.BaseTool instances.
// Supports builtin (via tool.Manager), skill:// (via skill.Manager), and
// mcp:// (via MCP Registry + MCPToolAdapter) refs.
func (b *AgentBuilder) resolveEinoTools(ctx context.Context, refs []resolvedTool) []einotool.BaseTool {
	var result []einotool.BaseTool
	for _, ref := range refs {
		switch ref.scheme {
		case "builtin":
			if b.toolManager == nil {
				continue
			}
			t, ok := b.toolManager.Get(ref.target)
			if !ok {
				continue
			}
			result = append(result, t)
		case "skill":
			if b.skillManager == nil {
				continue
			}
			t, ok := b.skillManager.GetTool(ref.target)
			if !ok {
				continue
			}
			result = append(result, t)
		case "mcp":
			if b.mcpRegistry == nil {
				continue
			}
			// target format: "server-name/tool-name"
			parts := strings.SplitN(ref.target, "/", 2)
			if len(parts) != 2 {
				continue
			}
			serverName, toolName := parts[0], parts[1]
			client, ok := b.mcpRegistry.GetClient(serverName)
			if !ok {
				continue
			}
			tools, err := client.ListTools(ctx)
			if err != nil {
				continue
			}
			for _, td := range tools {
				if td.Name == toolName {
					result = append(result, mcp.NewMCPToolAdapter(client, td))
					break
				}
			}
		}
	}
	return result
}

// resolvedTool carries the parsed tool reference and optional tool name.
type resolvedTool struct {
	scheme string // "builtin", "mcp", "skill"
	target string // tool identifier within the scheme
	config map[string]any
}

// resolveToolRef parses a ToolRef URI into a resolvedTool.
// Supported formats:
//   - builtin/web_search   → scheme=builtin, target=web_search
//   - mcp://server/tool    → scheme=mcp,     target=server/tool
//   - skill://name         → scheme=skill,   target=name
func (b *AgentBuilder) resolveToolRef(ref ToolRef) (resolvedTool, error) {
	r := ref.Ref
	switch {
	case strings.HasPrefix(r, "builtin/"):
		return resolvedTool{scheme: "builtin", target: strings.TrimPrefix(r, "builtin/"), config: ref.Config}, nil
	case strings.HasPrefix(r, "mcp://"):
		return resolvedTool{scheme: "mcp", target: strings.TrimPrefix(r, "mcp://"), config: ref.Config}, nil
	case strings.HasPrefix(r, "skill://"):
		return resolvedTool{scheme: "skill", target: strings.TrimPrefix(r, "skill://"), config: ref.Config}, nil
	default:
		return resolvedTool{}, fmt.Errorf("unrecognised tool ref scheme in %q", r)
	}
}

// ─── orchestration builders ───────────────────────────────────────────────────

// resolveSubAgents resolves all SubAgentRefs in def using the agent registry.
// Returns an error if the registry is not configured or a ref cannot be found.
func (b *AgentBuilder) resolveSubAgents(def *AgentDefinition) (map[string]Agent, error) {
	if b.agentRegistry == nil {
		return nil, fmt.Errorf("agentdef: Build %q: no agent registry configured; use WithAgentRegistry", def.Metadata.Name)
	}
	resolved := make(map[string]Agent, len(def.Spec.SubAgents))
	for _, ref := range def.Spec.SubAgents {
		agent, ok := b.agentRegistry(ref.Ref)
		if !ok {
			return nil, fmt.Errorf("agentdef: Build %q: sub-agent %q not found in registry", def.Metadata.Name, ref.Ref)
		}
		resolved[ref.Ref] = agent
	}
	return resolved, nil
}

// resolveSubAgentList returns sub-agents as an ordered slice matching the
// declaration order in def.Spec.SubAgents.
func (b *AgentBuilder) resolveSubAgentList(def *AgentDefinition) ([]Agent, error) {
	if b.agentRegistry == nil {
		return nil, fmt.Errorf("agentdef: Build %q: no agent registry configured; use WithAgentRegistry", def.Metadata.Name)
	}
	agents := make([]Agent, 0, len(def.Spec.SubAgents))
	for _, ref := range def.Spec.SubAgents {
		agent, ok := b.agentRegistry(ref.Ref)
		if !ok {
			return nil, fmt.Errorf("agentdef: Build %q: sub-agent %q not found in registry", def.Metadata.Name, ref.Ref)
		}
		agents = append(agents, agent)
	}
	return agents, nil
}

// buildSupervisor constructs a SupervisorAgent.  It builds the main LLM agent
// for the supervisor definition itself, then resolves sub-agent references.
func (b *AgentBuilder) buildSupervisor(ctx context.Context, def *AgentDefinition) (Agent, error) {
	subAgents, err := b.resolveSubAgents(def)
	if err != nil {
		return nil, err
	}

	// Build the supervisor's own LLM as a chat_model_agent so it has an
	// underlying model to generate responses.
	syntheticDef := &AgentDefinition{
		APIVersion: def.APIVersion,
		Kind:       def.Kind,
		Metadata:   def.Metadata,
		Spec: AgentSpec{
			Type:          "chat_model_agent",
			Model:         def.Spec.Model,
			SystemPrompt:  def.Spec.SystemPrompt,
			Observability: def.Spec.Observability,
		},
	}
	mainAgent, err := b.Build(ctx, syntheticDef)
	if err != nil {
		return nil, fmt.Errorf("agentdef: buildSupervisor %q: build main agent: %w", def.Metadata.Name, err)
	}

	maxRounds := 5
	if def.Spec.Orchestration != nil && def.Spec.Orchestration.MaxRounds > 0 {
		maxRounds = def.Spec.Orchestration.MaxRounds
	}

	return &SupervisorAgent{
		name:        def.Metadata.Name,
		description: def.Spec.SystemPrompt,
		mainAgent:   mainAgent,
		subAgents:   subAgents,
		maxRounds:   maxRounds,
		def:         def,
	}, nil
}

// buildSequential constructs a SequentialAgent from ordered sub-agent refs.
func (b *AgentBuilder) buildSequential(_ context.Context, def *AgentDefinition) (Agent, error) {
	agents, err := b.resolveSubAgentList(def)
	if err != nil {
		return nil, err
	}
	return &SequentialAgent{
		name:        def.Metadata.Name,
		description: def.Spec.SystemPrompt,
		agents:      agents,
		def:         def,
	}, nil
}

// buildParallel constructs a ParallelAgent from sub-agent refs.
func (b *AgentBuilder) buildParallel(_ context.Context, def *AgentDefinition) (Agent, error) {
	agents, err := b.resolveSubAgentList(def)
	if err != nil {
		return nil, err
	}
	return &ParallelAgent{
		name:        def.Metadata.Name,
		description: def.Spec.SystemPrompt,
		agents:      agents,
		def:         def,
	}, nil
}

// buildPlanExecute constructs a PlanExecuteAgent.  It builds the main LLM agent
// for planning and resolves sub-agent references as executors.
func (b *AgentBuilder) buildPlanExecute(ctx context.Context, def *AgentDefinition) (Agent, error) {
	executors, err := b.resolveSubAgentList(def)
	if err != nil {
		return nil, err
	}

	// Build the planner's own LLM as a chat_model_agent.
	syntheticDef := &AgentDefinition{
		APIVersion: def.APIVersion,
		Kind:       def.Kind,
		Metadata:   def.Metadata,
		Spec: AgentSpec{
			Type:          "chat_model_agent",
			Model:         def.Spec.Model,
			SystemPrompt:  def.Spec.SystemPrompt,
			Observability: def.Spec.Observability,
		},
	}
	mainAgent, err := b.Build(ctx, syntheticDef)
	if err != nil {
		return nil, fmt.Errorf("agentdef: buildPlanExecute %q: build main agent: %w", def.Metadata.Name, err)
	}

	maxSteps := 10
	if def.Spec.Orchestration != nil && def.Spec.Orchestration.MaxRounds > 0 {
		maxSteps = def.Spec.Orchestration.MaxRounds
	}

	return &PlanExecuteAgent{
		name:        def.Metadata.Name,
		description: def.Spec.SystemPrompt,
		mainAgent:   mainAgent,
		executors:   executors,
		def:         def,
		maxSteps:    maxSteps,
	}, nil
}

// buildAgentLoop constructs an AgentLoopAgent. It builds a chat_model_agent
// (with tools if configured) and wraps it in the autonomous loop.
func (b *AgentBuilder) buildAgentLoop(ctx context.Context, def *AgentDefinition) (Agent, error) {
	syntheticDef := &AgentDefinition{
		APIVersion: def.APIVersion,
		Kind:       def.Kind,
		Metadata:   def.Metadata,
		Spec: AgentSpec{
			Type:          "chat_model_agent",
			Model:         def.Spec.Model,
			SystemPrompt:  def.Spec.SystemPrompt,
			Tools:         def.Spec.Tools,
			Memory:        def.Spec.Memory,
			Observability: def.Spec.Observability,
		},
	}
	mainAgent, err := b.Build(ctx, syntheticDef)
	if err != nil {
		return nil, fmt.Errorf("agentdef: buildAgentLoop %q: build main agent: %w", def.Metadata.Name, err)
	}

	maxTurns := defaultMaxTurns
	if def.Spec.MaxTurns > 0 {
		maxTurns = def.Spec.MaxTurns
	}

	agent := &AgentLoopAgent{
		name:        def.Metadata.Name,
		description: def.Spec.SystemPrompt,
		mainAgent:   mainAgent,
		def:         def,
		maxTurns:    maxTurns,
	}
	return b.maybeWrapInterruptable(agent, def), nil
}

// buildWorkflow constructs a WorkflowAgent from a workflow spec.
func (b *AgentBuilder) buildWorkflow(def *AgentDefinition) (Agent, error) {
	if def.Spec.Workflow == nil {
		return nil, fmt.Errorf("agentdef: buildWorkflow %q: spec.workflow is required for type=workflow", def.Metadata.Name)
	}
	wa := &WorkflowAgent{
		name:        def.Metadata.Name,
		description: def.Spec.SystemPrompt,
		nodes:       def.Spec.Workflow.Nodes,
		edges:       def.Spec.Workflow.Edges,
		variables:   def.Spec.Workflow.Variables,
		registry:    b.agentRegistry,
		modelCfg:    b.modelConfig,
		def:         def,
	}
	// Inject evolution collector for node-level signal collection when enabled.
	if b.evolutionCollector != nil && def.Spec.Evolution != nil && def.Spec.Evolution.Enabled {
		wa.collector = b.evolutionCollector
	}
	return wa, nil
}

// buildEinoGraph constructs an einoGraphAgent from a named entry in the
// pkg/graphs registry.  The graph is compiled once at build time; the
// resulting Runnable is reused across all Chat calls.
func (b *AgentBuilder) buildEinoGraph(ctx context.Context, def *AgentDefinition) (Agent, error) {
	graphName := def.Spec.Graph
	if graphName == "" {
		return nil, fmt.Errorf("agentdef: buildEinoGraph %q: spec.graph is required for type=eino_graph", def.Metadata.Name)
	}
	factory, ok := graphs.Get(graphName)
	if !ok {
		return nil, fmt.Errorf("agentdef: buildEinoGraph %q: graph %q not found in registry (registered: %v)", def.Metadata.Name, graphName, graphs.List())
	}
	runnable, err := factory(ctx)
	if err != nil {
		return nil, fmt.Errorf("agentdef: buildEinoGraph %q: compile graph %q: %w", def.Metadata.Name, graphName, err)
	}
	return &einoGraphAgent{
		def:          def,
		systemPrompt: def.Spec.SystemPrompt,
		runnable:     runnable,
	}, nil
}

// einoGraphAgent wraps a compiled Eino graph Runnable as a superagent Agent.
// It converts the string Chat interface to []*schema.Message → *schema.Message.
type einoGraphAgent struct {
	def          *AgentDefinition
	systemPrompt string
	runnable     compose.Runnable[[]*schema.Message, *schema.Message]
}

func (a *einoGraphAgent) Name() string                    { return a.def.Metadata.Name }
func (a *einoGraphAgent) Description() string             { return a.systemPrompt }
func (a *einoGraphAgent) GetDefinition() *AgentDefinition { return a.def }

func (a *einoGraphAgent) Chat(ctx context.Context, _ string, message string) (<-chan string, error) {
	msgs := make([]*schema.Message, 0, 2)
	if a.systemPrompt != "" {
		msgs = append(msgs, schema.SystemMessage(a.systemPrompt))
	}
	msgs = append(msgs, schema.UserMessage(message))

	reader, err := a.runnable.Stream(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("agentdef: eino_graph %q: stream: %w", a.def.Metadata.Name, err)
	}

	ch := make(chan string, 64)
	go func() {
		defer close(ch)
		defer reader.Close()
		for {
			msg, recvErr := reader.Recv()
			if errors.Is(recvErr, io.EOF) {
				return
			}
			if recvErr != nil {
				select {
				case ch <- fmt.Sprintf("[error] %v", recvErr):
				case <-ctx.Done():
				}
				return
			}
			if msg != nil && msg.Content != "" {
				select {
				case ch <- msg.Content:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch, nil
}

// ─── stub agent (no model config) ────────────────────────────────────────────

// chatAgent is the stub implementation used when no real model endpoint is
// configured.  It satisfies the Agent interface for testing purposes.
type chatAgent struct {
	def        *AgentDefinition
	modelID    string
	tools      []resolvedTool
	memBackend memory.Backend
}

func (a *chatAgent) Name() string                    { return a.def.Metadata.Name }
func (a *chatAgent) Description() string             { return a.def.Spec.SystemPrompt }
func (a *chatAgent) GetDefinition() *AgentDefinition { return a.def }

func (a *chatAgent) Chat(_ context.Context, _ string, message string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- fmt.Sprintf("[%s] placeholder response for: %s", a.modelID, message)
	close(ch)
	return ch, nil
}

// ─── Eino simple chat agent (no tools) ───────────────────────────────────────

// einoChatAgent calls an Eino ChatModel directly (no tool loop).
type einoChatAgent struct {
	def          *AgentDefinition
	modelID      string
	provider     string // protocol / provider type for metrics labeling
	memBackend   memory.Backend
	chatModel    model.ToolCallingChatModel
	systemPrompt string
}

func (a *einoChatAgent) Name() string                    { return a.def.Metadata.Name }
func (a *einoChatAgent) Description() string             { return a.systemPrompt }
func (a *einoChatAgent) GetDefinition() *AgentDefinition { return a.def }

func (a *einoChatAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	// Build message list: system prompt + history + new user message
	msgs := make([]*schema.Message, 0, 8)
	if a.systemPrompt != "" {
		msgs = append(msgs, schema.SystemMessage(a.systemPrompt))
	}

	// Load conversation history from memory if available
	if a.memBackend != nil && sessionID != "" {
		history, err := a.memBackend.GetMessages(ctx, sessionID, memory.GetMessagesOpts{Limit: 20})
		if err == nil {
			for _, m := range history {
				switch m.Role {
				case "user":
					msgs = append(msgs, schema.UserMessage(m.Content))
				case "assistant":
					msgs = append(msgs, schema.AssistantMessage(m.Content, nil))
				}
			}
		}
	}

	// Append current user message
	msgs = append(msgs, schema.UserMessage(message))

	// Save user message to memory
	if a.memBackend != nil && sessionID != "" {
		_ = a.memBackend.AddMessage(ctx, sessionID, memory.Message{
			Role:      "user",
			Content:   message,
			Timestamp: time.Now().Unix(),
		})
	}

	reader, err := a.chatModel.Stream(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("agentdef: chat: stream: %w", err)
	}

	ch := make(chan string, 64)
	go func() {
		defer close(ch)
		defer reader.Close()
		var fullResponse strings.Builder
		streamStart := time.Now()
		firstToken := true
		for {
			chunk, err := reader.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					select {
					case ch <- fmt.Sprintf("[error] %v", err):
					case <-ctx.Done():
					}
				}
				// Save assistant response to memory
				if a.memBackend != nil && sessionID != "" && fullResponse.Len() > 0 {
					_ = a.memBackend.AddMessage(ctx, sessionID, memory.Message{
						Role:      "assistant",
						Content:   fullResponse.String(),
						Timestamp: time.Now().Unix(),
					})
				}
				return
			}
			if chunk != nil && chunk.Content != "" {
				if firstToken {
					modelrouter.RecordModelLatency(a.modelID, a.provider, time.Since(streamStart))
					firstToken = false
				}
				fullResponse.WriteString(chunk.Content)
				select {
				case ch <- chunk.Content:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch, nil
}

// ─── StreamToolCallChecker ────────────────────────────────────────────────────

// fullStreamToolCallChecker reads the entire stream to detect tool calls.
// Unlike the default firstChunkStreamToolCallChecker which only checks the first
// non-empty chunk, this implementation handles models (e.g. MiniMax, Claude) that
// emit content (think tags, text) before tool_calls deltas in the same response.
func fullStreamToolCallChecker(_ context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
	defer sr.Close()

	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if len(msg.ToolCalls) > 0 {
			return true, nil
		}
	}
}

// ─── Eino ReAct agent (with tools) ───────────────────────────────────────────

// einoReactAgent wraps an Eino react.Agent for tool-using interactions.
type einoReactAgent struct {
	def          *AgentDefinition
	modelID      string
	provider     string // protocol / provider type for metrics labeling
	memBackend   memory.Backend
	agent        *react.Agent
	systemPrompt string
}

func (a *einoReactAgent) Name() string                    { return a.def.Metadata.Name }
func (a *einoReactAgent) Description() string             { return a.systemPrompt }
func (a *einoReactAgent) GetDefinition() *AgentDefinition { return a.def }

func (a *einoReactAgent) Chat(ctx context.Context, _ string, message string) (<-chan string, error) {
	msgs := buildMessages(a.systemPrompt, message)

	reader, err := a.agent.Stream(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("agentdef: react chat: stream: %w", err)
	}

	ch := make(chan string, 64)
	go func() {
		defer close(ch)
		defer reader.Close()
		streamStart := time.Now()
		firstToken := true
		for {
			chunk, err := reader.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					select {
					case ch <- fmt.Sprintf("[error] %v", err):
					case <-ctx.Done():
					}
				}
				return
			}
			if chunk != nil && chunk.Content != "" {
				if firstToken {
					modelrouter.RecordModelLatency(a.modelID, a.provider, time.Since(streamStart))
					firstToken = false
				}
				select {
				case ch <- chunk.Content:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch, nil
}

// ─── provider dispatch ───────────────────────────────────────────────────────

// createChatModel creates an Eino ChatModel for the given protocol.
// Protocols deepseek, ollama, qwen are OpenAI-compatible (just different BaseURL).
// Claude, Gemini, and ARK have dedicated SDK adapters.
func (b *AgentBuilder) createChatModel(ctx context.Context, protocol, baseURL, apiKey, modelID string) (model.ToolCallingChatModel, error) {
	switch strings.ToLower(protocol) {
	case "claude":
		return einoclaude.NewChatModel(ctx, &einoclaude.Config{
			BaseURL: &baseURL,
			APIKey:  apiKey,
			Model:   modelID,
		})

	case "gemini":
		// Gemini requires google.golang.org/genai client. For simplicity, use
		// OpenAI-compatible endpoint (e.g. via LiteLLM proxy) when base_url is set.
		if baseURL != "" {
			return einoopenai.NewChatModel(ctx, &einoopenai.ChatModelConfig{
				BaseURL: baseURL,
				APIKey:  apiKey,
				Model:   modelID,
			})
		}
		return nil, fmt.Errorf("gemini protocol requires base_url pointing to an OpenAI-compatible proxy (e.g. LiteLLM)")

	case "ark":
		return einoark.NewChatModel(ctx, &einoark.ChatModelConfig{
			BaseURL: baseURL,
			APIKey:  apiKey,
			Model:   modelID,
		})

	case "deepseek":
		// DeepSeek uses OpenAI-compatible protocol with its own endpoint.
		if baseURL == "" || baseURL == b.modelConfig.BaseURL {
			baseURL = "https://api.deepseek.com/v1"
		}
		return einodeepseek.NewChatModel(ctx, &einodeepseek.ChatModelConfig{
			BaseURL: baseURL,
			APIKey:  apiKey,
			Model:   modelID,
		})

	case "ollama":
		// Ollama uses its own SDK at localhost:11434 (no /v1 suffix).
		ollamaURL := baseURL
		if ollamaURL == "" || ollamaURL == b.modelConfig.BaseURL {
			ollamaURL = "http://localhost:11434"
		}
		// Strip /v1 suffix if present (Ollama SDK adds its own paths).
		ollamaURL = strings.TrimSuffix(ollamaURL, "/v1")
		return einoollama.NewChatModel(ctx, &einoollama.ChatModelConfig{
			BaseURL: ollamaURL,
			Model:   modelID,
		})

	case "openai", "qwen", "":
		// OpenAI protocol (also used by Qwen DashScope compatible endpoint).
		return einoopenai.NewChatModel(ctx, &einoopenai.ChatModelConfig{
			BaseURL: baseURL,
			APIKey:  apiKey,
			Model:   modelID,
		})

	default:
		return nil, fmt.Errorf("unsupported model protocol %q (supported: openai, claude, deepseek, gemini, ark, ollama, qwen)", protocol)
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// buildMessages assembles the message slice sent to the model.
func buildMessages(systemPrompt, userMessage string) []*schema.Message {
	msgs := make([]*schema.Message, 0, 2)
	if systemPrompt != "" {
		msgs = append(msgs, schema.SystemMessage(systemPrompt))
	}
	msgs = append(msgs, schema.UserMessage(userMessage))
	return msgs
}

// shallowCopyDefWithSystemPrompt returns a copy of def with an overridden system prompt.
// The original def is not mutated.
func shallowCopyDefWithSystemPrompt(def *AgentDefinition, prompt string) *AgentDefinition {
	cp := *def
	cp.Spec.SystemPrompt = prompt
	return &cp
}

// buildEvolutionPromptPrefix formats Gene recommendations into a concise prompt prefix.
func buildEvolutionPromptPrefix(recs []evolution.Recommendation) string {
	if len(recs) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("# Accumulated Experience (from local Gene store)\n")
	sb.WriteString("The following strategies have been validated in previous executions:\n\n")
	for i, r := range recs {
		sb.WriteString(fmt.Sprintf("%d. [confidence=%.2f, success_rate=%.0f%%] %v\n",
			i+1, r.Confidence, r.SuccessRate*100, r.Strategy))
	}
	sb.WriteString("\nApply these strategies when relevant to the current task.\n\n")
	return sb.String()
}
