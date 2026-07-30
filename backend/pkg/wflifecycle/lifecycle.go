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

// Package wflifecycle defines the unified lifecycle for workflow nodes.
//
// Instead of each node type implementing its own execution logic, they
// all implement the NodeExecutor interface. The workflow engine calls
// Prepare → Execute → Finalize in order, with error handling and
// observability built in.
//
// This reduces the boilerplate for adding new node types from ~5 files
// to ~1 file (just implement NodeExecutor).
package wflifecycle

import (
	"context"
	"fmt"
	"time"
)

// NodeType identifies the kind of workflow node.
type NodeType string

const (
	NodeTypeLLM            NodeType = "llm_call"
	NodeTypeAgent          NodeType = "agent_call"
	NodeTypeTool           NodeType = "tool_call"
	NodeTypeCode           NodeType = "code"
	NodeTypeCondition      NodeType = "condition"
	NodeTypeEntry          NodeType = "entry"
	NodeTypeExit           NodeType = "exit"
	NodeTypeSubWorkflow    NodeType = "subworkflow"
	NodeTypeHTTP           NodeType = "httprequester"
	NodeTypeKnowledge      NodeType = "knowledge"
	NodeTypePlugin         NodeType = "plugin"
	NodeTypeBatch          NodeType = "batch"
	NodeTypeLoop           NodeType = "loop"
	NodeTypeTextProcessor  NodeType = "text_processor"
	NodeTypeVarAssigner    NodeType = "variable_assigner"
	NodeTypeVarAggregator  NodeType = "variable_aggregator"
	NodeTypeSelector       NodeType = "selector"
	NodeTypeIntentDetector NodeType = "intent_detector"
	NodeTypeJSON           NodeType = "json"
	NodeTypeDatabase       NodeType = "database"
	NodeTypeQA             NodeType = "question_answer"
	NodeTypeConversation   NodeType = "conversation"
	NodeTypeEmitter        NodeType = "emitter"
)

// NodeState represents the execution state of a node.
type NodeState string

const (
	StatePending   NodeState = "pending"
	StateRunning   NodeState = "running"
	StateSuccess   NodeState = "success"
	StateFailed    NodeState = "failed"
	StateSkipped   NodeState = "skipped"
	StateCancelled NodeState = "cancelled"
)

// NodeContext carries all the information a node needs to execute.
type NodeContext struct {
	// NodeID is the unique identifier for this node instance.
	NodeID string
	// NodeType is the kind of node.
	NodeType NodeType
	// Inputs contains the output values from upstream nodes.
	Inputs map[string]any
	// Variables contains the current workflow variable state.
	Variables map[string]any
	// Config contains the node-specific configuration from the YAML definition.
	Config map[string]any
	// SessionID is the current conversation session ID.
	SessionID string
	// AgentID is the current agent ID (if applicable).
	AgentID int64
	// UserID is the current user ID.
	UserID string
	// SpaceID is the current tenant/space ID.
	SpaceID int64
}

// NodeResult contains the output of a node execution.
type NodeResult struct {
	// Outputs contains the values produced by this node.
	Outputs map[string]any
	// State is the final state of the node.
	State NodeState
	// Error is set when State is StateFailed.
	Error error
	// Duration is how long the node took to execute.
	Duration time.Duration
	// Metadata contains additional information about the execution.
	Metadata map[string]any
}

// NodeExecutor is the interface that all workflow node types must implement.
// This is the single point of extension for new node types.
type NodeExecutor interface {
	// Type returns the node type this executor handles.
	Type() NodeType

	// Prepare validates the node configuration and prepares for execution.
	// Called before Execute. Return an error to abort the workflow.
	Prepare(ctx context.Context, nodeCtx *NodeContext) error

	// Execute runs the node logic and returns the result.
	Execute(ctx context.Context, nodeCtx *NodeContext) (*NodeResult, error)

	// Finalize is called after execution (success or failure) for cleanup.
	// Called even if Execute returns an error.
	Finalize(ctx context.Context, nodeCtx *NodeContext, result *NodeResult)
}

// NodeRegistry maps node types to their executors.
type NodeRegistry struct {
	executors map[NodeType]NodeExecutor
}

// NewNodeRegistry creates a new empty node registry.
func NewNodeRegistry() *NodeRegistry {
	return &NodeRegistry{
		executors: make(map[NodeType]NodeExecutor),
	}
}

// Register adds a node executor to the registry.
// Panics if an executor for the same type is already registered.
func (r *NodeRegistry) Register(executor NodeExecutor) {
	nt := executor.Type()
	if _, exists := r.executors[nt]; exists {
		panic(fmt.Sprintf("wflifecycle: executor for node type %q already registered", nt))
	}
	r.executors[nt] = executor
}

// Get returns the executor for the given node type, or nil if not registered.
func (r *NodeRegistry) Get(nt NodeType) NodeExecutor {
	return r.executors[nt]
}

// Has returns true if an executor is registered for the given node type.
func (r *NodeRegistry) Has(nt NodeType) bool {
	_, ok := r.executors[nt]
	return ok
}

// Types returns all registered node types.
func (r *NodeRegistry) Types() []NodeType {
	types := make([]NodeType, 0, len(r.executors))
	for t := range r.executors {
		types = append(types, t)
	}
	return types
}

// ExecuteNode runs the full node lifecycle: Prepare → Execute → Finalize.
// This is the single entry point the workflow engine should call.
func (r *NodeRegistry) ExecuteNode(ctx context.Context, nodeCtx *NodeContext) *NodeResult {
	executor := r.Get(nodeCtx.NodeType)
	if executor == nil {
		return &NodeResult{
			State:  StateFailed,
			Error:  fmt.Errorf("wflifecycle: no executor registered for node type %q", nodeCtx.NodeType),
		}
	}

	start := time.Now()

	// Prepare
	if err := executor.Prepare(ctx, nodeCtx); err != nil {
		result := &NodeResult{
			State:    StateFailed,
			Error:    fmt.Errorf("prepare failed: %w", err),
			Duration: time.Since(start),
		}
		executor.Finalize(ctx, nodeCtx, result)
		return result
	}

	// Execute
	result, err := executor.Execute(ctx, nodeCtx)
	if result == nil {
		result = &NodeResult{}
	}
	if err != nil {
		result.State = StateFailed
		result.Error = err
	}
	result.Duration = time.Since(start)

	// Finalize (always called)
	executor.Finalize(ctx, nodeCtx, result)

	return result
}
