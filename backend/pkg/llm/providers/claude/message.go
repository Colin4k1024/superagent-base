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

package claude

import (
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

// systemTextFromMessages extracts leading system messages and returns the
// remaining conversation messages. Anthropic puts the system prompt in a
// top-level field rather than as a message role.
func splitSystemMessages(msgs []*wfcompose.Message) (string, []*wfcompose.Message) {
	var systemText string
	var rest []*wfcompose.Message
	for _, m := range msgs {
		if m == nil {
			continue
		}
		if m.Role == wfcompose.RoleSystem && len(rest) == 0 {
			if systemText != "" {
				systemText += "\n"
			}
			systemText += m.Content
			continue
		}
		rest = append(rest, m)
	}
	return systemText, rest
}

// toAnthropicMessages converts wfcompose messages (with system already split
// out) into Anthropic MessageParam slice. System and tool roles are handled
// specially; only user/assistant go into the messages array.
func toAnthropicMessages(msgs []*wfcompose.Message) []anthropic.MessageParam {
	out := make([]anthropic.MessageParam, 0, len(msgs))
	for _, m := range msgs {
		if m == nil {
			continue
		}
		// Skip stray system messages (already extracted).
		if m.Role == wfcompose.RoleSystem {
			continue
		}
		switch m.Role {
		case wfcompose.RoleTool:
			// Anthropic represents tool results as user-role messages with
			// tool_result content blocks.
			out = append(out, anthropic.NewUserMessage(anthropic.ContentBlockParamUnion{
				OfToolResult: &anthropic.ToolResultBlockParam{
					ToolUseID: m.ToolCallID,
					Content: []anthropic.ToolResultBlockParamContentUnion{
						{OfText: &anthropic.TextBlockParam{Text: m.Content}},
					},
				},
			}))
		case wfcompose.RoleAssistant:
			blocks := toAssistantBlocks(m)
			out = append(out, anthropic.NewAssistantMessage(blocks...))
		default: // user (and any unknown role treated as user)
			blocks := toUserBlocks(m)
			out = append(out, anthropic.NewUserMessage(blocks...))
		}
	}
	return out
}

func toUserBlocks(m *wfcompose.Message) []anthropic.ContentBlockParamUnion {
	var blocks []anthropic.ContentBlockParamUnion
	if len(m.MultiContent) > 0 {
		for _, part := range m.MultiContent {
			switch part.Type {
			case wfcompose.ChatMessagePartTypeText:
				blocks = append(blocks, anthropic.ContentBlockParamUnion{
					OfText: &anthropic.TextBlockParam{Text: part.Text},
				})
			case wfcompose.ChatMessagePartTypeImageURL:
				if part.ImageURL != nil && part.ImageURL.URL != "" {
					blocks = append(blocks, anthropic.ContentBlockParamUnion{
						OfImage: &anthropic.ImageBlockParam{
							Source: anthropic.ImageBlockParamSourceUnion{
								OfURL: &anthropic.URLImageSourceParam{URL: part.ImageURL.URL},
							},
						},
					})
				}
			}
		}
	}
	if m.Content != "" && len(blocks) == 0 {
		blocks = append(blocks, anthropic.ContentBlockParamUnion{
			OfText: &anthropic.TextBlockParam{Text: m.Content},
		})
	}
	if len(blocks) == 0 {
		blocks = append(blocks, anthropic.ContentBlockParamUnion{
			OfText: &anthropic.TextBlockParam{Text: ""},
		})
	}
	return blocks
}

func toAssistantBlocks(m *wfcompose.Message) []anthropic.ContentBlockParamUnion {
	var blocks []anthropic.ContentBlockParamUnion
	if m.ReasoningContent != "" {
		// Reasoning content is carried over as a thinking block so multi-turn
		// extended-thinking sessions remain valid.
		blocks = append(blocks, anthropic.ContentBlockParamUnion{
			OfThinking: &anthropic.ThinkingBlockParam{
				Thinking: m.ReasoningContent,
			},
		})
	}
	if m.Content != "" {
		blocks = append(blocks, anthropic.ContentBlockParamUnion{
			OfText: &anthropic.TextBlockParam{Text: m.Content},
		})
	}
	for _, tc := range m.ToolCalls {
		blocks = append(blocks, anthropic.ContentBlockParamUnion{
			OfToolUse: &anthropic.ToolUseBlockParam{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: parseToolInput(tc.Function.Arguments),
			},
		})
	}
	if len(blocks) == 0 {
		blocks = append(blocks, anthropic.ContentBlockParamUnion{
			OfText: &anthropic.TextBlockParam{Text: ""},
		})
	}
	return blocks
}

func parseToolInput(args string) any {
	if args == "" {
		return map[string]any{}
	}
	var v any
	if err := json.Unmarshal([]byte(args), &v); err != nil {
		return map[string]any{}
	}
	return v
}

// toAnthropicTools converts wfcompose ToolInfo into Anthropic ToolUnionParam.
func toAnthropicTools(tools []*wfcompose.ToolInfo) []anthropic.ToolUnionParam {
	out := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		if t == nil {
			continue
		}
		tp := anthropic.ToolParam{
			Name: t.Name,
		}
		if t.Desc != "" {
			tp.Description = param.NewOpt(t.Desc)
		}
		if t.ParamsOneOf != nil && t.ParamsOneOf.Params() != nil {
			tp.InputSchema = paramsToInputSchema(t.ParamsOneOf.Params())
		}
		out = append(out, anthropic.ToolUnionParam{OfTool: &tp})
	}
	return out
}

