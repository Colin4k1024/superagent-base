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

// adk_facade provides native agent runtime types that replace eino's ADK.
// This file has ZERO cloudwego/eino imports.
package einobridge

import (
	"context"

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

// AdkChatModelAgent is a native agent runtime backed by wfcompose.
type AdkChatModelAgent struct {
	model  wfcompose.ToolCallingChatModel
	tools  []wfcompose.BaseTool
	maxStep int
}

// NewAdkChatModelAgent creates a native ADK-style agent.
func NewAdkChatModelAgent(model wfcompose.ToolCallingChatModel, tools []wfcompose.BaseTool, maxStep int) *AdkChatModelAgent {
	if maxStep <= 0 {
		maxStep = 10
	}
	return &AdkChatModelAgent{model: model, tools: tools, maxStep: maxStep}
}

// AdkAgentInput is the input to an agent run.
type AdkAgentInput struct {
	Messages        []*wfcompose.Message
	EnableStreaming  bool
}

// AdkAgentEvent represents a single event in the agent run.
type AdkAgentEvent struct {
	Err    error
	Action *AdkAgentAction
	Output *AdkAgentOutput
}

// AdkAgentAction describes side-effects during the run.
type AdkAgentAction struct {
	Interrupted *AdkInterruptInfo
	ToolCalls   []wfcompose.ToolCall
}

// AdkInterruptInfo carries interrupt context.
type AdkInterruptInfo struct {
	Info string
	Data []byte
}

// AdkAgentOutput holds the model's output.
type AdkAgentOutput struct {
	MessageOutput *AdkMessageOutput
}

// AdkMessageOutput holds the model's output message.
type AdkMessageOutput struct {
	Message      *wfcompose.Message
	MessageStream *wfcompose.StreamReader[*wfcompose.Message]
	IsStreaming   bool
}

// AdkCheckPointStore is a checkpoint store for agent runs.
type AdkCheckPointStore = wfcompose.CheckPointStore

// Run executes the agent and returns an event iterator.
func (a *AdkChatModelAgent) Run(ctx context.Context, input *AdkAgentInput) *AdkAsyncIterator[*AdkAgentEvent] {
	events := make([]*AdkAgentEvent, 0)

	msgs := input.Messages
	for step := 0; step < a.maxStep; step++ {
		result, err := a.model.Generate(ctx, msgs)
		if err != nil {
			events = append(events, &AdkAgentEvent{Err: err})
			break
		}
		msgs = append(msgs, result)

		if len(result.ToolCalls) == 0 {
			events = append(events, &AdkAgentEvent{
				Output: &AdkAgentOutput{
					MessageOutput: &AdkMessageOutput{
						Message:     result,
						IsStreaming: input.EnableStreaming,
					},
				},
			})
			break
		}

		events = append(events, &AdkAgentEvent{
			Action: &AdkAgentAction{
				ToolCalls: result.ToolCalls,
			},
			Output: &AdkAgentOutput{
				MessageOutput: &AdkMessageOutput{
					Message: result,
				},
			},
		})

		// Add tool results to messages
		for _, tc := range result.ToolCalls {
			msgs = append(msgs, wfcompose.ToolMessage("", tc.ID))
		}
	}

	return NewAdkAsyncIteratorFromSlice(events)
}

// AdkAsyncIterator is a simple iterator over agent events.
type AdkAsyncIterator[T any] struct {
	items []T
	idx   int
}

// Next returns the next item and whether it exists.
func (it *AdkAsyncIterator[T]) Next() (T, bool) {
	var zero T
	if it == nil || it.idx >= len(it.items) {
		return zero, false
	}
	item := it.items[it.idx]
	it.idx++
	return item, true
}

// NewAdkAsyncIteratorPair creates a connected iterator/generator pair.
func NewAdkAsyncIteratorPair[T any]() (*AdkAsyncIterator[T], *AdkAsyncGenerator[T]) {
	gen := &AdkAsyncGenerator[T]{items: make([]T, 0)}
	iter := &AdkAsyncIterator[T]{}
	gen.iter = iter
	return iter, gen
}

// NewAdkAsyncIteratorFromSlice creates an iterator from a slice.
func NewAdkAsyncIteratorFromSlice[T any](items []T) *AdkAsyncIterator[T] {
	return &AdkAsyncIterator[T]{items: items}
}

// AdkAsyncGenerator is the write side of an iterator pair.
type AdkAsyncGenerator[T any] struct {
	items []T
	iter  *AdkAsyncIterator[T]
}

// Send adds an item to the generator.
func (g *AdkAsyncGenerator[T]) Send(item T) {
	g.items = append(g.items, item)
	g.iter.items = g.items
}

// Close signals end of generation.
func (g *AdkAsyncGenerator[T]) Close() {
	// no-op
}
