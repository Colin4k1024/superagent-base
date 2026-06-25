package sdk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type AgentBuilder struct {
	config Config
}

func NewBuilder(opts ...Option) *AgentBuilder {
	cfg := Config{
		AgentsDir: "configs/agents",
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &AgentBuilder{config: cfg}
}

func (b *AgentBuilder) LoadAgent(name string) (Agent, error) {
	path := filepath.Join(b.config.AgentsDir, name+".yaml")
	return b.LoadAgentFromFile(path)
}

func (b *AgentBuilder) LoadAgentFromFile(path string) (Agent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read agent file: %w", err)
	}

	var def AgentDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse agent yaml: %w", err)
	}

	return b.Build(&def)
}

func (b *AgentBuilder) Build(def *AgentDefinition) (Agent, error) {
	switch def.Spec.Type {
	case "chat_model_agent", "deep_agent":
		return b.buildChatAgent(def)
	case "supervisor":
		return b.buildSupervisorAgent(def)
	case "sequential":
		return b.buildSequentialAgent(def)
	case "parallel":
		return b.buildParallelAgent(def)
	case "workflow":
		return b.buildWorkflowAgent(def)
	case "agentloop":
		return b.buildAgentLoopAgent(def)
	default:
		return b.buildChatAgent(def)
	}
}

func (b *AgentBuilder) ListAgents() ([]string, error) {
	entries, err := os.ReadDir(b.config.AgentsDir)
	if err != nil {
		return nil, fmt.Errorf("read agents dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			name := strings.TrimSuffix(e.Name(), ".yaml")
			names = append(names, name)
		}
	}
	return names, nil
}

func (b *AgentBuilder) buildChatAgent(def *AgentDefinition) (Agent, error) {
	return &chatAgent{
		def:    def,
		config: b.config,
	}, nil
}

func (b *AgentBuilder) buildSupervisorAgent(def *AgentDefinition) (Agent, error) {
	return &supervisorAgent{
		def:    def,
		config: b.config,
	}, nil
}

func (b *AgentBuilder) buildSequentialAgent(def *AgentDefinition) (Agent, error) {
	return &sequentialAgent{
		def:    def,
		config: b.config,
	}, nil
}

func (b *AgentBuilder) buildParallelAgent(def *AgentDefinition) (Agent, error) {
	return &parallelAgent{
		def:    def,
		config: b.config,
	}, nil
}

func (b *AgentBuilder) buildWorkflowAgent(def *AgentDefinition) (Agent, error) {
	return &workflowAgent{
		def:    def,
		config: b.config,
	}, nil
}

func (b *AgentBuilder) buildAgentLoopAgent(def *AgentDefinition) (Agent, error) {
	return &agentLoopAgent{
		def:    def,
		config: b.config,
	}, nil
}

type chatAgent struct {
	def    *AgentDefinition
	config Config
}

func (a *chatAgent) Name() string                    { return a.def.Metadata.Name }
func (a *chatAgent) Description() string             { return a.def.Spec.SystemPrompt }
func (a *chatAgent) GetDefinition() *AgentDefinition { return a.def }

func (a *chatAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	ch := make(chan string, 100)
	go func() {
		defer close(ch)
		ch <- fmt.Sprintf("[%s] Response to: %s", a.Name(), message)
	}()
	return ch, nil
}

func (a *chatAgent) ChatSync(ctx context.Context, sessionID string, message string) (string, error) {
	ch, err := a.Chat(ctx, sessionID, message)
	if err != nil {
		return "", err
	}
	var result string
	for chunk := range ch {
		result += chunk
	}
	return result, nil
}

type supervisorAgent struct {
	def    *AgentDefinition
	config Config
}

func (a *supervisorAgent) Name() string                    { return a.def.Metadata.Name }
func (a *supervisorAgent) Description() string             { return a.def.Spec.SystemPrompt }
func (a *supervisorAgent) GetDefinition() *AgentDefinition { return a.def }

