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

package gemini

import (
	"encoding/json"

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
	"google.golang.org/genai"
)

// toGeminiContents converts wfcompose messages to genai Content slice.
// System messages are extracted and returned separately so the caller can
// set them on GenerateContentConfig.SystemInstruction (Gemini handles
// system instructions outside the contents list).
func toGeminiContents(msgs []*wfcompose.Message) ([]*genai.Content, *genai.Content) {
	var (
		contents     []*genai.Content
		systemParts  []*genai.Part
	)
	for _, msg := range msgs {
		if msg == nil {
			continue
		}
		if msg.Role == wfcompose.RoleSystem {
			if msg.Content != "" {
				systemParts = append(systemParts, genai.NewPartFromText(msg.Content))
			}
			for _, p := range msg.MultiContent {
				if p.Type == wfcompose.ChatMessagePartTypeText && p.Text != "" {
					systemParts = append(systemParts, genai.NewPartFromText(p.Text))
				}
			}
			continue
		}
		contents = append(contents, toGeminiContent(msg))
	}
	var sys *genai.Content
	if len(systemParts) > 0 {
		sys = &genai.Content{Parts: systemParts, Role: genai.RoleUser}
	}
	return contents, sys
}

// toGeminiContent converts a single wfcompose Message to a genai Content.
func toGeminiContent(m *wfcompose.Message) *genai.Content {
	role := genai.RoleUser
	if m.Role == wfcompose.RoleAssistant {
		role = genai.RoleModel
	}
	return &genai.Content{Parts: toGeminiParts(m), Role: role}
}

// toGeminiParts converts a wfcompose Message's content to genai Parts.
func toGeminiParts(m *wfcompose.Message) []*genai.Part {
	var parts []*genai.Part

	// Tool messages become FunctionResponse parts.
	if m.Role == wfcompose.RoleTool {
		var response map[string]any
		if m.Content != "" {
			if err := json.Unmarshal([]byte(m.Content), &response); err != nil {
				response = map[string]any{"output": m.Content}
			}
		}
		name := m.ToolName
		if name == "" {
			name = m.ToolCallID
		}
		parts = append(parts, genai.NewPartFromFunctionResponse(name, response))
		return parts
	}

	// Multimodal content.
	if len(m.MultiContent) > 0 {
		for _, p := range m.MultiContent {
			if part := toGeminiPart(p); part != nil {
				parts = append(parts, part)
			}
		}
	} else if m.Content != "" {
		parts = append(parts, genai.NewPartFromText(m.Content))
	}

	// Tool calls become FunctionCall parts.
	for _, tc := range m.ToolCalls {
		var args map[string]any
		if tc.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				args = map[string]any{}
			}
		}
		parts = append(parts, genai.NewPartFromFunctionCall(tc.Function.Name, args))
	}

	return parts
}

// toGeminiPart converts a wfcompose ChatMessagePart to a genai Part.
func toGeminiPart(p wfcompose.ChatMessagePart) *genai.Part {
	switch p.Type {
	case wfcompose.ChatMessagePartTypeText:
		if p.Text == "" {
			return nil
		}
		return genai.NewPartFromText(p.Text)
	case wfcompose.ChatMessagePartTypeImageURL:
		if p.ImageURL != nil && p.ImageURL.URL != "" {
			return genai.NewPartFromURI(p.ImageURL.URL, p.ImageURL.MIMEType)
		}
	case wfcompose.ChatMessagePartTypeAudioURL:
		if p.AudioURL != nil && p.AudioURL.URL != "" {
			return genai.NewPartFromURI(p.AudioURL.URL, p.AudioURL.MIMEType)
		}
	case wfcompose.ChatMessagePartTypeVideoURL:
		if p.VideoURL != nil && p.VideoURL.URL != "" {
			return genai.NewPartFromURI(p.VideoURL.URL, p.VideoURL.MIMEType)
		}
	case wfcompose.ChatMessagePartTypeFileURL:
		if p.FileURL != nil && p.FileURL.URL != "" {
			return genai.NewPartFromURI(p.FileURL.URL, p.FileURL.MIMEType)
		}
	}
	return nil
}

// fromGeminiResponse converts a genai GenerateContentResponse to a wfcompose Message,
// extracting text, tool calls, usage, and finish reason.
func fromGeminiResponse(resp *genai.GenerateContentResponse) *wfcompose.Message {
	if resp == nil || len(resp.Candidates) == 0 {
		return &wfcompose.Message{Role: wfcompose.RoleAssistant}
	}
	msg := fromGeminiCandidate(resp.Candidates[0])
	if resp.UsageMetadata != nil {
		if msg.ResponseMeta == nil {
			msg.ResponseMeta = &wfcompose.ResponseMeta{}
		}
		msg.ResponseMeta.Usage = &wfcompose.TokenUsage{
			PromptTokens:     int(resp.UsageMetadata.PromptTokenCount),
			CompletionTokens: int(resp.UsageMetadata.CandidatesTokenCount),
			TotalTokens:      int(resp.UsageMetadata.TotalTokenCount),
		}
	}
	return msg
}

// fromGeminiCandidate converts a genai Candidate to a wfcompose Message.
func fromGeminiCandidate(candidate *genai.Candidate) *wfcompose.Message {
	msg := &wfcompose.Message{Role: wfcompose.RoleAssistant}
	if candidate.Content != nil {
		for _, part := range candidate.Content.Parts {
			if part == nil {
				continue
			}
			applyGeminiPart(msg, part)
		}
	}
	msg.ResponseMeta = &wfcompose.ResponseMeta{
		FinishReason: string(candidate.FinishReason),
	}
	return msg
}

