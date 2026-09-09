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
// and wrapper functions. This file has ZERO cloudwego/eino imports.
package einobridge

import (
	"context"

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"

	nativecompose "github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/compose"
)

// ---------------------------------------------------------------------------
// Type aliases
// ---------------------------------------------------------------------------

type Chain[I, O any] = nativecompose.Graph[I, O]
type Parallel = nativecompose.Parallel
type FieldMapping = nativecompose.FieldMapping
type FieldMappingOption = nativecompose.FieldMappingOption
type FieldPath = nativecompose.FieldPath
type NodePath = nativecompose.NodePath
type StateModifier = func(ctx context.Context, path NodePath, state any) error
type WorkflowAddInputOpt = nativecompose.NewGraphOption
type WorkflowBranch = nativecompose.GraphBranch
type WorkflowNode = nativecompose.Lambda
type GraphCompileCallback = func(ctx context.Context, info any) error
type GraphInfo = nativecompose.GraphInfo
type GraphNodeInfo = nativecompose.GraphNodeInfo
type GraphBranchCondition[T any] = nativecompose.GraphBranchCondition[T]
type GraphMultiBranchCondition[T any] = nativecompose.GraphMultiBranchCondition[T]
type StreamGraphMultiBranchCondition[T any] = nativecompose.StreamGraphMultiBranchCondition[T]
type ToolsNodeOption = nativecompose.ToolsNodeOption
type ToolInput = any
type ToolOutput = any
type StreamToolOutput = any
type Serializer = nativecompose.Serializer

// Workflow is a generic workflow builder.
type Workflow[I, O any] = nativecompose.Workflow[I, O]

// Generic func type aliases
type CollectWOOpt[I, O any] = nativecompose.CollectWOOpt[I, O]
type Collect[I, O, TOption any] = nativecompose.Collect[I, O, TOption]
type Invoke2[I, O, TOption any] = nativecompose.Invoke[I, O, TOption]
type StreamWOOpt[I, O any] = nativecompose.StreamWOOpt[I, O]
type StreamFunc[I, O, TOption any] = nativecompose.Stream[I, O, TOption]
type Transform2[I, O, TOption any] = nativecompose.Transform[I, O, TOption]
type StreamStatePostHandler[O, S any] = func(ctx context.Context, output *StreamReader[O], state S) (*StreamReader[O], error)
type StreamStatePreHandler[I, S any] = func(ctx context.Context, input *StreamReader[I], state S) (*StreamReader[I], error)

// ---------------------------------------------------------------------------
// Constants & variables
// ---------------------------------------------------------------------------

const (
	ComponentOfUnknown  = nativecompose.ComponentOfUnknown
	ComponentOfLambda    = nativecompose.ComponentOfLambda
	ComponentOfWorkflow = nativecompose.ComponentOfWorkflow
	ComponentOfChain     = nativecompose.ComponentOfChain
)

var (
	InterruptAndRerun = nativecompose.InterruptAndRerun
	DAGInvalidLoopErr  = nativecompose.DAGInvalidLoopErr
	ErrChainCompiled   = nativecompose.ErrChainCompiled
	ErrExceedMaxSteps  = nativecompose.ErrExceedMaxSteps
	ErrGraphCompiled   = nativecompose.ErrGraphCompiled
)

// ---------------------------------------------------------------------------
// Workflow builders
// ---------------------------------------------------------------------------

func NewWorkflow[I, O any](opts ...NewGraphOption) *Workflow[I, O] {
	return nativecompose.NewWorkflow[I, O]()
}

func NewChain[I, O any](opts ...NewGraphOption) *Chain[I, O] {
	return nativecompose.NewGraph[I, O](opts...)
}

func NewParallel() *Parallel {
	return nativecompose.NewParallel()
}

func NewGraphMultiBranch[T any](condition GraphMultiBranchCondition[T], endNodes map[string]bool) *GraphBranch {
	return nativecompose.NewGraphMultiBranch[T](condition, endNodes)
}

