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
	"sync"
)

// ExecutionContext carries per-node outputs and shared state during a
// graph run. It is not safe to use across concurrent graph runs, but
// concurrent branches within a single run share the same instance
// (protected by mutex).
type ExecutionContext struct {
	mu      sync.RWMutex
	outputs map[NodeKey]map[string]any // node → output fields
	state   map[string]any             // arbitrary shared state
	usage   Usage
}

// NewExecutionContext creates a fresh context with the given initial state.
func NewExecutionContext(initial map[string]any) *ExecutionContext {
	ec := &ExecutionContext{
		outputs: make(map[NodeKey]map[string]any),
		state:   make(map[string]any),
	}
	for k, v := range initial {
		ec.state[k] = v
	}
	return ec
}

// SetOutput stores a node's output.
func (ec *ExecutionContext) SetOutput(key NodeKey, output map[string]any) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.outputs[key] = output
}

// GetOutput retrieves a node's output, or nil if not yet executed.
func (ec *ExecutionContext) GetOutput(key NodeKey) map[string]any {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.outputs[key]
}

// SetState stores an arbitrary value in shared state.
func (ec *ExecutionContext) SetState(name string, value any) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.state[name] = value
}

// GetState retrieves a value from shared state.
func (ec *ExecutionContext) GetState(name string) (any, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	v, ok := ec.state[name]
	return v, ok
}

// AddUsage merges token usage from a node.
func (ec *ExecutionContext) AddUsage(u *Usage) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.usage.Add(u)
}

// Usage returns the accumulated token usage.
func (ec *ExecutionContext) Usage() Usage {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	return ec.usage
}

// resolveInput builds a node's input map by merging outputs from all
// predecessor nodes according to the edge field mappings.
func (ec *ExecutionContext) resolveInput(key NodeKey, inEdges []*Edge) map[string]any {
	input := make(map[string]any)
	for _, e := range inEdges {
		srcOutput := ec.GetOutput(e.From)
		if srcOutput == nil {
			continue
		}
		if len(e.FieldMapping) == 0 {
			// pass through all fields
			for k, v := range srcOutput {
				input[k] = v
			}
		} else {
			for srcField, dstField := range e.FieldMapping {
				if v, ok := srcOutput[srcField]; ok {
					input[dstField] = v
				}
			}
		}
	}
	return input
}

// contextKey is an unexported type for context.Context keys.
type contextKey int

const ctxKeyExecutionContext contextKey = iota

// WithExecutionContext attaches an ExecutionContext to a context.Context.
func WithExecutionContext(ctx context.Context, ec *ExecutionContext) context.Context {
	return context.WithValue(ctx, ctxKeyExecutionContext, ec)
}

// FromContext retrieves the ExecutionContext from a context.Context.
// Returns nil if not present.
func FromContext(ctx context.Context) *ExecutionContext {
	v, _ := ctx.Value(ctxKeyExecutionContext).(*ExecutionContext)
	return v
}
