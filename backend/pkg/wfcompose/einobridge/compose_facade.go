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

// compose_facade re-exports wfcompose/compose types and functions as
// type aliases and thin wrappers so callers avoid importing eino directly.
// This file has ZERO cloudwego/eino imports.
package einobridge

import (
	"context"

	nativecompose "github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/compose"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

// ---------------------------------------------------------------------------
// Type aliases
// ---------------------------------------------------------------------------

type NewGraphOption = nativecompose.NewGraphOption
type GraphAddNodeOpt = nativecompose.GraphAddNodeOpt
type GraphCompileOption = nativecompose.GraphCompileOption
type NodeTriggerMode = nativecompose.NodeTriggerMode
type Graph[I, O any] = nativecompose.Graph[I, O]
type Lambda = nativecompose.Lambda
type GraphBranch = nativecompose.GraphBranch
type ToolsNode = nativecompose.ToolsNode
type ToolsNodeConfig = nativecompose.ToolsNodeConfig
type CheckPointStore = wfcompose.CheckPointStore
type InterruptInfo = nativecompose.InterruptInfo
type Option = wfcompose.Option
type LambdaOpt = nativecompose.LambdaOpt

type GenLocalState[S any] = func(ctx context.Context) S
type StatePreHandler[I, S any] = func(ctx context.Context, input I, state S) (I, error)
type StatePostHandler[O, S any] = func(ctx context.Context, output O, state S) (O, error)
type StreamGraphBranchCondition[T any] = nativecompose.StreamGraphBranchCondition[T]
type InvokeWOOpt[I, O any] = nativecompose.InvokeWOOpt[I, O]
type TransformWOOpts[I, O any] = nativecompose.TransformWOOpts[I, O]

// ---------------------------------------------------------------------------
// Constants & variables
// ---------------------------------------------------------------------------

const (
	START                = nativecompose.START
	END                  = nativecompose.END
	ComponentOfGraph     = nativecompose.ComponentOfGraph
	ComponentOfToolsNode = nativecompose.ComponentOfToolsNode
)

var (
	AllPredecessor NodeTriggerMode = nativecompose.AllPredecessor
	AnyPredecessor NodeTriggerMode = nativecompose.AnyPredecessor
)

// ---------------------------------------------------------------------------
// Runnable adapter
// ---------------------------------------------------------------------------

type RunnableAdapter[I, O any] struct {
	Inner *nativecompose.Runnable[I, O]
}

func NewRunnableAdapter[I, O any](r *nativecompose.Runnable[I, O]) *RunnableAdapter[I, O] {
	return &RunnableAdapter[I, O]{Inner: r}
}

func (a *RunnableAdapter[I, O]) Stream(ctx context.Context, input I, opts ...wfcompose.Option) (*wfcompose.StreamReader[O], error) {
	return a.Inner.Stream(ctx, input, opts...)
}

// ---------------------------------------------------------------------------
// Graph builder + lambda helpers
// ---------------------------------------------------------------------------

func NewGraph[I, O any](opts ...NewGraphOption) *Graph[I, O] {
	return nativecompose.NewGraph[I, O](opts...)
}

func InvokableLambda[I, O any](fn InvokeWOOpt[I, O], opts ...LambdaOpt) *Lambda {
	return nativecompose.InvokableLambda[I, O](fn, opts...)
}

func TransformableLambda[I, O any](fn TransformWOOpts[I, O], opts ...LambdaOpt) *Lambda {
	return nativecompose.TransformableLambda[I, O](fn, opts...)
}

func ToList[I any](opts ...LambdaOpt) *Lambda {
	return nativecompose.ToList[I](opts...)
}

func NewToolNode(ctx context.Context, conf *ToolsNodeConfig) (*ToolsNode, error) {
	return nativecompose.NewToolNode(ctx, conf)
}

func NewStreamGraphBranch[T any](condition StreamGraphBranchCondition[T], endNodes map[string]bool) *GraphBranch {
	return nativecompose.NewStreamGraphBranch[T](condition, endNodes)
}

func ProcessState[S any](ctx context.Context, handler func(context.Context, S) error) error {
	return nativecompose.ProcessState[S](ctx, handler)
}

func ExtractInterruptInfo(err error) (*InterruptInfo, bool) {
	return nativecompose.ExtractInterruptInfo(err)
}

func RegisterSerializableType[T any](name string) error {
	return nativecompose.RegisterSerializableType[T](name)
}

// ---------------------------------------------------------------------------
// Option wrappers
// ---------------------------------------------------------------------------

func WithCheckPointID(id string) Option {
	return wfcompose.NewOption(id)
}

func WithGenLocalState[S any](gls GenLocalState[S]) NewGraphOption {
	return nativecompose.WithGenLocalState[S](gls)
}

func WithStatePreHandler[I, S any](pre StatePreHandler[I, S]) GraphAddNodeOpt {
	return func(nc *nativecompose.NodeConfig) {
		nc.StatePreHandler = pre
	}
}

func WithStatePostHandler[O, S any](post StatePostHandler[O, S]) GraphAddNodeOpt {
	return func(nc *nativecompose.NodeConfig) {
		nc.StatePostHandler = post
	}
}

func WithOutputKey(key string) GraphAddNodeOpt {
	return nativecompose.WithOutputKey(key)
}

func WithNodeName(name string) GraphAddNodeOpt {
	return nativecompose.WithNodeName(name)
}

func WithNodeTriggerMode(mode NodeTriggerMode) GraphCompileOption {
	return nativecompose.WithNodeTriggerMode(mode)
}

func WithGraphCompileOptions(opts ...GraphCompileOption) GraphAddNodeOpt {
	return nativecompose.WithGraphCompileOptions(opts...)
}

func WithMaxRunSteps(steps int) GraphCompileOption {
	return nativecompose.WithMaxRunSteps(steps)
}

func WithGraphName(name string) GraphCompileOption {
	return nativecompose.WithGraphName(name)
}

func WithCheckPointStore(store CheckPointStore) GraphCompileOption {
	return nativecompose.WithCheckPointStore(store)
}

// Compile wraps nativecompose.Graph.Compile, returning a RunnableAdapter.
func Compile[I, O any](g *Graph[I, O], ctx context.Context, opts ...GraphCompileOption) (*RunnableAdapter[I, O], error) {
	r, err := nativecompose.Compile[I, O](g, ctx, opts...)
	if err != nil {
		return nil, err
	}
	return NewRunnableAdapter[I, O](r), nil
}
