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

// This file defines framework-agnostic chat message types that mirror the
// public field surface of cloudwego/eino/schema's Message family. They let
// business code carry conversation messages without importing eino, and let
// engine adapters (see pkg/wfcompose/einobridge) convert to/from eino's
// concrete types at the boundary. Field names and JSON tags intentionally match
// eino's so that future migration is a mechanical type swap.

// RoleType is the role of a message participant.
type RoleType string

const (
	RoleSystem    RoleType = "system"
	RoleUser      RoleType = "user"
	RoleAssistant RoleType = "assistant"
	RoleTool      RoleType = "tool"
)

// ChatMessagePartType is the type of a multimodal message part.
type ChatMessagePartType string

const (
	ChatMessagePartTypeText       ChatMessagePartType = "text"
	ChatMessagePartTypeImageURL   ChatMessagePartType = "image_url"
	ChatMessagePartTypeAudioURL   ChatMessagePartType = "audio_url"
	ChatMessagePartTypeVideoURL   ChatMessagePartType = "video_url"
	ChatMessagePartTypeFileURL    ChatMessagePartType = "file_url"
	ChatMessagePartTypeReasoning  ChatMessagePartType = "reasoning"
)

// ImageURLDetail is the fidelity hint for an image part.
type ImageURLDetail string

const (
	ImageURLDetailHigh ImageURLDetail = "high"
	ImageURLDetailLow  ImageURLDetail = "low"
	ImageURLDetailAuto ImageURLDetail = "auto"
)

// ChatMessageImageURL is the image payload of an image_url part.
type ChatMessageImageURL struct {
	URL      string              `json:"url,omitempty"`
	URI      string              `json:"uri,omitempty"`
	Detail   ImageURLDetail      `json:"detail,omitempty"`
	MIMEType string              `json:"mime_type,omitempty"`
	Extra    map[string]any      `json:"extra,omitempty"`
}

// ChatMessageAudioURL is the audio payload of an audio_url part.
type ChatMessageAudioURL struct {
	URL      string         `json:"url,omitempty"`
	URI      string         `json:"uri,omitempty"`
	MIMEType string         `json:"mime_type,omitempty"`
	Extra    map[string]any `json:"extra,omitempty"`
}

// ChatMessageVideoURL is the video payload of a video_url part.
type ChatMessageVideoURL struct {
	URL      string         `json:"url,omitempty"`
	URI      string         `json:"uri,omitempty"`
	MIMEType string         `json:"mime_type,omitempty"`
	Extra    map[string]any `json:"extra,omitempty"`
}

// ChatMessageFileURL is the file payload of a file_url part.
type ChatMessageFileURL struct {
	URL      string         `json:"url,omitempty"`
	URI      string         `json:"uri,omitempty"`
	MIMEType string         `json:"mime_type,omitempty"`
	Name     string         `json:"name,omitempty"`
	Extra    map[string]any `json:"extra,omitempty"`
}

// ChatMessagePart is one part of a multimodal message (Message.MultiContent).
type ChatMessagePart struct {
	Type     ChatMessagePartType      `json:"type,omitempty"`
	Text     string                   `json:"text,omitempty"`
	ImageURL *ChatMessageImageURL     `json:"image_url,omitempty"`
	AudioURL *ChatMessageAudioURL     `json:"audio_url,omitempty"`
	VideoURL *ChatMessageVideoURL     `json:"video_url,omitempty"`
	FileURL  *ChatMessageFileURL      `json:"file_url,omitempty"`
}

// FunctionCall is the function invocation requested by the assistant.
type FunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// ToolCall is a single tool/function call requested by the assistant.
type ToolCall struct {
	Index    *int          `json:"index,omitempty"`
	ID       string        `json:"id"`
	Type     string        `json:"type"`
	Function FunctionCall  `json:"function"`
	Extra    map[string]any `json:"extra,omitempty"`
}

// Message is the framework-agnostic chat message. It mirrors eino's
// schema.Message public field surface so engine adapters can convert
// losslessly and migration is a mechanical type swap.
type Message struct {
	Role         RoleType          `json:"role"`
	Content      string            `json:"content"`
	MultiContent []ChatMessagePart `json:"multi_content,omitempty"`
	Name         string            `json:"name,omitempty"`
	ToolCalls    []ToolCall        `json:"tool_calls,omitempty"`
	ToolCallID   string            `json:"tool_call_id,omitempty"`
	ToolName     string            `json:"tool_name,omitempty"`
	Extra        map[string]any    `json:"extra,omitempty"`
}

// Document is a framework-agnostic retrieved document.
type Document struct {
	ID       string         `json:"id"`
	Content  string         `json:"content"`
	MetaData map[string]any `json:"meta_data"`
}

// String returns the document content.
func (d *Document) String() string {
	if d == nil {
		return ""
	}
	return d.Content
}
