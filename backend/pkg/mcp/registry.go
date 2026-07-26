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
	"os"
	"sync"

	"gorm.io/gorm"
	"gopkg.in/yaml.v3"
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
	mu          sync.RWMutex
	clients     map[string]*Client
	configs     map[string]ServerConfig
	configStore *ConfigStore // nil → pure-memory mode (no persistence)
}

// NewRegistry returns an empty Registry.
// Pass a non-nil *gorm.DB to enable configuration persistence across restarts.
// Pass nil to run in pure-memory mode.
func NewRegistry(db *gorm.DB) *Registry {
	r := &Registry{
		clients: make(map[string]*Client),
		configs: make(map[string]ServerConfig),
	}
	if db != nil {
		cs, err := NewConfigStore(db)
		if err == nil {
			r.configStore = cs
		}
	}
	return r
}

// Connect creates a transport from cfg, initializes a Client, and registers it
// under cfg.Name. An existing client with the same name is disconnected first.
// If a ConfigStore is attached the configuration is persisted after a successful
// connection so it can be restored on the next startup.
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
	r.configs[cfg.Name] = cfg
	r.mu.Unlock()

	// Persist configuration so it survives restarts.
	if r.configStore != nil {
		if saveErr := r.configStore.Save(cfg); saveErr != nil {
			// Non-fatal: connection is live, only persistence failed.
			_ = saveErr
		}
	}

	return nil
}

// GetClient returns the Client registered under name, or false if not found.
func (r *Registry) GetClient(name string) (*Client, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.clients[name]
	return c, ok
}

// GetConfig returns the ServerConfig for the named server, or false if not found.
func (r *Registry) GetConfig(name string) (ServerConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cfg, ok := r.configs[name]
	return cfg, ok
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
// is not registered. If a ConfigStore is attached the persisted configuration
// is soft-deleted so the server is not reconnected on the next startup.
func (r *Registry) Disconnect(name string) error {
	r.mu.Lock()
	client, ok := r.clients[name]
	if ok {
		delete(r.clients, name)
		delete(r.configs, name)
	}
	r.mu.Unlock()

	var closeErr error
	if ok {
		closeErr = client.Close()
	}

	// Remove from persistent store regardless of close error.
	if r.configStore != nil {
		if delErr := r.configStore.Delete(name); delErr != nil {
			// Non-fatal: connection is closed, only persistence cleanup failed.
			_ = delErr
		}
	}

	return closeErr
}

// ReconnectAll loads all enabled server configurations from the persistent
// store and attempts to reconnect each one. Errors for individual servers are
// logged but do not abort the reconnection of subsequent servers.
// This should be called once after all services have initialised.
func (r *Registry) ReconnectAll(ctx context.Context) error {
	if r.configStore == nil {
		return nil
	}

	cfgs, err := r.configStore.ListEnabled()
	if err != nil {
		return fmt.Errorf("mcp registry: reload configs: %w", err)
	}

	for _, cfg := range cfgs {
		// Skip servers that are already connected (e.g. connected during this session).
		r.mu.RLock()
		_, already := r.clients[cfg.Name]
		r.mu.RUnlock()
		if already {
			continue
		}

		if connErr := r.Connect(ctx, cfg); connErr != nil {
			// Non-fatal per-server failure: continue with remaining configs.
			_ = connErr
		}
	}
	return nil
}

// LoadFromConfig reads a YAML config file containing a list of MCP server
// configurations and attempts to connect each one. Errors for individual
// servers are logged but do not abort the loading of subsequent servers.
// Servers that are already connected are skipped.
func (r *Registry) LoadFromConfig(ctx context.Context, configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("mcp registry: read config %s: %w", configPath, err)
	}

	var cfg struct {
		Servers []ServerConfig `yaml:"servers"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("mcp registry: parse config %s: %w", configPath, err)
	}

	for _, sc := range cfg.Servers {
		if sc.Name == "" {
			continue
		}
		r.mu.RLock()
		_, already := r.clients[sc.Name]
		r.mu.RUnlock()
		if already {
			continue
		}
		if connErr := r.Connect(ctx, sc); connErr != nil {
			// Non-fatal: log and continue.
			_ = connErr
		}
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
