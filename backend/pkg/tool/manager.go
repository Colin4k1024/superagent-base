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

package tool

import (
	"context"
	"fmt"
	"sync"

	einotool "github.com/cloudwego/eino/components/tool"

	"github.com/superagent-ai/superagent-base/backend/pkg/tool/builtin"
)

// Manager keeps a registry of named Eino tools and optionally wraps invocations
// with a middleware chain.
type Manager struct {
	mu         sync.RWMutex
	tools      map[string]einotool.InvokableTool
	middleware Middleware
}

// NewManager creates a Manager with an optional middleware chain.
// When multiple middlewares are provided they are applied via Chain.
func NewManager(middlewares ...Middleware) *Manager {
	var mw Middleware
	if len(middlewares) > 0 {
		mw = Chain(middlewares...)
	}
	return &Manager{
		tools:      make(map[string]einotool.InvokableTool),
		middleware: mw,
	}
}

// Register adds t to the manager. Returns an error if a tool with the same
// name is already registered.
func (m *Manager) Register(t einotool.InvokableTool) error {
	info, err := t.Info(context.Background())
	if err != nil {
		return fmt.Errorf("tool manager: get info: %w", err)
	}
	name := info.Name

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.tools[name]; exists {
		return fmt.Errorf("tool manager: tool %q already registered", name)
	}
	m.tools[name] = t
	return nil
}

// Unregister removes the tool identified by name. Returns an error if not found.
func (m *Manager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.tools[name]; !exists {
		return fmt.Errorf("tool manager: tool %q not found", name)
	}
	delete(m.tools, name)
	return nil
}

// Get returns the tool registered under name.
func (m *Manager) Get(name string) (einotool.InvokableTool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tools[name]
	return t, ok
}

// List returns a snapshot of all registered tools.
func (m *Manager) List() []einotool.InvokableTool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]einotool.InvokableTool, 0, len(m.tools))
	for _, t := range m.tools {
		out = append(out, t)
	}
	return out
}

// RegisterBuiltins registers all built-in tools. Errors from individual
// registrations (e.g. duplicate name) are collected and returned together.
func (m *Manager) RegisterBuiltins() error {
	var errs []error
	for _, t := range builtin.GetAllBuiltinTools() {
		if err := m.Register(t); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("tool manager: register builtins: %v", errs)
	}
	return nil
}