func NewStreamGraphMultiBranch[T any](condition StreamGraphMultiBranchCondition[T], endNodes map[string]bool) *GraphBranch {
	return nativecompose.NewStreamGraphMultiBranch[T](condition, endNodes)
}

// ---------------------------------------------------------------------------
// Functions
// ---------------------------------------------------------------------------

func GetToolCallID(ctx context.Context) string {
	return nativecompose.GetToolCallID(ctx)
}

func IsInterruptRerunError(err error) (any, bool) {
	return nativecompose.IsInterruptRerunError(err)
}

func NewInterruptAndRerunErr(extra any) error {
	return nativecompose.NewInterruptAndRerunErr(extra)
}

func ToFieldPath(toFieldPath FieldPath, opts ...FieldMappingOption) *FieldMapping {
	return nativecompose.ToFieldPath(toFieldPath, opts...)
}

func FromFieldPath(fromFieldPath FieldPath) *FieldMapping {
	return nativecompose.FromFieldPath(fromFieldPath)
}

func FromField(from string) *FieldMapping {
	return nativecompose.FromField(from)
}

func ToField(to string, opts ...FieldMappingOption) *FieldMapping {
	return nativecompose.ToField(to, opts...)
}

func MapFields(from, to string) *FieldMapping {
	return nativecompose.MapFields(from, to)
}

func MapFieldPaths(fromFieldPath, toFieldPath FieldPath) *FieldMapping {
	return nativecompose.MapFieldPaths(fromFieldPath, toFieldPath)
}

func NewNodePath(nodeKeyPath ...string) *NodePath {
	return nativecompose.NewNodePath(nodeKeyPath...)
}

// ---------------------------------------------------------------------------
// Additional option functions
// ---------------------------------------------------------------------------

func WithCustomExtractor(extractor func(input any) (any, error)) FieldMappingOption {
	return nativecompose.WithCustomExtractor(extractor)
}

func WithLambdaCallbackEnable(enable bool) LambdaOpt {
	return nativecompose.WithLambdaCallbackEnable(enable)
}

func WithLambdaType(t string) LambdaOpt {
	return nativecompose.WithLambdaType(t)
}

func WithNoDirectDependency() WorkflowAddInputOpt {
	return func(gc *nativecompose.GraphConfigShim) {}
}

func WithStateModifier(modifier StateModifier) Option {
	return wfcompose.NewOption(modifier)
}

func WithStreamStatePostHandler[O, S any](post StreamStatePostHandler[O, S]) GraphAddNodeOpt {
	return func(nc *nativecompose.NodeConfig) {}
}

func WithStreamStatePreHandler[I, S any](pre StreamStatePreHandler[I, S]) GraphAddNodeOpt {
	return func(nc *nativecompose.NodeConfig) {}
}

func WithToolOption(opts ...ToolOption) ToolsNodeOption {
	return nativecompose.WithToolOption()
}

func WithToolsNodeOption(opts ...ToolsNodeOption) Option {
	return nativecompose.WithToolsNodeOption(opts...)
}

type Runnable[I, O any] = *nativecompose.Runnable[I, O]

func AnyLambda[I, O, TOption any](i Invoke2[I, O, TOption], s StreamFunc[I, O, TOption],
	c Collect[I, O, TOption], t Transform2[I, O, TOption], opts ...LambdaOpt) (*Lambda, error) {
	return nativecompose.AnyLambda[I, O, TOption](i, s, c, t, opts...)
}

func WithLambdaOption(opts ...any) Option {
	return wfcompose.NewOption(opts)
}


// AnyGraph is an untyped graph that any Graph can be assigned to.
type AnyGraph any


// ToolsInterruptAndRerunExtra holds extra data for tool interrupt and rerun.
type ToolsInterruptAndRerunExtra struct {
	ToolCallID       string
	ArgumentsInJSON  string
	RerunExtraMap    map[string]any
}
// WithCallbacks returns an Option that registers callback handlers.
func WithCallbacks(handlers ...Handler) Option {
	return wfcompose.NewOption(nativecompose.WithCallbacks(handlers))
}
