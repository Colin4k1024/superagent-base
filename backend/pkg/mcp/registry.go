/*
 * Copyright 2025 coze-dev Authors
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

package mcp

import (
	"context"
	"fmt"
	"sync"
)

// ServerConfig holds configuration for a single MCP server.
type ServerConfig struct {
	Name      string            `yaml:"name"`
	Transport string            `yaml:"transport"` // "stdio" or "sse"
	// Command is the executable path, required for stdio transport.
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
	// URL is the SSE endpoint, required for sse transport.
	URL string            `yaml:"url"`
	Env map[string]string `yaml:"env"`
}

// Registry manages a pool of named MCP clients.
type Registry struct {
	mu      sync.RWMutex
	clients map[string]*Client
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{clients: make(map[string]*Client)}
}

// Connect creates a transport from cfg, initializes a Client, and registers it
// under cfg.Name. An existing client with the same name is disconnected first.
func (r *Registry) Connect(ctx context.Context, cfg ServerConfig) error {
	transport, err := buildTransport(cfg)
	if err != nil {
		return fmt.Errorf("mcp registry: build transport for %q: %w", cfg.Name, err)
	}

	client := NewClient(transport)
	if _, err := client.Initialize(ctx); err != nil {
		_ = client.Close()
		return fmt.Errorf("mcp registry: initialize %q: %w", cfg.Name, err)
	}

	r.mu.Lock()
	if old, ok := r.clients[cfg.Name]; ok {
		_ = old.Close()
	}
	r.clients[cfg.Name] = client
	r.mu.Unlock()

	return nil
}

// GetClient returns the Client registered under name, or false if not found.
func (r *Registry) GetClient(name string) (*Client, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.clients[name]
	return c, ok
}

// ListServers returns the names of all registered servers.
func (r *Registry) ListServers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.clients))
	for k := range r.clients {
		names = append(names, k)
	}
	return names
}

// Disconnect closes and removes the named client. It is a no-op if the name
// is not registered.
func (r *Registry) Disconnect(name string) error {
	r.mu.Lock()
	client, ok := r.clients[name]
	if ok {
		delete(r.clients, name)
	}
	r.mu.Unlock()

	if ok {
		return client.Close()
	}
	return nil
}

// DisconnectAll closes and removes every registered client.
func (r *Registry) DisconnectAll() error {
	r.mu.Lock()
	clients := r.clients
	r.clients = make(map[string]*Client)
	r.mu.Unlock()

	var firstErr error
	for _, c := range clients {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// buildTransport constructs the appropriate Transport from ServerConfig.
func buildTransport(cfg ServerConfig) (Transport, error) {
	switch cfg.Transport {
	case "stdio":
		if cfg.Command == "" {
			return nil, fmt.Errorf("stdio transport requires command")
		}
		env := envMapToSlice(cfg.Env)
		return NewStdioTransport(cfg.Command, cfg.Args, env)
	case "sse":
		if cfg.URL == "" {
			return nil, fmt.Errorf("sse transport requires url")
		}
		return NewSSETransport(cfg.URL, "", nil), nil
	default:
		return nil, fmt.Errorf("unsupported transport %q, must be stdio or sse", cfg.Transport)
	}
}

// envMapToSlice converts a map of env vars to KEY=VALUE strings expected by
// exec.Cmd.Env.
func envMapToSlice(m map[string]string) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}
