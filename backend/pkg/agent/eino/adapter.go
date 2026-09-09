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

// Package eino provides an adapter that bridges eino's einobridge.AdkChatModelAgent
// and adk.Runner to the framework-agnostic pkg/agent.AgentRuntime interface.
package eino

import (
	"context"
	"fmt"
	"strings"

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"

	"github.com/superagent-ai/superagent-base/backend/pkg/agent"
	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
	einollm "github.com/superagent-ai/superagent-base/backend/pkg/llm/eino"
)

// AgentAdapter wraps an eino einobridge.AdkChatModelAgent as an agent.AgentRuntime.
type AgentAdapter struct {
	name        string
	description string
	agent       *einobridge.AdkChatModelAgent
	store       einobridge.AdkCheckPointStore
}

// NewAgentAdapter creates an agent.AgentRuntime from an eino ChatModelAgent.
// store may be nil (no interrupt/resume persistence).
func NewAgentAdapter(a *einobridge.AdkChatModelAgent, name, description string, store einobridge.AdkCheckPointStore) *AgentAdapter {
	return &AgentAdapter{
		name:        name,
		description: description,
		agent:       a,
		store:       store,
	}
}

func (a *AgentAdapter) Name() string       { return a.name }
func (a *AgentAdapter) Description() string { return a.description }

// Run executes the agent with the given input.
func (a *AgentAdapter) Run(ctx context.Context, input *agent.AgentInput) (*agent.EventIterator, error) {
	einoMsgs := einollm.ToEinoMessages(input.Messages)

	iter := a.agent.Run(ctx, &einobridge.AdkAgentInput{
		Messages:       einoMsgs,
		EnableStreaming: input.EnableStreaming,
	})

	return adaptIterator(iter), nil
}

// Resume continues a previously interrupted agent run.
func (a *AgentAdapter) Resume(ctx context.Context, input *agent.ResumeInput) (*agent.EventIterator, error) {
	msgs := []*llm.Message{llm.UserMessage(input.UserMessage)}
	einoMsgs := einollm.ToEinoMessages(msgs)

	iter := a.agent.Run(ctx, &einobridge.AdkAgentInput{
		Messages:       einoMsgs,
		EnableStreaming: true,
	})

	return adaptIterator(iter), nil
}

// adaptIterator converts an eino einobridge.AdkAsyncIterator to agent.EventIterator.
func adaptIterator(iter *einobridge.AdkAsyncIterator[*einobridge.AdkAgentEvent]) *agent.EventIterator {
	return agent.NewEventIterator(
		func() (*agent.AgentEvent, bool) {
			event, ok := iter.Next()
			if !ok {
				return nil, false
			}
			return fromEinoEvent(event), true
		},
		nil,
	)
}

// fromEinoEvent converts an eino einobridge.AdkAgentEvent to agent.AgentEvent.
func fromEinoEvent(event *einobridge.AdkAgentEvent) *agent.AgentEvent {
	if event == nil {
		return nil
	}
	result := &agent.AgentEvent{}
	if event.Err != nil {
		result.Err = event.Err
		return result
	}

	if event.Action != nil && event.Action.Interrupted != nil {
		// ChatModelAgentInterruptInfo has Info and Data []byte fields.
		result.Action = &agent.EventAction{
			Interrupted: &agent.InterruptAction{
				CheckpointData: toBytes(event.Action.Interrupted.Data),
			},
		}
	}

	if event.Output != nil && event.Output.MessageOutput != nil {
		mv := event.Output.MessageOutput
		result.MessageOutput = &agent.MessageOutput{
			IsStreaming: mv.IsStreaming,
		}
		if mv.Message != nil {
			result.MessageOutput.Message = einollm.FromEinoMessage(mv.Message)
		}
		if mv.IsStreaming && mv.MessageStream != nil {
			result.MessageOutput.MessageStream = adaptMessageStream(mv.MessageStream)
		}
	}

	return result
}

// adaptMessageStream wraps an eino einobridge.StreamReader as llm.StreamReader.
func adaptMessageStream(stream *einobridge.StreamReader[*einobridge.Message]) *llm.StreamReader {
	if stream == nil {
		return nil
	}
	return llm.NewStreamReader(
		func() (*llm.Message, error) {
			chunk, err := stream.Recv()
			if err != nil {
				return nil, err
			}
			return einollm.FromEinoMessage(chunk), nil
		},
		func() error {
			stream.Close()
			return nil
		},
	)
}

// drainIterator consumes remaining events to allow internal goroutines to exit.
func drainIterator(iter *einobridge.AdkAsyncIterator[*einobridge.AdkAgentEvent]) {
	go func() {
		for {
			if _, ok := iter.Next(); !ok {
				break
			}
		}
	}()
}

var _ agent.AgentRuntime = (*AgentAdapter)(nil)

// Avoid unused import warnings during partial implementation.
var (
	_ = fmt.Sprintf
	_ = strings.Builder{}
)

// toBytes extracts a []byte from an any value.
func toBytes(v any) []byte {
	if v == nil {
		return nil
	}
	if b, ok := v.([]byte); ok {
		return b
	}
	return nil
}
