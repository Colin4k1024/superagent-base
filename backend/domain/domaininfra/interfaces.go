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

// Package domaininfra defines the infrastructure interfaces that the domain
// layer depends on. These interfaces are the ONLY contract between domain
// logic and infrastructure — domain code MUST import this package instead of
// importing infra/ packages directly.
//
// Implementations live in backend/infra/ and are wired at startup via
// application.Init().
package domaininfra

import (
	"context"
	"io"
	"time"
)

// IDGenerator generates unique IDs for domain entities.
type IDGenerator interface {
	GenID(ctx context.Context) (int64, error)
}

// Cache provides a simplified key-value cache abstraction.
// Domain code should use this instead of the Redis-specific Cmdable interface.
type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) (int64, error)
	Exists(ctx context.Context, keys ...string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) (bool, error)
	HGet(ctx context.Context, key, field string) (string, error)
	HSet(ctx context.Context, key string, values ...interface{}) (int64, error)
	HDel(ctx context.Context, key string, fields ...string) (int64, error)
	LPush(ctx context.Context, key string, values ...interface{}) (int64, error)
	RPush(ctx context.Context, key string, values ...interface{}) (int64, error)
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	Close() error
}

// Storage provides an object storage abstraction.
// Domain code should use this instead of the infra/storage.Storage interface.
type Storage interface {
	// PutObject stores content with the given key.
	PutObject(ctx context.Context, objectKey string, content []byte) error
	// PutObjectWithReader stores content from a reader with the given key.
	PutObjectWithReader(ctx context.Context, objectKey string, content io.Reader) error
	// GetObject retrieves content by key.
	GetObject(ctx context.Context, objectKey string) ([]byte, error)
	// GetObjectReader returns a reader for the object.
	GetObjectReader(ctx context.Context, objectKey string) (io.ReadCloser, error)
	// DeleteObject removes an object by key.
	DeleteObject(ctx context.Context, objectKey string) error
	// GetObjectUrl returns a presigned URL for the object.
	GetObjectUrl(ctx context.Context, objectKey string) (string, error)
}

// EventBus provides a publish-subscribe abstraction for domain events.
type EventBus interface {
	Publish(ctx context.Context, topic string, payload interface{}) error
}
