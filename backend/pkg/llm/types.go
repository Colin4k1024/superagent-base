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

package llm

import (
	"context"
	"io"
)

// Role constants for message types.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// Message is the framework-agnostic chat message type.
// It abstracts away eino's schema.Message so business code does not
// import cloudwego/eino directly.
type Message struct {
	Role       string      // system, user, assistant, tool
	Content    string      // text content
	Name       string      // optional participant name
	ToolCalls  []ToolCall  // tool calls requested by the assistant
	ToolCallID string      // ID of the tool call this message responds to (role=tool)
}

// ToolCall represents a single tool/function call requested by the model.
type ToolCall struct {
	ID       string // unique call ID
	Name     string // function name
	ArgsJSON string // JSON-encoded arguments
}

// StreamReader is a generic streaming reader for model output chunks.
// Each Recv call returns the next chunk; io.EOF signals end-of-stream.
type StreamReader struct {
	recv  func() (*Message, error)
	close func() error
}

// NewStreamReader creates a StreamReader from recv/close functions.
// close may be nil.
func NewStreamReader(recv func() (*Message, error), close func() error) *StreamReader {
	return &StreamReader{recv: recv, close: close}
}

// Recv returns the next streamed message chunk.
func (r *StreamReader) Recv() (*Message, error) {
	if r == nil || r.recv == nil {
		return nil, io.EOF
	}
	return r.recv()
}

// Close releases resources associated with the stream.
func (r *StreamReader) Close() error {
	if r == nil || r.close == nil {
		return nil
	}
	return r.close()
}

// ToolInfo describes a tool's metadata for the model.
type ToolInfo struct {
	Extra map[string]any // optional extra metadata
	Name        string      // tool identifier
	Desc        string      // human-readable description
	ParamsOneOf ParamsOneOf // parameter schema
}

// ParamsOneOf abstracts tool parameter schemas.
// Concrete implementations may wrap JSON schema or OpenAPI specs.
type ParamsOneOf interface {
	isParamsOneOf()
}

// JSONObject is a generic map for JSON data.
type JSONObject = map[string]any

// ToolOptionConfig holds optional parameters for tool invocations.
type ToolOptionConfig struct {
	Ctx context.Context // context for the invocation
}

// ToolOption is a variadic option passed to tool invocations.
type ToolOption func(*ToolOptionConfig)

// ToolOptionWithContext returns a ToolOption that injects a context.
func ToolOptionWithContext(ctx context.Context) ToolOption {
	return func(c *ToolOptionConfig) { c.Ctx = ctx }
}

// --- Convenience message constructors ---

// SystemMessage creates a system-role message.
func SystemMessage(content string) *Message {
	return &Message{Role: RoleSystem, Content: content}
}

// UserMessage creates a user-role message.
func UserMessage(content string) *Message {
	return &Message{Role: RoleUser, Content: content}
}

// AssistantMessage creates an assistant-role message with optional tool calls.
func AssistantMessage(content string, toolCalls []ToolCall) *Message {
	return &Message{Role: RoleAssistant, Content: content, ToolCalls: toolCalls}
}

// ToolMessage creates a tool-role message responding to a specific tool call.
func ToolMessage(content, toolCallID string) *Message {
	return &Message{Role: RoleTool, Content: content, ToolCallID: toolCallID}
}
