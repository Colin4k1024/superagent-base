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

// ext_model_facade re-exports cloudwego/eino-ext model provider types as
// prefixed aliases so callers avoid importing eino-ext directly (S4).
package einobridge

import (
	"context"

	arkmodel "github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/claude"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/gemini"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/model/qwen"
)


// ---------------------------------------------------------------------------
// ark model provider

// ---------------------------------------------------------------------------

type ArkChatModelConfig = arkmodel.ChatModelConfig
type ArkResponseFormat = arkmodel.ResponseFormat
type ArkChatModel = arkmodel.ChatModel

func ArkNewChatModel(ctx context.Context, config *ArkChatModelConfig) (*ArkChatModel, error) {
	return arkmodel.NewChatModel(ctx, config)
}


// ---------------------------------------------------------------------------
// claude model provider

// ---------------------------------------------------------------------------

type ClaudeConfig = claude.Config
type ClaudeThinking = claude.Thinking
type ClaudeChatModel = claude.ChatModel

func ClaudeNewChatModel(ctx context.Context, config *ClaudeConfig) (*ClaudeChatModel, error) {
	return claude.NewChatModel(ctx, config)
}


// ---------------------------------------------------------------------------
// deepseek model provider

// ---------------------------------------------------------------------------

type DeepSeekChatModelConfig = deepseek.ChatModelConfig
type DeepSeekChatModel = deepseek.ChatModel
type DeepSeekResponseFormatType = deepseek.ResponseFormatType

const (
	DeepSeekResponseFormatTypeText         = deepseek.ResponseFormatTypeText
	DeepSeekResponseFormatTypeJSONObject   = deepseek.ResponseFormatTypeJSONObject
)

func DeepSeekNewChatModel(ctx context.Context, config *DeepSeekChatModelConfig) (*DeepSeekChatModel, error) {
	return deepseek.NewChatModel(ctx, config)
}


// ---------------------------------------------------------------------------
// gemini model provider

// ---------------------------------------------------------------------------

type GeminiModelConfig = gemini.Config
type GeminiChatModel = gemini.ChatModel

func GeminiNewChatModel(ctx context.Context, config *GeminiModelConfig) (*GeminiChatModel, error) {
	return gemini.NewChatModel(ctx, config)
}


// ---------------------------------------------------------------------------
// ollama model provider

// ---------------------------------------------------------------------------

type OllamaChatModelConfig = ollama.ChatModelConfig
type OllamaChatModel = ollama.ChatModel

func OllamaNewChatModel(ctx context.Context, config *OllamaChatModelConfig) (*OllamaChatModel, error) {
	return ollama.NewChatModel(ctx, config)
}


// ---------------------------------------------------------------------------
// openai model provider

// ---------------------------------------------------------------------------

type OpenAIChatModelConfig = openai.ChatModelConfig
type OpenAIChatCompletionResponseFormat = openai.ChatCompletionResponseFormat
type OpenAIChatModel = openai.ChatModel

const (
	OpenAIChatCompletionResponseFormatTypeText         = openai.ChatCompletionResponseFormatTypeText
	OpenAIChatCompletionResponseFormatTypeJSONObject   = openai.ChatCompletionResponseFormatTypeJSONObject
)

func OpenAINewChatModel(ctx context.Context, config *OpenAIChatModelConfig) (*OpenAIChatModel, error) {
	return openai.NewChatModel(ctx, config)
}


// ---------------------------------------------------------------------------
// qwen model provider

// ---------------------------------------------------------------------------

type QwenChatModelConfig = qwen.ChatModelConfig
type QwenChatModel = qwen.ChatModel

func QwenNewChatModel(ctx context.Context, config *QwenChatModelConfig) (*QwenChatModel, error) {
	return qwen.NewChatModel(ctx, config)
}
