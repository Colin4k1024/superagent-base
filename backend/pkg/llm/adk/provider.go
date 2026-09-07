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

package adk

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/model/gemini"
	"google.golang.org/adk/v2/model/openaimodel"
	"google.golang.org/genai"

	aclllm "github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

// Provider is the ADK Go-backed implementation of aclllm.ModelProvider.
// It creates model.LLM instances via ADK Go's native provider constructors.
type Provider struct {
	protocol string
	create   func(ctx context.Context, cfg aclllm.ModelConfig) (model.LLM, error)
}

// Protocol returns the provider protocol name.
func (p *Provider) Protocol() string { return p.protocol }

// Create builds a ChatModel from the given config.
func (p *Provider) Create(ctx context.Context, cfg aclllm.ModelConfig) (aclllm.ChatModel, error) {
	llm, err := p.create(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("adk provider %s: create model: %w", p.protocol, err)
	}
	return NewChatModelAdapter(llm, cfg.ModelID), nil
}

// NewDefaultRegistry creates a ModelProviderRegistry with ADK Go native
// providers (OpenAI + Gemini). Other providers (Claude, DeepSeek, Qwen,
// Ollama, Ark) can be registered separately or bridged via OpenAI-compatible
// endpoints.
func NewDefaultRegistry() *aclllm.ModelProviderRegistry {
	reg := aclllm.NewModelProviderRegistry()

	// OpenAI: native ADK Go provider, supports OpenAI-compatible endpoints.
	reg.Register(&Provider{
		protocol: "openai",
		create: func(ctx context.Context, cfg aclllm.ModelConfig) (model.LLM, error) {
			return openaimodel.NewModel(ctx, cfg.ModelID, &openaimodel.ClientConfig{
				APIKey:  cfg.APIKey,
				BaseURL: cfg.BaseURL,
			})
		},
	})

	// Qwen: OpenAI-compatible API, use openaimodel with custom BaseURL.
	reg.Register(&Provider{
		protocol: "qwen",
		create: func(ctx context.Context, cfg aclllm.ModelConfig) (model.LLM, error) {
			if cfg.BaseURL == "" {
				return nil, fmt.Errorf("qwen protocol requires base_url")
			}
			return openaimodel.NewModel(ctx, cfg.ModelID, &openaimodel.ClientConfig{
				APIKey:  cfg.APIKey,
				BaseURL: cfg.BaseURL,
			})
		},
	})

	// DeepSeek: OpenAI-compatible API.
	reg.Register(&Provider{
		protocol: "deepseek",
		create: func(ctx context.Context, cfg aclllm.ModelConfig) (model.LLM, error) {
			baseURL := cfg.BaseURL
			if baseURL == "" {
				baseURL = "https://api.deepseek.com/v1"
			}
			return openaimodel.NewModel(ctx, cfg.ModelID, &openaimodel.ClientConfig{
				APIKey:  cfg.APIKey,
				BaseURL: baseURL,
			})
		},
	})

	// Ollama: OpenAI-compatible API (no API key needed).
	reg.Register(&Provider{
		protocol: "ollama",
		create: func(ctx context.Context, cfg aclllm.ModelConfig) (model.LLM, error) {
			baseURL := cfg.BaseURL
			if baseURL == "" {
				baseURL = "http://localhost:11434/v1"
			}
			baseURL = strings.TrimSuffix(baseURL, "/v1") + "/v1"
			return openaimodel.NewModel(ctx, cfg.ModelID, &openaimodel.ClientConfig{
				BaseURL: baseURL,
			})
		},
	})

	// Gemini: native ADK Go provider.
	reg.Register(&Provider{
		protocol: "gemini",
		create: func(ctx context.Context, cfg aclllm.ModelConfig) (model.LLM, error) {
			clientCfg := &genai.ClientConfig{
				APIKey: cfg.APIKey,
			}
			if cfg.BaseURL != "" {
				clientCfg.HTTPOptions = genai.HTTPOptions{BaseURL: cfg.BaseURL}
			}
			return gemini.NewModel(ctx, cfg.ModelID, clientCfg)
		},
	})

	// Claude: OpenAI-compatible proxy (e.g. LiteLLM).
	reg.Register(&Provider{
		protocol: "claude",
		create: func(ctx context.Context, cfg aclllm.ModelConfig) (model.LLM, error) {
			if cfg.BaseURL == "" {
				return nil, fmt.Errorf("claude protocol requires base_url pointing to an OpenAI-compatible proxy")
			}
			return openaimodel.NewModel(ctx, cfg.ModelID, &openaimodel.ClientConfig{
				APIKey:  cfg.APIKey,
				BaseURL: cfg.BaseURL,
			})
		},
	})

	// Ark (火山方舟): OpenAI-compatible API.
	reg.Register(&Provider{
		protocol: "ark",
		create: func(ctx context.Context, cfg aclllm.ModelConfig) (model.LLM, error) {
			if cfg.BaseURL == "" {
				return nil, fmt.Errorf("ark protocol requires base_url")
			}
			return openaimodel.NewModel(ctx, cfg.ModelID, &openaimodel.ClientConfig{
				APIKey:  cfg.APIKey,
				BaseURL: cfg.BaseURL,
			})
		},
	})

	return reg
}
