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

package eino

import (
	"context"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
	"fmt"
	"strings"

	einoark "github.com/cloudwego/eino-ext/components/model/ark"
	einoclaude "github.com/cloudwego/eino-ext/components/model/claude"
	einodeepseek "github.com/cloudwego/eino-ext/components/model/deepseek"
	einoollama "github.com/cloudwego/eino-ext/components/model/ollama"
	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"

	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

// Provider is the eino-backed implementation of llm.ModelProvider.
// It delegates to eino-ext's provider-specific constructors.
type Provider struct {
	protocol string
	create   func(ctx context.Context, cfg llm.ModelConfig) (einobridge.ToolCallingChatModel, error)
}

// Protocol returns the provider protocol name.
func (p *Provider) Protocol() string { return p.protocol }

// Create builds a ChatModel from the given config.
func (p *Provider) Create(ctx context.Context, cfg llm.ModelConfig) (llm.ChatModel, error) {
	chatModel, err := p.create(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("eino provider %s: create model: %w", p.protocol, err)
	}
	return NewChatModelAdapter(chatModel, cfg.ModelID), nil
}

// NewDefaultRegistry creates a ModelProviderRegistry with all 7 eino-ext
// providers pre-registered (openai, claude, ark, deepseek, ollama, gemini, qwen).
func NewDefaultRegistry(defaultBaseURL string) *llm.ModelProviderRegistry {
	reg := llm.NewModelProviderRegistry()

	reg.Register(&Provider{
		protocol: "openai",
		create: func(ctx context.Context, cfg llm.ModelConfig) (einobridge.ToolCallingChatModel, error) {
			return einoopenai.NewChatModel(ctx, &einoopenai.ChatModelConfig{
				BaseURL: cfg.BaseURL,
				APIKey:  cfg.APIKey,
				Model:   cfg.ModelID,
			})
		},
	})

	reg.Register(&Provider{
		protocol: "qwen",
		create: func(ctx context.Context, cfg llm.ModelConfig) (einobridge.ToolCallingChatModel, error) {
			return einoopenai.NewChatModel(ctx, &einoopenai.ChatModelConfig{
				BaseURL: cfg.BaseURL,
				APIKey:  cfg.APIKey,
				Model:   cfg.ModelID,
			})
		},
	})

	reg.Register(&Provider{
		protocol: "claude",
		create: func(ctx context.Context, cfg llm.ModelConfig) (einobridge.ToolCallingChatModel, error) {
			return einoclaude.NewChatModel(ctx, &einoclaude.Config{
				BaseURL: &cfg.BaseURL,
				APIKey:  cfg.APIKey,
				Model:   cfg.ModelID,
			})
		},
	})

	reg.Register(&Provider{
		protocol: "ark",
		create: func(ctx context.Context, cfg llm.ModelConfig) (einobridge.ToolCallingChatModel, error) {
			return einoark.NewChatModel(ctx, &einoark.ChatModelConfig{
				BaseURL: cfg.BaseURL,
				APIKey:  cfg.APIKey,
				Model:   cfg.ModelID,
			})
		},
	})

	reg.Register(&Provider{
		protocol: "deepseek",
		create: func(ctx context.Context, cfg llm.ModelConfig) (einobridge.ToolCallingChatModel, error) {
			baseURL := cfg.BaseURL
			if baseURL == "" || baseURL == defaultBaseURL {
				baseURL = "https://api.deepseek.com/v1"
			}
			return einodeepseek.NewChatModel(ctx, &einodeepseek.ChatModelConfig{
				BaseURL: baseURL,
				APIKey:  cfg.APIKey,
				Model:   cfg.ModelID,
			})
		},
	})

	reg.Register(&Provider{
		protocol: "ollama",
		create: func(ctx context.Context, cfg llm.ModelConfig) (einobridge.ToolCallingChatModel, error) {
			ollamaURL := cfg.BaseURL
			if ollamaURL == "" || ollamaURL == defaultBaseURL {
				ollamaURL = "http://localhost:11434"
			}
			ollamaURL = strings.TrimSuffix(ollamaURL, "/v1")
			return einoollama.NewChatModel(ctx, &einoollama.ChatModelConfig{
				BaseURL: ollamaURL,
				Model:   cfg.ModelID,
			})
		},
	})

	reg.Register(&Provider{
		protocol: "gemini",
		create: func(ctx context.Context, cfg llm.ModelConfig) (einobridge.ToolCallingChatModel, error) {
			// Gemini requires an OpenAI-compatible proxy (e.g. LiteLLM).
			if cfg.BaseURL == "" {
				return nil, fmt.Errorf("gemini protocol requires base_url pointing to an OpenAI-compatible proxy")
			}
			return einoopenai.NewChatModel(ctx, &einoopenai.ChatModelConfig{
				BaseURL: cfg.BaseURL,
				APIKey:  cfg.APIKey,
				Model:   cfg.ModelID,
			})
		},
	})

	return reg
}
