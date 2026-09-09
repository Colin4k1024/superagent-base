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

// schema_facade re-exports cloudwego/eino/schema types and functions as
// type aliases and thin wrappers so agentflow can avoid importing
// cloudwego/eino/schema directly (S3 acceptance criterion).
package einobridge

import (
	"github.com/cloudwego/eino/schema"
)

// ---------------------------------------------------------------------------
// Type aliases
// ---------------------------------------------------------------------------

type Message = schema.Message
type StreamReader[T any] = schema.StreamReader[T]
type StreamWriter[T any] = schema.StreamWriter[T]
type ToolInfo = schema.ToolInfo
type ParamsOneOf = schema.ParamsOneOf
type ParameterInfo = schema.ParameterInfo
type ToolCall = schema.ToolCall
type MessagesTemplate = schema.MessagesTemplate
type FormatType = schema.FormatType
type ConvertOption = schema.ConvertOption

// ---------------------------------------------------------------------------
// Constants & variables
// ---------------------------------------------------------------------------

const Jinja2 FormatType = schema.Jinja2

var ErrNoValue = schema.ErrNoValue

// ---------------------------------------------------------------------------
// Message constructors
// ---------------------------------------------------------------------------

func SystemMessage(content string) *Message {
	return schema.SystemMessage(content)
}

func UserMessage(content string) *Message {
	return schema.UserMessage(content)
}

func MessagesPlaceholder(key string, optional bool) MessagesTemplate {
	return schema.MessagesPlaceholder(key, optional)
}

// ---------------------------------------------------------------------------
// Stream helpers
// ---------------------------------------------------------------------------

func Pipe[T any](cap int) (*StreamReader[T], *StreamWriter[T]) {
	return schema.Pipe[T](cap)
}

func StreamReaderWithConvert[T, D any](sr *StreamReader[T], convert func(T) (D, error), opts ...ConvertOption) *StreamReader[D] {
	return schema.StreamReaderWithConvert[T, D](sr, convert, opts...)
}

func ConcatMessages(msgs []*Message) (*Message, error) {
	return schema.ConcatMessages(msgs)
}

// ---------------------------------------------------------------------------
// Serialization
// ---------------------------------------------------------------------------

func RegisterName[T any](name string) {
	schema.RegisterName[T](name)
}

// ---------------------------------------------------------------------------
// Tool helpers
// ---------------------------------------------------------------------------

func NewParamsOneOfByParams(params map[string]*ParameterInfo) *ParamsOneOf {
	return schema.NewParamsOneOfByParams(params)
}

func AssistantMessage(content string, toolCalls []ToolCall) *Message {
	return schema.AssistantMessage(content, toolCalls)
}

// ---------------------------------------------------------------------------
// Additional type aliases for domain/workflow/ migration
// ---------------------------------------------------------------------------

type ChatMessagePart = schema.ChatMessagePart
type ChatMessagePartType = schema.ChatMessagePartType
type ChatMessageImageURL = schema.ChatMessageImageURL
type ChatMessageAudioURL = schema.ChatMessageAudioURL
type ChatMessageVideoURL = schema.ChatMessageVideoURL
type ChatMessageFileURL = schema.ChatMessageFileURL
type RoleType = schema.RoleType
type ResponseMeta = schema.ResponseMeta
type TokenUsage = schema.TokenUsage
type Document = schema.Document

// Role constants
const (
	Assistant RoleType = schema.Assistant
	User      RoleType = schema.User
	System    RoleType = schema.System
	Tool      RoleType = schema.Tool
)

// ChatMessagePartType constants
const (
	ChatMessagePartTypeText      ChatMessagePartType = schema.ChatMessagePartTypeText
	ChatMessagePartTypeImageURL  ChatMessagePartType = schema.ChatMessagePartTypeImageURL
	ChatMessagePartTypeAudioURL  ChatMessagePartType = schema.ChatMessagePartTypeAudioURL
	ChatMessagePartTypeVideoURL  ChatMessagePartType = schema.ChatMessagePartTypeVideoURL
	ChatMessagePartTypeFileURL   ChatMessagePartType = schema.ChatMessagePartTypeFileURL
)

// Additional stream helpers
func StreamReaderFromArray[T any](arr []T) *StreamReader[T] {
	return schema.StreamReaderFromArray(arr)
}

func MergeStreamReaders[T any](readers []*StreamReader[T]) *StreamReader[T] {
	return schema.MergeStreamReaders(readers)
}

// DataType and JSON schema constants
type DataType = schema.DataType

const (
	Object  DataType = schema.Object
	Number  DataType = schema.Number
	Integer DataType = schema.Integer
	String  DataType = schema.String
	Array   DataType = schema.Array
	Null    DataType = schema.Null
	Boolean DataType = schema.Boolean
)

// Additional type aliases for S4 migration
type FunctionCall = schema.FunctionCall

// ToolMessage constructs a tool-result message.
func ToolMessage(content string, toolCallID string) *Message {
	return schema.ToolMessage(content, toolCallID)
}

