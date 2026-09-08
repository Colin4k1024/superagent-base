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

// Package dagbridge adapts workflow node interfaces (InvokableNode,
// StreamableNodeWOpt, etc.) to the framework-agnostic dag.NodeExecutor
// and dag.StreamingNodeExecutor interfaces.
//
// This is the bridge between the existing node implementations and the
// new self-built DAG engine. It is the only place that needs to know
// about both the workflow node interfaces and the dag package.
package dagbridge

import (
	"context"
	"io"

	"github.com/superagent-ai/superagent-base/backend/domain/workflow/internal/nodes"
	"github.com/superagent-ai/superagent-base/backend/pkg/dag"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

// Wrap converts a workflow node into a dag.NodeExecutor. It inspects
// which interfaces the node implements and picks the right adapter.
// opts are passed to Invoke/Stream when the node accepts NodeOption.
func Wrap(node any, opts ...nodes.NodeOption) dag.NodeExecutor {
	// Try streaming first (if the node implements StreamableNodeWOpt).
	if sn, ok := node.(nodes.StreamableNodeWOpt); ok {
		return &streamWOptAdapter{node: sn, opts: opts}
	}
	if sn, ok := node.(nodes.StreamableNode); ok {
		return &streamAdapter{node: sn}
	}
	// Then invokable with options.
	if in, ok := node.(nodes.InvokableNodeWOpt); ok {
		return &invokeWOptAdapter{node: in, opts: opts}
	}
	// Finally plain invokable.
	if in, ok := node.(nodes.InvokableNode); ok {
		return &invokeAdapter{node: in}
	}
	// Fallback: treat as a raw dag.NodeExecutor.
	if ne, ok := node.(dag.NodeExecutor); ok {
		return ne
	}
	return nil
}

// invokeAdapter wraps an InvokableNode as a dag.NodeExecutor.
type invokeAdapter struct {
	node nodes.InvokableNode
}

func (a *invokeAdapter) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	return a.node.Invoke(ctx, input)
}

// invokeWOptAdapter wraps an InvokableNodeWOpt as a dag.NodeExecutor.
type invokeWOptAdapter struct {
	node nodes.InvokableNodeWOpt
	opts []nodes.NodeOption
}

func (a *invokeWOptAdapter) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	return a.node.Invoke(ctx, input, a.opts...)
}

// streamAdapter wraps a StreamableNode as a dag.StreamingNodeExecutor.
type streamAdapter struct {
	node nodes.StreamableNode
}

func (a *streamAdapter) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	sr, err := a.node.Stream(ctx, input)
	if err != nil {
		return nil, err
	}
	return drainStream(sr)
}

func (a *streamAdapter) Stream(ctx context.Context, input map[string]any) (<-chan dag.StreamChunk, error) {
	sr, err := a.node.Stream(ctx, input)
	if err != nil {
		return nil, err
	}
	return streamToChan(sr), nil
}

// streamWOptAdapter wraps a StreamableNodeWOpt as a dag.StreamingNodeExecutor.
type streamWOptAdapter struct {
	node nodes.StreamableNodeWOpt
	opts []nodes.NodeOption
}

func (a *streamWOptAdapter) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	sr, err := a.node.Stream(ctx, input, a.opts...)
	if err != nil {
		return nil, err
	}
	return drainStream(sr)
}

func (a *streamWOptAdapter) Stream(ctx context.Context, input map[string]any) (<-chan dag.StreamChunk, error) {
	sr, err := a.node.Stream(ctx, input, a.opts...)
	if err != nil {
		return nil, err
	}
	return streamToChan(sr), nil
}

// drainStream consumes an eino StreamReader and returns the merged
// final output map.
func drainStream(sr *wfcompose.StreamReader[map[string]any]) (map[string]any, error) {
	if sr == nil {
		return nil, nil
	}
	defer sr.Close()
	var result map[string]any
	for {
		chunk, err := sr.Recv()
		if err != nil {
			if err == io.EOF {
				return result, nil
			}
			return result, err
		}
		if chunk == nil {
			continue
		}
		// Merge: later chunks override earlier ones for the same key.
		if result == nil {
			result = make(map[string]any)
		}
		for k, v := range chunk {
			result[k] = v
		}
	}
}

// streamToChan converts an eino StreamReader into a dag.StreamChunk
// channel. The channel is closed when the stream is exhausted.
func streamToChan(sr *wfcompose.StreamReader[map[string]any]) <-chan dag.StreamChunk {
	// Wrap to wfcompose for potential lossless unwrap, then use as a
	// plain reader — we just need Recv/Close.
	_ = sr

	ch := make(chan dag.StreamChunk, 10)
	go func() {
		defer close(ch)
		if sr == nil {
			return
		}
		defer sr.Close()
		for {
			chunk, err := sr.Recv()
			if err != nil {
				if err != io.EOF {
					ch <- dag.StreamChunk{Err: err}
				}
				return
			}
			if chunk == nil {
				continue
			}
			ch <- dag.StreamChunk{Data: chunk}
		}
	}()
	return ch
}
