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

// ext_embedding_facade re-exports cloudwego/eino-ext embedding provider
// types as prefixed aliases so callers avoid importing eino-ext directly (S4).
package einobridge

import (
	"context"

	arkembed "github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino-ext/components/embedding/gemini"
	"github.com/cloudwego/eino-ext/components/embedding/ollama"
	"github.com/cloudwego/eino-ext/components/embedding/openai"
)

// ---------------------------------------------------------------------------
// ark embedding provider
// ---------------------------------------------------------------------------

type ArkAPIType = arkembed.APIType
type ArkEmbeddingConfig = arkembed.EmbeddingConfig
type ArkEmbedder = arkembed.Embedder

const (
	ArkAPITypeText        ArkAPIType = arkembed.APITypeText
	ArkAPITypeMultiModal  ArkAPIType = arkembed.APITypeMultiModal
)

func ArkNewEmbedder(ctx context.Context, config *ArkEmbeddingConfig) (*ArkEmbedder, error) {
	return arkembed.NewEmbedder(ctx, config)
}

// ---------------------------------------------------------------------------
// gemini embedding provider
// ---------------------------------------------------------------------------

type GeminiEmbeddingConfig = gemini.EmbeddingConfig
type GeminiEmbedder = gemini.Embedder

func GeminiNewEmbedder(ctx context.Context, config *GeminiEmbeddingConfig) (*GeminiEmbedder, error) {
	return gemini.NewEmbedder(ctx, config)
}

// ---------------------------------------------------------------------------
// ollama embedding provider
// ---------------------------------------------------------------------------

type OllamaEmbeddingConfig = ollama.EmbeddingConfig
type OllamaEmbedder = ollama.Embedder

func OllamaNewEmbedder(ctx context.Context, config *OllamaEmbeddingConfig) (*OllamaEmbedder, error) {
	return ollama.NewEmbedder(ctx, config)
}

// ---------------------------------------------------------------------------
// openai embedding provider
// ---------------------------------------------------------------------------

type OpenAIEmbeddingConfig = openai.EmbeddingConfig
type OpenAIEmbedder = openai.Embedder

func OpenAINewEmbedder(ctx context.Context, config *OpenAIEmbeddingConfig) (*OpenAIEmbedder, error) {
	return openai.NewEmbedder(ctx, config)
}
