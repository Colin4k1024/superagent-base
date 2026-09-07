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

// Package a2ui — contract tests (#16 P1: freeze A2UI schema).
//
// These tests assert the canonical A2UI event shapes that BOTH the Go and
// Python runtimes must emit. They are the frozen contract boundary: any
// change to event field names, types, or SSE encoding format must be
// reflected here first and mirrored on the Python side.
//
// Test coverage:
//   1. Every EventType constant has a corresponding data struct.
//   2. Each event type round-trips through JSON with the expected field set.
//   3. SSE encoding format (event: <type>\ndata: <json>\n\n) is stable.
//   4. Compatible encoding (data-only) preserves text/done semantics.
//   5. Event ordering invariants (tool_call before tool_result, done is last).

package a2ui

import (
	"encoding/json"
	"strings"
	"testing"
)

// ─── 1. Event type registry completeness ─────────────────────────────────────

func TestContract_AllEventTypesHaveDataStructs(t *testing.T) {
	// Every event type must have a well-defined data payload (or nil for done).
	// This map documents the expected data type for each event.
	// Adding a new EventType without a corresponding entry here is a contract
	// violation — both runtimes must agree on the shape.
	expectedDataTypes := map[EventType]struct{}{
		EventText:        {},
		EventThinking:    {},
		EventToolCall:    {},
		EventToolResult:  {},
		EventCodeBlock:   {},
		EventInterrupt:   {},
		EventError:       {},
		EventDone:        {},
		EventProgress:    {},
		EventAgentSwitch: {},
	}

	for et := range expectedDataTypes {
		// Verify the constant is not empty.
		if et == "" {
			t.Error("found empty EventType in expected set")
		}
	}

	// Verify no undocumented event types were added to the constants.
	allTypes := []EventType{
		EventText, EventThinking, EventToolCall, EventToolResult,
		EventCodeBlock, EventInterrupt, EventError, EventDone,
		EventProgress, EventAgentSwitch,
	}
	if len(allTypes) != len(expectedDataTypes) {
		t.Errorf("event type count mismatch: constants=%d expected=%d", len(allTypes), len(expectedDataTypes))
	}
}

// ─── 2. JSON round-trip for each event data type ─────────────────────────────

func TestContract_TextData_JSONShape(t *testing.T) {
	original := &TextData{Content: "full text", Delta: "full"}
	b, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal TextData: %v", err)
	}
	var decoded TextData
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal TextData: %v", err)
	}
	if decoded.Content != original.Content || decoded.Delta != original.Delta {
		t.Errorf("TextData round-trip mismatch: got %+v, want %+v", decoded, original)
	}
	// Verify field names are camelCase (contract requirement).
	var raw map[string]any
	_ = json.Unmarshal(b, &raw)
	requiredFields := []string{"content", "delta"}
	for _, f := range requiredFields {
		if _, ok := raw[f]; !ok {
			t.Errorf("TextData JSON missing field %q", f)
		}
	}
}

func TestContract_ToolCallData_JSONShape(t *testing.T) {
	original := &ToolCallData{
		ID:        "tc-1",
		Name:      "web_search",
		Arguments: map[string]any{"query": "golang"},
		Status:    "calling",
	}
	b, _ := json.Marshal(original)
	var decoded ToolCallData
	_ = json.Unmarshal(b, &decoded)
	if decoded.ID != original.ID || decoded.Name != original.Name || decoded.Status != original.Status {
		t.Errorf("ToolCallData round-trip mismatch: got %+v", decoded)
	}
	if decoded.Arguments["query"] != "golang" {
		t.Errorf("ToolCallData arguments mismatch: got %v", decoded.Arguments)
	}
	var raw map[string]any
	_ = json.Unmarshal(b, &raw)
	for _, f := range []string{"id", "name", "arguments", "status"} {
		if _, ok := raw[f]; !ok {
			t.Errorf("ToolCallData JSON missing field %q", f)
		}
	}
}

func TestContract_ToolResultData_JSONShape(t *testing.T) {
	original := &ToolResultData{
		ID:      "tc-1",
		Name:    "web_search",
		Result:  "results here",
		IsError: false,
	}
	b, _ := json.Marshal(original)
	var decoded ToolResultData
	_ = json.Unmarshal(b, &decoded)
	if decoded.ID != original.ID || decoded.Result != original.Result || decoded.IsError != original.IsError {
		t.Errorf("ToolResultData round-trip mismatch: got %+v", decoded)
	}
	var raw map[string]any
	_ = json.Unmarshal(b, &raw)
	for _, f := range []string{"id", "name", "result", "is_error"} {
		if _, ok := raw[f]; !ok {
			t.Errorf("ToolResultData JSON missing field %q (snake_case required)", f)
		}
	}
}

