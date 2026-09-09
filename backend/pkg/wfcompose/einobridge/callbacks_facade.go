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

// callbacks_facade re-exports cloudwego/eino/callbacks and
// cloudwego/eino/utils/callbacks types as aliases so callers avoid
// importing eino directly (S4 migration).
package einobridge

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	ucallbacks "github.com/cloudwego/eino/utils/callbacks"
)

// ---------------------------------------------------------------------------
// eino/callbacks type aliases
// ---------------------------------------------------------------------------

type Handler = callbacks.Handler
type RunInfo = callbacks.RunInfo
type CallbackInput = callbacks.CallbackInput
type CallbackOutput = callbacks.CallbackOutput
type HandlerBuilder = callbacks.HandlerBuilder

// ---------------------------------------------------------------------------
// eino/callbacks functions
// ---------------------------------------------------------------------------

func EnsureRunInfo(ctx context.Context, component string, comp Component) context.Context {
	return callbacks.EnsureRunInfo(ctx, component, comp)
}

func AppendGlobalHandlers(handlers ...Handler) {
	callbacks.AppendGlobalHandlers(handlers...)
}

func NewHandlerBuilder() *HandlerBuilder {
	return callbacks.NewHandlerBuilder()
}

func OnStart[T any](ctx context.Context, input T) context.Context {
	return callbacks.OnStart[T](ctx, input)
}

func OnEnd[T any](ctx context.Context, output T) context.Context {
	return callbacks.OnEnd[T](ctx, output)
}

func OnError(ctx context.Context, err error) context.Context {
	return callbacks.OnError(ctx, err)
}

func OnEndWithStreamOutput[T any](ctx context.Context, output *StreamReader[T]) (context.Context, *StreamReader[T]) {
	return callbacks.OnEndWithStreamOutput[T](ctx, output)
}

func OnStartWithStreamInput[T any](ctx context.Context, input *StreamReader[T]) (context.Context, *StreamReader[T]) {
	return callbacks.OnStartWithStreamInput[T](ctx, input)
}

// ---------------------------------------------------------------------------
// eino/utils/callbacks type aliases
// ---------------------------------------------------------------------------

type HandlerHelper = ucallbacks.HandlerHelper
type ToolCallbackHandler = ucallbacks.ToolCallbackHandler
type ModelCallbackHandler = ucallbacks.ModelCallbackHandler

func NewHandlerHelper() *HandlerHelper {
	return ucallbacks.NewHandlerHelper()
}
