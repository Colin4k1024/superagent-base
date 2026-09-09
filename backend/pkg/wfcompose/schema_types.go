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

package wfcompose

import (
	"context"
	"errors"
	
)

// DataType is the JSON-Schema type of a tool parameter.
type DataType string

const (
	Object  DataType = "object"
	Number  DataType = "number"
	Integer DataType = "integer"
	String  DataType = "string"
	Array   DataType = "array"
	Null    DataType = "null"
	Boolean DataType = "boolean"
)

// ParameterInfo describes a single tool parameter.
type ParameterInfo struct {
	Type      DataType             `json:"type"`
	ElemInfo *ParameterInfo        `json:"elem_info,omitempty"`
	SubParams map[string]*ParameterInfo `json:"sub_params,omitempty"`
	Desc      string               `json:"desc,omitempty"`
	Enum      []string             `json:"enum,omitempty"`
	Required  bool                `json:"required,omitempty"`
}

// ParamsOneOf holds either named parameters or a JSON schema for a tool.
// It mirrors eino's schema.ParamsOneOf so that tool definitions remain
// structurally compatible.
type ParamsOneOf struct {
	params map[string]*ParameterInfo
}

// NewParamsOneOfByParams creates a ParamsOneOf from a map of named parameters.
func NewParamsOneOfByParams(params map[string]*ParameterInfo) *ParamsOneOf {
	return &ParamsOneOf{params: params}
}

// Params returns the underlying named-parameter map, or nil if none.
func (p *ParamsOneOf) Params() map[string]*ParameterInfo {
	if p == nil {
		return nil
	}
	return p.params
}

// ToolInfo describes a tool that can be passed to a ChatModel.
type ToolInfo struct {
	Name string
	Desc string
	Extra map[string]any

	// ParamsOneOf may be nil for tools that take no arguments.
	*ParamsOneOf
}

// FormatType selects the template engine for message formatting.
type FormatType uint8

const (
	FString    FormatType = 0
	GoTemplate FormatType = 1
	Jinja2     FormatType = 2
)

// MessagesTemplate is a template that renders into a slice of messages.
type MessagesTemplate interface {
	Format(ctx context.Context, vs map[string]any, formatType FormatType) ([]*Message, error)
}

type messagesPlaceholder struct {
	key      string
	optional bool
}

// MessagesPlaceholder creates a placeholder that renders the messages
// stored under the given key in the format variables.
func MessagesPlaceholder(key string, optional bool) MessagesTemplate {
	return &messagesPlaceholder{key: key, optional: optional}
}

func (m *messagesPlaceholder) Format(ctx context.Context, vs map[string]any, _ FormatType) ([]*Message, error) {
	val, ok := vs[m.key]
	if !ok {
		if m.optional {
			return nil, nil
		}
		return nil, nil
	}
	switch t := val.(type) {
	case []*Message:
		return t, nil
	case []Message:
		out := make([]*Message, len(t))
		for i := range t {
			out[i] = &t[i]
		}
		return out, nil
	case *Message:
		return []*Message{t}, nil
	case Message:
		return []*Message{&t}, nil
	default:
		return nil, nil
	}
}

// ConvertOption is a filter/wrapper option for StreamReaderWithConvert.
type ConvertOption func(*convertOptions)

type convertOptions struct {
	errWrapper func(error) error
}

// ErrNoValue is a sentinel returned from the convert function passed to
// StreamReaderWithConvert to silently skip a stream element.
var ErrNoValue = errors.New("no value")

// --- Message constructors ---

func SystemMessage(content string) *Message {
	return &Message{Role: System, Content: content}
}

func UserMessage(content string) *Message {
	return &Message{Role: User, Content: content}
}

func AssistantMessage(content string, toolCalls []ToolCall) *Message {
	return &Message{Role: Assistant, Content: content, ToolCalls: toolCalls}
}

func ToolMessage(content string, toolCallID string) *Message {
	return &Message{Role: Tool, Content: content, ToolCallID: toolCallID}
}

// --- Role constants (matching eino naming for facade compatibility) ---

const (
	Assistant RoleType = "assistant"
	User      RoleType = "user"
	System    RoleType = "system"
	Tool      RoleType = "tool"
)

// ConcatMessages merges a slice of same-role messages into a single message
// by concatenating their content and reasoning content.
func ConcatMessages(msgs []*Message) (*Message, error) {
	if len(msgs) == 0 {
		return nil, errors.New("concat messages: empty input")
	}
	var (
		content     string
		reasoning   string
		role        = msgs[0].Role
	)
	for _, m := range msgs {
		if m.Role != role {
			return nil, errors.New("concat messages: all messages must have the same role")
		}
		content += m.Content
		reasoning += m.ReasoningContent
	}
	return &Message{
		Role:             role,
		Content:          content,
		ReasoningContent: reasoning,
	}, nil
}

// RegisterName is a no-op stub for type registration compatibility.
// It mirrors eino's schema.RegisterName but does nothing in the wfcompose
// context since there is no serialization registry.
func RegisterName[T any](name string) {
	// no-op
}
