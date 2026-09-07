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

package llm

import "context"

// ChatModel is the framework-agnostic chat model interface.
// It abstracts eino's components/model.ToolCallingChatModel
// and Google ADK Go's model.LLM into a single contract.
//
// Implementations must be safe for concurrent use.
type ChatModel interface {
	// ID returns the model identifier (e.g. "gpt-4o", "doubao-pro-4k").
	ID() string

	// Generate produces a single (non-streaming) response.
	Generate(ctx context.Context, msgs []*Message) (*Message, error)

	// Stream produces a streaming response reader.
	Stream(ctx context.Context, msgs []*Message) (*StreamReader, error)

	// BindTools associates tool definitions with the model so the model
	// can emit tool calls during generation. Returns a new ChatModel
	// instance; the original is unmodified.
	BindTools(tools []Tool) (ChatModel, error)
}

// ModelConfig holds provider connection parameters.
// It is intentionally generic so different providers (OpenAI, Ark, Claude,
// DeepSeek, Ollama, Gemini, Qwen) can be configured uniformly.
type ModelConfig struct {
	Protocol string // provider protocol: openai, claude, ark, deepseek, ollama, gemini, qwen
	BaseURL  string // API base URL
	APIKey   string // API key
	ModelID  string // model identifier
}

// ModelProvider is a factory that creates ChatModel instances.
// Each eino-ext or ADK model adapter registers a ModelProvider.
type ModelProvider interface {
	// Protocol returns the provider protocol this factory handles.
	Protocol() string

	// Create builds a ChatModel from the given config.
	Create(ctx context.Context, cfg ModelConfig) (ChatModel, error)
}

// ModelProviderRegistry maps protocol names to ModelProvider factories.
type ModelProviderRegistry struct {
	providers map[string]ModelProvider
}

// NewModelProviderRegistry creates an empty registry.
func NewModelProviderRegistry() *ModelProviderRegistry {
	return &ModelProviderRegistry{providers: make(map[string]ModelProvider)}
}

// Register adds a provider for the given protocol.
func (r *ModelProviderRegistry) Register(p ModelProvider) {
	r.providers[p.Protocol()] = p
}

// Create resolves a provider by protocol and builds a ChatModel.
func (r *ModelProviderRegistry) Create(ctx context.Context, cfg ModelConfig) (ChatModel, error) {
	p, ok := r.providers[cfg.Protocol]
	if !ok {
		return nil, &UnsupportedProtocolError{Protocol: cfg.Protocol}
	}
	return p.Create(ctx, cfg)
}

// UnsupportedProtocolError is returned when no provider is registered
// for the requested protocol.
type UnsupportedProtocolError struct {
	Protocol string
}

func (e *UnsupportedProtocolError) Error() string {
	return "llm: unsupported model protocol: " + e.Protocol
}
