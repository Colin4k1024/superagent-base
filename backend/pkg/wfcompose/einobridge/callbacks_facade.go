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

// callbacks_facade provides native callback types that replace
// cloudwego/eino/callbacks. This file has ZERO cloudwego/eino imports.
package einobridge

import (
	"context"

)

// Handler is the callback handler interface.
type Handler interface {
	OnStart(ctx context.Context, info *RunInfo, input CallbackInput) context.Context
	OnEnd(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context
	OnError(ctx context.Context, info *RunInfo, err error) context.Context
	OnStartWithStreamInput(ctx context.Context, info *RunInfo, input *StreamReader[CallbackInput]) context.Context
	OnEndWithStreamOutput(ctx context.Context, info *RunInfo, output *StreamReader[CallbackOutput]) context.Context
}

// RunInfo contains metadata about the current execution step.
type RunInfo struct {
	Name      string
	Type      string
	Component Component
}

// CallbackInput is an opaque input type for callbacks.
type CallbackInput any

// CallbackOutput is an opaque output type for callbacks.
type CallbackOutput any

// HandlerBuilder builds callback handlers.
type HandlerBuilder struct {
	onStart  func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context
	onEnd    func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context
	onError  func(ctx context.Context, info *RunInfo, err error) context.Context
}

func NewHandlerBuilder() *HandlerBuilder { return &HandlerBuilder{} }

func (b *HandlerBuilder) OnStart(fn func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context) *HandlerBuilder {
	b.onStart = fn
	return b
}

func (b *HandlerBuilder) OnEnd(fn func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context) *HandlerBuilder {
	b.onEnd = fn
	return b
}

func (b *HandlerBuilder) OnError(fn func(ctx context.Context, info *RunInfo, err error) context.Context) *HandlerBuilder {
	b.onError = fn
	return b
}

type builtHandler struct {
	onStart func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context
	onEnd   func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context
	onError func(ctx context.Context, info *RunInfo, err error) context.Context
}

func (h *builtHandler) OnStart(ctx context.Context, info *RunInfo, input CallbackInput) context.Context {
	if h.onStart != nil {
		return h.onStart(ctx, info, input)
	}
	return ctx
}

func (h *builtHandler) OnEnd(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context {
	if h.onEnd != nil {
		return h.onEnd(ctx, info, output)
	}
	return ctx
}

func (h *builtHandler) OnError(ctx context.Context, info *RunInfo, err error) context.Context {
	if h.onError != nil {
		return h.onError(ctx, info, err)
	}
	return ctx
}

// OnStartWithStreamInput is a no-op in native compose.
func (h *builtHandler) OnStartWithStreamInput(ctx context.Context, info *RunInfo, input *StreamReader[CallbackInput]) context.Context {
	return ctx
}

func (h *builtHandler) OnEndWithStreamOutput(ctx context.Context, info *RunInfo, output *StreamReader[CallbackOutput]) context.Context {
	return ctx
}

func (b *HandlerBuilder) Build() Handler {
	return &builtHandler{onStart: b.onStart, onEnd: b.onEnd, onError: b.onError}
}

// Native callback functions (no-ops, since we don't have a callback dispatch system yet)

func EnsureRunInfo(ctx context.Context, component string, comp Component) context.Context {
	return ctx
}

func AppendGlobalHandlers(handlers ...Handler) {
	// no-op: native compose doesn't have global handlers yet
}

func OnStart[T any](ctx context.Context, input T) context.Context {
	return ctx
}

func OnEnd[T any](ctx context.Context, output T) context.Context {
	return ctx
}

func OnError(ctx context.Context, err error) context.Context {
	return ctx
}

func OnEndWithStreamOutput[T any](ctx context.Context, output *StreamReader[T]) (context.Context, *StreamReader[T]) {
	return ctx, output
}

func OnStartWithStreamInput[T any](ctx context.Context, input *StreamReader[T]) (context.Context, *StreamReader[T]) {
	return ctx, input
}

// HandlerHelper provides a fluent API for building callback handlers.
type HandlerHelper struct {
	builder *HandlerBuilder
}

func NewHandlerHelper() *HandlerHelper {
	return &HandlerHelper{builder: NewHandlerBuilder()}
}

// ToolCallbackHandler is a callback handler for tool operations.
type ToolCallbackHandler struct {
	OnStart               func(ctx context.Context, info *RunInfo, input *ToolCallbackInput) context.Context
	OnEnd                 func(ctx context.Context, info *RunInfo, output *ToolCallbackOutput) context.Context
	OnEndWithStreamOutput func(ctx context.Context, info *RunInfo, output *StreamReader[*ToolCallbackOutput]) context.Context
	OnError               func(ctx context.Context, info *RunInfo, err error) context.Context
}

// ModelCallbackHandler is a callback handler for model operations.
type ModelCallbackHandler struct {
	OnStart               func(ctx context.Context, info *RunInfo, input *ModelCallbackInput) context.Context
	OnEnd                 func(ctx context.Context, info *RunInfo, output *ModelCallbackOutput) context.Context
	OnEndWithStreamOutput func(ctx context.Context, info *RunInfo, output *StreamReader[*ModelCallbackOutput]) context.Context
	OnError               func(ctx context.Context, info *RunInfo, err error) context.Context
}

// OnStartFn sets the OnStart function (alias for OnStart).
func (b *HandlerBuilder) OnStartFn(fn func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context) *HandlerBuilder {
	return b.OnStart(fn)
}

// OnEndFn sets the OnEnd function (alias for OnEnd).
func (b *HandlerBuilder) OnEndFn(fn func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context) *HandlerBuilder {
	return b.OnEnd(fn)
}

// OnErrorFn sets the OnError function (alias for OnError).
func (b *HandlerBuilder) OnErrorFn(fn func(ctx context.Context, info *RunInfo, err error) context.Context) *HandlerBuilder {
	return b.OnError(fn)
}

// OnEndWithStreamOutputFn sets a no-op OnEndWithStreamOutput handler.
func (b *HandlerBuilder) OnEndWithStreamOutputFn(fn func(ctx context.Context, info *RunInfo, output *StreamReader[CallbackOutput]) context.Context) *HandlerBuilder {
	// no-op: native compose doesn't support stream callbacks yet
	return b
}

// Tool configures the builder with a ToolCallbackHandler and returns the helper.
func (h *HandlerHelper) Tool(th *ToolCallbackHandler) *HandlerHelper {
	if th != nil {
		h.builder.onStart = func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context {
			if th.OnStart != nil {
				ci := ToolConvCallbackInput(input)
				if ci != nil {
					return th.OnStart(ctx, info, ci)
				}
			}
			return ctx
		}
		h.builder.onEnd = func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context {
			if th.OnEnd != nil {
				co := ToolConvCallbackOutput(output)
				if co != nil {
					return th.OnEnd(ctx, info, co)
				}
			}
			return ctx
		}
		h.builder.onError = th.OnError
	}
	return h
}

// ChatModel configures the builder with a ModelCallbackHandler and returns the helper.
func (h *HandlerHelper) ChatModel(mh *ModelCallbackHandler) *HandlerHelper {
	if mh != nil {
		h.builder.onStart = func(ctx context.Context, info *RunInfo, input CallbackInput) context.Context {
			if mh.OnStart != nil {
				ci := ModelConvCallbackInput(input)
				if ci != nil {
					return mh.OnStart(ctx, info, ci)
				}
			}
			return ctx
		}
		h.builder.onEnd = func(ctx context.Context, info *RunInfo, output CallbackOutput) context.Context {
			if mh.OnEnd != nil {
				co := ModelConvCallbackOutput(output)
				if co != nil {
					return mh.OnEnd(ctx, info, co)
				}
			}
			return ctx
		}
		h.builder.onError = mh.OnError
	}
	return h
}

// Handler returns the built Handler.
func (h *HandlerHelper) Handler() Handler {
	return h.builder.Build()
}
