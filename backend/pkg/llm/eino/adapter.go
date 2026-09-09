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

// Package eino provides adapters that bridge cloudwego/eino's model and tool
// implementations to the framework-agnostic pkg/llm interfaces. This is the
// anti-corruption layer (ACL): business code imports pkg/llm, never
// cloudwego/eino directly.
package eino

import (
	"context"
	"fmt"

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"

	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

// ChatModelAdapter wraps an eino ToolCallingChatModel as an llm.ChatModel.
type ChatModelAdapter struct {
	chatModel einobridge.ToolCallingChatModel
	modelID   string
}

// NewChatModelAdapter creates an llm.ChatModel from an eino chat model.
func NewChatModelAdapter(chatModel einobridge.ToolCallingChatModel, modelID string) *ChatModelAdapter {
	return &ChatModelAdapter{chatModel: chatModel, modelID: modelID}
}

// ID returns the model identifier.
func (a *ChatModelAdapter) ID() string { return a.modelID }

// Generate produces a single (non-streaming) response.
func (a *ChatModelAdapter) Generate(ctx context.Context, msgs []*llm.Message) (*llm.Message, error) {
	einoMsgs := ToEinoMessages(msgs)
	resp, err := a.chatModel.Generate(ctx, einoMsgs)
	if err != nil {
		return nil, fmt.Errorf("eino adapter: generate: %w", err)
	}
	return FromEinoMessage(resp), nil
}

// Stream produces a streaming response reader.
func (a *ChatModelAdapter) Stream(ctx context.Context, msgs []*llm.Message) (*llm.StreamReader, error) {
	einoMsgs := ToEinoMessages(msgs)
	reader, err := a.chatModel.Stream(ctx, einoMsgs)
	if err != nil {
		return nil, fmt.Errorf("eino adapter: stream: %w", err)
	}
	return llm.NewStreamReader(
		func() (*llm.Message, error) {
			chunk, recvErr := reader.Recv()
			if recvErr != nil {
				return nil, recvErr
			}
			return FromEinoMessage(chunk), nil
		},
		func() error {
			reader.Close(); return nil
		},
	), nil
}

// BindTools associates tool definitions with the model.
// This adapter converts llm.Tool to eino einobridge.ToolInfo via ToolAdapter.Info,
// then calls eino's WithTools which returns a new model instance.
func (a *ChatModelAdapter) BindTools(tools []llm.Tool) (llm.ChatModel, error) {
	einoToolInfos := make([]*einobridge.ToolInfo, 0, len(tools))
	for _, t := range tools {
		info, err := t.Info(context.Background())
		if err != nil {
			return nil, fmt.Errorf("eino adapter: bind tools: get info for %T: %w", t, err)
		}
		einoToolInfos = append(einoToolInfos, &einobridge.ToolInfo{
			Name: info.Name,
			Desc: info.Desc,
		})
	}
	bound, err := a.chatModel.WithTools(einoToolInfos)
	if err != nil {
		return nil, fmt.Errorf("eino adapter: with tools: %w", err)
	}
	return &ChatModelAdapter{chatModel: bound, modelID: a.modelID}, nil
}

// --- Message conversion helpers ---

// ToEinoMessages converts a slice of llm.Message to eino einobridge.Message.
func ToEinoMessages(msgs []*llm.Message) []*einobridge.Message {
	result := make([]*einobridge.Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, toEinoMessage(m))
	}
	return result
}

// toEinoMessage converts a single llm.Message to eino einobridge.Message.
func toEinoMessage(m *llm.Message) *einobridge.Message {
	if m == nil {
		return nil
	}
	switch m.Role {
	case llm.RoleSystem:
		return einobridge.SystemMessage(m.Content)
	case llm.RoleUser:
		return einobridge.UserMessage(m.Content)
	case llm.RoleAssistant:
		var toolCalls []einobridge.ToolCall
		for _, tc := range m.ToolCalls {
			toolCalls = append(toolCalls, einobridge.ToolCall{
				ID:       tc.ID,
				Function: einobridge.FunctionCall{Name: tc.Name, Arguments: tc.ArgsJSON},
			})
		}
		return einobridge.AssistantMessage(m.Content, toolCalls)
	case llm.RoleTool:
		return einobridge.ToolMessage(m.Content, m.ToolCallID)
	default:
		return &einobridge.Message{Role: einobridge.RoleType(m.Role), Content: m.Content}
	}
}

// FromEinoMessage converts an eino einobridge.Message to llm.Message.
func FromEinoMessage(m *einobridge.Message) *llm.Message {
	if m == nil {
		return nil
	}
	result := &llm.Message{
		Role:    string(m.Role),
		Content: m.Content,
	}
	if m.Name != "" {
		result.Name = m.Name
	}
	if m.ToolCallID != "" {
		result.ToolCallID = m.ToolCallID
	}
	for _, tc := range m.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, llm.ToolCall{
			ID:       tc.ID,
			Name:     tc.Function.Name,
			ArgsJSON: tc.Function.Arguments,
		})
	}
	return result
}

var _ llm.ChatModel = (*ChatModelAdapter)(nil)

// UnwrapEinoModel returns the underlying eino ToolCallingChatModel.
// This is used during the migration transition period where eino's adk
// still requires the raw eino model type. Once the migration to the
// target framework is complete, this method will be removed.
func (a *ChatModelAdapter) UnwrapEinoModel() einobridge.ToolCallingChatModel {
	return a.chatModel
}
