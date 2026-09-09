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

package a2ui

import (
	"context"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/tool"

	"github.com/superagent-ai/superagent-base/backend/pkg/observe"
)

// streamKey is a private context key type for storing an EventStream in context.
type streamKey struct{}

// WithEventStream attaches an EventStream to the context so the A2UI callback
// can emit events during tool/model execution.
func WithEventStream(ctx context.Context, stream *EventStream) context.Context {
	return context.WithValue(ctx, streamKey{}, stream)
}

// streamFromCtx retrieves the EventStream from context. Returns nil if absent.
func streamFromCtx(ctx context.Context) *EventStream {
	s, _ := ctx.Value(streamKey{}).(*EventStream)
	return s
}

// NewA2UICallback creates an eino callback handler that injects tool_call and
// tool_result events into the A2UI EventStream stored in context.
// This enables structured rendering of tool usage on the frontend.
func NewA2UICallback() einobridge.Handler {
	cb := &a2uiCallback{}
	return einobridge.NewHandlerBuilder().
		OnStartFn(cb.OnStart).
		OnEndFn(cb.OnEnd).
		OnErrorFn(cb.OnError).
		Build()
}

type a2uiCallback struct{}

func (c *a2uiCallback) OnStart(ctx context.Context, info *einobridge.RunInfo, input einobridge.CallbackInput) context.Context {
	stream := streamFromCtx(ctx)
	if stream == nil {
		return ctx
	}
	aclInfo := einoToA2UIRunInfo(info)
	if aclInfo.Component == observe.ComponentTool {
		args := extractToolArgs(input)
		stream.SendToolCall(info.Name, info.Name, args)
	}
	return ctx
}

func (c *a2uiCallback) OnEnd(ctx context.Context, info *einobridge.RunInfo, output einobridge.CallbackOutput) context.Context {
	stream := streamFromCtx(ctx)
	if stream == nil {
		return ctx
	}
	aclInfo := einoToA2UIRunInfo(info)
	if aclInfo.Component == observe.ComponentTool {
		result := extractToolResult(output)
		stream.SendToolResult(info.Name, info.Name, result, false)
	}
	return ctx
}

func (c *a2uiCallback) OnError(ctx context.Context, info *einobridge.RunInfo, err error) context.Context {
	stream := streamFromCtx(ctx)
	if stream == nil {
		return ctx
	}
	aclInfo := einoToA2UIRunInfo(info)
	if aclInfo.Component == observe.ComponentTool {
		stream.SendToolResult(info.Name, info.Name, err.Error(), true)
	}
	return ctx
}

// einoToA2UIRunInfo converts eino's einobridge.RunInfo to ACL CallbackRunInfo.
func einoToA2UIRunInfo(info *einobridge.RunInfo) observe.CallbackRunInfo {
	comp := observe.ComponentOther
	switch info.Component {
	case components.ComponentOfChatModel:
		comp = observe.ComponentChatModel
	case components.ComponentOfTool:
		comp = observe.ComponentTool
	}
	return observe.CallbackRunInfo{Component: comp, Name: info.Name}
}

// extractToolArgs attempts to extract tool arguments from callback input.
func extractToolArgs(input einobridge.CallbackInput) map[string]any {
	if input == nil {
		return nil
	}
	in := tool.ConvCallbackInput(input)
	if in == nil {
		return nil
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(in.ArgumentsInJSON), &args); err == nil {
		return args
	}
	if in.ArgumentsInJSON != "" {
		return map[string]any{"input": in.ArgumentsInJSON}
	}
	return nil
}

// extractToolResult attempts to extract a string result from callback output.
func extractToolResult(output einobridge.CallbackOutput) string {
	if output == nil {
		return ""
	}
	out := tool.ConvCallbackOutput(output)
	if out == nil {
		return fmt.Sprintf("%v", output)
	}
	result := out.Response
	if len(result) > 2000 {
		result = result[:2000] + "...(truncated)"
	}
	return result
}
