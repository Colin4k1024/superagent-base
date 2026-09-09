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
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	goopenai "github.com/meguminnnnnnnnn/go-openai"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

// ChatModel is an OpenAI-compatible chat model that implements
// wfcompose.ToolCallingChatModel without any eino dependency.
type ChatModel struct {
	cli     *goopenai.Client
	conf    *ChatModelConfig
	tools   []*wfcompose.ToolInfo
}

var _ wfcompose.ToolCallingChatModel = (*ChatModel)(nil)

// NewChatModel creates a new OpenAI-compatible chat model from the given config.
func NewChatModel(ctx context.Context, conf *ChatModelConfig) (*ChatModel, error) {
	if conf == nil {
		return nil, fmt.Errorf("openai: config is nil")
	}
	clientConf := goopenai.DefaultConfig(conf.APIKey)
	if conf.BaseURL != "" {
		clientConf.BaseURL = conf.BaseURL
	}
	httpClient := &http.Client{}
	if conf.Timeout > 0 {
		httpClient.Timeout = conf.Timeout
	} else {
		httpClient.Timeout = 120 * time.Second
	}
	clientConf.HTTPClient = httpClient

	if conf.ByAzure {
		if conf.APIVersion == "" {
			return nil, fmt.Errorf("openai: api_version is required for Azure")
		}
		azureConf := goopenai.DefaultAzureConfig(conf.APIKey, conf.BaseURL)
		azureConf.APIVersion = conf.APIVersion
		azureConf.HTTPClient = httpClient
		clientConf = azureConf
	}

	cli := goopenai.NewClientWithConfig(clientConf)
	return &ChatModel{cli: cli, conf: conf}, nil
}

func (m *ChatModel) buildRequest(input []*wfcompose.Message, opts ...wfcompose.ModelOption) (goopenai.ChatCompletionRequest, error) {
	req := goopenai.ChatCompletionRequest{
		Model: m.conf.Model,
	}
	for _, msg := range input {
		req.Messages = append(req.Messages, toOpenAIMessage(msg))
	}
	merged := wfcompose.GetModelOptions(&wfcompose.ModelOptions{
		Temperature: m.conf.Temperature,
		TopP:        m.conf.TopP,
		MaxTokens:   m.conf.MaxTokens,
		Stop:        m.conf.Stop,
		Tools:       m.tools,
	}, opts...)

	if merged.Temperature != nil {
		req.Temperature = merged.Temperature
	}
	if merged.TopP != nil {
		req.TopP = *merged.TopP
	}
	if merged.MaxTokens != nil {
		req.MaxTokens = *merged.MaxTokens
	}
	if len(merged.Stop) > 0 {
		req.Stop = merged.Stop
	}
	if m.conf.PresencePenalty != nil {
		req.PresencePenalty = *m.conf.PresencePenalty
	}
	if m.conf.FrequencyPenalty != nil {
		req.FrequencyPenalty = *m.conf.FrequencyPenalty
	}
	if m.conf.MaxCompletionTokens != nil {
		req.MaxCompletionTokens = *m.conf.MaxCompletionTokens
	}
	if m.conf.ResponseFormat != nil {
		req.ResponseFormat = &goopenai.ChatCompletionResponseFormat{
			Type: goopenai.ChatCompletionResponseFormatType(m.conf.ResponseFormat.Type),
		}
	}
	if len(merged.Tools) > 0 {
		req.Tools = toOpenAITools(merged.Tools)
	}
	if merged.ToolChoice != nil {
		req.ToolChoice = *merged.ToolChoice
	}
	if m.conf.ExtraFields != nil {
		req.SetExtraFields(m.conf.ExtraFields)
	}
	return req, nil
}

func toOpenAITools(tools []*wfcompose.ToolInfo) []goopenai.Tool {
	out := make([]goopenai.Tool, 0, len(tools))
	for _, t := range tools {
		if t == nil {
			continue
		}
		fd := goopenai.FunctionDefinition{
			Name:        t.Name,
			Description: t.Desc,
		}
		if t.ParamsOneOf != nil && t.ParamsOneOf.Params() != nil {
			fd.Parameters = paramsToSchema(t.ParamsOneOf.Params())
		}
		out = append(out, goopenai.Tool{
			Type:     goopenai.ToolTypeFunction,
			Function: &fd,
		})
	}
	return out
}

func paramsToSchema(params map[string]*wfcompose.ParameterInfo) map[string]any {
	if len(params) == 0 {
		return nil
	}
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
		if p.Type == "object" && len(p.SubParams) > 0 {
			prop["properties"] = paramsToSchema(p.SubParams)
		}
		if p.Type == "array" && p.ElemInfo != nil {
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
	return map[string]any{
		"type":       "object",
		"properties": props,
		"required":   required,
	}
}

// Generate produces a single (non-streaming) response.
func (m *ChatModel) Generate(ctx context.Context, input []*wfcompose.Message, opts ...wfcompose.ModelOption) (*wfcompose.Message, error) {
	req, err := m.buildRequest(input, opts...)
	if err != nil {
		return nil, fmt.Errorf("openai: build request: %w", err)
	}
	resp, err := m.cli.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai: create chat completion: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai: no choices in response")
	}
	msg := fromOpenAIChoice(resp.Choices[0])
	if resp.Usage.TotalTokens > 0 {
		msg.ResponseMeta.Usage = &wfcompose.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}
	return msg, nil
}

// Stream produces a streaming response reader.
func (m *ChatModel) Stream(ctx context.Context, input []*wfcompose.Message, opts ...wfcompose.ModelOption) (*wfcompose.StreamReader[*wfcompose.Message], error) {
	req, err := m.buildRequest(input, opts...)
	if err != nil {
		return nil, fmt.Errorf("openai: build request: %w", err)
	}
	req.Stream = true

	stream, err := m.cli.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("openai: create chat completion stream: %w", err)
	}

	return wfcompose.NewStreamReader[*wfcompose.Message](
		func() (*wfcompose.Message, error) {
			resp, recvErr := stream.Recv()
			if recvErr != nil {
				if recvErr == io.EOF {
					return nil, io.EOF
				}
				return nil, recvErr
			}
			if len(resp.Choices) == 0 {
				return &wfcompose.Message{}, nil
			}
			return fromOpenAIStreamDelta(resp.Choices[0].Delta, resp.Choices[0].FinishReason), nil
		},
		func() { stream.Close() },
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
