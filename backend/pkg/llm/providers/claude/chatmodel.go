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
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

// ChatModel is a Claude chat model that implements wfcompose.ToolCallingChatModel
// using the Anthropic SDK directly (no eino dependency).
type ChatModel struct {
	cli   *anthropic.Client
	conf  *Config
	tools []*wfcompose.ToolInfo
}

var _ wfcompose.ToolCallingChatModel = (*ChatModel)(nil)

// NewChatModel creates a new Claude chat model from the given config.
func NewChatModel(ctx context.Context, conf *Config) (*ChatModel, error) {
	if conf == nil {
		return nil, fmt.Errorf("claude: config is nil")
	}
	if conf.Model == "" {
		return nil, fmt.Errorf("claude: model is required")
	}
	maxTokens := conf.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	opts := []option.RequestOption{
		option.WithAPIKey(conf.APIKey),
		option.WithHTTPClient(&http.Client{Timeout: 120 * time.Second}),
	}
	if conf.BaseURL != nil && *conf.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(*conf.BaseURL))
	}

	cli := anthropic.NewClient(opts...)
	return &ChatModel{
		cli:  &cli,
		conf: conf,
	}, nil
}

func (m *ChatModel) buildParams(input []*wfcompose.Message, opts ...wfcompose.ModelOption) (anthropic.MessageNewParams, error) {
	maxTokens := m.conf.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	merged := wfcompose.GetModelOptions(&wfcompose.ModelOptions{
		Temperature: m.conf.Temperature,
		TopP:        topPPtr(m.conf.TopP),
		MaxTokens:   &maxTokens,
		Tools:       m.tools,
	}, opts...)

	finalMaxTokens := int64(maxTokens)
	if merged.MaxTokens != nil && *merged.MaxTokens > 0 {
		finalMaxTokens = int64(*merged.MaxTokens)
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(m.conf.Model),
		MaxTokens: finalMaxTokens,
	}

	if merged.Temperature != nil {
		params.Temperature = param.NewOpt(float64(*merged.Temperature))
	}
	if merged.TopP != nil {
		params.TopP = param.NewOpt(float64(*merged.TopP))
	}
	if m.conf.TopK > 0 {
		params.TopK = param.NewOpt(int64(m.conf.TopK))
	}
	if len(merged.Stop) > 0 {
		params.StopSequences = merged.Stop
	}
	if merged.Model != nil && *merged.Model != "" {
		params.Model = anthropic.Model(*merged.Model)
	}

	systemText, convMsgs := splitSystemMessages(input)
	if systemText != "" {
		params.System = []anthropic.TextBlockParam{{Text: systemText}}
	}
	params.Messages = toAnthropicMessages(convMsgs)

	if len(merged.Tools) > 0 {
		params.Tools = toAnthropicTools(merged.Tools)
	}
	if merged.ToolChoice != nil {
		params.ToolChoice = toToolChoice(merged.ToolChoice, merged.AllowedToolNames)
	}

	if m.conf.Thinking != nil && m.conf.Thinking.Enable {
		budget := finalMaxTokens
		if budget > 10000 {
			budget = 10000
		}
		params.Thinking = anthropic.ThinkingConfigParamOfEnabled(budget)
	}

	return params, nil
}

func topPPtr(v float32) *float32 {
	if v == 0 {
		return nil
	}
	return &v
}

func toToolChoice(choice *string, allowedNames []string) anthropic.ToolChoiceUnionParam {
	switch *choice {
	case "none":
		return anthropic.ToolChoiceUnionParam{OfAuto: &anthropic.ToolChoiceAutoParam{}}
	case "any", "required":
		return anthropic.ToolChoiceUnionParam{OfAny: &anthropic.ToolChoiceAnyParam{}}
	case "auto":
		return anthropic.ToolChoiceUnionParam{OfAuto: &anthropic.ToolChoiceAutoParam{}}
	default:
		if len(allowedNames) > 0 {
			return anthropic.ToolChoiceUnionParam{OfTool: &anthropic.ToolChoiceToolParam{Name: allowedNames[0]}}
		}
		return anthropic.ToolChoiceUnionParam{OfTool: &anthropic.ToolChoiceToolParam{Name: *choice}}
	}
}

// Generate produces a single (non-streaming) response.
func (m *ChatModel) Generate(ctx context.Context, input []*wfcompose.Message, opts ...wfcompose.ModelOption) (*wfcompose.Message, error) {
	params, err := m.buildParams(input, opts...)
	if err != nil {
		return nil, fmt.Errorf("claude: build params: %w", err)
	}
	resp, err := m.cli.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("claude: create message: %w", err)
	}
	return fromAnthropicMessage(resp), nil
}

// Stream produces a streaming response reader.
func (m *ChatModel) Stream(ctx context.Context, input []*wfcompose.Message, opts ...wfcompose.ModelOption) (*wfcompose.StreamReader[*wfcompose.Message], error) {
	params, err := m.buildParams(input, opts...)
	if err != nil {
		return nil, fmt.Errorf("claude: build params: %w", err)
	}

	stream := m.cli.Messages.NewStreaming(ctx, params)

	return wfcompose.NewStreamReader[*wfcompose.Message](
		func() (*wfcompose.Message, error) {
			for stream.Next() {
				msg := fromAnthropicStreamDelta(stream.Current())
				if msg == nil {
					continue
				}
				return msg, nil
			}
			if err := stream.Err(); err != nil {
				return nil, err
			}
			return nil, io.EOF
		},
		func() { _ = stream.Close() },
	), nil
}

// WithTools returns a new ChatModel instance with the given tools bound.
func (m *ChatModel) WithTools(tools []*wfcompose.ToolInfo) (wfcompose.ToolCallingChatModel, error) {
	copied := make([]*wfcompose.ToolInfo, len(tools))
	copy(copied, tools)
	return &ChatModel{
		cli:   m.cli,
		conf:  m.conf,
		tools: copied,
	}, nil
}