func (a *supervisorAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	ch := make(chan string, 100)
	go func() {
		defer close(ch)
		ch <- fmt.Sprintf("[supervisor:%s] Delegating: %s", a.Name(), message)
	}()
	return ch, nil
}

func (a *supervisorAgent) ChatSync(ctx context.Context, sessionID string, message string) (string, error) {
	ch, err := a.Chat(ctx, sessionID, message)
	if err != nil {
		return "", err
	}
	var result string
	for chunk := range ch {
		result += chunk
	}
	return result, nil
}

type sequentialAgent struct {
	def    *AgentDefinition
	config Config
}

func (a *sequentialAgent) Name() string                    { return a.def.Metadata.Name }
func (a *sequentialAgent) Description() string             { return a.def.Spec.SystemPrompt }
func (a *sequentialAgent) GetDefinition() *AgentDefinition { return a.def }

func (a *sequentialAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	ch := make(chan string, 100)
	go func() {
		defer close(ch)
		ch <- fmt.Sprintf("[sequential:%s] Pipeline: %s", a.Name(), message)
	}()
	return ch, nil
}

func (a *sequentialAgent) ChatSync(ctx context.Context, sessionID string, message string) (string, error) {
	ch, err := a.Chat(ctx, sessionID, message)
	if err != nil {
		return "", err
	}
	var result string
	for chunk := range ch {
		result += chunk
	}
	return result, nil
}

type parallelAgent struct {
	def    *AgentDefinition
	config Config
}

func (a *parallelAgent) Name() string                    { return a.def.Metadata.Name }
func (a *parallelAgent) Description() string             { return a.def.Spec.SystemPrompt }
func (a *parallelAgent) GetDefinition() *AgentDefinition { return a.def }

func (a *parallelAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	ch := make(chan string, 100)
	go func() {
		defer close(ch)
		ch <- fmt.Sprintf("[parallel:%s] Fan-out: %s", a.Name(), message)
	}()
	return ch, nil
}

func (a *parallelAgent) ChatSync(ctx context.Context, sessionID string, message string) (string, error) {
	ch, err := a.Chat(ctx, sessionID, message)
	if err != nil {
		return "", err
	}
	var result string
	for chunk := range ch {
		result += chunk
	}
	return result, nil
}

type workflowAgent struct {
	def    *AgentDefinition
	config Config
}

func (a *workflowAgent) Name() string                    { return a.def.Metadata.Name }
func (a *workflowAgent) Description() string             { return a.def.Spec.SystemPrompt }
func (a *workflowAgent) GetDefinition() *AgentDefinition { return a.def }

func (a *workflowAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	ch := make(chan string, 100)
	go func() {
		defer close(ch)
		ch <- fmt.Sprintf("[workflow:%s] DAG: %s", a.Name(), message)
	}()
	return ch, nil
}

func (a *workflowAgent) ChatSync(ctx context.Context, sessionID string, message string) (string, error) {
	ch, err := a.Chat(ctx, sessionID, message)
	if err != nil {
		return "", err
	}
	var result string
	for chunk := range ch {
		result += chunk
	}
	return result, nil
}

type agentLoopAgent struct {
	def    *AgentDefinition
	config Config
}

func (a *agentLoopAgent) Name() string                    { return a.def.Metadata.Name }
func (a *agentLoopAgent) Description() string             { return a.def.Spec.SystemPrompt }
func (a *agentLoopAgent) GetDefinition() *AgentDefinition { return a.def }

func (a *agentLoopAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	ch := make(chan string, 100)
	go func() {
		defer close(ch)
		ch <- fmt.Sprintf("[agentloop:%s] Loop: %s", a.Name(), message)
	}()
	return ch, nil
}

func (a *agentLoopAgent) ChatSync(ctx context.Context, sessionID string, message string) (string, error) {
	ch, err := a.Chat(ctx, sessionID, message)
	if err != nil {
		return "", err
	}
	var result string
	for chunk := range ch {
		result += chunk
	}
	return result, nil
}
