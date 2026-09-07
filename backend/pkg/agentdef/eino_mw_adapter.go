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

package agentdef

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	aclagent "github.com/superagent-ai/superagent-base/backend/pkg/agent"
	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

// einoMwAdapter wraps an ACL agent.Middleware as an eino
// adk.ChatModelAgentMiddleware so it can be used with the eino ChatModelAgent.
type einoMwAdapter struct {
	*adk.BaseChatModelAgentMiddleware
	mw aclagent.Middleware
}

var _ adk.ChatModelAgentMiddleware = (*einoMwAdapter)(nil)

// newEinoMwAdapter wraps an ACL middleware for use with eino's ChatModelAgent.
func newEinoMwAdapter(mw aclagent.Middleware) adk.ChatModelAgentMiddleware {
	return &einoMwAdapter{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		mw:                          mw,
	}
}

func (a *einoMwAdapter) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	mCtx := &aclagent.MiddlewareContext{
		Instruction: runCtx.Instruction,
	}
	newCtx, err := a.mw.BeforeAgent(ctx, mCtx)
	return newCtx, runCtx, err
}

func (a *einoMwAdapter) AfterAgent(ctx context.Context, state *adk.ChatModelAgentState) (context.Context, error) {
	mCtx := &aclagent.MiddlewareContext{Messages: schemaToACLMessages(state.Messages)}
	return ctx, a.mw.AfterAgent(ctx, mCtx)
}

func (a *einoMwAdapter) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	mState := &aclagent.MiddlewareState{Messages: schemaToACLMessages(state.Messages)}
	newCtx, err := a.mw.BeforeModel(ctx, mState)
	if err != nil {
		return ctx, state, err
	}
	state.Messages = aclToSchemaMessages(mState.Messages)
	return newCtx, state, nil
}

func (a *einoMwAdapter) AfterModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	mState := &aclagent.MiddlewareState{Messages: schemaToACLMessages(state.Messages)}
	err := a.mw.AfterModel(ctx, mState)
	if err != nil {
		return ctx, state, err
	}
	state.Messages = aclToSchemaMessages(mState.Messages)
	return ctx, state, nil
}

func (a *einoMwAdapter) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	wrapper := &einoToolWrapper{name: tCtx.Name}
	wrapped, err := a.mw.WrapTool(ctx, wrapper)
	if err != nil {
		return nil, err
	}
	if wrapped == wrapper {
		return endpoint, nil
	}
	wt, ok := wrapped.(*einoToolWrapper)
	if !ok || !wt.denied {
		return endpoint, nil
	}
	return func(_ context.Context, _ string, _ ...tool.Option) (string, error) {
		return "", fmt.Errorf("tool %q denied by middleware", tCtx.Name)
	}, nil
}

func (a *einoMwAdapter) WrapStreamableToolCall(ctx context.Context, endpoint adk.StreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.StreamableToolCallEndpoint, error) {
	wrapper := &einoToolWrapper{name: tCtx.Name}
	wrapped, err := a.mw.WrapTool(ctx, wrapper)
	if err != nil {
		return nil, err
	}
	if wrapped == wrapper {
		return endpoint, nil
	}
	wt, ok := wrapped.(*einoToolWrapper)
	if !ok || !wt.denied {
		return endpoint, nil
	}
	return func(_ context.Context, _ string, _ ...tool.Option) (*schema.StreamReader[string], error) {
		return nil, fmt.Errorf("tool %q denied by middleware", tCtx.Name)
	}, nil
}

func (a *einoMwAdapter) WrapEnhancedInvokableToolCall(ctx context.Context, endpoint adk.EnhancedInvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.EnhancedInvokableToolCallEndpoint, error) {
	wrapper := &einoToolWrapper{name: tCtx.Name}
	wrapped, err := a.mw.WrapTool(ctx, wrapper)
	if err != nil {
		return nil, err
	}
	if wrapped == wrapper {
		return endpoint, nil
	}
	wt, ok := wrapped.(*einoToolWrapper)
	if !ok || !wt.denied {
		return endpoint, nil
	}
	return func(_ context.Context, _ *schema.ToolArgument, _ ...tool.Option) (*schema.ToolResult, error) {
		return nil, fmt.Errorf("tool %q denied by middleware", tCtx.Name)
	}, nil
}

func (a *einoMwAdapter) WrapEnhancedStreamableToolCall(ctx context.Context, endpoint adk.EnhancedStreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.EnhancedStreamableToolCallEndpoint, error) {
	wrapper := &einoToolWrapper{name: tCtx.Name}
	wrapped, err := a.mw.WrapTool(ctx, wrapper)
	if err != nil {
		return nil, err
	}
	if wrapped == wrapper {
		return endpoint, nil
	}
	wt, ok := wrapped.(*einoToolWrapper)
	if !ok || !wt.denied {
		return endpoint, nil
	}
	return func(_ context.Context, _ *schema.ToolArgument, _ ...tool.Option) (*schema.StreamReader[*schema.ToolResult], error) {
		return nil, fmt.Errorf("tool %q denied by middleware", tCtx.Name)
	}, nil
}

// einoToolWrapper is a minimal llm.Tool used to check whether a middleware
// wants to deny or wrap a tool call.
type einoToolWrapper struct {
	name   string
	denied bool
}

func (t *einoToolWrapper) Info(_ context.Context) (*llm.ToolInfo, error) {
	return &llm.ToolInfo{Name: t.name}, nil
}

func (t *einoToolWrapper) Run(_ context.Context, _ string, _ ...llm.ToolOption) (string, error) {
	return "", nil
}

// --- conversion helpers ---

func schemaToACLMessages(msgs []*schema.Message) []*llm.Message {
	result := make([]*llm.Message, len(msgs))
	for i, m := range msgs {
		result[i] = &llm.Message{
			Role:       string(m.Role),
			Content:    m.Content,
			Name:       m.Name,
			ToolCallID: m.ToolCallID,
		}
		if len(m.ToolCalls) > 0 {
			result[i].ToolCalls = make([]llm.ToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				result[i].ToolCalls[j] = llm.ToolCall{
					ID:       tc.ID,
					Name:     tc.Function.Name,
					ArgsJSON: tc.Function.Arguments,
				}
			}
		}
	}
	return result
}

func aclToSchemaMessages(msgs []*llm.Message) []*schema.Message {
	result := make([]*schema.Message, len(msgs))
	for i, m := range msgs {
		result[i] = &schema.Message{
			Role:       schema.RoleType(m.Role),
			Content:    m.Content,
			Name:       m.Name,
			ToolCallID: m.ToolCallID,
		}
		if len(m.ToolCalls) > 0 {
			result[i].ToolCalls = make([]schema.ToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				result[i].ToolCalls[j] = schema.ToolCall{
					ID: tc.ID,
					Function: schema.FunctionCall{
						Name:      tc.Name,
						Arguments: tc.ArgsJSON,
					},
				}
			}
		}
	}
	return result
}
