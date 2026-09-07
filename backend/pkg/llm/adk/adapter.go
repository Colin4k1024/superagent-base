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

// Package adk provides adapters that bridge Google ADK Go's model.LLM and
// runner.Runner to the framework-agnostic pkg/llm interfaces. This is the
// target-side anti-corruption layer: once eino is fully removed, business
// code imports pkg/llm and the concrete implementation is this ADK adapter.
package adk

import (
	"context"
	"fmt"
	"encoding/json"
	"iter"
	"io"

	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"

	aclllm "github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

// ChatModelAdapter wraps a Google ADK Go model.LLM as an aclllm.ChatModel.
// This is the target implementation that will replace the eino adapter
// once the migration is complete.
type ChatModelAdapter struct {
	adkLLM   model.LLM
	modelID  string
}

// NewChatModelAdapter creates an aclllm.ChatModel from a Google ADK Go LLM.
func NewChatModelAdapter(llm model.LLM, modelID string) *ChatModelAdapter {
	return &ChatModelAdapter{adkLLM: llm, modelID: modelID}
}

// ID returns the model identifier.
func (a *ChatModelAdapter) ID() string { return a.modelID }

// Generate produces a single (non-streaming) response.
func (a *ChatModelAdapter) Generate(ctx context.Context, msgs []*aclllm.Message) (*aclllm.Message, error) {
	req, err := buildLLMRequest(msgs, nil)
	if err != nil {
		return nil, fmt.Errorf("adk adapter: build request: %w", err)
	}

	for resp, err := range a.adkLLM.GenerateContent(ctx, req, false) {
		if err != nil {
			return nil, fmt.Errorf("adk adapter: generate: %w", err)
		}
		if resp != nil && resp.Content != nil {
			return fromGenaiContent(resp.Content), nil
		}
	}

	return nil, fmt.Errorf("adk adapter: generate: no response")
}

// Stream produces a streaming response reader.
func (a *ChatModelAdapter) Stream(ctx context.Context, msgs []*aclllm.Message) (*aclllm.StreamReader, error) {
	req, err := buildLLMRequest(msgs, nil)
	if err != nil {
		return nil, fmt.Errorf("adk adapter: build request: %w", err)
	}

	seq := a.adkLLM.GenerateContent(ctx, req, true)
	pull, stop := iter.Pull2(seq)
	defer stop()
	defer func() { _ = pull }()

	// Channel-based bridge: pull first chunk synchronously to check for errors.
	first, firstErr, ok := pull()
	if !ok {
		return nil, fmt.Errorf("adk adapter: stream: no data")
	}
	if firstErr != nil {
		return nil, fmt.Errorf("adk adapter: stream: %w", firstErr)
	}

	// Create a channel that feeds chunks from the iterator.
	ch := make(chan *aclllm.Message, 64)
	go func() {
		defer close(ch)
		// Push first chunk.
		if first != nil && first.Content != nil {
			ch <- fromGenaiContent(first.Content)
		}
		// Pull remaining chunks.
		for {
			resp, err, ok := pull()
			if !ok {
				return
			}
			if err != nil {
				return
			}
			if resp != nil && resp.Content != nil {
				select {
				case ch <- fromGenaiContent(resp.Content):
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return aclllm.NewStreamReader(
		func() (*aclllm.Message, error) {
			msg, ok := <-ch
			if !ok {
				return nil, io.EOF
			}
			return msg, nil
		},
		nil,
	), nil
}

// BindTools associates tool definitions with the model.
// ADK Go uses genai.FunctionDeclaration for tool schemas.
func (a *ChatModelAdapter) BindTools(tools []aclllm.Tool) (aclllm.ChatModel, error) {
	// ADK Go handles tools at the agent level, not the model level.
	// The tools are passed via LLMRequest.Tools which is set at call time.
	// This adapter stores the tools and applies them in Generate/Stream.
	return &chatModelWithTools{
		ChatModelAdapter: a,
		tools:           tools,
	}, nil
}

// chatModelWithTools extends ChatModelAdapter with bound tools.
type chatModelWithTools struct {
	*ChatModelAdapter
	tools []aclllm.Tool
}

func (c *chatModelWithTools) Generate(ctx context.Context, msgs []*aclllm.Message) (*aclllm.Message, error) {
	req, err := buildLLMRequest(msgs, c.tools)
	if err != nil {
		return nil, fmt.Errorf("adk adapter: build request: %w", err)
	}
	for resp, err := range c.adkLLM.GenerateContent(ctx, req, false) {
		if err != nil {
			return nil, fmt.Errorf("adk adapter: generate: %w", err)
		}
		if resp != nil && resp.Content != nil {
			return fromGenaiContent(resp.Content), nil
		}
	}
	return nil, fmt.Errorf("adk adapter: generate: no response")
}

func (c *chatModelWithTools) Stream(ctx context.Context, msgs []*aclllm.Message) (*aclllm.StreamReader, error) {
	req, err := buildLLMRequest(msgs, c.tools)
	if err != nil {
		return nil, fmt.Errorf("adk adapter: build request: %w", err)
	}
	seq := c.adkLLM.GenerateContent(ctx, req, true)
	pull, stop := iter.Pull2(seq)
	defer stop()

	first, firstErr, ok := pull()
	if !ok {
		return nil, fmt.Errorf("adk adapter: stream: no data")
	}
	if firstErr != nil {
		return nil, fmt.Errorf("adk adapter: stream: %w", firstErr)
	}

	ch := make(chan *aclllm.Message, 64)
	go func() {
		defer close(ch)
		if first != nil && first.Content != nil {
			ch <- fromGenaiContent(first.Content)
		}
		for {
			resp, err, ok := pull()
			if !ok {
				return
			}
			if err != nil {
				return
			}
			if resp != nil && resp.Content != nil {
				select {
				case ch <- fromGenaiContent(resp.Content):
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return aclllm.NewStreamReader(
		func() (*aclllm.Message, error) {
			msg, ok := <-ch
			if !ok {
				return nil, io.EOF
			}
			return msg, nil
		},
		nil,
	), nil
}

// --- Message conversion helpers ---

// buildLLMRequest converts ACL messages to an ADK Go LLMRequest.
func buildLLMRequest(msgs []*aclllm.Message, tools []aclllm.Tool) (*model.LLMRequest, error) {
	contents := make([]*genai.Content, 0, len(msgs))
	var systemInstruction *genai.Content

	for _, m := range msgs {
		if m == nil {
			continue
		}
		switch m.Role {
		case aclllm.RoleSystem:
			// ADK Go handles system messages via system instruction.
			systemInstruction = &genai.Content{
				Parts: []*genai.Part{{Text: m.Content}},
				Role:  string(genai.RoleUser),
			}
		case aclllm.RoleUser:
			contents = append(contents, &genai.Content{
				Parts: []*genai.Part{{Text: m.Content}},
				Role:  string(genai.RoleUser),
			})
		case aclllm.RoleAssistant:
			parts := make([]*genai.Part, 0, 1+len(m.ToolCalls))
			if m.Content != "" {
				parts = append(parts, &genai.Part{Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				parts = append(parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						ID:   tc.ID,
						Name: tc.Name,
						Args: parseArgsJSON(tc.ArgsJSON),
					},
				})
			}
			contents = append(contents, &genai.Content{
				Parts: parts,
				Role:  string(genai.RoleModel),
			})
		case aclllm.RoleTool:
			contents = append(contents, &genai.Content{
				Parts: []*genai.Part{{
					FunctionResponse: &genai.FunctionResponse{
						ID:     m.ToolCallID,
						Name:   m.Name,
						Response: map[string]any{"result": m.Content},
					},
				}},
				Role: string(genai.RoleUser),
			})
		}
	}

	req := &model.LLMRequest{
		Model:    "",
		Contents: contents,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: systemInstruction,
		},
	}

	// Add tools as function declarations.
	if len(tools) > 0 {
		toolDefs := make([]*genai.FunctionDeclaration, 0, len(tools))
		for _, t := range tools {
			info, err := t.Info(context.Background())
			if err != nil {
				continue
			}
			toolDefs = append(toolDefs, &genai.FunctionDeclaration{
				Name:        info.Name,
				Description: info.Desc,
			})
		}
		req.Config.Tools = []*genai.Tool{{FunctionDeclarations: toolDefs}}
	}

	return req, nil
}

// fromGenaiContent converts a genai.Content to an ACL Message.
func fromGenaiContent(c *genai.Content) *aclllm.Message {
	if c == nil {
		return nil
	}

	role := aclllm.RoleAssistant
	if c.Role == string(genai.RoleUser) {
		role = aclllm.RoleUser
	}

	msg := &aclllm.Message{Role: role}

	for _, part := range c.Parts {
		if part == nil {
			continue
		}
		if part.Text != "" {
			msg.Content += part.Text
		}
		if part.FunctionCall != nil {
			argsBytes, _ := json.Marshal(part.FunctionCall.Args)
			argsJSON := string(argsBytes)
			msg.ToolCalls = append(msg.ToolCalls, aclllm.ToolCall{
				ID:       part.FunctionCall.ID,
				Name:     part.FunctionCall.Name,
				ArgsJSON: argsJSON,
			})
		}
	}

	return msg
}

// parseArgsJSON parses a JSON string into a map.
func parseArgsJSON(s string) map[string]any {
	if s == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return m
}


// UnwrapADKModel returns the underlying Google ADK Go model.LLM.
// This is used when the agent runtime adapter needs the raw ADK model
// to construct an llmagent.
func (a *ChatModelAdapter) UnwrapADKModel() model.LLM {
	return a.adkLLM
}
