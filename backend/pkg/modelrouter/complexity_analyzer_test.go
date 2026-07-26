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

package modelrouter

import (
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestParseComplexityResponse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"valid low", `{"complexity": "low"}`, "low", false},
		{"valid medium", `{"complexity": "medium"}`, "medium", false},
		{"valid high", `{"complexity": "high"}`, "high", false},
		{"json with whitespace", `  {"complexity": "high"}  `, "high", false},
		{"fallback keyword high", "The task is high complexity", "high", false},
		{"fallback keyword low", "This is a low effort task", "low", false},
		{"fallback keyword medium", "medium difficulty", "medium", false},
		{"invalid", "I don't know", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseComplexityResponse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseComplexityResponse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseComplexityResponse(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractRecentUserMessages(t *testing.T) {
	msgs := []*schema.Message{
		{Role: schema.System, Content: "system"},
		{Role: schema.User, Content: "hello"},
		{Role: schema.Assistant, Content: "hi there"},
		{Role: schema.User, Content: "help me code"},
		{Role: schema.Assistant, Content: "sure"},
		{Role: schema.User, Content: "write a parser"},
	}

	got := extractRecentUserMessages(msgs, 3)
	want := []string{"hello", "help me code", "write a parser"}
	if len(got) != len(want) {
		t.Fatalf("got %d messages, want %d", len(got), len(want))
	}
	for i, m := range got {
		if m != want[i] {
			t.Errorf("message[%d] = %q, want %q", i, m, want[i])
		}
	}
}

func TestExtractRecentUserMessages_LimitN(t *testing.T) {
	msgs := []*schema.Message{
		{Role: schema.User, Content: "a"},
		{Role: schema.User, Content: "b"},
		{Role: schema.User, Content: "c"},
		{Role: schema.User, Content: "d"},
	}

	got := extractRecentUserMessages(msgs, 2)
	if len(got) != 2 {
		t.Fatalf("got %d messages, want 2", len(got))
	}
	if got[0] != "c" || got[1] != "d" {
		t.Errorf("got %v, want [c d]", got)
	}
}

func TestExtractRecentUserMessages_Empty(t *testing.T) {
	msgs := []*schema.Message{
		{Role: schema.Assistant, Content: "only assistant"},
	}
	got := extractRecentUserMessages(msgs, 3)
	if len(got) != 0 {
		t.Errorf("got %d messages, want 0", len(got))
	}
}

func TestNewLLMComplexityAnalyzer_Defaults(t *testing.T) {
	a := NewLLMComplexityAnalyzer(ComplexityConfig{
		Model:  "qwen-turbo",
		BaseURL: "https://example.com",
		APIKey:  "test-key",
	})

	if a.fallback != "medium" {
		t.Errorf("default fallback = %q, want %q", a.fallback, "medium")
	}
	if a.cacheTTL != 5*60*1e9 { // 5 minutes in nanoseconds
		t.Errorf("default cacheTTL = %v, want 5m", a.cacheTTL)
	}
}
