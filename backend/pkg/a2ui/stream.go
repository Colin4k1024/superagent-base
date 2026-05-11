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

import "sync"

// EventStream is a thread-safe, buffered channel of A2UI events.
// Producers call Send* helpers; consumers read from Chan().
// Close is idempotent: the first call closes the underlying channel.
type EventStream struct {
	ch     chan *Event
	closed bool
	mu     sync.Mutex
}

// NewEventStream creates an EventStream with the given channel buffer size.
func NewEventStream(bufSize int) *EventStream {
	return &EventStream{ch: make(chan *Event, bufSize)}
}

// Send enqueues an event. Calls after Close are silently dropped.
func (s *EventStream) Send(evt *Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.ch <- evt
	}
}

// SendText emits a text delta event.
func (s *EventStream) SendText(delta string) {
	s.Send(NewEvent(EventText, &TextData{Delta: delta}))
}

// SendToolCall emits a tool_call event with status "calling".
func (s *EventStream) SendToolCall(id, name string, args map[string]any) {
	s.Send(NewEvent(EventToolCall, &ToolCallData{
		ID:        id,
		Name:      name,
		Arguments: args,
		Status:    "calling",
	}))
}

// SendToolResult emits a tool_result event.
func (s *EventStream) SendToolResult(id, name, result string, isError bool) {
	s.Send(NewEvent(EventToolResult, &ToolResultData{
		ID:      id,
		Name:    name,
		Result:  result,
		IsError: isError,
	}))
}

// SendThinking emits a thinking delta event.
func (s *EventStream) SendThinking(delta string) {
	s.Send(NewEvent(EventThinking, &ThinkingData{Delta: delta}))
}

// SendDone emits a done event and then closes the stream.
func (s *EventStream) SendDone() {
	s.Send(NewEvent(EventDone, nil))
	s.Close()
}

// SendError emits an error event.
func (s *EventStream) SendError(code, msg string) {
	s.Send(NewEvent(EventError, &ErrorData{Code: code, Message: msg}))
}

// Close closes the underlying channel. Subsequent Send calls are no-ops.
// Close is safe to call multiple times.
func (s *EventStream) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		close(s.ch)
	}
}

// Chan returns the read-only event channel for consumers.
func (s *EventStream) Chan() <-chan *Event { return s.ch }
