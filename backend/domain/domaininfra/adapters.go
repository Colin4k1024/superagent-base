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

// Package domaininfra provides adapter functions that wrap existing infra
// implementations into domain-layer interfaces. These adapters are called
// once at startup in application.Init() — after that, domain code only
// sees the interfaces defined in domain/domaininfra/interfaces.go.
package domaininfra

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/superagent-ai/superagent-base/backend/infra/cache"
	"github.com/superagent-ai/superagent-base/backend/infra/idgen"
	"github.com/superagent-ai/superagent-base/backend/infra/storage"
)

// ── IDGenerator adapter ─────────────────────────────────────────────────────

type idGenAdapter struct {
	inner idgen.IDGenerator
}

// WrapIDGenerator wraps an infra/idgen.IDGenerator into a domain IDGenerator.
func WrapIDGenerator(g idgen.IDGenerator) IDGenerator {
	return &idGenAdapter{inner: g}
}

func (a *idGenAdapter) GenID(ctx context.Context) (int64, error) {
	return a.inner.GenID(ctx)
}

// ── Cache adapter ───────────────────────────────────────────────────────────

type cacheAdapter struct {
	inner cache.Cmdable
}

// WrapCache wraps an infra/cache.Cmdable into a domain Cache.
func WrapCache(c cache.Cmdable) Cache {
	return &cacheAdapter{inner: c}
}

func (a *cacheAdapter) Get(ctx context.Context, key string) (string, error) {
	cmd := a.inner.Get(ctx, key)
	return cmd.Val(), cmd.Err()
}

func (a *cacheAdapter) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return a.inner.Set(ctx, key, value, expiration).Err()
}

func (a *cacheAdapter) Del(ctx context.Context, keys ...string) (int64, error) {
	cmd := a.inner.Del(ctx, keys...)
	return cmd.Result()
}

func (a *cacheAdapter) Exists(ctx context.Context, keys ...string) (int64, error) {
	cmd := a.inner.Exists(ctx, keys...)
	return cmd.Result()
}

func (a *cacheAdapter) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	cmd := a.inner.Expire(ctx, key, expiration)
	return cmd.Result()
}

func (a *cacheAdapter) HGet(ctx context.Context, key, field string) (string, error) {
	all := a.inner.HGetAll(ctx, key)
	m, err := all.Result()
	if err != nil {
		return "", err
	}
	val, ok := m[field]
	if !ok {
		return "", cache.Nil
	}
	return val, nil
}

func (a *cacheAdapter) HSet(ctx context.Context, key string, values ...interface{}) (int64, error) {
	cmd := a.inner.HSet(ctx, key, values...)
	return cmd.Result()
}

func (a *cacheAdapter) HDel(ctx context.Context, key string, fields ...string) (int64, error) {
	cmd := a.inner.HDel(ctx, key, fields...)
	return cmd.Result()
}

func (a *cacheAdapter) LPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	cmd := a.inner.LPush(ctx, key, values...)
	return cmd.Result()
}

func (a *cacheAdapter) RPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	cmd := a.inner.RPush(ctx, key, values...)
	return cmd.Result()
}

func (a *cacheAdapter) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	cmd := a.inner.LRange(ctx, key, start, stop)
	return cmd.Result()
}

func (a *cacheAdapter) Close() error {
	// Cmdable doesn't have Close; the underlying client handles lifecycle.
	return nil
}

// ── Storage adapter ─────────────────────────────────────────────────────────

type storageAdapter struct {
	inner storage.Storage
}

// WrapStorage wraps an infra/storage.Storage into a domain Storage.
func WrapStorage(s storage.Storage) Storage {
	return &storageAdapter{inner: s}
}

func (a *storageAdapter) PutObject(ctx context.Context, objectKey string, content []byte) error {
	return a.inner.PutObject(ctx, objectKey, content)
}

func (a *storageAdapter) PutObjectWithReader(ctx context.Context, objectKey string, content io.Reader) error {
	return a.inner.PutObjectWithReader(ctx, objectKey, content)
}

func (a *storageAdapter) GetObject(ctx context.Context, objectKey string) ([]byte, error) {
	return a.inner.GetObject(ctx, objectKey)
}

func (a *storageAdapter) GetObjectReader(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	data, err := a.inner.GetObject(ctx, objectKey)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (a *storageAdapter) DeleteObject(ctx context.Context, objectKey string) error {
	return a.inner.DeleteObject(ctx, objectKey)
}

func (a *storageAdapter) GetObjectUrl(ctx context.Context, objectKey string) (string, error) {
	return a.inner.GetObjectUrl(ctx, objectKey)
}
