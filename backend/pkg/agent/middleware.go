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

package agent

import (
	"context"

	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

// Middleware is the framework-agnostic agent middleware interface.
// It hooks into agent execution at agent-level, model-level, and tool-level
// points without depending on any specific LLM framework (eino, ADK Go, etc.).
//
// Implementations embed BaseMiddleware to get default no-op implementations
// for hooks they do not need.
type Middleware interface {
	// BeforeAgent is called before the agent starts executing.
	// The returned context is passed to subsequent hooks.
	// Returning a non-nil error aborts the run.
	BeforeAgent(ctx context.Context, mCtx *MiddlewareContext) (context.Context, error)

	// AfterAgent is called after the agent completes (or errors).
	AfterAgent(ctx context.Context, mCtx *MiddlewareContext) error

	// BeforeModel is called before each model invocation.
	// The returned context is passed to AfterModel.
	// The middleware may modify mState.Messages in place.
	BeforeModel(ctx context.Context, mState *MiddlewareState) (context.Context, error)

	// AfterModel is called after each model invocation.
	// The middleware may modify mState.Messages in place.
	AfterModel(ctx context.Context, mState *MiddlewareState) error

	// WrapTool wraps a tool for permission checking, logging, etc.
	// Return the original tool unchanged if no wrapping is needed.
	WrapTool(ctx context.Context, tool llm.Tool) (llm.Tool, error)

	// WrapModel wraps the chat model for failover, caching, etc.
	// Return the original model unchanged if no wrapping is needed.
	WrapModel(ctx context.Context, model llm.ChatModel) (llm.ChatModel, error)
}

// BaseMiddleware provides default no-op implementations for Middleware.
// Embed this to only override the hooks you need.
type BaseMiddleware struct{}

func (BaseMiddleware) BeforeAgent(ctx context.Context, _ *MiddlewareContext) (context.Context, error) {
	return ctx, nil
}
func (BaseMiddleware) AfterAgent(context.Context, *MiddlewareContext) error { return nil }
func (BaseMiddleware) BeforeModel(ctx context.Context, _ *MiddlewareState) (context.Context, error) {
	return ctx, nil
}
func (BaseMiddleware) AfterModel(context.Context, *MiddlewareState) error { return nil }
func (BaseMiddleware) WrapTool(ctx context.Context, t llm.Tool) (llm.Tool, error) {
	return t, nil
}
func (BaseMiddleware) WrapModel(ctx context.Context, m llm.ChatModel) (llm.ChatModel, error) {
	return m, nil
}

// MiddlewareContext carries agent-level execution context.
type MiddlewareContext struct {
	Instruction string         // user instruction/prompt
	Tools       []llm.Tool     // available tools
	Messages    []*llm.Message // conversation history
}

// MiddlewareState carries model-level execution state.
// Messages is mutable: middlewares may prepend, append, or modify messages.
type MiddlewareState struct {
	Messages []*llm.Message
}
