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

package openai

import (
	goopenai "github.com/meguminnnnnnnnn/go-openai"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

func toOpenAIMessage(m *wfcompose.Message) goopenai.ChatCompletionMessage {
	if m == nil {
		return goopenai.ChatCompletionMessage{}
	}
	msg := goopenai.ChatCompletionMessage{
		Role:             string(m.Role),
		Content:          m.Content,
		Name:             m.Name,
		ReasoningContent: m.ReasoningContent,
		ToolCallID:       m.ToolCallID,
	}
	if len(m.MultiContent) > 0 {
		msg.MultiContent = toOpenAIMultiContent(m.MultiContent)
	}
	if len(m.ToolCalls) > 0 {
		msg.ToolCalls = toOpenAIToolCalls(m.ToolCalls)
	}
	return msg
}

func toOpenAIMultiContent(parts []wfcompose.ChatMessagePart) []goopenai.ChatMessagePart {
	out := make([]goopenai.ChatMessagePart, 0, len(parts))
	for _, p := range parts {
		mp := goopenai.ChatMessagePart{
			Type: goopenai.ChatMessagePartType(p.Type),
			Text: p.Text,
		}
		if p.ImageURL != nil {
			mp.ImageURL = &goopenai.ChatMessageImageURL{
				URL:    p.ImageURL.URL,
				Detail: goopenai.ImageURLDetail(p.ImageURL.Detail),
			}
		}
		if p.AudioURL != nil {
			mp.InputAudio = &goopenai.ChatMessageInputAudio{
				Data:    p.AudioURL.URL,
				Format:  p.AudioURL.MIMEType,
			}
		}
		out = append(out, mp)
	}
	return out
}

func toOpenAIToolCalls(tcs []wfcompose.ToolCall) []goopenai.ToolCall {
	out := make([]goopenai.ToolCall, 0, len(tcs))
	for _, tc := range tcs {
		call := goopenai.ToolCall{
			ID:   tc.ID,
			Type: goopenai.ToolType(tc.Type),
			Function: goopenai.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
		if tc.Index != nil {
			idx := *tc.Index
			call.Index = &idx
		}
		out = append(out, call)
	}
	return out
}

func fromOpenAIMessage(msg goopenai.ChatCompletionMessage) *wfcompose.Message {
	out := &wfcompose.Message{
		Role:             wfcompose.RoleType(msg.Role),
		Content:          msg.Content,
		Name:             msg.Name,
		ReasoningContent: msg.ReasoningContent,
		ToolCallID:       msg.ToolCallID,
	}
	if len(msg.ToolCalls) > 0 {
		out.ToolCalls = make([]wfcompose.ToolCall, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			idx := tc.Index
			out.ToolCalls[i] = wfcompose.ToolCall{
				Index: idx,
				ID:    tc.ID,
				Type:  string(tc.Type),
				Function: wfcompose.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}
	return out
}

func fromOpenAIChoice(choice goopenai.ChatCompletionChoice) *wfcompose.Message {
	msg := fromOpenAIMessage(choice.Message)
	msg.ResponseMeta = &wfcompose.ResponseMeta{
		FinishReason: string(choice.FinishReason),
	}
	return msg
}

func fromOpenAIStreamDelta(delta goopenai.ChatCompletionStreamChoiceDelta, finishReason goopenai.FinishReason) *wfcompose.Message {
	out := &wfcompose.Message{
		Role:             wfcompose.RoleType(delta.Role),
		Content:          delta.Content,
		ReasoningContent: delta.ReasoningContent,
	}
	if len(delta.ToolCalls) > 0 {
		out.ToolCalls = make([]wfcompose.ToolCall, len(delta.ToolCalls))
		for i, tc := range delta.ToolCalls {
			out.ToolCalls[i] = wfcompose.ToolCall{
				Index: tc.Index,
				ID:    tc.ID,
				Type:  string(tc.Type),
				Function: wfcompose.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
	}
	out.ResponseMeta = &wfcompose.ResponseMeta{
		FinishReason: string(finishReason),
	}
	return out
}
