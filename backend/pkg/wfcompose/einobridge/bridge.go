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

// Package einobridge provides the bridge between wfcompose native types and
// the facade API. Now that all type aliases point to wfcompose types, most
// bridge functions are identity functions or thin wrappers.
package einobridge

import (
	"context"

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

// WrapOption wraps a compose option as a framework-agnostic wfcompose.Option.
func WrapOption(opt any) wfcompose.Option {
	return wfcompose.NewOption(opt)
}

// UnwrapOption recovers the underlying option from a framework-agnostic option.
func UnwrapOption(opt wfcompose.Option) (wfcompose.Option, bool) {
	return opt, true
}

// UnwrapOptionSlice recovers a variadic slice of options.
func UnwrapOptionSlice(opts ...wfcompose.Option) []wfcompose.Option {
	return opts
}

// WrapStreamReader is an identity function (wfcompose → wfcompose).
func WrapStreamReader[T any](sr *wfcompose.StreamReader[T]) *wfcompose.StreamReader[T] {
	return sr
}

// UnwrapStreamReader is an identity function.
func UnwrapStreamReader[T any](sr *wfcompose.StreamReader[T]) *wfcompose.StreamReader[T] {
	return sr
}

// WrapMessageStreamReader is an identity function.
func WrapMessageStreamReader(sr *wfcompose.StreamReader[*wfcompose.Message]) *wfcompose.StreamReader[*wfcompose.Message] {
	return sr
}

// UnwrapMessageStreamReader is an identity function.
func UnwrapMessageStreamReader(sr *wfcompose.StreamReader[*wfcompose.Message]) *wfcompose.StreamReader[*wfcompose.Message] {
	return sr
}

// WrapStreamWriter is an identity function.
func WrapStreamWriter[T any](sw *wfcompose.StreamWriter[T]) *wfcompose.StreamWriter[T] {
	return sw
}

// CheckPointStoreAdapter is a no-op adapter (wfcompose.CheckPointStore is already native).
type CheckPointStoreAdapter struct {
	Store wfcompose.CheckPointStore
}

func (a CheckPointStoreAdapter) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	return a.Store.Get(ctx, checkPointID)
}

func (a CheckPointStoreAdapter) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	return a.Store.Set(ctx, checkPointID, checkPoint)
}

// AdaptCheckPointStore returns the given store directly (no conversion needed).
func AdaptCheckPointStore(store wfcompose.CheckPointStore) wfcompose.CheckPointStore {
	return store
}

// EinoCheckPointStore is a compatibility alias (no longer wraps eino).
type EinoCheckPointStore struct {
	Store wfcompose.CheckPointStore
}

func (e EinoCheckPointStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	return e.Store.Get(ctx, checkPointID)
}

func (e EinoCheckPointStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	return e.Store.Set(ctx, checkPointID, checkPoint)
}
