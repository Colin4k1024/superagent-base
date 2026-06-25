package sdk

import (
	"context"
)

type Agent interface {
	Name() string
	Description() string
	Chat(ctx context.Context, sessionID string, message string) (<-chan string, error)
	ChatSync(ctx context.Context, sessionID string, message string) (string, error)
	GetDefinition() *AgentDefinition
}

type AgentDefinition struct {
	APIVersion string            `yaml:"apiVersion" json:"apiVersion"`
	Kind       string            `yaml:"kind"       json:"kind"`
	Metadata   Metadata          `yaml:"metadata"   json:"metadata"`
	Spec       AgentSpec         `yaml:"spec"       json:"spec"`
}

type Metadata struct {
	Name    string            `yaml:"name"              json:"name"`
	Version string            `yaml:"version,omitempty" json:"version,omitempty"`
	Tags    []string          `yaml:"tags,omitempty"    json:"tags,omitempty"`
	Labels  map[string]string `yaml:"labels,omitempty"  json:"labels,omitempty"`
}

type AgentSpec struct {
	Type         string           `yaml:"type"                    json:"type"`
	Model        ModelSpec        `yaml:"model"                   json:"model"`
	SystemPrompt string           `yaml:"system_prompt,omitempty" json:"system_prompt,omitempty"`
	Tools        []ToolRef        `yaml:"tools,omitempty"         json:"tools,omitempty"`
	Memory       MemorySpec       `yaml:"memory,omitempty"        json:"memory,omitempty"`
	SubAgents    []SubAgentRef    `yaml:"sub_agents,omitempty"    json:"sub_agents,omitempty"`
	Workflow     *WorkflowSpec    `yaml:"workflow,omitempty"      json:"workflow,omitempty"`
	MaxTurns     int              `yaml:"max_turns,omitempty"     json:"max_turns,omitempty"`
}

type ModelSpec struct {
	Primary  string `yaml:"primary"            json:"primary"`
	Fallback string `yaml:"fallback,omitempty" json:"fallback,omitempty"`
	BaseURL  string `yaml:"base_url,omitempty" json:"base_url,omitempty"`
	APIKey   string `yaml:"api_key,omitempty"  json:"api_key,omitempty"`
}

type ToolRef struct {
	Ref    string         `yaml:"ref"              json:"ref"`
	Config map[string]any `yaml:"config,omitempty" json:"config,omitempty"`
}

type MemorySpec struct {
	Backend string         `yaml:"backend,omitempty" json:"backend,omitempty"`
	Config  map[string]any `yaml:"config,omitempty"  json:"config,omitempty"`
}

type SubAgentRef struct {
	Ref  string `yaml:"ref"          json:"ref"`
	Role string `yaml:"role,omitempty" json:"role,omitempty"`
}

type WorkflowSpec struct {
	Nodes []WorkflowNode `yaml:"nodes" json:"nodes"`
	Edges []WorkflowEdge `yaml:"edges" json:"edges"`
}

type WorkflowNode struct {
	ID        string `yaml:"id"                  json:"id"`
	Type      string `yaml:"type"                json:"type"`
	Agent     string `yaml:"agent,omitempty"     json:"agent,omitempty"`
	Tool      string `yaml:"tool,omitempty"      json:"tool,omitempty"`
	Code      string `yaml:"code,omitempty"      json:"code,omitempty"`
	Condition string `yaml:"condition,omitempty" json:"condition,omitempty"`
}

type WorkflowEdge struct {
	From      string `yaml:"from"                json:"from"`
	To        string `yaml:"to"                  json:"to"`
	Condition string `yaml:"condition,omitempty" json:"condition,omitempty"`
}

type Event struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
	Delta   string `json:"delta,omitempty"`
	Name    string `json:"name,omitempty"`
	Error   string `json:"error,omitempty"`
}
