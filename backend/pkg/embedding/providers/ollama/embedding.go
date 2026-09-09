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

// Package ollama provides a native embedding provider for Ollama,
// implementing the eino embedding.Embedder interface without any eino-ext imports.
package ollama

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/ollama/ollama/api"
)

var defaultBaseURL = "http://localhost:11434"

type EmbeddingConfig struct {
	Timeout    time.Duration    `json:"timeout"`
	HTTPClient *http.Client     `json:"http_client"`
	BaseURL    string            `json:"base_url"`
	Model      string            `json:"model"`
	Truncate   *bool            `json:"truncate,omitempty"`
	KeepAlive  *time.Duration   `json:"keep_alive,omitempty"`
	Options    map[string]any   `json:"options,omitempty"`
}

var _ embedding.Embedder = (*Embedder)(nil)

type Embedder struct {
	cli  *api.Client
	conf *EmbeddingConfig
}

func NewEmbedder(ctx context.Context, config *EmbeddingConfig) (*Embedder, error) {
	if config == nil {
		return nil, fmt.Errorf("embedding config must not be nil")
	}
	if len(config.BaseURL) == 0 {
		config.BaseURL = defaultBaseURL
	}

	var httpClient *http.Client
	if config.HTTPClient != nil {
		httpClient = config.HTTPClient
	} else {
		httpClient = &http.Client{Timeout: config.Timeout}
	}

	baseURL, err := url.Parse(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	cli := api.NewClient(baseURL, httpClient)
	return &Embedder{cli: cli, conf: config}, nil
}

func (e *Embedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	req := &api.EmbedRequest{
		Model:    e.conf.Model,
		Input:    texts,
		Truncate: e.conf.Truncate,
		Options:  e.conf.Options,
	}
	if e.conf.KeepAlive != nil {
		req.KeepAlive = &api.Duration{Duration: *e.conf.KeepAlive}
	}

	resp, err := e.cli.Embed(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("[Ollama] EmbedStrings error: %v", err)
	}

	result := make([][]float64, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		result[i] = make([]float64, len(emb))
		for j, v := range emb {
			result[i][j] = float64(v)
		}
	}
	return result, nil
}
