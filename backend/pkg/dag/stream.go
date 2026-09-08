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

package dag

import (
	"context"
	"io"
)

// StreamChunk is one element of streaming output from a node.
type StreamChunk struct {
	NodeKey NodeKey
	Data    map[string]any
	Err     error
}

// StreamingNodeExecutor is an optional interface that nodes can implement
// to provide streaming output. If a node implements both NodeExecutor and
// StreamingNodeExecutor, the executor will prefer Stream over Execute
// when a StreamHandler is configured.
type StreamingNodeExecutor interface {
	// Stream produces streaming output chunks. The channel must be closed
	// when the stream is complete or an error occurs. If io.EOF is sent
	// as the error, it signals successful completion.
	Stream(ctx context.Context, input map[string]any) (<-chan StreamChunk, error)
}

// isStreamingNode returns true if the node's executor implements
// StreamingNodeExecutor.
func isStreamingNode(n NodeExecutor) bool {
	_, ok := n.(StreamingNodeExecutor)
	return ok
}

// drainStream consumes a streaming node's output channel, calls the
// stream handler for each chunk, and returns the merged final output.
// The last non-nil chunk's Data is returned as the node's output.
func drainStream(ctx context.Context, ch <-chan StreamChunk, handler StreamHandler, key NodeKey) (map[string]any, error) {
	var lastData map[string]any
	for {
		select {
		case <-ctx.Done():
			return lastData, ctx.Err()
		case chunk, ok := <-ch:
			if !ok {
				return lastData, nil
			}
			if chunk.Err != nil {
				if chunk.Err == io.EOF {
					return lastData, nil
				}
				return lastData, chunk.Err
			}
			lastData = chunk.Data
			if handler != nil && chunk.Data != nil {
				handler(key, chunk.Data)
			}
		}
	}
}
