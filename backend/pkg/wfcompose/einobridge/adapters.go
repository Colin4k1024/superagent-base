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

// adapters.go provides the bridge between native wfcompose model providers
// (which implement wfcompose.ToolCallingChatModel) and the eino compose/react
// framework (which expects model.ToolCallingChatModel).  The adapter converts
// schema.Message ↔ wfcompose.Message at every call site so that native
// providers can be used inside the eino orchestration layer without importing
// eino-ext.
package einobridge

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

// chatModelAdapter wraps a wfcompose.ToolCallingChatModel as a
// model.ToolCallingChatModel.
type chatModelAdapter struct {
	inner wfcompose.ToolCallingChatModel
}

// NewModelAdapter wraps a native wfcompose.ToolCallingChatModel as an eino
// model.ToolCallingChatModel so it can be used with the eino compose/react/adk
// framework.  This is the anti-corruption boundary: callers that still depend
// on eino's model interface receive a fully-compliant adapter while the actual
// provider implementation has zero eino-ext imports.
func NewModelAdapter(inner wfcompose.ToolCallingChatModel) model.ToolCallingChatModel {
	if inner == nil {
		return nil
	}
	return &chatModelAdapter{inner: inner}
}

// Generate calls the wrapped provider's Generate, converting messages.
func (a *chatModelAdapter) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	wfMsgs := schemaMsgsToWfcompose(input)
	wfOpts := modelOptsToWfcompose(opts)
	resp, err := a.inner.Generate(ctx, wfMsgs, wfOpts...)
	if err != nil {
		return nil, fmt.Errorf("model adapter: generate: %w", err)
	}
	return wfcomposeMsgToSchema(resp), nil
}

// Stream calls the wrapped provider's Stream and bridges the wfcompose
// StreamReader into a schema.StreamReader using a Pipe + goroutine pump.
func (a *chatModelAdapter) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	wfMsgs := schemaMsgsToWfcompose(input)
	wfOpts := modelOptsToWfcompose(opts)
	reader, err := a.inner.Stream(ctx, wfMsgs, wfOpts...)
	if err != nil {
		return nil, fmt.Errorf("model adapter: stream: %w", err)
	}
	sr, sw := schema.Pipe[*schema.Message](0)
	go func() {
		defer sw.Close()
		for {
			chunk, recvErr := reader.Recv()
			if recvErr != nil {
				if !errors.Is(recvErr, io.EOF) {
					sw.Send(nil, recvErr)
				}
				return
			}
			sw.Send(wfcomposeMsgToSchema(chunk), nil)
		}
	}()
	return sr, nil
}

// WithTools converts schema.ToolInfo to wfcompose.ToolInfo and delegates to the
// wrapped provider, returning a new adapter instance.
func (a *chatModelAdapter) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	wfTools := schemaToolInfosToWfcompose(tools)
	bound, err := a.inner.WithTools(wfTools)
	if err != nil {
		return nil, fmt.Errorf("model adapter: with tools: %w", err)
	}
	return &chatModelAdapter{inner: bound}, nil
}

// ---------------------------------------------------------------------------
// Message conversion: schema.Message ↔ wfcompose.Message
// ---------------------------------------------------------------------------

func schemaMsgsToWfcompose(msgs []*schema.Message) []*wfcompose.Message {
	out := make([]*wfcompose.Message, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, schemaMsgToWfcompose(m))
	}
	return out
}

func schemaMsgToWfcompose(m *schema.Message) *wfcompose.Message {
	if m == nil {
		return nil
	}
	return &wfcompose.Message{
		Role:             wfcompose.RoleType(m.Role),
		Content:          m.Content,
		MultiContent:     schemaPartsToWfcompose(m.MultiContent),
		Name:             m.Name,
		ToolCalls:        schemaToolCallsToWfcompose(m.ToolCalls),
		ToolCallID:       m.ToolCallID,
		ToolName:         m.ToolName,
		Extra:            m.Extra,
		ResponseMeta:     schemaResponseMetaToWfcompose(m.ResponseMeta),
		ReasoningContent: m.ReasoningContent,
	}
}

func wfcomposeMsgsToSchema(msgs []*wfcompose.Message) []*schema.Message {
	out := make([]*schema.Message, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, wfcomposeMsgToSchema(m))
	}
	return out
}