func paramsToInputSchema(params map[string]*wfcompose.ParameterInfo) anthropic.ToolInputSchemaParam {
	schema := anthropic.ToolInputSchemaParam{}
	props := make(map[string]any)
	required := []string{}
	for name, p := range params {
		if p == nil {
			continue
		}
		prop := map[string]any{
			"type":        string(p.Type),
			"description": p.Desc,
		}
		if len(p.Enum) > 0 {
			prop["enum"] = p.Enum
		}
		if p.Type == wfcompose.Object && len(p.SubParams) > 0 {
			sub := paramsToInputSchema(p.SubParams)
			prop["properties"] = sub.Properties
			if len(sub.Required) > 0 {
				prop["required"] = sub.Required
			}
		}
		if p.Type == wfcompose.Array && p.ElemInfo != nil {
			prop["items"] = map[string]any{
				"type":        string(p.ElemInfo.Type),
				"description": p.ElemInfo.Desc,
			}
		}
		props[name] = prop
		if p.Required {
			required = append(required, name)
		}
	}
	schema.Properties = props
	schema.Required = required
	return schema
}

// fromAnthropicMessage converts an Anthropic response Message into a
// wfcompose.Message.
func fromAnthropicMessage(resp *anthropic.Message) *wfcompose.Message {
	msg := &wfcompose.Message{
		Role: wfcompose.RoleAssistant,
		ResponseMeta: &wfcompose.ResponseMeta{
			FinishReason: string(resp.StopReason),
		},
	}
	if resp.Usage.InputTokens > 0 || resp.Usage.OutputTokens > 0 {
		msg.ResponseMeta.Usage = &wfcompose.TokenUsage{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		}
	}
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			if msg.Content != "" {
				msg.Content += "\n"
			}
			msg.Content += block.Text
		case "thinking":
			if msg.ReasoningContent != "" {
				msg.ReasoningContent += "\n"
			}
			msg.ReasoningContent += block.Thinking
		case "tool_use":
			msg.ToolCalls = append(msg.ToolCalls, wfcompose.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: wfcompose.FunctionCall{
					Name:      block.Name,
					Arguments: string(block.Input),
				},
			})
		}
	}
	return msg
}

// fromAnthropicStreamDelta converts a single streaming event union into a
// partial wfcompose.Message. It returns nil to signal "skip this event".
func fromAnthropicStreamDelta(event anthropic.MessageStreamEventUnion) *wfcompose.Message {
	switch event.Type {
	case "message_start":
		ev := event.AsMessageStart()
		msg := &wfcompose.Message{Role: wfcompose.RoleAssistant}
		if ev.Message.StopReason != "" {
			msg.ResponseMeta = &wfcompose.ResponseMeta{
				FinishReason: string(ev.Message.StopReason),
			}
		}
		return msg
	case "content_block_start":
		ev := event.AsContentBlockStart()
		switch ev.ContentBlock.Type {
		case "text":
			return &wfcompose.Message{Content: ev.ContentBlock.Text}
		case "thinking":
			return &wfcompose.Message{ReasoningContent: ev.ContentBlock.Thinking}
		case "tool_use":
			return &wfcompose.Message{
				ToolCalls: []wfcompose.ToolCall{{
					ID:   ev.ContentBlock.ID,
					Type: "function",
					Function: wfcompose.FunctionCall{
						Name: ev.ContentBlock.Name,
					},
				}},
			}
		}
	case "content_block_delta":
		ev := event.AsContentBlockDelta()
		switch ev.Delta.Type {
		case "text_delta":
			return &wfcompose.Message{Content: ev.Delta.Text}
		case "thinking_delta":
			return &wfcompose.Message{ReasoningContent: ev.Delta.Thinking}
		case "input_json_delta":
			return &wfcompose.Message{
				ToolCalls: []wfcompose.ToolCall{{
					Function: wfcompose.FunctionCall{
						Arguments: ev.Delta.PartialJSON,
					},
				}},
			}
		}
	case "content_block_stop":
		return nil
	case "message_delta":
		ev := event.AsMessageDelta()
		msg := &wfcompose.Message{
			ResponseMeta: &wfcompose.ResponseMeta{
				FinishReason: string(ev.Delta.StopReason),
			},
		}
		if ev.Usage.OutputTokens > 0 {
			msg.ResponseMeta.Usage = &wfcompose.TokenUsage{
				CompletionTokens: int(ev.Usage.OutputTokens),
			}
		}
		return msg
	case "message_stop":
		return nil
	}
	return nil
}
