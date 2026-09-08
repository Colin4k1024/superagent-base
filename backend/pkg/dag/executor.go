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

package dag

import (
	"context"
	"fmt"
	"sync"

	"github.com/superagent-ai/superagent-base/backend/pkg/logs"
)

// StreamHandler is called by streaming nodes to push partial output.
type StreamHandler func(key NodeKey, data map[string]any)

// ExecutorOption configures the Executor.
type ExecutorOption func(*executorConfig)

type executorConfig struct {
	streamHandler  StreamHandler
	maxConcurrency int
	failFast       bool
}

// WithStreamHandler sets a handler that receives streaming output events.
func WithStreamHandler(h StreamHandler) ExecutorOption {
	return func(c *executorConfig) { c.streamHandler = h }
}

// WithMaxConcurrency limits parallel branch execution (0 = unlimited).
func WithMaxConcurrency(n int) ExecutorOption {
	return func(c *executorConfig) { c.maxConcurrency = n }
}

// WithFailFast stops the entire graph on the first node error.
func WithFailFast() ExecutorOption {
	return func(c *executorConfig) { c.failFast = true }
}

// Executor runs a Graph.
type Executor struct {
	graph *Graph
	cfg   executorConfig
}

// NewExecutor creates an Executor for the given graph.
func NewExecutor(g *Graph, opts ...ExecutorOption) *Executor {
	cfg := executorConfig{
		maxConcurrency: 0,
		failFast:       true,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return &Executor{graph: g, cfg: cfg}
}

// Execute runs the graph synchronously and returns the exit node's output.
func (ex *Executor) Execute(ctx context.Context, input map[string]any) (*ExecutionResult, error) {
	ec := NewExecutionContext(input)
	ctx = WithExecutionContext(ctx, ec)

	order, err := ex.graph.TopologicalSort()
	if err != nil {
		return nil, err
	}

	executed := make(map[NodeKey]bool)

	for _, key := range order {
		if executed[key] {
			continue
		}

		node := ex.graph.Node(key)
		if node == nil {
			return nil, fmt.Errorf("dag: node %q not found", key)
		}

		// Entry node receives the graph input; all others get
		// resolved input from predecessor outputs.
		var nodeInput map[string]any
		if key == ex.graph.entry {
			nodeInput = input
		} else {
			nodeInput = ec.resolveInput(key, ex.graph.InEdges(key))
		}

		// Check if all predecessors have executed.
		ready := true
		for _, e := range ex.graph.InEdges(key) {
			if !executed[e.From] {
				ready = false
				break
			}
		}
		if !ready {
			continue
		}

		// Execute the node.
		output, err := node.Executor.Execute(ctx, nodeInput)
		if err != nil {
			if ex.cfg.failFast {
				return nil, fmt.Errorf("dag: node %q failed: %w", key, err)
			}
			logs.CtxWarnf(ctx, "dag: node %q failed (non-fatal): %v", key, err)
			output = nil
		}

		ec.SetOutput(key, output)
		executed[key] = true

		// Stream output if handler is set and output is non-empty.
		if ex.cfg.streamHandler != nil && output != nil {
			ex.cfg.streamHandler(key, output)
		}

		// Handle conditional branching: if the node produced a port selection,
		// deactivate edges that don't match the selected port.
		if selectedPort, ok := getSelectedPort(output); ok {
			for _, e := range ex.graph.OutEdges(key) {
				if e.Port != "" && PortLabel(e.Port) != selectedPort {
					// Mark the target as executed (skip) to prevent
					// downstream execution of inactive branches.
					executed[e.To] = true
				}
			}
		}
	}

	result := &ExecutionResult{
		Usage: nil,
	}
	if ex.graph.exit != "" {
		result.Output = ec.GetOutput(ex.graph.exit)
	}
	u := ec.Usage()
	result.Usage = &u

	return result, nil
}

// ExecuteAsync runs the graph and streams results via the configured
// StreamHandler. It blocks until the graph completes or errors.
func (ex *Executor) ExecuteAsync(ctx context.Context, input map[string]any) (*ExecutionResult, error) {
	return ex.Execute(ctx, input)
}

// getSelectedPort checks if the node output contains a port selection
// (used by selector/branch nodes). The convention is that the output
// map contains a special key "_selected_port" with the port value.
func getSelectedPort(output map[string]any) (PortLabel, bool) {
	if output == nil {
		return "", false
	}
	v, ok := output["_selected_port"]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	return PortLabel(s), true
}

// ParallelExecutor runs independent branches concurrently.
type ParallelExecutor struct {
	graph *Graph
	cfg   executorConfig
}

// NewParallelExecutor creates an executor that runs independent branches
// concurrently (respecting topological order).
func NewParallelExecutor(g *Graph, opts ...ExecutorOption) *ParallelExecutor {
	cfg := executorConfig{
		maxConcurrency: 0,
		failFast:       true,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return &ParallelExecutor{graph: g, cfg: cfg}
}

// Execute runs the graph with parallel branch execution.
func (px *ParallelExecutor) Execute(ctx context.Context, input map[string]any) (*ExecutionResult, error) {
	ec := NewExecutionContext(input)
	ctx = WithExecutionContext(ctx, ec)

	executed := make(map[NodeKey]bool)

	var mu sync.Mutex
	var execErr error

	// BFS-like wave execution: process nodes level by level.
	for {
		// Find all ready nodes (all predecessors executed).
		var ready []NodeKey
		for key := range px.graph.nodes {
			if executed[key] {
				continue
			}
			allDone := true
			for _, e := range px.graph.InEdges(key) {
				if !executed[e.From] {
					allDone = false
					break
				}
			}
			if allDone {
				ready = append(ready, key)
			}
		}

		if len(ready) == 0 {
			break
		}

		var wg sync.WaitGroup
		sem := make(chan struct{}, px.cfg.maxConcurrency)
		if px.cfg.maxConcurrency == 0 {
			sem = nil
		}

		for _, key := range ready {
			wg.Add(1)
			go func(k NodeKey) {
				defer wg.Done()
				if sem != nil {
					sem <- struct{}{}
					defer func() { <-sem }()
				}

				node := px.graph.Node(k)
				if node == nil {
					mu.Lock()
					if execErr == nil {
						execErr = fmt.Errorf("dag: node %q not found", k)
					}
					mu.Unlock()
					return
				}

				var nodeInput map[string]any
				if k == px.graph.entry {
					nodeInput = input
				} else {
					nodeInput = ec.resolveInput(k, px.graph.InEdges(k))
				}
				output, err := node.Executor.Execute(ctx, nodeInput)
				if err != nil {
					mu.Lock()
					if px.cfg.failFast && execErr == nil {
						execErr = fmt.Errorf("dag: node %q failed: %w", k, err)
					}
					mu.Unlock()
					if px.cfg.failFast {
						return
					}
				}

				ec.SetOutput(k, output)
				mu.Lock()
				executed[k] = true
				mu.Unlock()

				if px.cfg.streamHandler != nil && output != nil {
					px.cfg.streamHandler(k, output)
				}

				if selectedPort, ok := getSelectedPort(output); ok {
					for _, e := range px.graph.OutEdges(k) {
						if e.Port != "" && PortLabel(e.Port) != selectedPort {
							mu.Lock()
							executed[e.To] = true
							mu.Unlock()
						}
					}
				}
			}(key)
		}
		wg.Wait()

		if execErr != nil && px.cfg.failFast {
			break
		}
	}

	if execErr != nil {
		return nil, execErr
	}

	result := &ExecutionResult{}
	if px.graph.exit != "" {
		result.Output = ec.GetOutput(px.graph.exit)
	}
	u := ec.Usage()
	result.Usage = &u
	return result, nil
}
