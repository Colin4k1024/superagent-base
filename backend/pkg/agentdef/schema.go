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
	// Router is the named routing strategy (e.g. "capability-based").
	// When empty, Primary is used directly without routing.
	Router   string `yaml:"router,omitempty"   json:"router,omitempty"`
	// Primary is the default model ID (e.g. "gpt-4o", "deepseek-r1").
	Primary  string `yaml:"primary"            json:"primary"`
	// Fallback is the model ID to use when Primary is unavailable.
	Fallback string `yaml:"fallback,omitempty" json:"fallback,omitempty"`
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