func wfcomposeMsgToSchema(m *wfcompose.Message) *schema.Message {
	if m == nil {
		return nil
	}
	return &schema.Message{
		Role:             schema.RoleType(m.Role),
		Content:          m.Content,
		MultiContent:     wfcomposePartsToSchema(m.MultiContent),
		Name:             m.Name,
		ToolCalls:        wfcomposeToolCallsToSchema(m.ToolCalls),
		ToolCallID:       m.ToolCallID,
		ToolName:         m.ToolName,
		Extra:            m.Extra,
		ResponseMeta:     wfcomposeResponseMetaToSchema(m.ResponseMeta),
		ReasoningContent: m.ReasoningContent,
	}
}

// ---------------------------------------------------------------------------
// MultiContent (ChatMessagePart) conversion
// ---------------------------------------------------------------------------

func schemaPartsToWfcompose(parts []schema.ChatMessagePart) []wfcompose.ChatMessagePart {
	if len(parts) == 0 {
		return nil
	}
	out := make([]wfcompose.ChatMessagePart, 0, len(parts))
	for _, p := range parts {
		wp := wfcompose.ChatMessagePart{
			Type: wfcompose.ChatMessagePartType(p.Type),
			Text: p.Text,
		}
		if p.ImageURL != nil {
			wp.ImageURL = &wfcompose.ChatMessageImageURL{
				URL:      p.ImageURL.URL,
				URI:      p.ImageURL.URI,
				Detail:   wfcompose.ImageURLDetail(p.ImageURL.Detail),
				MIMEType: p.ImageURL.MIMEType,
				Extra:    p.ImageURL.Extra,
			}
		}
		if p.AudioURL != nil {
			wp.AudioURL = &wfcompose.ChatMessageAudioURL{
				URL:      p.AudioURL.URL,
				URI:      p.AudioURL.URI,
				MIMEType: p.AudioURL.MIMEType,
				Extra:    p.AudioURL.Extra,
			}
		}
		if p.VideoURL != nil {
			wp.VideoURL = &wfcompose.ChatMessageVideoURL{
				URL:      p.VideoURL.URL,
				URI:      p.VideoURL.URI,
				MIMEType: p.VideoURL.MIMEType,
				Extra:    p.VideoURL.Extra,
			}
		}
		if p.FileURL != nil {
			wp.FileURL = &wfcompose.ChatMessageFileURL{
				URL:      p.FileURL.URL,
				URI:      p.FileURL.URI,
				MIMEType: p.FileURL.MIMEType,
				Name:     p.FileURL.Name,
				Extra:    p.FileURL.Extra,
			}
		}
		out = append(out, wp)
	}
	return out
}

func wfcomposePartsToSchema(parts []wfcompose.ChatMessagePart) []schema.ChatMessagePart {
	if len(parts) == 0 {
		return nil
	}
	out := make([]schema.ChatMessagePart, 0, len(parts))
	for _, p := range parts {
		sp := schema.ChatMessagePart{
			Type: schema.ChatMessagePartType(p.Type),
			Text: p.Text,
		}
		if p.ImageURL != nil {
			sp.ImageURL = &schema.ChatMessageImageURL{
				URL:      p.ImageURL.URL,
				URI:      p.ImageURL.URI,
				Detail:   schema.ImageURLDetail(p.ImageURL.Detail),
				MIMEType: p.ImageURL.MIMEType,
				Extra:    p.ImageURL.Extra,
			}
		}
		if p.AudioURL != nil {
			sp.AudioURL = &schema.ChatMessageAudioURL{
				URL:      p.AudioURL.URL,
				URI:      p.AudioURL.URI,
				MIMEType: p.AudioURL.MIMEType,
				Extra:    p.AudioURL.Extra,
			}
		}
		if p.VideoURL != nil {
			sp.VideoURL = &schema.ChatMessageVideoURL{
				URL:      p.VideoURL.URL,
				URI:      p.VideoURL.URI,
				MIMEType: p.VideoURL.MIMEType,
				Extra:    p.VideoURL.Extra,
			}
		}
		if p.FileURL != nil {
			sp.FileURL = &schema.ChatMessageFileURL{
				URL:      p.FileURL.URL,
				URI:      p.FileURL.URI,
				MIMEType: p.FileURL.MIMEType,
				Name:     p.FileURL.Name,
				Extra:    p.FileURL.Extra,
			}
		}
		out = append(out, sp)
	}
	return out
}

// ---------------------------------------------------------------------------
// ToolCall conversion
// ---------------------------------------------------------------------------

func schemaToolCallsToWfcompose(tcs []schema.ToolCall) []wfcompose.ToolCall {
	if len(tcs) == 0 {
		return nil
	}
	out := make([]wfcompose.ToolCall, 0, len(tcs))
	for _, tc := range tcs {
		out = append(out, wfcompose.ToolCall{
			Index: tc.Index,
			ID:    tc.ID,
			Type:  tc.Type,
			Function: wfcompose.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
			Extra: tc.Extra,
		})
	}
	return out
}

