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

// ext_embedding_facade defines native config types and constructor functions
// for all embedding providers.  Each constructor creates a native provider
// (which implements eino's embedding.Embedder with zero eino-ext imports) and
// returns it directly.
//
// This file has ZERO cloudwego/eino-ext imports.
package einobridge

import (
	"context"

	"github.com/cloudwego/eino/components/embedding"

	arkprovider "github.com/superagent-ai/superagent-base/backend/pkg/embedding/providers/ark"
	geminiprovider "github.com/superagent-ai/superagent-base/backend/pkg/embedding/providers/gemini"
	ollamaprovider "github.com/superagent-ai/superagent-base/backend/pkg/embedding/providers/ollama"
	openaiprovider "github.com/superagent-ai/superagent-base/backend/pkg/embedding/providers/openai"
)

// ---------------------------------------------------------------------------
// ark embedding provider
// ---------------------------------------------------------------------------

type ArkAPIType = arkprovider.APIType
type ArkEmbeddingConfig = arkprovider.EmbeddingConfig

const (
	ArkAPITypeText       ArkAPIType = arkprovider.APITypeText
	ArkAPITypeMultiModal ArkAPIType = arkprovider.APITypeMultiModal
)

func ArkNewEmbedder(ctx context.Context, config *ArkEmbeddingConfig) (embedding.Embedder, error) {
	return arkprovider.NewEmbedder(ctx, config)
}

// ---------------------------------------------------------------------------
// gemini embedding provider
// ---------------------------------------------------------------------------

type GeminiEmbeddingConfig = geminiprovider.EmbeddingConfig

func GeminiNewEmbedder(ctx context.Context, config *GeminiEmbeddingConfig) (embedding.Embedder, error) {
	return geminiprovider.NewEmbedder(ctx, config)
}

// ---------------------------------------------------------------------------
// ollama embedding provider
// ---------------------------------------------------------------------------

type OllamaEmbeddingConfig = ollamaprovider.EmbeddingConfig

func OllamaNewEmbedder(ctx context.Context, config *OllamaEmbeddingConfig) (embedding.Embedder, error) {
	return ollamaprovider.NewEmbedder(ctx, config)
}

// ---------------------------------------------------------------------------
// openai embedding provider
// ---------------------------------------------------------------------------

type OpenAIEmbeddingConfig = openaiprovider.EmbeddingConfig

func OpenAINewEmbedder(ctx context.Context, config *OpenAIEmbeddingConfig) (embedding.Embedder, error) {
	return openaiprovider.NewEmbedder(ctx, config)
}
