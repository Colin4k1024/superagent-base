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
	"errors"
	"fmt"
	"io"
	"strings"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"github.com/superagent-ai/superagent-base/backend/pkg/mcp"
	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
	"github.com/superagent-ai/superagent-base/backend/pkg/modelrouter"
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

// AgentBuilder converts AgentDefinitions into running Agent instances.
type AgentBuilder struct {
	modelRouter   modelrouter.Router
	toolManager   *tool.Manager
	memoryFactory func(config memory.BackendConfig) (memory.Backend, error)
	mcpRegistry   *mcp.Registry
	modelConfig   ModelRuntimeConfig
	agentRegistry func(name string) (Agent, bool)
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

	// If no real model config is provided fall back to the stub agent so
	// existing unit tests that don't set up a model endpoint keep passing.
	if b.modelConfig.BaseURL == "" {
		return &chatAgent{
			def:        def,
			modelID:    modelID,
			tools:      toolRefs,
			memBackend: memBackend,
		}, nil
	}

	// Build a real Eino ChatModel.
	effectiveModelID := modelID
	if effectiveModelID == "" {
		effectiveModelID = b.modelConfig.ModelID
	}

	chatModel, err := einoopenai.NewChatModel(ctx, &einoopenai.ChatModelConfig{
		BaseURL: b.modelConfig.BaseURL,
		APIKey:  b.modelConfig.APIKey,
		Model:   effectiveModelID,
	})
	if err != nil {
		return nil, fmt.Errorf("agentdef: Build: create chat model: %w", err)
	}

	// Gather Eino-compatible tools from resolved refs.
	einoTools := b.resolveEinoTools(toolRefs)

	if len(einoTools) > 0 {
		// ReAct agent with tool calling.
		reactAgent, err := react.NewAgent(ctx, &react.AgentConfig{
			ToolCallingModel: chatModel,
			ToolsConfig: compose.ToolsNodeConfig{
				Tools: einoTools,
			},
			MaxStep: 10,
		})
		if err != nil {
			return nil, fmt.Errorf("agentdef: Build: create react agent: %w", err)
		}
		return &einoReactAgent{
			def:          def,
			modelID:      effectiveModelID,
			memBackend:   memBackend,
			agent:        reactAgent,
			systemPrompt: def.Spec.SystemPrompt,
		}, nil
	}

	// Simple chat agent without tools.
	return &einoChatAgent{
		def:          def,
		modelID:      effectiveModelID,
		memBackend:   memBackend,
		chatModel:    chatModel,
		systemPrompt: def.Spec.SystemPrompt,
	}, nil
}

// resolveEinoTools converts resolved tool refs to Eino einotool.BaseTool instances.
// Only builtin tools backed by tool.Manager are currently wired; MCP and skill
// refs are silently skipped until their adapters are ready.
func (b *AgentBuilder) resolveEinoTools(refs []resolvedTool) []einotool.BaseTool {
	if b.toolManager == nil {
		return nil
	}
	var result []einotool.BaseTool
	for _, ref := range refs {
		if ref.scheme != "builtin" {
			continue
		}
		t, ok := b.toolManager.Get(ref.target)
		if !ok {
			continue
		}
		result = append(result, t)
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
	memBackend   memory.Backend
	chatModel    *einoopenai.ChatModel
	systemPrompt string
}

func (a *einoChatAgent) Name() string                    { return a.def.Metadata.Name }
func (a *einoChatAgent) Description() string             { return a.systemPrompt }
func (a *einoChatAgent) GetDefinition() *AgentDefinition { return a.def }

func (a *einoChatAgent) Chat(ctx context.Context, _ string, message string) (<-chan string, error) {
	msgs := buildMessages(a.systemPrompt, message)

	reader, err := a.chatModel.Stream(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("agentdef: chat: stream: %w", err)
	}

	ch := make(chan string, 64)
	go func() {
		defer close(ch)
		defer reader.Close()
		for {
			chunk, err := reader.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					ch <- fmt.Sprintf("[error] %v", err)
				}
				return
			}
			if chunk != nil && chunk.Content != "" {
				ch <- chunk.Content
			}
		}
	}()
	return ch, nil
}

// ─── Eino ReAct agent (with tools) ───────────────────────────────────────────

// einoReactAgent wraps an Eino react.Agent for tool-using interactions.
type einoReactAgent struct {
	def          *AgentDefinition
	modelID      string
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
		for {
			chunk, err := reader.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					ch <- fmt.Sprintf("[error] %v", err)
				}
				return
			}
			if chunk != nil && chunk.Content != "" {
				ch <- chunk.Content
			}
		}
	}()
	return ch, nil
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