func wfcomposeToolCallsToSchema(tcs []wfcompose.ToolCall) []schema.ToolCall {
	if len(tcs) == 0 {
		return nil
	}
	out := make([]schema.ToolCall, 0, len(tcs))
	for _, tc := range tcs {
		out = append(out, schema.ToolCall{
			Index: tc.Index,
			ID:    tc.ID,
			Type:  tc.Type,
			Function: schema.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
			Extra: tc.Extra,
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// ResponseMeta conversion
// ---------------------------------------------------------------------------

func schemaResponseMetaToWfcompose(rm *schema.ResponseMeta) *wfcompose.ResponseMeta {
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

func wfcomposeResponseMetaToSchema(rm *wfcompose.ResponseMeta) *schema.ResponseMeta {
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

// ---------------------------------------------------------------------------
// ToolInfo conversion
// ---------------------------------------------------------------------------

func schemaToolInfosToWfcompose(tools []*schema.ToolInfo) []*wfcompose.ToolInfo {
	if len(tools) == 0 {
		return nil
	}
	out := make([]*wfcompose.ToolInfo, 0, len(tools))
	for _, t := range tools {
		if t == nil {
			continue
		}
		wt := &wfcompose.ToolInfo{
			Name:  t.Name,
			Desc:  t.Desc,
			Extra: t.Extra,
		}
		if t.ParamsOneOf != nil {
			wt.ParamsOneOf = schemaParamsOneOfToWfcompose(t.ParamsOneOf)
		}
		out = append(out, wt)
	}
	return out
}

// schemaParamsOneOfToWfcompose converts a schema.ParamsOneOf to
// wfcompose.ParamsOneOf via ToJSONSchema so we don't need to access the
// private params field.
func schemaParamsOneOfToWfcompose(p *schema.ParamsOneOf) *wfcompose.ParamsOneOf {
	if p == nil {
		return nil
	}
	js, err := p.ToJSONSchema()
	if err != nil || js == nil || js.Properties == nil {
		return nil
	}
	requiredSet := make(map[string]bool, len(js.Required))
	for _, r := range js.Required {
		requiredSet[r] = true
	}
	params := make(map[string]*wfcompose.ParameterInfo)
	for pair := js.Properties.Oldest(); pair != nil; pair = pair.Next() {
		v := pair.Value
		if v == nil {
			continue
		}
		pi := &wfcompose.ParameterInfo{
			Type:     wfcompose.DataType(v.Type),
			Desc:     v.Description,
			Required: requiredSet[pair.Key],
		}
		if len(v.Enum) > 0 {
			pi.Enum = make([]string, 0, len(v.Enum))
			for _, e := range v.Enum {
				pi.Enum = append(pi.Enum, fmt.Sprint(e))
			}
		}
		params[pair.Key] = pi
	}
	if len(params) == 0 {
		return nil
	}
	return wfcompose.NewParamsOneOfByParams(params)
}

// ---------------------------------------------------------------------------
// model.Option → wfcompose.ModelOption conversion
// ---------------------------------------------------------------------------

func modelOptsToWfcompose(opts []model.Option) []wfcompose.ModelOption {
	if len(opts) == 0 {
		return nil
	}
	merged := model.GetCommonOptions(&model.Options{}, opts...)
	var wfOpts []wfcompose.ModelOption
	if merged.Temperature != nil {
		wfOpts = append(wfOpts, wfcompose.WithTemperature(*merged.Temperature))
	}
	if merged.Model != nil {
		wfOpts = append(wfOpts, wfcompose.WithModel(*merged.Model))
	}
	if merged.TopP != nil {
		wfOpts = append(wfOpts, wfcompose.WithTopP(*merged.TopP))
	}
	if merged.MaxTokens != nil {
		wfOpts = append(wfOpts, wfcompose.WithMaxTokens(*merged.MaxTokens))
	}
	if len(merged.Stop) > 0 {
		wfOpts = append(wfOpts, wfcompose.WithStop(merged.Stop))
	}
	if len(merged.Tools) > 0 {
		wfOpts = append(wfOpts, wfcompose.WithTools(schemaToolInfosToWfcompose(merged.Tools)))
	}
	if merged.ToolChoice != nil {
		wfOpts = append(wfOpts, wfcompose.WithToolChoice(string(*merged.ToolChoice), merged.AllowedToolNames...))
	}
	return wfOpts
}
