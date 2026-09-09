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

// compose_facade_ext extends compose_facade.go with additional type aliases
// and wrapper functions for the full eino/compose API surface used by the
// codebase.
package einobridge

import (
	"context"

	"github.com/cloudwego/eino/compose"
)

// ---------------------------------------------------------------------------
// Additional type aliases
// ---------------------------------------------------------------------------

type Chain[I, O any] = compose.Chain[I, O]
type Parallel = compose.Parallel
type FieldMapping = compose.FieldMapping
type FieldMappingOption = compose.FieldMappingOption
type FieldPath = compose.FieldPath
type NodePath = compose.NodePath
type StateModifier = compose.StateModifier
type WorkflowAddInputOpt = compose.WorkflowAddInputOpt
type WorkflowBranch = compose.WorkflowBranch
type WorkflowNode = compose.WorkflowNode
type GraphCompileCallback = compose.GraphCompileCallback
type GraphInfo = compose.GraphInfo
type GraphNodeInfo = compose.GraphNodeInfo
type GraphBranchCondition[T any] = compose.GraphBranchCondition[T]
type GraphMultiBranchCondition[T any] = compose.GraphMultiBranchCondition[T]
type StreamGraphMultiBranchCondition[T any] = compose.StreamGraphMultiBranchCondition[T]
type ToolsNodeOption = compose.ToolsNodeOption
type ToolInput = compose.ToolInput
type ToolOutput = compose.ToolOutput
type StreamToolOutput = compose.StreamToolOutput
type Serializer = compose.Serializer

// Workflow is a generic workflow builder.
type Workflow[I, O any] = compose.Workflow[I, O]

// Generic func type aliases
type CollectWOOpt[I, O any] = compose.CollectWOOpt[I, O]
type Collect[I, O, TOption any] = compose.Collect[I, O, TOption]
type Invoke2[I, O, TOption any] = compose.Invoke[I, O, TOption]
type StreamWOOpt[I, O any] = compose.StreamWOOpt[I, O]
type StreamFunc[I, O, TOption any] = compose.Stream[I, O, TOption]
type Transform2[I, O, TOption any] = compose.Transform[I, O, TOption]
type StreamStatePostHandler[O, S any] = compose.StreamStatePostHandler[O, S]
type StreamStatePreHandler[I, S any] = compose.StreamStatePreHandler[I, S]

// ---------------------------------------------------------------------------
// Constants & variables
// ---------------------------------------------------------------------------

const (
	ComponentOfUnknown  = compose.ComponentOfUnknown
	ComponentOfLambda    = compose.ComponentOfLambda
	ComponentOfWorkflow = compose.ComponentOfWorkflow
	ComponentOfChain     = compose.ComponentOfChain
)

var (
	InterruptAndRerun = compose.InterruptAndRerun
	DAGInvalidLoopErr  = compose.DAGInvalidLoopErr
	ErrChainCompiled   = compose.ErrChainCompiled
	ErrExceedMaxSteps  = compose.ErrExceedMaxSteps
	ErrGraphCompiled   = compose.ErrGraphCompiled
)

// ---------------------------------------------------------------------------
// Workflow builders
// ---------------------------------------------------------------------------

func NewWorkflow[I, O any](opts ...NewGraphOption) *Workflow[I, O] {
	return compose.NewWorkflow[I, O](opts...)
}

func NewChain[I, O any](opts ...NewGraphOption) *Chain[I, O] {
	return compose.NewChain[I, O](opts...)
	
}

func NewParallel() *Parallel {
	return compose.NewParallel()
}

func NewGraphMultiBranch[T any](condition GraphMultiBranchCondition[T], endNodes map[string]bool) *GraphBranch {
	return compose.NewGraphMultiBranch[T](condition, endNodes)
}

func NewStreamGraphMultiBranch[T any](condition StreamGraphMultiBranchCondition[T], endNodes map[string]bool) *GraphBranch {
	return compose.NewStreamGraphMultiBranch[T](condition, endNodes)
}

// ---------------------------------------------------------------------------
// Functions
// ---------------------------------------------------------------------------

func GetToolCallID(ctx context.Context) string {
	return compose.GetToolCallID(ctx)
}

func IsInterruptRerunError(err error) (any, bool) {
	return compose.IsInterruptRerunError(err)
}

func NewInterruptAndRerunErr(extra any) error {
	return compose.NewInterruptAndRerunErr(extra)
}

func ToFieldPath(toFieldPath FieldPath, opts ...FieldMappingOption) *FieldMapping {
	return compose.ToFieldPath(toFieldPath, opts...)
}

func FromFieldPath(fromFieldPath FieldPath) *FieldMapping {
	return compose.FromFieldPath(fromFieldPath)
}

func FromField(from string) *FieldMapping {
	return compose.FromField(from)
}

func ToField(to string, opts ...FieldMappingOption) *FieldMapping {
	return compose.ToField(to, opts...)
}

func MapFields(from, to string) *FieldMapping {
	return compose.MapFields(from, to)
}

func MapFieldPaths(fromFieldPath, toFieldPath FieldPath) *FieldMapping {
	return compose.MapFieldPaths(fromFieldPath, toFieldPath)
}

func NewNodePath(nodeKeyPath ...string) *NodePath {
	return compose.NewNodePath(nodeKeyPath...)
}

// ---------------------------------------------------------------------------
// Additional option functions
// ---------------------------------------------------------------------------

func WithCustomExtractor(extractor func(input any) (any, error)) FieldMappingOption {
	return compose.WithCustomExtractor(extractor)
}


func WithLambdaCallbackEnable(enable bool) LambdaOpt {
	return compose.WithLambdaCallbackEnable(enable)
}

func WithLambdaType(t string) LambdaOpt {
	return compose.WithLambdaType(t)
}


func WithNoDirectDependency() WorkflowAddInputOpt {
	return compose.WithNoDirectDependency()
}


func WithStateModifier(modifier StateModifier) Option {
	return compose.WithStateModifier(modifier)
}

func WithStreamStatePostHandler[O, S any](post StreamStatePostHandler[O, S]) GraphAddNodeOpt {
	return compose.WithStreamStatePostHandler[O, S](post)
}

func WithStreamStatePreHandler[I, S any](pre StreamStatePreHandler[I, S]) GraphAddNodeOpt {
	return compose.WithStreamStatePreHandler[I, S](pre)
}

func WithToolOption(opts ...ToolOption) ToolsNodeOption {
	return compose.WithToolOption(opts...)
}

func WithToolsNodeOption(opts ...ToolsNodeOption) Option {
	return compose.WithToolsNodeOption(opts...)
}
type Runnable[I, O any] = compose.Runnable[I, O]

// AnyLambda creates a Lambda with any lambda function.
func AnyLambda[I, O, TOption any](i Invoke2[I, O, TOption], s StreamFunc[I, O, TOption],
	c Collect[I, O, TOption], t Transform2[I, O, TOption], opts ...LambdaOpt) (*Lambda, error) {
	return compose.AnyLambda[I, O, TOption](i, s, c, t, opts...)
}
func WithLambdaOption(opts ...any) Option {
	return compose.WithLambdaOption(opts...)
}
