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

package observe

import (
	"context"

	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

// CallbackComponent identifies the type of component that triggered the callback.
type CallbackComponent string

const (
	ComponentChatModel CallbackComponent = "chat_model"
	ComponentTool      CallbackComponent = "tool"
	ComponentAgent     CallbackComponent = "agent"
	ComponentOther     CallbackComponent = "other"
)

// CallbackRunInfo carries framework-agnostic run metadata.
// It replaces eino's einobridge.RunInfo so business code does not import eino.
type CallbackRunInfo struct {
	Component CallbackComponent
	Name      string // model ID, tool name, or agent name
}

// CallbackInput carries input data for a callback event.
type CallbackInput struct {
	// Messages are the input messages (for chat model components).
	Messages []*llm.Message
	// Raw is the raw input for non-model components.
	Raw any
}

// CallbackOutput carries output data for a callback event.
type CallbackOutput struct {
	// Message is the output message (for chat model components).
	Message *llm.Message
	// PromptTokens is the number of input tokens consumed.
	PromptTokens int
	// CompletionTokens is the number of output tokens generated.
	CompletionTokens int
	// Raw is the raw output for non-model components.
	Raw any
}

// ObservabilityCallback is the framework-agnostic callback interface
// for feeding OpenTelemetry tracing and Prometheus metrics from
// component lifecycle events.  It abstracts eino's einobridge.Handler
// and Google ADK Go's BeforeAgentCallbacks/AfterAgentCallbacks.
type ObservabilityCallback interface {
	// OnStart is called when a component starts executing.
	OnStart(ctx context.Context, info CallbackRunInfo, input CallbackInput) context.Context

	// OnEnd is called when a component finishes (non-streaming).
	OnEnd(ctx context.Context, info CallbackRunInfo, output CallbackOutput) context.Context

	// OnEndWithStream is called when a component finishes (streaming).
	// The caller drains the stream reader to collect token counts.
	OnEndWithStream(ctx context.Context, info CallbackRunInfo, reader *llm.StreamReader) context.Context

	// OnError is called when a component errors.
	OnError(ctx context.Context, info CallbackRunInfo, err error) context.Context
}

// defaultCallback is the global ACL callback instance.
var defaultCallback ObservabilityCallback

// SetCallback sets the global ACL callback for observability.
func SetCallback(cb ObservabilityCallback) { defaultCallback = cb }

// GetCallback returns the global ACL callback.
func GetCallback() ObservabilityCallback { return defaultCallback }
