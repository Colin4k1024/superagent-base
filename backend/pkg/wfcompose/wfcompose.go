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

package wfcompose

import (
	"context"
	"errors"
	"io"
)

// Package wfcompose provides framework-agnostic types for workflow graph
// orchestration. It decouples the workflow public API surface from any
// specific orchestration engine (e.g. cloudwego/eino).
//
// Engine-specific adapters live in sub-packages (see einobridge) and are the
// only place that imports a concrete engine. All other code should depend on
// the types declared here.

// errStreamClosed is returned by StreamReader.Recv when no recv function is
// configured (e.g. a nil-backed reader).
var errStreamClosed = errors.New("wfcompose: stream is closed")

// IsEOF reports whether err signals end-of-stream. It matches io.EOF and any
// error that wraps it, regardless of the engine's concrete sentinel type.
func IsEOF(err error) bool { return errors.Is(err, io.EOF) }

// Option is an opaque, framework-agnostic workflow execution option.
//
// It is created by workflow providers (e.g. WithExecuteConfig,
// WithMessagePipe) and consumed by the workflow runtime. Consumers must not
// inspect its contents; engine adapters unwrap it via Unwrap when feeding the
// concrete engine.
type Option struct {
	inner any
}

// NewOption wraps an engine-specific option value.
func NewOption(inner any) Option { return Option{inner: inner} }

// Unwrap returns the underlying engine-specific option.
// It is intended for use by engine adapters only.
func (o Option) Unwrap() (any, bool) {
	if o.inner == nil {
		return nil, false
	}
	return o.inner, true
}

// IsEmpty reports whether the option carries no wrapped value.
func (o Option) IsEmpty() bool { return o.inner == nil }

// MergeOptions combines multiple Options into a single Option whose wrapped
// value is a slice of the underlying engine options (in order). Engine adapters
// that expect a variadic spread should call Unwrap and type-assert to []any.
func MergeOptions(opts ...Option) Option {
	merged := make([]any, 0, len(opts))
	for _, o := range opts {
		if v, ok := o.Unwrap(); ok {
			merged = append(merged, v)
		}
	}
	return NewOption(merged)
}

// StreamReader is a framework-agnostic streaming reader.
//
// It mirrors the minimal consumption surface (Recv/Close) of common engine
// stream readers, so callers can iterate engine streams without importing the
// engine package.
type StreamReader[T any] struct {
	recv      func() (T, error)
	close     func()
	autoClose func()
	src       any
}

// NewStreamReader builds a StreamReader from the given recv/close functions.
// close may be nil.
func NewStreamReader[T any](recv func() (T, error), close func()) *StreamReader[T] {
	return &StreamReader[T]{recv: recv, close: close}
}

// Recv receives the next streamed value.
// It returns io.EOF (or an error wrapping it) when the stream is exhausted.
func (sr *StreamReader[T]) Recv() (T, error) {
	if sr == nil || sr.recv == nil {
		var zero T
		return zero, errStreamClosed
	}
	return sr.recv()
}

// Close releases the underlying stream resources. It is safe to call on a nil
// receiver or to call it multiple times.
func (sr *StreamReader[T]) Close() {
	if sr == nil {
		return
	}
	if sr.autoClose != nil {
		sr.autoClose()
	}
	if sr.close != nil {
		sr.close()
	}
}

// SetAutomaticClose marks the reader so that Close is invoked automatically
// when the stream is fully consumed (Recv returns io.EOF). Engines that do not
// support automatic close ignore this call.
func (sr *StreamReader[T]) SetAutomaticClose() {
	if sr != nil && sr.autoClose != nil {
		sr.autoClose()
	}
}

// WithAutomaticClose sets the hook invoked by SetAutomaticClose.
func (sr *StreamReader[T]) WithAutomaticClose(auto func()) *StreamReader[T] {
	if sr != nil {
		sr.autoClose = auto
	}
	return sr
}

// WithSource attaches the underlying engine-native reader so that engine
// adapters can recover it losslessly via Source. It is intended for use by
// engine adapters only.
func (sr *StreamReader[T]) WithSource(src any) *StreamReader[T] {
	if sr != nil {
		sr.src = src
	}
	return sr
}

// Source returns the underlying engine-native reader (if any) attached via
// WithSource. It is intended for use by engine adapters only.
func (sr *StreamReader[T]) Source() any {
	if sr == nil {
		return nil
	}
	return sr.src
}

// StreamWriter is a framework-agnostic streaming writer.
type StreamWriter[T any] struct {
	send  func(T, error)
	close func()
}

// NewStreamWriter builds a StreamWriter from the given send/close functions.
func NewStreamWriter[T any](send func(T, error), close func()) *StreamWriter[T] {
	return &StreamWriter[T]{send: send, close: close}
}

// Send sends a value (or a terminal error) to the stream.
func (sw *StreamWriter[T]) Send(v T, err error) {
	if sw == nil || sw.send == nil {
		return
	}
	sw.send(v, err)
}

// Close closes the write side of the stream.
func (sw *StreamWriter[T]) Close() {
	if sw == nil || sw.close == nil {
		return
	}
	sw.close()
}

// CheckPointStore is a framework-agnostic checkpoint store for workflow
// execution state. Implementations persist serialized checkpoint state keyed
// by a checkpoint ID, enabling interrupt/resume semantics.
type CheckPointStore interface {
	Get(ctx context.Context, checkPointID string) ([]byte, bool, error)
	Set(ctx context.Context, checkPointID string, checkPoint []byte) error
}

// CheckPointDeleter is an optional interface that CheckPointStore
// implementations may implement to support explicit checkpoint deletion.
type CheckPointDeleter interface {
	Delete(ctx context.Context, checkPointID string) error
}

// FieldPath represents a path into a nested map, mirroring
// compose.FieldPath from eino. It is a simple []string that
// names successive map keys.
type FieldPath []string

// Runnable is the framework-agnostic runner interface, mirroring
// compose.Runnable from eino. It is the contract returned by graph
// compilation and consumed by the agent/service layer. Only the Stream
// method is required for the current agentflow usage; additional methods
// (Invoke, Collect, Transform) can be added when needed.
type Runnable[I, O any] interface {
	Stream(ctx context.Context, input I, opts ...Option) (*StreamReader[O], error)
}
