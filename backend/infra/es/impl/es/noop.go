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

package es

import (
	"context"
	"fmt"
)

// noopClient is a no-op implementation of es.Client for when ES is not configured.
type noopClient struct{}

// NewNoop creates a new no-op ES client.
func NewNoop() Client {
	return &noopClient{}
}

func (c *noopClient) Create(ctx context.Context, index, id string, document any) error {
	return fmt.Errorf("elasticsearch not configured")
}

func (c *noopClient) Update(ctx context.Context, index, id string, document any) error {
	return fmt.Errorf("elasticsearch not configured")
}

func (c *noopClient) Delete(ctx context.Context, index, id string) error {
	return fmt.Errorf("elasticsearch not configured")
}

func (c *noopClient) Search(ctx context.Context, index string, req *Request) (*Response, error) {
	return &Response{}, nil
}

func (c *noopClient) Exists(ctx context.Context, index string) (bool, error) {
	return false, nil
}

func (c *noopClient) CreateIndex(ctx context.Context, index string, properties map[string]any) error {
	return fmt.Errorf("elasticsearch not configured")
}

func (c *noopClient) DeleteIndex(ctx context.Context, index string) error {
	return fmt.Errorf("elasticsearch not configured")
}

func (c *noopClient) Types() Types {
	return &noopTypes{}
}

func (c *noopClient) NewBulkIndexer(index string) (BulkIndexer, error) {
	return nil, fmt.Errorf("elasticsearch not configured")
}

type noopTypes struct{}

func (t *noopTypes) NewLongNumberProperty() any {
	return nil
}

func (t *noopTypes) NewTextProperty() any {
	return nil
}

func (t *noopTypes) NewUnsignedLongNumberProperty() any {
	return nil
}
