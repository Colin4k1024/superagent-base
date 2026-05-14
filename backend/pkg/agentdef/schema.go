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

// Package agentdef provides YAML-based declarative Agent definitions.
//
// An AgentDefinition follows a Kubernetes-like schema:
//
//	apiVersion: superagent/v1
//	kind: Agent
//	metadata:
//	  name: research-agent
//	  version: "1.0.0"
//	spec:
//	  type: chat_model_agent
//	  model:
//	    primary: gpt-4o
//	  ...
package agentdef

// AgentDefinition is the root object for an Agent YAML file.
type AgentDefinition struct {
	// APIVersion must be "superagent/v1".
	APIVersion string    `yaml:"apiVersion" json:"apiVersion"`
	// Kind must be "Agent".
	Kind       string    `yaml:"kind"       json:"kind"`
	// Metadata holds the agent identity and labels.
	Metadata   Metadata  `yaml:"metadata"   json:"metadata"`
	// Spec describes the agent behaviour.
	Spec       AgentSpec `yaml:"spec"       json:"spec"`
}

// Metadata carries identity information for the agent.
type Metadata struct {
	// Name is the unique human-readable identifier; must match [a-z0-9-]+.
	Name    string            `yaml:"name"              json:"name"`
	// Version is the semantic version string for this agent definition.
	Version string            `yaml:"version,omitempty" json:"version,omitempty"`
	// Tags are free-form labels for grouping/filtering.
	Tags    []string          `yaml:"tags,omitempty"    json:"tags,omitempty"`
	// Labels are arbitrary key/value pairs.
	Labels  map[string]string `yaml:"labels,omitempty"  json:"labels,omitempty"`
}

// AgentSpec holds the behavioural configuration for the agent.
type AgentSpec struct {
	// Type selects the agent execution mode.
	// Must be one of: chat_model_agent, deep_agent, workflow, supervisor, sequential, parallel, plan_execute.
	Type         string           `yaml:"type"                    json:"type"`
	// Model configures model selection and routing.
	Model        ModelSpec        `yaml:"model"                   json:"model"`
	// SystemPrompt is the system-level instruction for the agent.
	SystemPrompt string           `yaml:"system_prompt,omitempty" json:"system_prompt,omitempty"`
	// Tools lists tool references the agent may invoke.
	Tools        []ToolRef        `yaml:"tools,omitempty"         json:"tools,omitempty"`
	// Memory configures the memory backend.
	Memory       MemorySpec       `yaml:"memory,omitempty"        json:"memory,omitempty"`
	// Middleware is an ordered list of middleware components.
	Middleware    []MiddlewareSpec  `yaml:"middleware,omitempty"    json:"middleware,omitempty"`
	// Observability controls tracing, metrics, and log verbosity.
	Observability ObsSpec          `yaml:"observability,omitempty" json:"observability,omitempty"`
	// SubAgents lists the sub-agent references for orchestration types.
	SubAgents     []SubAgentRef      `yaml:"sub_agents,omitempty"    json:"sub_agents,omitempty"`
	// Orchestration defines the multi-agent coordination strategy.
	Orchestration *OrchestrationSpec `yaml:"orchestration,omitempty" json:"orchestration,omitempty"`
	// Workflow defines a graph-based workflow for type=workflow.
	Workflow      *WorkflowSpec      `yaml:"workflow,omitempty"      json:"workflow,omitempty"`
	// Interrupt controls interrupt/resume behaviour for this agent.
	Interrupt     *InterruptConfig   `yaml:"interrupt,omitempty"     json:"interrupt,omitempty"`
	// Graph is the registered Eino graph name for type=eino_graph.
	// The name must match a factory registered via pkg/graphs.Register().
	Graph         string             `yaml:"graph,omitempty"         json:"graph,omitempty"`
	// Evolution enables experience self-evolution via the Oris SDK.
	Evolution     *EvolutionSpec     `yaml:"evolution,omitempty"     json:"evolution,omitempty"`
}

