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

// Package einobridge adapts cloudwego/eino compose + schema types to the
// framework-agnostic pkg/wfcompose types. It is the ONLY package outside the
// workflow internals that should import eino for orchestration concerns.
package einobridge

import (
	"context"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

// WrapOption wraps an eino compose.Option as a framework-agnostic wfcompose.Option.
func WrapOption(opt compose.Option) wfcompose.Option {
	return wfcompose.NewOption(opt)
}

// UnwrapOption recovers the underlying eino compose.Option from a
// framework-agnostic option. Returns ok=false when the option is empty or does
// not carry a single eino Option.
func UnwrapOption(opt wfcompose.Option) (compose.Option, bool) {
	inner, ok := opt.Unwrap()
	if !ok {
		return compose.Option{}, false
	}
	eo, ok := inner.(compose.Option)
	if !ok {
		return compose.Option{}, false
	}
	return eo, true
}

// UnwrapOptionSlice recovers a variadic slice of eino compose.Option from
// framework-agnostic options, including options produced by MergeOptions.
func UnwrapOptionSlice(opts ...wfcompose.Option) []compose.Option {
	out := make([]compose.Option, 0, len(opts))
	for _, o := range opts {
		inner, ok := o.Unwrap()
		if !ok {
			continue
		}
		switch v := inner.(type) {
		case compose.Option:
			out = append(out, v)
		case []any:
			for _, item := range v {
				if eo, ok := item.(compose.Option); ok {
					out = append(out, eo)
				}
			}
		case []compose.Option:
			out = append(out, v...)
		}
	}
	return out
}

// WrapStreamReader wraps an eino *schema.StreamReader[T] as a framework-agnostic
// *wfcompose.StreamReader[T]. The returned reader delegates Recv/Close to the
// underlying eino reader and propagates SetAutomaticClose.
func WrapStreamReader[T any](sr *schema.StreamReader[T]) *wfcompose.StreamReader[T] {
	if sr == nil {
		return nil
	}
	w := wfcompose.NewStreamReader[T](sr.Recv, sr.Close)
	w.WithAutomaticClose(sr.SetAutomaticClose)
	w.WithSource(sr) // lossless unwrap support for engine-internal consumers
	return w
}

// UnwrapStreamReader recovers the underlying eino *schema.StreamReader[T] from
// a framework-agnostic reader produced by WrapStreamReader. It is lossless: the
// returned reader is the very same eino reader that was wrapped, so Recv/Close
// are shared and no goroutine or pipe is introduced.
//
// It is intended for engine-internal consumers (e.g. application handlers that
// must feed eino conversion helpers like schema.StreamReaderWithConvert). Call
// it only at the boundary where a wfcompose reader must re-enter eino APIs.
func UnwrapStreamReader[T any](sr *wfcompose.StreamReader[T]) *schema.StreamReader[T] {
	if sr == nil {
		return nil
	}
	if e, ok := sr.Source().(*schema.StreamReader[T]); ok {
		return e
	}
	return nil
}

// WrapMessageStreamReader converts an eino *schema.StreamReader[*schema.Message]
// into a framework-agnostic *wfcompose.StreamReader[*wfcompose.Message] by
// wrapping each element via WrapMessage. nil in, nil out.
func WrapMessageStreamReader(sr *schema.StreamReader[*schema.Message]) *wfcompose.StreamReader[*wfcompose.Message] {
	if sr == nil {
		return nil
	}
	converted := schema.StreamReaderWithConvert(sr, func(m *schema.Message) (*wfcompose.Message, error) {
		return WrapMessage(m), nil
	})
	return WrapStreamReader[*wfcompose.Message](converted)
}

// UnwrapMessageStreamReader converts a framework-agnostic
// *wfcompose.StreamReader[*wfcompose.Message] back into an eino
// *schema.StreamReader[*schema.Message] by unwrapping each element via
// UnwrapMessage. nil in, nil out.
func UnwrapMessageStreamReader(sr *wfcompose.StreamReader[*wfcompose.Message]) *schema.StreamReader[*schema.Message] {
	if sr == nil {
		return nil
	}
	einoSR := UnwrapStreamReader[*wfcompose.Message](sr)
	if einoSR == nil {
		return nil
	}
	return schema.StreamReaderWithConvert(einoSR, func(m *wfcompose.Message) (*schema.Message, error) {
		return UnwrapMessage(m), nil
	})
}

// WrapStreamWriter wraps an eino *schema.StreamWriter[T] as a framework-agnostic
// *wfcompose.StreamWriter[T].
func WrapStreamWriter[T any](sw *schema.StreamWriter[T]) *wfcompose.StreamWriter[T] {
	if sw == nil {
		return nil
	}
	return wfcompose.NewStreamWriter[T](func(v T, err error) { sw.Send(v, err) }, sw.Close)
}

// CheckPointStoreAdapter adapts a framework-agnostic wfcompose.CheckPointStore
// to the eino compose.CheckPointStore interface.
type CheckPointStoreAdapter struct {
	Store wfcompose.CheckPointStore
}

// Get delegates to the wrapped store.
func (a CheckPointStoreAdapter) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	return a.Store.Get(ctx, checkPointID)
}

// Set delegates to the wrapped store.
func (a CheckPointStoreAdapter) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	return a.Store.Set(ctx, checkPointID, checkPoint)
}

// AdaptCheckPointStore returns an eino compose.CheckPointStore backed by the
// given framework-agnostic store, or nil when store is nil.
func AdaptCheckPointStore(store wfcompose.CheckPointStore) compose.CheckPointStore {
	if store == nil {
		return nil
	}
	return CheckPointStoreAdapter{Store: store}
}

// EinoCheckPointStore wraps an eino compose.CheckPointStore as a framework-
// agnostic wfcompose.CheckPointStore.
type EinoCheckPointStore struct {
	Store compose.CheckPointStore
}

// Get delegates to the wrapped eino store.
func (e EinoCheckPointStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	return e.Store.Get(ctx, checkPointID)
}

// Set delegates to the wrapped eino store.
func (e EinoCheckPointStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	return e.Store.Set(ctx, checkPointID, checkPoint)
}
