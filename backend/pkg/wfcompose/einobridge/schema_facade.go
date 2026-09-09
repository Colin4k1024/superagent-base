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

// schema_facade re-exports wfcompose schema types as prefixed aliases
// so callers avoid importing cloudwego/eino/schema directly.
// This file has ZERO cloudwego/eino imports.
package einobridge

import (
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

// ---------------------------------------------------------------------------
// Type aliases (all map to wfcompose native types)
// ---------------------------------------------------------------------------

type Message = wfcompose.Message
type StreamReader[T any] = wfcompose.StreamReader[T]
type StreamWriter[T any] = wfcompose.StreamWriter[T]
type ToolInfo = wfcompose.ToolInfo
type ParamsOneOf = wfcompose.ParamsOneOf
type ParameterInfo = wfcompose.ParameterInfo
type ToolCall = wfcompose.ToolCall
type MessagesTemplate = wfcompose.MessagesTemplate
type FormatType = wfcompose.FormatType
type ConvertOption = wfcompose.ConvertOption

// ---------------------------------------------------------------------------
// Constants & variables
// ---------------------------------------------------------------------------

const Jinja2 FormatType = wfcompose.Jinja2

var ErrNoValue = wfcompose.ErrNoValue

// ---------------------------------------------------------------------------
// Message constructors
// ---------------------------------------------------------------------------

func SystemMessage(content string) *Message {
	return wfcompose.SystemMessage(content)
}

func UserMessage(content string) *Message {
	return wfcompose.UserMessage(content)
}

func MessagesPlaceholder(key string, optional bool) MessagesTemplate {
	return wfcompose.MessagesPlaceholder(key, optional)
}

func AssistantMessage(content string, toolCalls []ToolCall) *Message {
	return wfcompose.AssistantMessage(content, toolCalls)
}

func ToolMessage(content string, toolCallID string) *Message {
	return wfcompose.ToolMessage(content, toolCallID)
}

func ConcatMessages(msgs []*Message) (*Message, error) {
	return wfcompose.ConcatMessages(msgs)
}

// ---------------------------------------------------------------------------
// Stream helpers
// ---------------------------------------------------------------------------

func Pipe[T any](cap int) (*StreamReader[T], *StreamWriter[T]) {
	return wfcompose.Pipe[T](cap)
}

func StreamReaderWithConvert[T, D any](sr *StreamReader[T], convert func(T) (D, error), opts ...ConvertOption) *StreamReader[D] {
	return wfcompose.StreamReaderWithConvert[T, D](sr, convert, opts...)
}

func StreamReaderFromArray[T any](arr []T) *StreamReader[T] {
	return wfcompose.StreamReaderFromArray(arr)
}

func MergeStreamReaders[T any](readers []*StreamReader[T]) *StreamReader[T] {
	return wfcompose.MergeStreamReaders(readers)
}

// ---------------------------------------------------------------------------
// Serialization
// ---------------------------------------------------------------------------

func RegisterName[T any](name string) {
	wfcompose.RegisterName[T](name)
}

// ---------------------------------------------------------------------------
// Tool helpers
// ---------------------------------------------------------------------------

func NewParamsOneOfByParams(params map[string]*ParameterInfo) *ParamsOneOf {
	return wfcompose.NewParamsOneOfByParams(params)
}

// ---------------------------------------------------------------------------
// Additional type aliases for domain/workflow/ migration
// ---------------------------------------------------------------------------

type ChatMessagePart = wfcompose.ChatMessagePart
type ChatMessagePartType = wfcompose.ChatMessagePartType
type ChatMessageImageURL = wfcompose.ChatMessageImageURL
type ChatMessageAudioURL = wfcompose.ChatMessageAudioURL
type ChatMessageVideoURL = wfcompose.ChatMessageVideoURL
type ChatMessageFileURL = wfcompose.ChatMessageFileURL
type RoleType = wfcompose.RoleType
type ResponseMeta = wfcompose.ResponseMeta
type TokenUsage = wfcompose.TokenUsage
type Document = wfcompose.Document
type FunctionCall = wfcompose.FunctionCall

// Role constants
const (
	Assistant RoleType = wfcompose.Assistant
	User      RoleType = wfcompose.User
	System    RoleType = wfcompose.System
	Tool      RoleType = wfcompose.Tool
)

// ChatMessagePartType constants
const (
	ChatMessagePartTypeText      ChatMessagePartType = wfcompose.ChatMessagePartTypeText
	ChatMessagePartTypeImageURL  ChatMessagePartType = wfcompose.ChatMessagePartTypeImageURL
	ChatMessagePartTypeAudioURL  ChatMessagePartType = wfcompose.ChatMessagePartTypeAudioURL
	ChatMessagePartTypeVideoURL  ChatMessagePartType = wfcompose.ChatMessagePartTypeVideoURL
	ChatMessagePartTypeFileURL   ChatMessagePartType = wfcompose.ChatMessagePartTypeFileURL
)

// DataType and JSON schema constants
type DataType = wfcompose.DataType

const (
	Object  DataType = wfcompose.Object
	Number  DataType = wfcompose.Number
	Integer DataType = wfcompose.Integer
	String  DataType = wfcompose.String
	Array   DataType = wfcompose.Array
	Null    DataType = wfcompose.Null
	Boolean DataType = wfcompose.Boolean
)
