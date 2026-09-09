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

package wrap

import (
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
	"context"


	contract "github.com/superagent-ai/superagent-base/backend/infra/embedding"
)

func NewGeminiEmbedder(ctx context.Context, config *einobridge.GeminiEmbeddingConfig, dimensions int64, batchSize int) (contract.Embedder, error) {
	emb, err := einobridge.GeminiNewEmbedder(ctx, config)
	if err != nil {
		return nil, err
	}
	return &denseOnlyWrap{dims: dimensions, batchSize: batchSize, Embedder: emb}, nil
}
