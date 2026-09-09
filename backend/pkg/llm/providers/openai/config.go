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

import "time"

// ChatModelConfig is the configuration for an OpenAI-compatible chat model.
// It works with any provider that exposes an OpenAI-compatible API:
// OpenAI, Azure OpenAI, Volcengine Ark, DeepSeek, Qwen, Ollama, etc.
type ChatModelConfig struct {
	APIKey string `json:"api_key"`
	BaseURL string `json:"base_url"`
	Model   string `json:"model"`

	Timeout     time.Duration `json:"timeout,omitempty"`

	Temperature       *float32 `json:"temperature,omitempty"`
	TopP              *float32 `json:"top_p,omitempty"`
	MaxTokens         *int     `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int   `json:"max_completion_tokens,omitempty"`
	Stop              []string `json:"stop,omitempty"`
	PresencePenalty   *float32 `json:"presence_penalty,omitempty"`
	FrequencyPenalty  *float32 `json:"frequency_penalty,omitempty"`

	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

	ByAzure     bool   `json:"by_azure,omitempty"`
	APIVersion  string `json:"api_version,omitempty"`

	ExtraFields map[string]any `json:"extra_fields,omitempty"`

	EnableThinking *bool `json:"enable_thinking,omitempty"`
}

// ResponseFormat controls the format of the model's response.
type ResponseFormat struct {
	Type       string `json:"type"`
	JSONSchema any   `json:"json_schema,omitempty"`
}

const (
	ResponseFormatTypeText       = "text"
	ResponseFormatTypeJSONObject = "json_object"
	ResponseFormatTypeJSONSchema = "json_schema"
)
