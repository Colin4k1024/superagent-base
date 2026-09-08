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

// Package dag provides a framework-agnostic directed acyclic graph (DAG)
// execution engine. It is designed to replace cloudwego/eino/compose for
// workflow orchestration without depending on any LLM framework.
//
// The engine supports:
//   - Topological execution ordering
//   - Conditional branching (selector nodes)
//   - Parallel branch execution
//   - Streaming output
//   - Interrupt/resume via checkpoints
//   - Error handling with configurable exception strategies
//
// Node implementations register as NodeExecutor; the engine handles
// scheduling, input routing, and output collection.
package dag

import "context"

// NodeKey uniquely identifies a node within a graph.
type NodeKey string

// PortLabel identifies an output port on a node (for conditional branching).
type PortLabel string

// NodeExecutor is the interface every workflow node must implement.
// The engine calls Execute with the node's resolved input (a map of field
// name to value) and expects a map of output fields.
type NodeExecutor interface {
	// Execute runs the node's logic and returns its output.
	Execute(ctx context.Context, input map[string]any) (map[string]any, error)
}

// NodeMeta provides static metadata about a node.
type NodeMeta struct {
	Key         NodeKey
	Name        string
	Type        string
	InputTypes  map[string]*TypeInfo
	OutputTypes map[string]*TypeInfo
}

// TypeInfo describes the type of a field.
type TypeInfo struct {
	Kind       string // "string", "int", "float", "bool", "object", "array", "file"
	Required   bool
	ItemType   *TypeInfo            // for arrays
	Properties map[string]*TypeInfo // for objects
}

// Node is a vertex in the DAG.
type Node struct {
	Meta     NodeMeta
	Executor NodeExecutor
}

// Edge is a directed connection from one node's output port to another
// node's input. An edge with an empty Port is unconditional; an edge with
// a Port is only active when the source node selects that port.
type Edge struct {
	From NodeKey
	To   NodeKey
	Port PortLabel
	// FieldMapping maps output field names to input field names.
	// nil means pass through all fields with the same name.
	FieldMapping map[string]string
}

// InterruptInfo carries interrupt/resume context.
type InterruptInfo struct {
	NodeKey NodeKey
	Reason  string
	Data    map[string]any
}

// ExecutionResult holds the final output and metadata of a graph run.
type ExecutionResult struct {
	Output    map[string]any
	Usage     *Usage
	Interrupt *InterruptInfo
}

// Usage tracks token usage across all nodes.
type Usage struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
}

// Add merges other usage into this one.
func (u *Usage) Add(other *Usage) {
	if u == nil || other == nil {
		return
	}
	u.PromptTokens += other.PromptTokens
	u.CompletionTokens += other.CompletionTokens
	u.TotalTokens += other.TotalTokens
}
