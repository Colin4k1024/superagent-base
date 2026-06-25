package sdk

import (
	"context"
	"fmt"
	"sync"
)

type Runtime struct {
	config  Config
	builder *AgentBuilder
	agents  map[string]Agent
	mu      sync.RWMutex
}

func NewRuntime(opts ...Option) (*Runtime, error) {
	cfg := Config{
		AgentsDir: "configs/agents",
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	rt := &Runtime{
		config:  cfg,
		builder: NewBuilder(opts...),
		agents:  make(map[string]Agent),
	}

	if err := rt.loadAll(); err != nil {
		return nil, fmt.Errorf("load agents: %w", err)
	}

	return rt, nil
}

func (rt *Runtime) loadAll() error {
	names, err := rt.builder.ListAgents()
	if err != nil {
		return err
	}

	for _, name := range names {
		agent, err := rt.builder.LoadAgent(name)
		if err != nil {
			continue
		}
		rt.agents[name] = agent
	}
	return nil
}

func (rt *Runtime) GetAgent(name string) (Agent, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	a, ok := rt.agents[name]
	return a, ok
}

func (rt *Runtime) ListAgents() []string {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	names := make([]string, 0, len(rt.agents))
	for name := range rt.agents {
		names = append(names, name)
	}
	return names
}

func (rt *Runtime) Reload() error {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.agents = make(map[string]Agent)
	return rt.loadAll()
}

func (rt *Runtime) Shutdown() error {
	return nil
}

func LoadAgent(path string) (Agent, error) {
	b := NewBuilder()
	return b.LoadAgentFromFile(path)
}

func Chat(ctx context.Context, agentName string, message string) (string, error) {
	rt, err := NewRuntime()
	if err != nil {
		return "", err
	}
	defer rt.Shutdown()

	agent, ok := rt.GetAgent(agentName)
	if !ok {
		return "", fmt.Errorf("agent %q not found", agentName)
	}

	return agent.ChatSync(ctx, "", message)
}
