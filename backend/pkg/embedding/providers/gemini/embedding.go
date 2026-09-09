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

// Package gemini provides a native embedding provider for Google Gemini,
// implementing the wfcompose.Embedder interface without any eino-ext imports.
package gemini

import (
	"context"

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
	"google.golang.org/genai"
)

type EmbeddingConfig struct {
	Client               *genai.Client
	Model                string
	TaskType             string
	Title                string
	OutputDimensionality *int32
	MIMEType             string `json:"mimeType,omitempty"`
	AutoTruncate         bool   `json:"autoTruncate,omitempty"`
}

var _ wfcompose.Embedder = (*Embedder)(nil)

type Embedder struct {
	cli  *genai.Client
	conf *EmbeddingConfig
}

func NewEmbedder(ctx context.Context, cfg *EmbeddingConfig) (*Embedder, error) {
	return &Embedder{cli: cfg.Client, conf: cfg}, nil
}

func (e *Embedder) EmbedStrings(ctx context.Context, texts []string, opts ...wfcompose.EmbeddingOption) ([][]float64, error) {
	options := wfcompose.GetEmbeddingOptions(&wfcompose.EmbeddingOptions{
		Model: &e.conf.Model,
	}, opts...)

	contents := make([]*genai.Content, 0, len(texts))
	for _, text := range texts {
		contents = append(contents, genai.NewContentFromText(text, genai.RoleUser))
	}

	embedContentConfig := &genai.EmbedContentConfig{
		TaskType:             e.conf.TaskType,
		Title:                e.conf.Title,
		OutputDimensionality: e.conf.OutputDimensionality,
		MIMEType:             e.conf.MIMEType,
		AutoTruncate:         e.conf.AutoTruncate,
	}

	resp, err := e.cli.Models.EmbedContent(ctx,
		*options.Model,
		contents,
		embedContentConfig,
	)
	if err != nil {
		return nil, err
	}

	embeddings := make([][]float64, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		embeddings[i] = make([]float64, len(emb.Values))
		for j, v := range emb.Values {
			embeddings[i][j] = float64(v)
		}
	}
	return embeddings, nil
}
