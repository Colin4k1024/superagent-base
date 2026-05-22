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
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
	"github.com/superagent-ai/superagent-base/backend/pkg/modelrouter"
)

// chatAgent is the stub implementation used when no real model endpoint is
// configured. It satisfies the Agent interface for testing purposes.
type chatAgent struct {
	def        *AgentDefinition
	modelID    string
	tools      []resolvedTool
	memBackend memory.Backend
}

func (a *chatAgent) Name() string                    { return a.def.Metadata.Name }
func (a *chatAgent) Description() string             { return a.def.Spec.SystemPrompt }
func (a *chatAgent) GetDefinition() *AgentDefinition { return a.def }

func (a *chatAgent) Chat(_ context.Context, _ string, message string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- fmt.Sprintf("[%s] placeholder response for: %s", a.modelID, message)
	close(ch)
	return ch, nil
}

// einoChatAgent calls an Eino ChatModel directly (no tool loop).
type einoChatAgent struct {
	def          *AgentDefinition
	modelID      string
	provider     string
	memBackend   memory.Backend
	chatModel    model.ToolCallingChatModel
	systemPrompt string
}

func (a *einoChatAgent) Name() string                    { return a.def.Metadata.Name }
func (a *einoChatAgent) Description() string             { return a.systemPrompt }
func (a *einoChatAgent) GetDefinition() *AgentDefinition { return a.def }

func (a *einoChatAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	msgs := make([]*schema.Message, 0, 8)
	if a.systemPrompt != "" {
		msgs = append(msgs, schema.SystemMessage(a.systemPrompt))
	}

	if a.memBackend != nil && sessionID != "" {
		history, err := a.memBackend.GetMessages(ctx, sessionID, memory.GetMessagesOpts{Limit: 20})
		if err == nil {
			for _, m := range history {
				switch m.Role {
				case "user":
					msgs = append(msgs, schema.UserMessage(m.Content))
				case "assistant":
					msgs = append(msgs, schema.AssistantMessage(m.Content, nil))
				}
			}
		}
	}

	msgs = append(msgs, schema.UserMessage(message))

	if a.memBackend != nil && sessionID != "" {
		_ = a.memBackend.AddMessage(ctx, sessionID, memory.Message{
			Role:      "user",
			Content:   message,
			Timestamp: time.Now().Unix(),
		})
	}

	reader, err := a.chatModel.Stream(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("agentdef: chat: stream: %w", err)
	}

	ch := make(chan string, 64)
	go func() {
		defer close(ch)
		defer reader.Close()
		var fullResponse strings.Builder
		streamStart := time.Now()
		firstToken := true
		for {
			chunk, err := reader.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					select {
					case ch <- fmt.Sprintf("[error] %v", err):
					case <-ctx.Done():
					}
				}
				if a.memBackend != nil && sessionID != "" && fullResponse.Len() > 0 {
					_ = a.memBackend.AddMessage(ctx, sessionID, memory.Message{
						Role:      "assistant",
						Content:   fullResponse.String(),
						Timestamp: time.Now().Unix(),
					})
				}
				return
			}
			if chunk != nil && chunk.Content != "" {
				if firstToken {
					modelrouter.RecordModelLatency(a.modelID, a.provider, time.Since(streamStart))
					firstToken = false
				}
				fullResponse.WriteString(chunk.Content)
				select {
				case ch <- chunk.Content:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch, nil
}

// adkChatModelAgent wraps an ADK ChatModelAgent for tool-using interactions.
type adkChatModelAgent struct {
	def          *AgentDefinition
	modelID      string
	provider     string
	memBackend   memory.Backend
	agent        *adk.ChatModelAgent
	systemPrompt string
}

func (a *adkChatModelAgent) Name() string                    { return a.def.Metadata.Name }
func (a *adkChatModelAgent) Description() string             { return a.systemPrompt }
func (a *adkChatModelAgent) GetDefinition() *AgentDefinition { return a.def }

func (a *adkChatModelAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	// Build history BEFORE persisting so the current message isn't loaded twice.
	msgs := buildMessageHistory(ctx, a.systemPrompt, sessionID, a.memBackend)
	msgs = append(msgs, schema.UserMessage(message))
	persistUserMessage(ctx, sessionID, message, a.memBackend)

	iter := a.agent.Run(ctx, &adk.AgentInput{
		Messages:       msgs,
		EnableStreaming: true,
	})

	ch := make(chan string, 64)
	params := streamConsumerParams{
		sessionID:  sessionID,
		modelID:    a.modelID,
		provider:   a.provider,
		memBackend: a.memBackend,
	}
	go consumeADKIterator(ctx, params, iter, ch, nil)
	return ch, nil
}
