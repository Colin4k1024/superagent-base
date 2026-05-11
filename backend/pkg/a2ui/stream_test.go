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

import (
	"sync"
	"testing"
)

func TestEventStream_SendAndReceive(t *testing.T) {
	s := NewEventStream(4)
	s.Send(NewEvent(EventText, &TextData{Delta: "a"}))
	s.Send(NewEvent(EventText, &TextData{Delta: "b"}))
	s.Close()

	var got []string
	for evt := range s.Chan() {
		if td, ok := evt.Data.(*TextData); ok {
			got = append(got, td.Delta)
		}
	}
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("unexpected events: %v", got)
	}
}

func TestEventStream_SendAfterCloseIsNoop(t *testing.T) {
	s := NewEventStream(4)
	s.Close()
	// Must not panic.
	s.Send(NewEvent(EventText, &TextData{Delta: "x"}))
}

func TestEventStream_CloseIsIdempotent(t *testing.T) {
	s := NewEventStream(4)
	s.Close()
	s.Close() // second close must not panic
}

func TestEventStream_SendDoneClosesChannel(t *testing.T) {
	s := NewEventStream(8)
	s.SendDone()

	var events []*Event
	for evt := range s.Chan() {
		events = append(events, evt)
	}
	if len(events) != 1 || events[0].Type != EventDone {
		t.Errorf("expected single done event, got %v", events)
	}
}

func TestEventStream_ConcurrentSends(t *testing.T) {
	const n = 50
	s := NewEventStream(n * 2)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.SendText("token")
		}()
	}
	wg.Wait()
	s.Close()

	count := 0
	for range s.Chan() {
		count++
	}
	if count != n {
		t.Errorf("expected %d events, got %d", n, count)
	}
}

func TestEventStream_SendTextHelper(t *testing.T) {
	s := NewEventStream(4)
	s.SendText("hello")
	s.Close()

	evt := <-s.Chan()
	if evt.Type != EventText {
		t.Errorf("expected text event, got %q", evt.Type)
	}
	td, ok := evt.Data.(*TextData)
	if !ok || td.Delta != "hello" {
		t.Errorf("unexpected text data: %v", evt.Data)
	}
}

func TestEventStream_SendErrorHelper(t *testing.T) {
	s := NewEventStream(4)
	s.SendError("E01", "something went wrong")
	s.Close()

	evt := <-s.Chan()
	if evt.Type != EventError {
		t.Errorf("expected error event, got %q", evt.Type)
	}
	ed, ok := evt.Data.(*ErrorData)
	if !ok || ed.Code != "E01" || ed.Message != "something went wrong" {
		t.Errorf("unexpected error data: %v", evt.Data)
	}
}

func TestEventStream_SendToolCallHelper(t *testing.T) {
	s := NewEventStream(4)
	s.SendToolCall("tc1", "search", map[string]any{"q": "go"})
	s.Close()

	evt := <-s.Chan()
	if evt.Type != EventToolCall {
		t.Errorf("expected tool_call event, got %q", evt.Type)
	}
	td, ok := evt.Data.(*ToolCallData)
	if !ok || td.ID != "tc1" || td.Name != "search" || td.Status != "calling" {
		t.Errorf("unexpected tool call data: %v", evt.Data)
	}
}

func TestEventStream_SendToolResultHelper(t *testing.T) {
	s := NewEventStream(4)
	s.SendToolResult("tc1", "search", "results", false)
	s.Close()

	evt := <-s.Chan()
	if evt.Type != EventToolResult {
		t.Errorf("expected tool_result event, got %q", evt.Type)
	}
	td, ok := evt.Data.(*ToolResultData)
	if !ok || td.ID != "tc1" || td.IsError {
		t.Errorf("unexpected tool result data: %v", evt.Data)
	}
}

func TestEventStream_SendThinkingHelper(t *testing.T) {
	s := NewEventStream(4)
	s.SendThinking("thinking...")
	s.Close()

	evt := <-s.Chan()
	if evt.Type != EventThinking {
		t.Errorf("expected thinking event, got %q", evt.Type)
	}
	td, ok := evt.Data.(*ThinkingData)
	if !ok || td.Delta != "thinking..." {
		t.Errorf("unexpected thinking data: %v", evt.Data)
	}
}