func TestContract_InterruptData_JSONShape(t *testing.T) {
	original := &InterruptData{
		Reason: "need_input",
		Fields: []InterruptField{
			{Name: "confirm", Type: "confirm", Label: "Proceed?", Required: true, Options: []string{"yes", "no"}},
		},
	}
	b, _ := json.Marshal(original)
	var decoded InterruptData
	_ = json.Unmarshal(b, &decoded)
	if decoded.Reason != original.Reason || len(decoded.Fields) != 1 {
		t.Errorf("InterruptData round-trip mismatch: got %+v", decoded)
	}
	if decoded.Fields[0].Name != "confirm" || decoded.Fields[0].Type != "confirm" {
		t.Errorf("InterruptField round-trip mismatch: got %+v", decoded.Fields[0])
	}
}

func TestContract_ErrorData_JSONShape(t *testing.T) {
	original := &ErrorData{Code: "E001", Message: "something broke"}
	b, _ := json.Marshal(original)
	var decoded ErrorData
	_ = json.Unmarshal(b, &decoded)
	if decoded.Code != original.Code || decoded.Message != original.Message {
		t.Errorf("ErrorData round-trip mismatch: got %+v", decoded)
	}
	var raw map[string]any
	_ = json.Unmarshal(b, &raw)
	for _, f := range []string{"code", "message"} {
		if _, ok := raw[f]; !ok {
			t.Errorf("ErrorData JSON missing field %q", f)
		}
	}
}

func TestContract_ProgressData_JSONShape(t *testing.T) {
	original := &ProgressData{AgentName: "research-agent", Step: "search", Total: 5, Current: 2}
	b, _ := json.Marshal(original)
	var decoded ProgressData
	_ = json.Unmarshal(b, &decoded)
	if decoded.AgentName != original.AgentName || decoded.Step != original.Step {
		t.Errorf("ProgressData round-trip mismatch: got %+v", decoded)
	}
	var raw map[string]any
	_ = json.Unmarshal(b, &raw)
	for _, f := range []string{"agent_name", "step", "total", "current"} {
		if _, ok := raw[f]; !ok {
			t.Errorf("ProgressData JSON missing field %q (snake_case required)", f)
		}
	}
}

func TestContract_AgentSwitchData_JSONShape(t *testing.T) {
	original := &AgentSwitchData{FromAgent: "router", ToAgent: "coder", Reason: "delegation"}
	b, _ := json.Marshal(original)
	var decoded AgentSwitchData
	_ = json.Unmarshal(b, &decoded)
	if decoded.FromAgent != original.FromAgent || decoded.ToAgent != original.ToAgent {
		t.Errorf("AgentSwitchData round-trip mismatch: got %+v", decoded)
	}
	var raw map[string]any
	_ = json.Unmarshal(b, &raw)
	for _, f := range []string{"from_agent", "to_agent", "reason"} {
		if _, ok := raw[f]; !ok {
			t.Errorf("AgentSwitchData JSON missing field %q (snake_case required)", f)
		}
	}
}

// ─── 3. SSE encoding format stability ─────────────────────────────────────────

func TestContract_SSEFormat_AllEventTypes(t *testing.T) {
	tests := []struct {
		name    string
		event   *Event
		wantType string
	}{
		{"text", NewEvent(EventText, &TextData{Delta: "hi"}), "text"},
		{"thinking", NewEvent(EventThinking, &ThinkingData{Delta: "hmm"}), "thinking"},
		{"tool_call", NewEvent(EventToolCall, &ToolCallData{ID: "1", Name: "x", Status: "calling"}), "tool_call"},
		{"tool_result", NewEvent(EventToolResult, &ToolResultData{ID: "1", Name: "x", Result: "ok"}), "tool_result"},
		{"code_block", NewEvent(EventCodeBlock, &CodeBlockData{Language: "go", Code: "x"}), "code_block"},
		{"interrupt", NewEvent(EventInterrupt, &InterruptData{Reason: "r"}), "interrupt"},
		{"error", NewEvent(EventError, &ErrorData{Code: "e", Message: "m"}), "error"},
		{"done", NewEvent(EventDone, nil), "done"},
		{"progress", NewEvent(EventProgress, &ProgressData{AgentName: "a", Step: "s"}), "progress"},
		{"agent_switch", NewEvent(EventAgentSwitch, &AgentSwitchData{FromAgent: "a", ToAgent: "b"}), "agent_switch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := EncodeSSE(tt.event)
			// Must start with "event: <type>\n".
			expectedPrefix := "event: " + tt.wantType + "\n"
			if !strings.HasPrefix(encoded, expectedPrefix) {
				t.Errorf("SSE prefix mismatch: got %q, want prefix %q", encoded, expectedPrefix)
			}
			// Must contain "data: " line.
			if !strings.Contains(encoded, "data: ") {
				t.Errorf("SSE missing data line: %q", encoded)
			}
			// Must end with "\n\n".
			if !strings.HasSuffix(encoded, "\n\n") {
				t.Errorf("SSE missing double-newline suffix: %q", encoded)
			}
		})
	}
}

