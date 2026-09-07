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

package agent

import (
	"context"

	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

// AgentRuntime is the framework-agnostic agent execution interface.
// It abstracts eino's adk.ChatModelAgent + adk.Runner
// and Google ADK Go's llmagent + runner.Runner.
type AgentRuntime interface {
	// Name returns the agent's unique identifier.
	Name() string

	// Description returns a human-readable description.
	Description() string

	// Run executes the agent with the given input and returns an event iterator.
	Run(ctx context.Context, input *AgentInput) (*EventIterator, error)

	// Resume continues a previously interrupted agent run.
	Resume(ctx context.Context, input *ResumeInput) (*EventIterator, error)
}

// AgentInput is the input to an agent run.
type AgentInput struct {
	Messages       []*llm.Message // conversation history (system + user + prior turns)
	EnableStreaming bool            // if true, stream token chunks
	Tools          []llm.Tool      // tools available to the agent
	MaxIterations  int             // max tool-calling iterations (0 = default)
}

// ResumeInput carries user response to an interrupted run.
type ResumeInput struct {
	SessionID   string
	UserMessage string
}

// EventIterator is a framework-agnostic iterator over agent run events.
type EventIterator struct {
	next  func() (*AgentEvent, bool)
	close func() error
}

// NewEventIterator creates an EventIterator from next/close functions.
func NewEventIterator(next func() (*AgentEvent, bool), close func() error) *EventIterator {
	return &EventIterator{next: next, close: close}
}

// Next returns the next event and whether it is valid.
func (it *EventIterator) Next() (*AgentEvent, bool) {
	if it == nil || it.next == nil {
		return nil, false
	}
	return it.next()
}

// Close releases any resources held by the iterator.
func (it *EventIterator) Close() error {
	if it == nil || it.close == nil {
		return nil
	}
	return it.close()
}

// AgentEvent represents a single event in an agent run stream.
type AgentEvent struct {
	// MessageOutput contains the model's output (may be nil for non-message events).
	MessageOutput *MessageOutput

	// Action describes side-effects (tool calls, interrupts).
	Action *EventAction

	// Err is non-nil if this event represents an error.
	Err error
}

// MessageOutput holds the model's output message, either streamed or complete.
type MessageOutput struct {
	// Message is the complete output (non-streaming).
	Message *llm.Message

	// MessageStream is the streaming reader (when EnableStreaming = true).
	MessageStream *llm.StreamReader

	// IsStreaming indicates whether MessageStream should be consumed.
	IsStreaming bool
}

// EventAction describes side-effects that occurred during the run.
type EventAction struct {
	// Interrupted is non-nil when the agent requests user confirmation.
	Interrupted *InterruptAction

	// ToolCalls is the list of tool calls made in this event.
	ToolCalls []llm.ToolCall
}

// InterruptAction carries interrupt context for human-in-the-loop.
type InterruptAction struct {
	// SessionID is the session that was interrupted.
	SessionID string

	// CheckpointData is opaque data needed to resume the run later.
	CheckpointData []byte
}
