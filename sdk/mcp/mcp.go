package mcp

import (
	"context"
	"fmt"
)

type ServerConfig struct {
	Name      string            `yaml:"name"`
	Transport string            `yaml:"transport"`
	Command   string            `yaml:"command"`
	Args      []string          `yaml:"args"`
	URL       string            `yaml:"url"`
	Env       map[string]string `yaml:"env"`
}

type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type Client struct {
	config ServerConfig
	tools  []ToolDefinition
}

func NewClient(cfg ServerConfig) *Client {
	return &Client{config: cfg}
}

func (c *Client) Connect(ctx context.Context) error {
	return nil
}

func (c *Client) ListTools(ctx context.Context) ([]ToolDefinition, error) {
	return c.tools, nil
}

func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *Client) Close() error {
	return nil
}

type Registry struct {
	clients map[string]*Client
}

func NewRegistry() *Registry {
	return &Registry{clients: make(map[string]*Client)}
}

func (r *Registry) Connect(ctx context.Context, cfg ServerConfig) error {
	client := NewClient(cfg)
	if err := client.Connect(ctx); err != nil {
		return err
	}
	r.clients[cfg.Name] = client
	return nil
}

func (r *Registry) GetClient(name string) (*Client, bool) {
	c, ok := r.clients[name]
	return c, ok
}

func (r *Registry) ListServers() []string {
	names := make([]string, 0, len(r.clients))
	for name := range r.clients {
		names = append(names, name)
	}
	return names
}

func (r *Registry) DisconnectAll() {
	for _, c := range r.clients {
		c.Close()
	}
	r.clients = make(map[string]*Client)
}
