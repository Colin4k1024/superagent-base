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

import "time"

// EventType identifies the kind of A2UI event.
type EventType string

const (
	EventText        EventType = "text"
	EventThinking    EventType = "thinking"
	EventToolCall    EventType = "tool_call"
	EventToolResult  EventType = "tool_result"
	EventCodeBlock   EventType = "code_block"
	EventInterrupt   EventType = "interrupt"
	EventError       EventType = "error"
	EventDone        EventType = "done"
	EventProgress    EventType = "progress"
	EventAgentSwitch EventType = "agent_switch"
)

// Event is the envelope for every A2UI message sent over the stream.
type Event struct {
	Type      EventType `json:"type"`
	Timestamp int64     `json:"timestamp"`
	Data      any       `json:"data"`
}

// NewEvent creates a new Event with the current time in milliseconds.
func NewEvent(t EventType, data any) *Event {
	return &Event{Type: t, Timestamp: time.Now().UnixMilli(), Data: data}
}

// TextData carries a streaming text token.
type TextData struct {
	Content string `json:"content"`
	Delta   string `json:"delta"`
}

// ThinkingData carries a streaming thinking/reasoning token.
type ThinkingData struct {
	Content string `json:"content"`
	Delta   string `json:"delta"`
}

// ToolCallData describes a tool invocation.
type ToolCallData struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
	// Status is one of: calling, success, error.
	Status string `json:"status"`
}

// ToolResultData carries the result of a tool invocation.
type ToolResultData struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Result  string `json:"result"`
	IsError bool   `json:"is_error"`
}

// CodeBlockData carries a streaming or complete code block.
type CodeBlockData struct {
	Language string `json:"language"`
	Code     string `json:"code"`
	Delta    string `json:"delta"`
}

// InterruptData requests input from the user before the agent can continue.
type InterruptData struct {
	Reason string           `json:"reason"`
	Fields []InterruptField `json:"fields"`
}

// InterruptField describes a single form field within an InterruptData.
type InterruptField struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"` // text, confirm, select
	Label    string   `json:"label"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
}

// ErrorData describes an error that occurred during agent execution.
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ProgressData reports execution progress for a named agent step.
type ProgressData struct {
	AgentName string `json:"agent_name"`
	Step      string `json:"step"`
	Total     int    `json:"total,omitempty"`
	Current   int    `json:"current,omitempty"`
}

// AgentSwitchData signals a handoff between agents.
type AgentSwitchData struct {
	FromAgent string `json:"from_agent"`
	ToAgent   string `json:"to_agent"`
	Reason    string `json:"reason"`
}
