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

	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
	"github.com/superagent-ai/superagent-base/backend/pkg/tool/builtin"
)

// Manager keeps a registry of named tools and optionally wraps invocations
// with a middleware chain. Tools are stored as framework-agnostic llm.Tool
// instances; ADK-native tools are wrapped via ReverseToolAdapter at
// registration time so business code never imports eino directly.
type Manager struct {
	mu         sync.RWMutex
	tools      map[string]llm.Tool
	middleware Middleware
}

// NewManager creates a Manager with an optional middleware chain.
func NewManager(middlewares ...Middleware) *Manager {
	var mw Middleware
	if len(middlewares) > 0 {
		mw = Chain(middlewares...)
	}
	return &Manager{
		tools:      make(map[string]llm.Tool),
		middleware: mw,
	}
}

// Register adds an llm.Tool to the manager.
func (m *Manager) Register(t llm.Tool) error {
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

// Unregister removes the tool identified by name.
func (m *Manager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.tools[name]; !exists {
		return fmt.Errorf("tool manager: tool %q not found", name)
	}
	delete(m.tools, name)
	return nil
}

// Get returns the tool registered under name as an llm.Tool.
func (m *Manager) Get(name string) (llm.Tool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tools[name]
	return t, ok
}

// List returns a snapshot of all registered tools as llm.Tool instances.
func (m *Manager) List() []llm.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]llm.Tool, 0, len(m.tools))
	for _, t := range m.tools {
		out = append(out, t)
	}
	return out
}

// RegisterBuiltins registers all built-in tools as framework-agnostic llm.Tool instances.
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
