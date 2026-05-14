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

// Package graphs is the drop zone for Eino Dev–generated graph code.
//
// # Workflow
//
//  1. Use the "Eino Dev" VS Code plugin to visually orchestrate your graph.
//  2. Export the generated Go code into this package (e.g. pkg/graphs/my_flow.go).
//  3. Add an init() call that registers the compiled graph under a name.
//  4. Reference the name from any agent YAML with type: eino_graph.
//
// # Example registration
//
//	func init() {
//	    graphs.Register("my-research-flow", graphs.CompileGraph(BuildMyResearchFlow))
//	}
//
// # Example YAML
//
//	spec:
//	  type: eino_graph
//	  graph: my-research-flow
//	  system_prompt: "You are a research assistant."
package graphs

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// GraphFactory builds and compiles a single Eino graph runnable.
// The context carries request-scoped values; the factory is called once per
// agent build (at startup or hot-reload) and the result is cached for the
// lifetime of the agent.
//
// The standard graph type is []*schema.Message → *schema.Message, which maps
// naturally to the Agent.Chat interface.  Use CompileGraph to wrap the
// *compose.Graph builder function Eino Dev generates.
type GraphFactory func(ctx context.Context) (compose.Runnable[[]*schema.Message, *schema.Message], error)

var (
	mu       sync.RWMutex
	registry = map[string]GraphFactory{}
)

// Register adds a named GraphFactory to the global registry.
// Panics on duplicate registration to surface configuration errors early.
// Call from package init() functions.
func Register(name string, factory GraphFactory) {
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("graphs: Register called twice for graph %q", name))
	}
	registry[name] = factory
}

// Get retrieves a registered GraphFactory by name.
func Get(name string) (GraphFactory, bool) {
	mu.RLock()
	defer mu.RUnlock()
	f, ok := registry[name]
	return f, ok
}

// List returns all registered graph names.
func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}

// CompileGraph is a convenience wrapper: it turns the *compose.Graph builder
// function that Eino Dev generates into a GraphFactory.
//
// Usage:
//
//	// Eino Dev generated:
//	func BuildMyFlow(ctx context.Context) (*compose.Graph[[]*schema.Message, *schema.Message], error) { ... }
//
//	// Register it:
//	func init() {
//	    graphs.Register("my-flow", graphs.CompileGraph(BuildMyFlow))
//	}
func CompileGraph(
	build func(ctx context.Context) (*compose.Graph[[]*schema.Message, *schema.Message], error),
) GraphFactory {
	return func(ctx context.Context) (compose.Runnable[[]*schema.Message, *schema.Message], error) {
		g, err := build(ctx)
		if err != nil {
			return nil, fmt.Errorf("graphs: build graph: %w", err)
		}
		return g.Compile(ctx)
	}
}