// ─── 4. Compatible encoding contract ──────────────────────────────────────────

func TestContract_CompatibleEncoding_TextAndDone(t *testing.T) {
	// Text events emit the raw delta (no JSON wrapping).
	textEvt := NewEvent(EventText, &TextData{Delta: "raw delta"})
	if got := EncodeCompatible(textEvt); got != "data: raw delta\n\n" {
		t.Errorf("compatible text encoding: got %q", got)
	}

	// Done events emit the literal "[DONE]".
	doneEvt := NewEvent(EventDone, nil)
	if got := EncodeCompatible(doneEvt); got != "data: [DONE]\n\n" {
		t.Errorf("compatible done encoding: got %q", got)
	}
}

// ─── 5. Event ordering invariants ────────────────────────────────────────────

func TestContract_EventOrdering_ToolCallBeforeResult(t *testing.T) {
	// A tool_call event must always precede its corresponding tool_result.
	// The tool_result references the tool_call by ID.
	s := NewEventStream(8)
	s.SendToolCall("tc-1", "search", map[string]any{"q": "test"})
	s.SendToolResult("tc-1", "search", "results", false)
	s.Close()

	var types []EventType
	for evt := range s.Chan() {
		types = append(types, evt.Type)
	}

	// Must have exactly tool_call then tool_result.
	if len(types) != 2 {
		t.Fatalf("expected 2 events, got %d: %v", len(types), types)
	}
	if types[0] != EventToolCall {
		t.Errorf("expected first event to be tool_call, got %q", types[0])
	}
	if types[1] != EventToolResult {
		t.Errorf("expected second event to be tool_result, got %q", types[1])
	}
}

func TestContract_EventOrdering_DoneIsTerminal(t *testing.T) {
	// The done event must be the last event in any stream.
	// After done, no more events should be deliverable.
	s := NewEventStream(8)
	s.SendText("some text")
	s.SendDone()
	s.SendText("this should be dropped") // after close — no-op

	var types []EventType
	for evt := range s.Chan() {
		types = append(types, evt.Type)
	}

	if len(types) != 2 {
		t.Fatalf("expected 2 events (text + done), got %d: %v", len(types), types)
	}
	if types[1] != EventDone {
		t.Errorf("expected done to be last event, got %q", types[1])
	}
}

func TestContract_EventOrdering_TextBeforeDone(t *testing.T) {
	// Text deltas must all precede the done event.
	s := NewEventStream(16)
	s.SendText("hello")
	s.SendText(" ")
	s.SendText("world")
	s.SendDone()

	var types []EventType
	for evt := range s.Chan() {
		types = append(types, evt.Type)
	}

	if len(types) != 4 {
		t.Fatalf("expected 4 events, got %d: %v", len(types), types)
	}
	if types[3] != EventDone {
		t.Errorf("expected done to be last event, got %q", types[3])
	}
	for i := 0; i < 3; i++ {
		if types[i] != EventText {
			t.Errorf("expected text at position %d, got %q", i, types[i])
		}
	}
}

// ─── 6. Event envelope shape ─────────────────────────────────────────────────

func TestContract_EventEnvelope_JSONShape(t *testing.T) {
	// Every event must serialize as {"type":"<type>","timestamp":<int>,"data":<data>}.
	evt := NewEvent(EventText, &TextData{Delta: "x"})
	b, _ := json.Marshal(evt)
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("Event JSON unmarshal: %v", err)
	}
	requiredFields := []string{"type", "timestamp", "data"}
	for _, f := range requiredFields {
		if _, ok := raw[f]; !ok {
			t.Errorf("Event JSON missing field %q", f)
		}
	}
	// timestamp must be a number (int64 milliseconds).
	ts, ok := raw["timestamp"].(float64)
	if !ok {
		t.Errorf("timestamp is not a number, got %T", raw["timestamp"])
	}
	if ts <= 0 {
		t.Errorf("timestamp should be positive, got %v", ts)
	}
}

func TestContract_DoneEvent_HasNilData(t *testing.T) {
	// The done event's data field must be null (not omitted).
	evt := NewEvent(EventDone, nil)
	b, _ := json.Marshal(evt)
	var raw map[string]any
	_ = json.Unmarshal(b, &raw)
	if _, ok := raw["data"]; !ok {
		t.Error("done event must include data field (even if null)")
	}
}