// EvolutionSpec configures experience self-evolution for an agent.
type EvolutionSpec struct {
	// Enabled is the per-agent switch (requires global engine to be initialised).
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Collect lists the signal types to contribute to the Experience Repo.
	// Valid values: tool_success, tool_error, model_invoke, agent_done.
	// Defaults to all types when empty.
	Collect []string `yaml:"collect,omitempty" json:"collect,omitempty"`
	// Advise configures Gene recommendation injection into the system prompt.
	Advise EvolutionAdviseSpec `yaml:"advise,omitempty" json:"advise,omitempty"`
}

// EvolutionAdviseSpec controls how Advisor recommendations are applied.
type EvolutionAdviseSpec struct {
	// MinConfidence overrides the global minimum confidence threshold for this agent.
	MinConfidence float64 `yaml:"min_confidence,omitempty" json:"min_confidence,omitempty"`
	// MaxSuggestions caps the number of Gene recommendations injected into the prompt.
	MaxSuggestions int `yaml:"max_suggestions,omitempty" json:"max_suggestions,omitempty"`
}

// WorkflowSpec defines a graph-based workflow composed of nodes and edges.
type WorkflowSpec struct {
	// Nodes are the processing steps in the workflow.
	Nodes     []WorkflowNode     `yaml:"nodes"               json:"nodes"`
	// Edges define the directed connections between nodes.
	Edges     []WorkflowEdge     `yaml:"edges"               json:"edges"`
	// Variables declare named references to node outputs for use in downstream nodes.
	Variables []WorkflowVariable `yaml:"variables,omitempty" json:"variables,omitempty"`
}

// WorkflowNode is a single processing step within a workflow.
type WorkflowNode struct {
	// ID is the unique identifier for this node within the workflow.
	ID           string            `yaml:"id"                       json:"id"`
	// Type is the node execution mode: llm_call, agent_call, tool_call, code, condition.
	Type         string            `yaml:"type"                     json:"type"`
	// Agent is the name of the agent to invoke for agent_call nodes.
	Agent        string            `yaml:"agent,omitempty"          json:"agent,omitempty"`
	// Tool is the tool reference URI for tool_call nodes.
	Tool         string            `yaml:"tool,omitempty"           json:"tool,omitempty"`
	// Prompt is the system/instruction prompt for llm_call nodes.
	Prompt       string            `yaml:"prompt,omitempty"         json:"prompt,omitempty"`
	// Code is the code to execute for code nodes.
	Code         string            `yaml:"code,omitempty"           json:"code,omitempty"`
	// Language is the programming language for code nodes (e.g. "python", "javascript").
	Language     string            `yaml:"language,omitempty"       json:"language,omitempty"`
	// Condition is a boolean expression for condition nodes.
	Condition    string            `yaml:"condition,omitempty"      json:"condition,omitempty"`
	// InputMapping maps node input fields to state expressions.
	InputMapping map[string]string `yaml:"input_mapping,omitempty"  json:"input_mapping,omitempty"`
}

// WorkflowEdge is a directed connection between two workflow nodes.
type WorkflowEdge struct {
	// From is the source node ID, or "START" to mark the workflow entry point.
	From      string `yaml:"from"                json:"from"`
	// To is the destination node ID, or "END" to mark the workflow exit point.
	To        string `yaml:"to"                  json:"to"`
	// Condition is an optional expression; when non-empty the edge is only
	// followed when the expression evaluates to true.
	Condition string `yaml:"condition,omitempty" json:"condition,omitempty"`
}

// WorkflowVariable declares a named alias that maps a node output to a
// state key consumable by downstream nodes via {{.name}} template syntax.
type WorkflowVariable struct {
	// Name is the template variable name.
	Name string `yaml:"name" json:"name"`
	// From identifies the source value as "node_id.output" or "node_id.field".
	From string `yaml:"from" json:"from"`
}

// SubAgentRef references another agent by name for use in orchestration.
type SubAgentRef struct {
	// Ref is the name of the sub-agent (must match an agent's metadata.name).
	Ref    string         `yaml:"ref"              json:"ref"`
	// Role describes the sub-agent's purpose within the orchestration.
	Role   string         `yaml:"role,omitempty"   json:"role,omitempty"`
	// Config holds optional per-sub-agent configuration overrides.
	Config map[string]any `yaml:"config,omitempty" json:"config,omitempty"`
}

