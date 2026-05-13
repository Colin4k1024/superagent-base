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

import "sync/atomic"

// EventStream is a thread-safe, buffered channel of A2UI events.
// Producers call Send* helpers; consumers read from Chan().
// Close is idempotent: the first call closes the underlying channel.
//
// The closed state is tracked via an atomic int32 so that Send never
// blocks while holding a lock — preventing the deadlock where a full
// channel + concurrent Close would starve each other.
type EventStream struct {
	ch     chan *Event
	closed atomic.Int32 // 0 = open, 1 = closed
}

// NewEventStream creates an EventStream with the given channel buffer size.
func NewEventStream(bufSize int) *EventStream {
	return &EventStream{ch: make(chan *Event, bufSize)}
}

// Send enqueues an event. Calls after Close are silently dropped.
// If the channel buffer is full, Send performs a non-blocking attempt:
// the event is dropped rather than deadlocking the stream.
func (s *EventStream) Send(evt *Event) {
	if s.closed.Load() != 0 {
		return
	}
	select {
	case s.ch <- evt:
	default:
		// Channel full — drop event to prevent deadlock.
		// In production this means back-pressure from a slow consumer;
		// callers should size the buffer appropriately.
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
	if s.closed.CompareAndSwap(0, 1) {
		close(s.ch)
	}
}

// Chan returns the read-only event channel for consumers.
func (s *EventStream) Chan() <-chan *Event { return s.ch }
