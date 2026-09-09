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
	"context"
	"fmt"
	"io"
	"iter"

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
	"google.golang.org/genai"
)

// ChatModel is a Gemini chat model that implements wfcompose.ToolCallingChatModel
// without any eino dependency, using the google.golang.org/genai SDK directly.
type ChatModel struct {
	client *genai.Client
	conf   *Config
	tools  []*wfcompose.ToolInfo
}

var _ wfcompose.ToolCallingChatModel = (*ChatModel)(nil)

// NewChatModel validates the config and returns a ChatModel backed by the
// genai.Client provided in the Config.
func NewChatModel(ctx context.Context, conf *Config) (*ChatModel, error) {
	_ = ctx
	if conf == nil {
		return nil, fmt.Errorf("gemini: config is nil")
	}
	if conf.Client == nil {
		return nil, fmt.Errorf("gemini: config.Client is nil")
	}
	if conf.Model == "" {
		return nil, fmt.Errorf("gemini: config.Model is empty")
	}
	return &ChatModel{client: conf.Client, conf: conf}, nil
}

// buildRequest merges config defaults with call-time options and returns the
// model name, contents, system instruction, and GenerateContentConfig.
func (m *ChatModel) buildRequest(input []*wfcompose.Message, opts ...wfcompose.ModelOption) (
	string, []*genai.Content, *genai.Content, *genai.GenerateContentConfig,
) {
	contents, sysInstruction := toGeminiContents(input)

	merged := wfcompose.GetModelOptions(&wfcompose.ModelOptions{
		MaxTokens: m.conf.MaxTokens,
		Tools:     m.tools,
	}, opts...)

	config := &genai.GenerateContentConfig{}

	// Temperature: Config uses *float64, genai uses *float32.
	if m.conf.Temperature != nil {
		t := float32(*m.conf.Temperature)
		config.Temperature = &t
	}
	if merged.Temperature != nil {
		// Call-time option overrides config default.
		config.Temperature = merged.Temperature
	}

	// TopP.
	if m.conf.TopP != nil {
		tp := *m.conf.TopP
		config.TopP = &tp
	}
	if merged.TopP != nil {
		config.TopP = merged.TopP
	}

	// TopK: Config uses *int, genai uses *float32.
	if m.conf.TopK != nil {
		tk := float32(*m.conf.TopK)
		config.TopK = &tk
	}

	// MaxTokens.
	if merged.MaxTokens != nil {
		config.MaxOutputTokens = int32(*merged.MaxTokens)
	}

	// ThinkingConfig.
	if m.conf.ThinkingConfig != nil {
		config.ThinkingConfig = m.conf.ThinkingConfig
	}

	// Stop sequences.
	if len(merged.Stop) > 0 {
		config.StopSequences = merged.Stop
	}

	// Tools.
	if len(merged.Tools) > 0 {
		config.Tools = toGeminiTools(merged.Tools)
	}

	// Tool choice.
	if merged.ToolChoice != nil {
		config.ToolConfig = buildGeminiToolConfig(*merged.ToolChoice, merged.AllowedToolNames)
	}

	// System instruction extracted from leading system messages.
	if sysInstruction != nil {
		config.SystemInstruction = sysInstruction
	}

	model := m.conf.Model
	if merged.Model != nil {
		model = *merged.Model
	}

	return model, contents, sysInstruction, config
}

// Generate produces a single (non-streaming) response.
func (m *ChatModel) Generate(ctx context.Context, input []*wfcompose.Message, opts ...wfcompose.ModelOption) (*wfcompose.Message, error) {
	model, contents, _, config := m.buildRequest(input, opts...)

	resp, err := m.client.Models.GenerateContent(ctx, model, contents, config)
	if err != nil {
		return nil, fmt.Errorf("gemini: generate content: %w", err)
	}

	return fromGeminiResponse(resp), nil
}

// Stream produces a streaming response reader.
func (m *ChatModel) Stream(ctx context.Context, input []*wfcompose.Message, opts ...wfcompose.ModelOption) (*wfcompose.StreamReader[*wfcompose.Message], error) {
	model, contents, _, config := m.buildRequest(input, opts...)

	stream := m.client.Models.GenerateContentStream(ctx, model, contents, config)
	next, stop := iter.Pull2(stream)

	return wfcompose.NewStreamReader[*wfcompose.Message](
		func() (*wfcompose.Message, error) {
			resp, err, ok := next()
			if !ok {
				// Stream exhausted.
				return nil, io.EOF
			}
			if err != nil {
				return nil, fmt.Errorf("gemini: stream recv: %w", err)
			}
			return fromGeminiStreamChunk(resp), nil
		},
		stop,
	), nil
}

// fromGeminiStreamChunk converts a streaming GenerateContentResponse chunk to
// a wfcompose Message delta. It iterates the candidate's parts and merges them
// using fromGeminiStreamDelta, then attaches usage metadata if present.
func fromGeminiStreamChunk(resp *genai.GenerateContentResponse) *wfcompose.Message {
	msg := &wfcompose.Message{Role: wfcompose.RoleAssistant}
	if resp == nil {
		return msg
	}
	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		finishReason := string(candidate.FinishReason)
		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				if part == nil {
					continue
				}
				delta := fromGeminiStreamDelta(part, finishReason)
				msg.Content += delta.Content
				msg.ReasoningContent += delta.ReasoningContent
				msg.ToolCalls = append(msg.ToolCalls, delta.ToolCalls...)
			}
		}
		msg.ResponseMeta = &wfcompose.ResponseMeta{
			FinishReason: finishReason,
		}
	}
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

// WithTools returns a new ChatModel instance with the given tools bound.
func (m *ChatModel) WithTools(tools []*wfcompose.ToolInfo) (wfcompose.ToolCallingChatModel, error) {
	copied := make([]*wfcompose.ToolInfo, len(tools))
	copy(copied, tools)
	return &ChatModel{
		client: m.client,
		conf:   m.conf,
		tools:  copied,
	}, nil
}