// fromGeminiStreamDelta converts a single streaming genai Part to a wfcompose
// Message delta. The finishReason is attached so callers can propagate it.
func fromGeminiStreamDelta(part *genai.Part, finishReason string) *wfcompose.Message {
	msg := &wfcompose.Message{Role: wfcompose.RoleAssistant}
	if part != nil {
		applyGeminiPart(msg, part)
	}
	msg.ResponseMeta = &wfcompose.ResponseMeta{
		FinishReason: finishReason,
	}
	return msg
}

// applyGeminiPart merges a single genai Part into the given Message.
func applyGeminiPart(msg *wfcompose.Message, part *genai.Part) {
	if part == nil {
		return
	}
	if part.Text != "" {
		if part.Thought {
			msg.ReasoningContent += part.Text
		} else {
			msg.Content += part.Text
		}
	}
	if part.FunctionCall != nil {
		args, _ := json.Marshal(part.FunctionCall.Args)
		msg.ToolCalls = append(msg.ToolCalls, wfcompose.ToolCall{
			ID:   part.FunctionCall.ID,
			Type: "function",
			Function: wfcompose.FunctionCall{
				Name:      part.FunctionCall.Name,
				Arguments: string(args),
			},
		})
	}
}

// toGeminiTools converts wfcompose ToolInfo slice to genai Tool slice.
func toGeminiTools(tools []*wfcompose.ToolInfo) []*genai.Tool {
	if len(tools) == 0 {
		return nil
	}
	decls := make([]*genai.FunctionDeclaration, 0, len(tools))
	for _, t := range tools {
		if t == nil {
			continue
		}
		fd := &genai.FunctionDeclaration{
			Name:        t.Name,
			Description: t.Desc,
		}
		if t.ParamsOneOf != nil && t.ParamsOneOf.Params() != nil {
			fd.Parameters = paramsToGeminiSchema(t.ParamsOneOf.Params())
		}
		decls = append(decls, fd)
	}
	return []*genai.Tool{{FunctionDeclarations: decls}}
}

// paramsToGeminiSchema converts a wfcompose parameter map to a genai Schema
// (object type).
func paramsToGeminiSchema(params map[string]*wfcompose.ParameterInfo) *genai.Schema {
	if len(params) == 0 {
		return nil
	}
	props := make(map[string]*genai.Schema, len(params))
	var required []string
	for name, p := range params {
		if p == nil {
			continue
		}
		props[name] = paramToGeminiSchema(p)
		if p.Required {
			required = append(required, name)
		}
	}
	return &genai.Schema{
		Type:       genai.TypeObject,
		Properties: props,
		Required:   required,
	}
}

// paramToGeminiSchema converts a single wfcompose ParameterInfo to a genai Schema.
func paramToGeminiSchema(p *wfcompose.ParameterInfo) *genai.Schema {
	if p == nil {
		return nil
	}
	schema := &genai.Schema{
		Type:        wfcomposeTypeToGemini(p.Type),
		Description: p.Desc,
	}
	if len(p.Enum) > 0 {
		schema.Enum = p.Enum
	}
	if p.Type == wfcompose.Object && len(p.SubParams) > 0 {
		schema.Properties = make(map[string]*genai.Schema, len(p.SubParams))
		for name, sp := range p.SubParams {
			if sp == nil {
				continue
			}
			schema.Properties[name] = paramToGeminiSchema(sp)
			if sp.Required {
				schema.Required = append(schema.Required, name)
			}
		}
	}
	if p.Type == wfcompose.Array && p.ElemInfo != nil {
		schema.Items = paramToGeminiSchema(p.ElemInfo)
	}
	return schema
}

// wfcomposeTypeToGemini maps a wfcompose DataType to a genai Type.
func wfcomposeTypeToGemini(t wfcompose.DataType) genai.Type {
	switch t {
	case wfcompose.Object:
		return genai.TypeObject
	case wfcompose.Number:
		return genai.TypeNumber
	case wfcompose.Integer:
		return genai.TypeInteger
	case wfcompose.String:
		return genai.TypeString
	case wfcompose.Array:
		return genai.TypeArray
	case wfcompose.Null:
		return genai.TypeNULL
	case wfcompose.Boolean:
		return genai.TypeBoolean
	default:
		return genai.TypeUnspecified
	}
}

// buildGeminiToolConfig converts a tool choice string and allowed names to a
// genai ToolConfig.
func buildGeminiToolConfig(choice string, allowedNames []string) *genai.ToolConfig {
	fc := &genai.FunctionCallingConfig{}
	switch choice {
	case "auto", "":
		fc.Mode = genai.FunctionCallingConfigModeAuto
	case "none":
		fc.Mode = genai.FunctionCallingConfigModeNone
	case "required":
		fc.Mode = genai.FunctionCallingConfigModeAny
		if len(allowedNames) > 0 {
			fc.AllowedFunctionNames = allowedNames
		}
	default:
		fc.Mode = genai.FunctionCallingConfigModeAny
		fc.AllowedFunctionNames = []string{choice}
	}
	return &genai.ToolConfig{FunctionCallingConfig: fc}
}
