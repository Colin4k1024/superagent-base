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

package agentdef

import (
	"context"
	"fmt"
	"strings"

	einoark "github.com/cloudwego/eino-ext/components/model/ark"
	einoclaude "github.com/cloudwego/eino-ext/components/model/claude"
	einodeepseek "github.com/cloudwego/eino-ext/components/model/deepseek"
	einoollama "github.com/cloudwego/eino-ext/components/model/ollama"
	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// createChatModel creates an Eino ChatModel for the given protocol.
func (b *AgentBuilder) createChatModel(ctx context.Context, protocol, baseURL, apiKey, modelID string) (model.ToolCallingChatModel, error) {
	switch strings.ToLower(protocol) {
	case "claude":
		return einoclaude.NewChatModel(ctx, &einoclaude.Config{
			BaseURL: &baseURL,
			APIKey:  apiKey,
			Model:   modelID,
		})

	case "gemini":
		if baseURL != "" {
			return einoopenai.NewChatModel(ctx, &einoopenai.ChatModelConfig{
				BaseURL: baseURL,
				APIKey:  apiKey,
				Model:   modelID,
			})
		}
		return nil, fmt.Errorf("gemini protocol requires base_url pointing to an OpenAI-compatible proxy (e.g. LiteLLM)")

	case "ark":
		return einoark.NewChatModel(ctx, &einoark.ChatModelConfig{
			BaseURL: baseURL,
			APIKey:  apiKey,
			Model:   modelID,
		})

	case "deepseek":
		if baseURL == "" || baseURL == b.modelConfig.BaseURL {
			baseURL = "https://api.deepseek.com/v1"
		}
		return einodeepseek.NewChatModel(ctx, &einodeepseek.ChatModelConfig{
			BaseURL: baseURL,
			APIKey:  apiKey,
			Model:   modelID,
		})

	case "ollama":
		ollamaURL := baseURL
		if ollamaURL == "" || ollamaURL == b.modelConfig.BaseURL {
			ollamaURL = "http://localhost:11434"
		}
		ollamaURL = strings.TrimSuffix(ollamaURL, "/v1")
		return einoollama.NewChatModel(ctx, &einoollama.ChatModelConfig{
			BaseURL: ollamaURL,
			Model:   modelID,
		})

	case "openai", "qwen", "":
		return einoopenai.NewChatModel(ctx, &einoopenai.ChatModelConfig{
			BaseURL: baseURL,
			APIKey:  apiKey,
			Model:   modelID,
		})

	default:
		return nil, fmt.Errorf("unsupported model protocol %q (supported: openai, claude, deepseek, gemini, ark, ollama, qwen)", protocol)
	}
}