// OrchestrationSpec defines the coordination strategy for multi-agent types.
type OrchestrationSpec struct {
	// Mode is the orchestration strategy: supervisor, sequential, parallel, plan_execute.
	Mode      string `yaml:"mode"                  json:"mode"`
	// MaxRounds limits the number of orchestration iterations.
	MaxRounds int    `yaml:"max_rounds,omitempty"  json:"max_rounds,omitempty"`
}

// ModelSpec configures model selection and routing for the agent.
type ModelSpec struct {
	// Protocol selects the LLM provider protocol.
	// Supported: openai (default), claude, deepseek, gemini, ark, ollama, qwen.
	// Protocols deepseek, ollama, qwen use OpenAI-compatible endpoints.
	Protocol string `yaml:"protocol,omitempty" json:"protocol,omitempty"`
	// Router is the named routing strategy (e.g. "capability-based").
	// When empty, Primary is used directly without routing.
	Router   string `yaml:"router,omitempty"   json:"router,omitempty"`
	// Primary is the default model ID (e.g. "gpt-4o", "deepseek-r1").
	Primary  string `yaml:"primary"            json:"primary"`
	// Fallback is the model ID to use when Primary is unavailable.
	Fallback string `yaml:"fallback,omitempty" json:"fallback,omitempty"`
	// BaseURL overrides the global model endpoint for this agent.
	// When empty, the global ModelRuntimeConfig.BaseURL is used.
	BaseURL  string `yaml:"base_url,omitempty" json:"base_url,omitempty"`
	// APIKeyEnv is the environment variable name holding the API key for this agent.
	// When empty, the global ModelRuntimeConfig.APIKey is used.
	APIKeyEnv string `yaml:"api_key_env,omitempty" json:"api_key_env,omitempty"`
}

// ToolRef identifies a tool by reference URI and optional per-tool config.
//
// Supported URI schemes:
//   - builtin/web_search  — built-in tool from pkg/tool/builtin
//   - mcp://server/tool   — tool exposed by a registered MCP server
//   - skill://name        — tool from the SkillsHub
type ToolRef struct {
	// Ref is the tool reference URI.
	Ref    string         `yaml:"ref"              json:"ref"`
	// Config holds tool-specific configuration.
	Config map[string]any `yaml:"config,omitempty" json:"config,omitempty"`
}

// MemorySpec selects and configures a memory backend.
type MemorySpec struct {
	// Backend is the memory backend type: builtin, mem0, zep, letta.
	Backend string         `yaml:"backend,omitempty" json:"backend,omitempty"`
	// Config holds backend-specific configuration.
	Config  map[string]any `yaml:"config,omitempty"  json:"config,omitempty"`
}

// MiddlewareSpec configures a single middleware component in the pipeline.
type MiddlewareSpec struct {
	// Name identifies the middleware; may be inferred from inline keys when absent.
	Name   string         `yaml:"name,omitempty" json:"name,omitempty"`
	// Config holds arbitrary middleware configuration via inline YAML keys.
	Config map[string]any `yaml:",inline"        json:"config,omitempty"`
}

// ObsSpec controls observability features for the agent.
type ObsSpec struct {
	// Tracing enables distributed tracing via OpenTelemetry.
	Tracing  bool   `yaml:"tracing,omitempty"   json:"tracing,omitempty"`
	// Metrics enables Prometheus metrics collection.
	Metrics  bool   `yaml:"metrics,omitempty"   json:"metrics,omitempty"`
	// LogLevel sets the log verbosity: trace, debug, info, warn, error.
	LogLevel string `yaml:"log_level,omitempty" json:"log_level,omitempty"`
}

// InterruptConfig controls interrupt/resume behaviour for an agent.
type InterruptConfig struct {
	// Enabled activates the interrupt/resume wrapper around the agent.
	Enabled bool `yaml:"enabled" json:"enabled"`
	// CheckpointBackend selects the persistence store: "redis" or "memory" (default).
	CheckpointBackend string `yaml:"checkpoint_backend,omitempty" json:"checkpoint_backend,omitempty"`
	// TimeoutSeconds controls how long an interrupt state is retained.
	// Defaults to 300 (5 minutes) when zero.
	TimeoutSeconds int `yaml:"timeout_seconds,omitempty" json:"timeout_seconds,omitempty"`
}
