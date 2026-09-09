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

// ext_model_facade defines native config types and constructor functions for
// all model providers.  Each constructor creates a native provider (which
// implements wfcompose.ToolCallingChatModel with zero eino-ext imports) and
// wraps it with NewModelAdapter so the result satisfies eino's
// model.ToolCallingChatModel interface expected by the compose/react/adk layer.
//
// This file has ZERO cloudwego/eino-ext imports.
package einobridge

import (
	"context"

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"

	claudeprovider "github.com/superagent-ai/superagent-base/backend/pkg/llm/providers/claude"
	geminiprovider "github.com/superagent-ai/superagent-base/backend/pkg/llm/providers/gemini"
	openaiprovider "github.com/superagent-ai/superagent-base/backend/pkg/llm/providers/openai"
	volcmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

// ---------------------------------------------------------------------------
// OpenAI model provider
// ---------------------------------------------------------------------------

type OpenAIChatModelConfig = openaiprovider.ChatModelConfig
type OpenAIChatCompletionResponseFormat = openaiprovider.ResponseFormat

const (
	OpenAIChatCompletionResponseFormatTypeText       = openaiprovider.ResponseFormatTypeText
	OpenAIChatCompletionResponseFormatTypeJSONObject = openaiprovider.ResponseFormatTypeJSONObject
	OpenAIChatCompletionResponseFormatTypeJSONSchema = openaiprovider.ResponseFormatTypeJSONSchema
)

func OpenAINewChatModel(ctx context.Context, config *OpenAIChatModelConfig) (wfcompose.ToolCallingChatModel, error) {
	provider, err := openaiprovider.NewChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// ---------------------------------------------------------------------------
// Ark model provider (OpenAI-compatible, uses volcengine SDK types)
// ---------------------------------------------------------------------------

type ArkChatModelConfig struct {
	APIKey              string                 `json:"api_key"`
	BaseURL             string                 `json:"base_url"`
	Region              string                 `json:"region"`
	Model               string                 `json:"model"`
	MaxTokens           *int                   `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int                   `json:"max_completion_tokens,omitempty"`
	Temperature         *float32               `json:"temperature,omitempty"`
	TopP                *float32               `json:"top_p,omitempty"`
	Stop                []string               `json:"stop,omitempty"`
	FrequencyPenalty    *float32               `json:"frequency_penalty,omitempty"`
	PresencePenalty     *float32               `json:"presence_penalty,omitempty"`
	ResponseFormat      *ArkResponseFormat     `json:"response_format,omitempty"`
	Thinking            *volcmodel.Thinking     `json:"thinking,omitempty"`
}

type ArkResponseFormat struct {
	Type       volcmodel.ResponseFormatType                `json:"type"`
	JSONSchema *volcmodel.ResponseFormatJSONSchemaJSONSchemaParam `json:"json_schema,omitempty"`
}

func ArkNewChatModel(ctx context.Context, config *ArkChatModelConfig) (wfcompose.ToolCallingChatModel, error) {
	openaiConf := &openaiprovider.ChatModelConfig{
		APIKey:            config.APIKey,
		BaseURL:           config.BaseURL,
		Model:             config.Model,
		Temperature:       config.Temperature,
		TopP:              config.TopP,
		MaxTokens:         config.MaxTokens,
		FrequencyPenalty:  config.FrequencyPenalty,
		PresencePenalty:   config.PresencePenalty,
	}
	if config.ResponseFormat != nil {
		openaiConf.ResponseFormat = &openaiprovider.ResponseFormat{
			Type: string(config.ResponseFormat.Type),
		}
	}
	if config.Thinking != nil {
		enabled := config.Thinking.Type != volcmodel.ThinkingTypeDisabled
		openaiConf.EnableThinking = &enabled
	}
	if config.Region != "" {
		openaiConf.ExtraFields = map[string]any{"region": config.Region}
	}
	provider, err := openaiprovider.NewChatModel(ctx, openaiConf)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// ---------------------------------------------------------------------------
// Claude model provider
// ---------------------------------------------------------------------------

type ClaudeConfig = claudeprovider.Config
type ClaudeThinking = claudeprovider.ThinkingConfig

func ClaudeNewChatModel(ctx context.Context, config *ClaudeConfig) (wfcompose.ToolCallingChatModel, error) {
	provider, err := claudeprovider.NewChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// ---------------------------------------------------------------------------
// DeepSeek model provider (OpenAI-compatible)
// ---------------------------------------------------------------------------

type DeepSeekChatModelConfig struct {
	APIKey              string                   `json:"api_key"`
	BaseURL             string                   `json:"base_url"`
	Model               string                   `json:"model"`
	MaxTokens           int                      `json:"max_tokens,omitempty"`
	Temperature         float32                  `json:"temperature,omitempty"`
	TopP                float32                  `json:"top_p,omitempty"`
	Stop                []string                 `json:"stop,omitempty"`
	FrequencyPenalty    float32                  `json:"frequency_penalty,omitempty"`
	PresencePenalty     float32                  `json:"presence_penalty,omitempty"`
	ResponseFormatType  DeepSeekResponseFormatType `json:"response_format_type,omitempty"`
}

type DeepSeekResponseFormatType string

const (
	DeepSeekResponseFormatTypeText       DeepSeekResponseFormatType = "text"
	DeepSeekResponseFormatTypeJSONObject DeepSeekResponseFormatType = "json_object"
)

func DeepSeekNewChatModel(ctx context.Context, config *DeepSeekChatModelConfig) (wfcompose.ToolCallingChatModel, error) {
	openaiConf := &openaiprovider.ChatModelConfig{
		APIKey:           config.APIKey,
		BaseURL:          config.BaseURL,
		Model:            config.Model,
		FrequencyPenalty: ptrOfFloat32(config.FrequencyPenalty),
		PresencePenalty:  ptrOfFloat32(config.PresencePenalty),
	}
	if config.Temperature != 0 {
		openaiConf.Temperature = ptrOfFloat32(config.Temperature)
	}
	if config.MaxTokens != 0 {
		openaiConf.MaxTokens = ptrOfInt(config.MaxTokens)
	}
	if config.TopP != 0 {
		openaiConf.TopP = ptrOfFloat32(config.TopP)
	}
	if config.ResponseFormatType != "" {
		openaiConf.ResponseFormat = &openaiprovider.ResponseFormat{
			Type: string(config.ResponseFormatType),
		}
	}
	provider, err := openaiprovider.NewChatModel(ctx, openaiConf)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// ---------------------------------------------------------------------------
// Gemini model provider
// ---------------------------------------------------------------------------

type GeminiModelConfig = geminiprovider.Config

func GeminiNewChatModel(ctx context.Context, config *GeminiModelConfig) (wfcompose.ToolCallingChatModel, error) {
	provider, err := geminiprovider.NewChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// ---------------------------------------------------------------------------
// Ollama model provider (OpenAI-compatible)
// ---------------------------------------------------------------------------

type OllamaChatModelConfig struct {
	BaseURL  string             `json:"base_url"`
	Model    string            `json:"model"`
	Options  *OllamaOptions    `json:"options,omitempty"`
	Thinking *OllamaThinkValue `json:"thinking,omitempty"`
}

type OllamaOptions struct {
	Temperature      float32 `json:"temperature,omitempty"`
	TopP             float32 `json:"top_p,omitempty"`
	TopK             int     `json:"top_k,omitempty"`
	FrequencyPenalty float32 `json:"frequency_penalty,omitempty"`
	PresencePenalty  float32 `json:"presence_penalty,omitempty"`
}

type OllamaThinkValue struct {
	Value *bool `json:"value,omitempty"`
}

func OllamaNewChatModel(ctx context.Context, config *OllamaChatModelConfig) (wfcompose.ToolCallingChatModel, error) {
	openaiConf := &openaiprovider.ChatModelConfig{
		APIKey:  "ollama",
		BaseURL: config.BaseURL,
		Model:   config.Model,
	}
	if config.Options != nil {
		if config.Options.Temperature != 0 {
			openaiConf.Temperature = ptrOfFloat32(config.Options.Temperature)
		}
		if config.Options.TopP != 0 {
			openaiConf.TopP = ptrOfFloat32(config.Options.TopP)
		}
		if config.Options.FrequencyPenalty != 0 {
			openaiConf.FrequencyPenalty = ptrOfFloat32(config.Options.FrequencyPenalty)
		}
		if config.Options.PresencePenalty != 0 {
			openaiConf.PresencePenalty = ptrOfFloat32(config.Options.PresencePenalty)
		}
	}
	if config.Thinking != nil && config.Thinking.Value != nil {
		openaiConf.EnableThinking = config.Thinking.Value
	}
	provider, err := openaiprovider.NewChatModel(ctx, openaiConf)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// ---------------------------------------------------------------------------
// Qwen model provider (OpenAI-compatible)
// ---------------------------------------------------------------------------

type QwenChatModelConfig struct {
	APIKey            string                              `json:"api_key"`
	BaseURL           string                              `json:"base_url"`
	Model             string                              `json:"model"`
	MaxTokens         *int                                `json:"max_tokens,omitempty"`
	Temperature       *float32                            `json:"temperature,omitempty"`
	TopP              *float32                            `json:"top_p,omitempty"`
	Stop              []string                            `json:"stop,omitempty"`
	FrequencyPenalty  *float32                            `json:"frequency_penalty,omitempty"`
	PresencePenalty   *float32                            `json:"presence_penalty,omitempty"`
	ResponseFormat    *OpenAIChatCompletionResponseFormat  `json:"response_format,omitempty"`
	EnableThinking   *bool                               `json:"enable_thinking,omitempty"`
}

func QwenNewChatModel(ctx context.Context, config *QwenChatModelConfig) (wfcompose.ToolCallingChatModel, error) {
	openaiConf := &openaiprovider.ChatModelConfig{
		APIKey:           config.APIKey,
		BaseURL:          config.BaseURL,
		Model:            config.Model,
		Temperature:      config.Temperature,
		TopP:             config.TopP,
		MaxTokens:        config.MaxTokens,
		FrequencyPenalty: config.FrequencyPenalty,
		PresencePenalty:  config.PresencePenalty,
		EnableThinking:   config.EnableThinking,
	}
	if config.ResponseFormat != nil {
		openaiConf.ResponseFormat = config.ResponseFormat
	}
	provider, err := openaiprovider.NewChatModel(ctx, openaiConf)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func ptrOfFloat32(v float32) *float32 { return &v }
func ptrOfInt(v int) *int             { return &v }
