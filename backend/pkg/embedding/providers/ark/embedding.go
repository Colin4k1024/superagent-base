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

// Package ark provides a native embedding provider for Volcengine Ark,
// implementing the wfcompose.Embedder interface without any eino-ext imports.
package ark

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

var (
	defaultBaseURL    = "https://ark.cn-beijing.volces.com/api/v3"
	defaultRegion     = "cn-beijing"
	defaultRetryTimes = 2
	defaultTimeout    = 10 * time.Minute
)

type APIType string

const (
	APITypeText       APIType = "text_api"
	APITypeMultiModal APIType = "multi_modal_api"
)

type EmbeddingConfig struct {
	Timeout               *time.Duration `json:"timeout"`
	HTTPClient            *http.Client    `json:"http_client"`
	RetryTimes            *int            `json:"retry_times"`
	BaseURL               string          `json:"base_url"`
	Region                string          `json:"region"`
	APIKey                string          `json:"api_key"`
	AccessKey             string          `json:"access_key"`
	SecretKey             string          `json:"secret_key"`
	Model                 string          `json:"model"`
	APIType               *APIType        `json:"api_type,omitempty"`
	MaxConcurrentRequests *int            `json:"max_concurrent_requests"`
}

var _ wfcompose.Embedder = (*Embedder)(nil)

type Embedder struct {
	client *arkruntime.Client
	conf   *EmbeddingConfig
}

func NewEmbedder(ctx context.Context, config *EmbeddingConfig) (*Embedder, error) {
	client := buildClient(config)
	return &Embedder{client: client, conf: config}, nil
}

func (e *Embedder) EmbedStrings(ctx context.Context, texts []string, opts ...wfcompose.EmbeddingOption) ([][]float64, error) {
	options := wfcompose.GetEmbeddingOptions(&wfcompose.EmbeddingOptions{
		Model: &e.conf.Model,
	}, opts...)

	modelName := dereferenceOrZero(options.Model)

	if e.conf.APIType == nil || *e.conf.APIType == APITypeText {
		resp, err := e.client.CreateEmbeddings(ctx, model.EmbeddingRequestStrings{
			Input:          texts,
			Model:          modelName,
			EncodingFormat: model.EmbeddingEncodingFormatFloat,
		})
		if err != nil {
			return nil, fmt.Errorf("[Ark] CreateEmbeddings error: %w", err)
		}
		embeddings := make([][]float64, len(resp.Data))
		for i, d := range resp.Data {
			embeddings[i] = toFloat64(d.Embedding)
		}
		return embeddings, nil
	}

	mu := sync.Mutex{}
	eg := errgroup.Group{}
	eg.SetLimit(*e.conf.MaxConcurrentRequests)
	embeddings := make([][]float64, len(texts))

	for i := 0; i < len(texts); i++ {
		idx := i
		text := texts[idx]
		eg.Go(func() error {
			res, err := e.client.CreateMultiModalEmbeddings(ctx, model.MultiModalEmbeddingRequest{
				Input: []model.MultimodalEmbeddingInput{
					{Type: model.MultiModalEmbeddingInputTypeText, Text: &text},
				},
				Model:          modelName,
				EncodingFormat: ptrOf(model.EmbeddingEncodingFormatFloat),
			})
			if err != nil {
				return fmt.Errorf("[Ark] CreateMultiModalEmbeddings error: %w", err)
			}
			mu.Lock()
			defer mu.Unlock()
			embeddings[idx] = toFloat64(res.Data.Embedding)
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return embeddings, nil
}

func buildClient(config *EmbeddingConfig) *arkruntime.Client {
	if len(config.BaseURL) == 0 {
		config.BaseURL = defaultBaseURL
	}
	if len(config.Region) == 0 {
		config.Region = defaultRegion
	}
	if config.Timeout == nil {
		config.Timeout = &defaultTimeout
	}
	if config.RetryTimes == nil {
		config.RetryTimes = &defaultRetryTimes
	}
	if config.APIType == nil {
		apiType := APITypeText
		config.APIType = &apiType
	} else if *config.APIType == APITypeMultiModal {
		if config.MaxConcurrentRequests == nil {
			defaultMaxConcurrentRequests := 5
			config.MaxConcurrentRequests = &defaultMaxConcurrentRequests
		}
	}

	opts := []arkruntime.ConfigOption{
		arkruntime.WithRetryTimes(*config.RetryTimes),
		arkruntime.WithBaseUrl(config.BaseURL),
		arkruntime.WithRegion(config.Region),
		arkruntime.WithTimeout(*config.Timeout),
	}
	if config.HTTPClient != nil {
		opts = append(opts, arkruntime.WithHTTPClient(config.HTTPClient))
	}

	if len(config.APIKey) > 0 {
		return arkruntime.NewClientWithApiKey(config.APIKey, opts...)
	}
	return arkruntime.NewClientWithAkSk(config.AccessKey, config.SecretKey, opts...)
}

func toFloat64(in []float32) []float64 {
	out := make([]float64, len(in))
	for i, v := range in {
		out[i] = float64(v)
	}
	return out
}

func dereferenceOrZero[T any](v *T) T {
	if v == nil {
		var t T
		return t
	}
	return *v
}

func ptrOf[T any](v T) *T {
	return &v
}
