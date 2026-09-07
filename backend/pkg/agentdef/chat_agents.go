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
	"errors"
	"fmt"
	"io"
	"strings"
	"time"


	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
	aclagent "github.com/superagent-ai/superagent-base/backend/pkg/agent"
	adkagent "github.com/superagent-ai/superagent-base/backend/pkg/agent/adk"
	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
	"github.com/superagent-ai/superagent-base/backend/pkg/modelrouter"
	"github.com/superagent-ai/superagent-base/backend/pkg/observe"
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

// einoChatAgent calls a ChatModel directly (no tool loop).
// Uses the framework-agnostic llm.ChatModel interface via the ACL.
type einoChatAgent struct {
	def           *AgentDefinition
	modelID       string
	provider      string
	memBackend    memory.Backend
	chatModel     llm.ChatModel
	systemPrompt  string
	modelSelector *DynamicModelSelector // nil when dynamic routing is not configured
}

func (a *einoChatAgent) Name() string                    { return a.def.Metadata.Name }
func (a *einoChatAgent) Description() string             { return a.systemPrompt }
func (a *einoChatAgent) GetDefinition() *AgentDefinition { return a.def }

func (a *einoChatAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	msgs := make([]*llm.Message, 0, 8)
	if a.systemPrompt != "" {
		msgs = append(msgs, llm.SystemMessage(a.systemPrompt))
	}

	if a.memBackend != nil && sessionID != "" {
		history, err := a.memBackend.GetMessages(ctx, sessionID, memory.GetMessagesOpts{Limit: 20})
		if err == nil {
			for _, m := range history {
				switch m.Role {
				case "user":
					msgs = append(msgs, llm.UserMessage(m.Content))
				case "assistant":
					msgs = append(msgs, llm.AssistantMessage(m.Content, nil))
				}
			}
		}
	}

	msgs = append(msgs, llm.UserMessage(message))

	if a.memBackend != nil && sessionID != "" {
		_ = a.memBackend.AddMessage(ctx, sessionID, memory.Message{
			Role:      "user",
			Content:   message,
			Timestamp: time.Now().Unix(),
		})
	}

	// Dynamic model selection based on message complexity.
	activeModel := a.chatModel
	activeModelID := a.modelID
	if a.modelSelector != nil {
		selected, complexity := a.modelSelector.SelectModel(ctx, msgs)
		if selected != nil {
			activeModel = selected
			if tierDef, ok := findModelTier(a.def, complexity); ok {
				activeModelID = tierDef.ModelID
			}
		}
	}

	ctx = observe.WithModelInfo(ctx, activeModelID, a.provider)
	reader, err := activeModel.Stream(ctx, msgs)
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
					modelrouter.RecordModelLatency(activeModelID, a.provider, time.Since(streamStart))
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

// findModelTier returns the ModelTier definition for the given complexity level.
func findModelTier(def *AgentDefinition, complexity string) (ModelTier, bool) {
	for _, tier := range def.Spec.Model.Models {
		if tier.Level == complexity {
			return tier, true
		}
	}
	return ModelTier{}, false
}

// adkRunnerAgent wraps a Google ADK Go AgentAdapter for tool-using interactions.
// This is the sole tool-using agent runtime after the eino removal.
type adkRunnerAgent struct {
	def          *AgentDefinition
	modelID      string
	provider     string
	memBackend   memory.Backend
	agent        *adkagent.AgentAdapter
	systemPrompt string
}

func (a *adkRunnerAgent) Name() string                    { return a.def.Metadata.Name }
func (a *adkRunnerAgent) Description() string             { return a.systemPrompt }
func (a *adkRunnerAgent) GetDefinition() *AgentDefinition { return a.def }

func (a *adkRunnerAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	msgs := buildMessageHistory(ctx, a.systemPrompt, sessionID, a.memBackend)
	msgs = append(msgs, llm.UserMessage(message))
	persistUserMessage(ctx, sessionID, message, a.memBackend)

	ctx = observe.WithModelInfo(ctx, a.modelID, a.provider)

	iter, err := a.agent.Run(ctx, &aclagent.AgentInput{
		Messages:       msgs,
		EnableStreaming: true,
		MaxIterations:  10,
	})
	if err != nil {
		return nil, fmt.Errorf("agentdef: adk runner chat: %w", err)
	}

	ch := make(chan string, 64)
	go func() {
		defer close(ch)
		var fullResponse strings.Builder
		for {
			event, ok := iter.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				select {
				case ch <- fmt.Sprintf("[error] %v", event.Err):
				case <-ctx.Done():
				}
				return
			}
			if event.MessageOutput != nil {
				if event.MessageOutput.IsStreaming && event.MessageOutput.MessageStream != nil {
					for {
						chunk, recvErr := event.MessageOutput.MessageStream.Recv()
						if errors.Is(recvErr, io.EOF) {
							break
						}
						if recvErr != nil {
							break
						}
						if chunk != nil && chunk.Content != "" {
							fullResponse.WriteString(chunk.Content)
							select {
							case ch <- chunk.Content:
							case <-ctx.Done():
								return
							}
						}
					}
				} else if event.MessageOutput.Message != nil && event.MessageOutput.Message.Content != "" {
					fullResponse.WriteString(event.MessageOutput.Message.Content)
					select {
					case ch <- event.MessageOutput.Message.Content:
					case <-ctx.Done():
						return
					}
				}
			}
		}
		if a.memBackend != nil && sessionID != "" && fullResponse.Len() > 0 {
			_ = a.memBackend.AddMessage(ctx, sessionID, memory.Message{
				Role:      "assistant",
				Content:   fullResponse.String(),
				Timestamp: time.Now().Unix(),
			})
		}
	}()
	return ch, nil
}
