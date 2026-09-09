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

// compose_facade re-exports cloudwego/eino/compose types and functions as
// type aliases and thin wrappers so agentflow can avoid importing
// cloudwego/eino/compose directly (S2 acceptance criterion).
package einobridge

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

// ---------------------------------------------------------------------------
// Type aliases
// ---------------------------------------------------------------------------

type NewGraphOption = compose.NewGraphOption
type GraphAddNodeOpt = compose.GraphAddNodeOpt
type GraphCompileOption = compose.GraphCompileOption
type NodeTriggerMode = compose.NodeTriggerMode
type AnyGraph = compose.AnyGraph
type Graph[I, O any] = compose.Graph[I, O]
type Lambda = compose.Lambda
type GraphBranch = compose.GraphBranch
type ToolsNode = compose.ToolsNode
type ToolsNodeConfig = compose.ToolsNodeConfig
type CheckPointStore = compose.CheckPointStore
type InterruptInfo = compose.InterruptInfo
type ToolsInterruptAndRerunExtra = compose.ToolsInterruptAndRerunExtra
type Option = compose.Option
type LambdaOpt = compose.LambdaOpt

type GenLocalState[S any] = compose.GenLocalState[S]
type StatePreHandler[I, S any] = compose.StatePreHandler[I, S]
type StatePostHandler[O, S any] = compose.StatePostHandler[O, S]
type StreamGraphBranchCondition[T any] = compose.StreamGraphBranchCondition[T]
type InvokeWOOpt[I, O any] = compose.InvokeWOOpt[I, O]
type TransformWOOpts[I, O any] = compose.TransformWOOpts[I, O]

// ---------------------------------------------------------------------------
// Constants & variables
// ---------------------------------------------------------------------------

const (
	START                = compose.START
	END                  = compose.END
	ComponentOfGraph     = compose.ComponentOfGraph
	ComponentOfToolsNode = compose.ComponentOfToolsNode
)

var (
	AllPredecessor NodeTriggerMode = compose.AllPredecessor
	AnyPredecessor NodeTriggerMode = compose.AnyPredecessor
)

// ---------------------------------------------------------------------------
// Runnable adapter
// ---------------------------------------------------------------------------

type RunnableAdapter[I, O any] struct {
	Inner compose.Runnable[I, O]
}

func NewRunnableAdapter[I, O any](r compose.Runnable[I, O]) *RunnableAdapter[I, O] {
	return &RunnableAdapter[I, O]{Inner: r}
}

func (a *RunnableAdapter[I, O]) Stream(ctx context.Context, input I, opts ...wfcompose.Option) (*wfcompose.StreamReader[O], error) {
	einoOpts := UnwrapOptionSlice(opts...)
	sr, err := a.Inner.Stream(ctx, input, einoOpts...)
	if err != nil {
		return nil, err
	}
	return WrapStreamReader[O](sr), nil
}

// ---------------------------------------------------------------------------
// Graph builder + lambda helpers
// ---------------------------------------------------------------------------

func NewGraph[I, O any](opts ...NewGraphOption) *Graph[I, O] {
	return compose.NewGraph[I, O](opts...)
}

func InvokableLambda[I, O any](fn InvokeWOOpt[I, O], opts ...LambdaOpt) *Lambda {
	return compose.InvokableLambda[I, O](fn, opts...)
}

func TransformableLambda[I, O any](fn TransformWOOpts[I, O], opts ...LambdaOpt) *Lambda {
	return compose.TransformableLambda[I, O](fn, opts...)
}

func ToList[I any](opts ...LambdaOpt) *Lambda {
	return compose.ToList[I](opts...)
}

func NewToolNode(ctx context.Context, conf *ToolsNodeConfig) (*ToolsNode, error) {
	return compose.NewToolNode(ctx, conf)
}

func NewStreamGraphBranch[T any](condition StreamGraphBranchCondition[T], endNodes map[string]bool) *GraphBranch {
	return compose.NewStreamGraphBranch[T](condition, endNodes)
}

func ProcessState[S any](ctx context.Context, handler func(context.Context, S) error) error {
	return compose.ProcessState[S](ctx, handler)
}

func ExtractInterruptInfo(err error) (*InterruptInfo, bool) {
	return compose.ExtractInterruptInfo(err)
}

func RegisterSerializableType[T any](name string) error {
	return compose.RegisterSerializableType[T](name)
}

// ---------------------------------------------------------------------------
// Option wrappers
// ---------------------------------------------------------------------------

func WithCallbacks(cbs ...callbacks.Handler) Option {
	return compose.WithCallbacks(cbs...)
}

func WithCheckPointID(id string) Option {
	return compose.WithCheckPointID(id)
}

func WithGenLocalState[S any](gls GenLocalState[S]) NewGraphOption {
	return compose.WithGenLocalState[S](gls)
}

func WithStatePreHandler[I, S any](pre StatePreHandler[I, S]) GraphAddNodeOpt {
	return compose.WithStatePreHandler[I, S](pre)
}

func WithStatePostHandler[O, S any](post StatePostHandler[O, S]) GraphAddNodeOpt {
	return compose.WithStatePostHandler[O, S](post)
}

func WithOutputKey(key string) GraphAddNodeOpt {
	return compose.WithOutputKey(key)
}

func WithNodeName(name string) GraphAddNodeOpt {
	return compose.WithNodeName(name)
}

func WithNodeTriggerMode(mode NodeTriggerMode) GraphCompileOption {
	return compose.WithNodeTriggerMode(mode)
}

func WithGraphCompileOptions(opts ...GraphCompileOption) GraphAddNodeOpt {
	return compose.WithGraphCompileOptions(opts...)
}

func WithMaxRunSteps(steps int) GraphCompileOption {
	return compose.WithMaxRunSteps(steps)
}

func WithGraphName(name string) GraphCompileOption {
	return compose.WithGraphName(name)
}

func WithCheckPointStore(store CheckPointStore) GraphCompileOption {
	return compose.WithCheckPointStore(store)
}

// Compile wraps compose.Graph.Compile, returning a RunnableAdapter
// that satisfies wfcompose.Runnable.
func Compile[I, O any](g *Graph[I, O], ctx context.Context, opts ...GraphCompileOption) (*RunnableAdapter[I, O], error) {
	r, err := g.Compile(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return NewRunnableAdapter[I, O](r), nil
}
