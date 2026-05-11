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
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewEvent_SetsTypeAndTimestamp(t *testing.T) {
	before := time.Now().UnixMilli()
	evt := NewEvent(EventText, &TextData{Delta: "hello"})
	after := time.Now().UnixMilli()

	if evt.Type != EventText {
		t.Errorf("got type %q, want %q", evt.Type, EventText)
	}
	if evt.Timestamp < before || evt.Timestamp > after {
		t.Errorf("timestamp %d outside [%d, %d]", evt.Timestamp, before, after)
	}
}

func TestNewEvent_NilData(t *testing.T) {
	evt := NewEvent(EventDone, nil)
	if evt.Data != nil {
		t.Errorf("expected nil data, got %v", evt.Data)
	}
}

func TestEncodeSSE_Format(t *testing.T) {
	evt := NewEvent(EventText, &TextData{Delta: "world"})
	encoded := EncodeSSE(evt)

	if !strings.HasPrefix(encoded, "event: text\n") {
		t.Errorf("expected SSE event line prefix, got: %q", encoded)
	}
	if !strings.Contains(encoded, "data: ") {
		t.Error("expected data line in SSE frame")
	}
	if !strings.HasSuffix(encoded, "\n\n") {
		t.Error("expected double newline at end of SSE frame")
	}

	// data line must be valid JSON containing the full event.
	lines := strings.Split(strings.TrimSpace(encoded), "\n")
	var dataLine string
	for _, l := range lines {
		if strings.HasPrefix(l, "data: ") {
			dataLine = strings.TrimPrefix(l, "data: ")
		}
	}
	var out Event
	if err := json.Unmarshal([]byte(dataLine), &out); err != nil {
		t.Fatalf("data line is not valid JSON: %v", err)
	}
	if out.Type != EventText {
		t.Errorf("decoded type %q, want %q", out.Type, EventText)
	}
}

func TestEncodeCompatible_TextEvent(t *testing.T) {
	evt := NewEvent(EventText, &TextData{Delta: "hello"})
	encoded := EncodeCompatible(evt)
	if encoded != "data: hello\n\n" {
		t.Errorf("unexpected compatible encoding: %q", encoded)
	}
}

func TestEncodeCompatible_DoneEvent(t *testing.T) {
	evt := NewEvent(EventDone, nil)
	encoded := EncodeCompatible(evt)
	if encoded != "data: [DONE]\n\n" {
		t.Errorf("unexpected compatible done encoding: %q", encoded)
	}
}

func TestEncodeCompatible_NonTextFallsBackToJSON(t *testing.T) {
	evt := NewEvent(EventError, &ErrorData{Code: "err", Message: "oops"})
	encoded := EncodeCompatible(evt)
	if !strings.HasPrefix(encoded, "data: ") {
		t.Error("expected data prefix for non-text event")
	}
	payload := strings.TrimPrefix(strings.TrimSuffix(encoded, "\n\n"), "data: ")
	var out Event
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		t.Fatalf("non-text compat encoding is not valid JSON: %v", err)
	}
}
