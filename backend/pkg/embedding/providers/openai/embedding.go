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

// Package openai provides a native embedding provider for OpenAI-compatible
// APIs, implementing the eino embedding.Embedder interface without any eino-ext imports.
package openai

import (
	"context"
	"net/http"
	"time"

	"github.com/cloudwego/eino/components/embedding"
	goopenai "github.com/meguminnnnnnnnn/go-openai"
)

type EmbeddingEncodingFormat string

const (
	EmbeddingEncodingFormatFloat  EmbeddingEncodingFormat = "float"
	EmbeddingEncodingFormatBase64 EmbeddingEncodingFormat = "base64"
)

type EmbeddingConfig struct {
	Timeout         time.Duration           `json:"timeout"`
	HTTPClient      *http.Client            `json:"http_client"`
	APIKey          string                  `json:"api_key"`
	ByAzure         bool                    `json:"by_azure"`
	BaseURL         string                  `json:"base_url"`
	APIVersion      string                  `json:"api_version"`
	Model           string                  `json:"model"`
	EncodingFormat  *EmbeddingEncodingFormat `json:"encoding_format,omitempty"`
	Dimensions      *int                    `json:"dimensions,omitempty"`
	User            *string                 `json:"user,omitempty"`
}

var _ embedding.Embedder = (*Embedder)(nil)

type Embedder struct {
	cli    *goopenai.Client
	config *EmbeddingConfig
}

func NewEmbedder(ctx context.Context, config *EmbeddingConfig) (*Embedder, error) {
	if config == nil {
		config = &EmbeddingConfig{Model: string(goopenai.AdaEmbeddingV2)}
	}

	var clientConf goopenai.ClientConfig
	if config.ByAzure {
		clientConf = goopenai.DefaultAzureConfig(config.APIKey, config.BaseURL)
		if config.APIVersion != "" {
			clientConf.APIVersion = config.APIVersion
		}
	} else {
		clientConf = goopenai.DefaultConfig(config.APIKey)
		if config.BaseURL != "" {
			clientConf.BaseURL = config.BaseURL
		}
	}

	if config.HTTPClient != nil {
		clientConf.HTTPClient = config.HTTPClient
	} else {
		clientConf.HTTPClient = &http.Client{Timeout: config.Timeout}
	}

	return &Embedder{
		cli:    goopenai.NewClientWithConfig(clientConf),
		config: config,
	}, nil
}

func (e *Embedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	options := embedding.GetCommonOptions(&embedding.Options{
		Model: &e.config.Model,
	}, opts...)

	req := &goopenai.EmbeddingRequest{
		Input:          texts,
		Model:          goopenai.EmbeddingModel(*options.Model),
		User:           dereferenceOrZero(e.config.User),
		EncodingFormat: goopenai.EmbeddingEncodingFormat(dereferenceOrDefault(e.config.EncodingFormat, EmbeddingEncodingFormatFloat)),
		Dimensions:     dereferenceOrZero(e.config.Dimensions),
	}

	resp, err := e.cli.CreateEmbeddings(ctx, *req)
	if err != nil {
		return nil, err
	}

	embeddings := make([][]float64, len(resp.Data))
	for i, d := range resp.Data {
		res := make([]float64, len(d.Embedding))
		for j, emb := range d.Embedding {
			res[j] = float64(emb)
		}
		embeddings[i] = res
	}
	return embeddings, nil
}

func dereferenceOrZero[T any](v *T) T {
	if v == nil {
		var t T
		return t
	}
	return *v
}

func dereferenceOrDefault[T any](v *T, def T) T {
	if v == nil {
		return def
	}
	return *v
}
