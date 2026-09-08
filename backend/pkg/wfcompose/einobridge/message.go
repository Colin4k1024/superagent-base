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

package einobridge

import (
	"github.com/cloudwego/eino/schema"

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

// wrapResponseMeta converts eino *schema.ResponseMeta to *wfcompose.ResponseMeta.
func wrapResponseMeta(rm *schema.ResponseMeta) *wfcompose.ResponseMeta {
	if rm == nil {
		return nil
	}
	out := &wfcompose.ResponseMeta{
		FinishReason: rm.FinishReason,
	}
	if rm.Usage != nil {
		out.Usage = &wfcompose.TokenUsage{
			PromptTokens:     rm.Usage.PromptTokens,
			CompletionTokens: rm.Usage.CompletionTokens,
			TotalTokens:      rm.Usage.TotalTokens,
		}
	}
	return out
}

// unwrapResponseMeta converts *wfcompose.ResponseMeta to eino *schema.ResponseMeta.
func unwrapResponseMeta(rm *wfcompose.ResponseMeta) *schema.ResponseMeta {
	if rm == nil {
		return nil
	}
	out := &schema.ResponseMeta{
		FinishReason: rm.FinishReason,
	}
	if rm.Usage != nil {
		out.Usage = &schema.TokenUsage{
			PromptTokens:     rm.Usage.PromptTokens,
			CompletionTokens: rm.Usage.CompletionTokens,
			TotalTokens:      rm.Usage.TotalTokens,
		}
	}
	return out
}

// WrapMessage converts an eino *schema.Message into a framework-agnostic
// *wfcompose.Message by copying the public field surface. nil in, nil out.
func WrapMessage(m *schema.Message) *wfcompose.Message {
	if m == nil {
		return nil
	}
	out := &wfcompose.Message{
		Role:             wfcompose.RoleType(m.Role),
		Content:          m.Content,
		Name:             m.Name,
		ToolCallID:       m.ToolCallID,
		ToolName:         m.ToolName,
		Extra:            m.Extra,
		ResponseMeta:     wrapResponseMeta(m.ResponseMeta),
		ReasoningContent: m.ReasoningContent,
	}
	if len(m.MultiContent) > 0 {
		out.MultiContent = make([]wfcompose.ChatMessagePart, len(m.MultiContent))
		for i, p := range m.MultiContent {
			out.MultiContent[i] = wrapChatMessagePart(p)
		}
	}
	if len(m.ToolCalls) > 0 {
		out.ToolCalls = make([]wfcompose.ToolCall, len(m.ToolCalls))
		for i, tc := range m.ToolCalls {
			out.ToolCalls[i] = wfcompose.ToolCall{
				Index: tc.Index,
				ID:    tc.ID,
				Type:  tc.Type,
				Function: wfcompose.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
				Extra: tc.Extra,
			}
		}
	}
	return out
}

// UnwrapMessage converts a framework-agnostic *wfcompose.Message back into an
// eino *schema.Message. nil in, nil out. It is the inverse of WrapMessage.
func UnwrapMessage(m *wfcompose.Message) *schema.Message {
	if m == nil {
		return nil
	}
	out := &schema.Message{
		Role:             schema.RoleType(m.Role),
		Content:          m.Content,
		Name:             m.Name,
		ToolCallID:       m.ToolCallID,
		ToolName:         m.ToolName,
		Extra:            m.Extra,
		ResponseMeta:     unwrapResponseMeta(m.ResponseMeta),
		ReasoningContent: m.ReasoningContent,
	}
	if len(m.MultiContent) > 0 {
		out.MultiContent = make([]schema.ChatMessagePart, len(m.MultiContent))
		for i, p := range m.MultiContent {
			out.MultiContent[i] = unwrapChatMessagePart(p)
		}
	}
	if len(m.ToolCalls) > 0 {
		out.ToolCalls = make([]schema.ToolCall, len(m.ToolCalls))
		for i, tc := range m.ToolCalls {
			out.ToolCalls[i] = schema.ToolCall{
				Index: tc.Index,
				ID:    tc.ID,
				Type:  tc.Type,
				Function: schema.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
				Extra: tc.Extra,
			}
		}
	}
	return out
}

// WrapMessageSlice converts a slice of eino messages to framework-agnostic ones.
func WrapMessageSlice(in []*schema.Message) []*wfcompose.Message {
	out := make([]*wfcompose.Message, len(in))
	for i, m := range in {
		out[i] = WrapMessage(m)
	}
	return out
}

// UnwrapMessageSlice converts a slice of framework-agnostic messages to eino ones.
func UnwrapMessageSlice(in []*wfcompose.Message) []*schema.Message {
	out := make([]*schema.Message, len(in))
	for i, m := range in {
		out[i] = UnwrapMessage(m)
	}
	return out
}

// WrapDocument converts an eino *schema.Document to a framework-agnostic one.
func WrapDocument(d *schema.Document) *wfcompose.Document {
	if d == nil {
		return nil
	}
	return &wfcompose.Document{
		ID:       d.ID,
		Content:  d.Content,
		MetaData: d.MetaData,
	}
}

// WrapDocumentSlice converts a slice of eino documents to framework-agnostic ones.
func WrapDocumentSlice(in []*schema.Document) []*wfcompose.Document {
	out := make([]*wfcompose.Document, len(in))
	for i, d := range in {
		out[i] = WrapDocument(d)
	}
	return out
}

// UnwrapDocumentSlice converts a slice of framework-agnostic documents to eino ones.
func UnwrapDocumentSlice(in []*wfcompose.Document) []*schema.Document {
	out := make([]*schema.Document, len(in))
	for i, d := range in {
		out[i] = UnwrapDocument(d)
	}
	return out
}

// UnwrapDocument converts a framework-agnostic document to an eino one.
func UnwrapDocument(d *wfcompose.Document) *schema.Document {
	if d == nil {
		return nil
	}
	return &schema.Document{
		ID:       d.ID,
		Content:  d.Content,
		MetaData: d.MetaData,
	}
}

func wrapChatMessagePart(p schema.ChatMessagePart) wfcompose.ChatMessagePart {
	out := wfcompose.ChatMessagePart{
		Type: wfcompose.ChatMessagePartType(p.Type),
		Text: p.Text,
	}
	if p.ImageURL != nil {
		out.ImageURL = &wfcompose.ChatMessageImageURL{
			URL:      p.ImageURL.URL,
			URI:      p.ImageURL.URI,
			Detail:   wfcompose.ImageURLDetail(p.ImageURL.Detail),
			MIMEType: p.ImageURL.MIMEType,
			Extra:    p.ImageURL.Extra,
		}
	}
	if p.AudioURL != nil {
		out.AudioURL = &wfcompose.ChatMessageAudioURL{
			URL:      p.AudioURL.URL,
			URI:      p.AudioURL.URI,
			MIMEType: p.AudioURL.MIMEType,
			Extra:    p.AudioURL.Extra,
		}
	}
	if p.VideoURL != nil {
		out.VideoURL = &wfcompose.ChatMessageVideoURL{
			URL:      p.VideoURL.URL,
			URI:      p.VideoURL.URI,
			MIMEType: p.VideoURL.MIMEType,
			Extra:    p.VideoURL.Extra,
		}
	}
	if p.FileURL != nil {
		out.FileURL = &wfcompose.ChatMessageFileURL{
			URL:      p.FileURL.URL,
			URI:      p.FileURL.URI,
			MIMEType: p.FileURL.MIMEType,
			Name:     p.FileURL.Name,
			Extra:    p.FileURL.Extra,
		}
	}
	return out
}

func unwrapChatMessagePart(p wfcompose.ChatMessagePart) schema.ChatMessagePart {
	out := schema.ChatMessagePart{
		Type: schema.ChatMessagePartType(p.Type),
		Text: p.Text,
	}
	if p.ImageURL != nil {
		out.ImageURL = &schema.ChatMessageImageURL{
			URL:      p.ImageURL.URL,
			URI:      p.ImageURL.URI,
			Detail:   schema.ImageURLDetail(p.ImageURL.Detail),
			MIMEType: p.ImageURL.MIMEType,
			Extra:    p.ImageURL.Extra,
		}
	}
	if p.AudioURL != nil {
		out.AudioURL = &schema.ChatMessageAudioURL{
			URL:      p.AudioURL.URL,
			URI:      p.AudioURL.URI,
			MIMEType: p.AudioURL.MIMEType,
			Extra:    p.AudioURL.Extra,
		}
	}
	if p.VideoURL != nil {
		out.VideoURL = &schema.ChatMessageVideoURL{
			URL:      p.VideoURL.URL,
			URI:      p.VideoURL.URI,
			MIMEType: p.VideoURL.MIMEType,
			Extra:    p.VideoURL.Extra,
		}
	}
	if p.FileURL != nil {
		out.FileURL = &schema.ChatMessageFileURL{
			URL:      p.FileURL.URL,
			URI:      p.FileURL.URI,
			MIMEType: p.FileURL.MIMEType,
			Name:     p.FileURL.Name,
			Extra:    p.FileURL.Extra,
		}
	}
	return out
}
