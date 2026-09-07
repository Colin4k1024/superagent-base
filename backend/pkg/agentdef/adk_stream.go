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

package agentdef

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"time"

	aclagent "github.com/superagent-ai/superagent-base/backend/pkg/agent"
	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
	"github.com/superagent-ai/superagent-base/backend/pkg/modelrouter"
)

// streamConsumerParams carries context for streaming consumption.
type streamConsumerParams struct {
	sessionID  string
	modelID    string
	provider   string
	memBackend memory.Backend
}

// consumeGoogleADKIterator drains a Google ADK Go EventIterator,
// streaming text content to ch. It handles both streaming and non-streaming
// message outputs, records latency metrics, persists the full assistant
// response to memory, and drains the iterator on context cancellation.
func consumeGoogleADKIterator(
	ctx context.Context,
	params streamConsumerParams,
	iter *aclagent.EventIterator,
	ch chan<- string,
) {
	defer close(ch)

	streamStart := time.Now()
	firstToken := true
	var fullResponse strings.Builder

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Printf("[agentdef] stream error session=%s: %v", params.sessionID, event.Err)
			select {
			case ch <- "[error] internal error occurred":
			case <-ctx.Done():
			}
			return
		}

		if event.MessageOutput == nil {
			continue
		}
		mv := event.MessageOutput

		if mv.IsStreaming && mv.MessageStream != nil {
			consumeACLMessageStream(ctx, mv.MessageStream, params, &fullResponse, &firstToken, streamStart, ch)
			if ctx.Err() != nil {
				return
			}
		} else if mv.Message != nil && mv.Message.Content != "" {
			if firstToken {
				modelrouter.RecordModelLatency(params.modelID, params.provider, time.Since(streamStart))
				firstToken = false
			}
			fullResponse.WriteString(mv.Message.Content)
			select {
			case ch <- mv.Message.Content:
			case <-ctx.Done():
				return
			}
		}
	}

	if params.memBackend != nil && params.sessionID != "" && fullResponse.Len() > 0 {
		_ = params.memBackend.AddMessage(ctx, params.sessionID, memory.Message{
			Role:      "assistant",
			Content:   fullResponse.String(),
			Timestamp: time.Now().Unix(),
		})
	}
}

// consumeACLMessageStream reads chunks from an ACL StreamReader and forwards them.
func consumeACLMessageStream(
	ctx context.Context,
	stream *llm.StreamReader,
	params streamConsumerParams,
	fullResponse *strings.Builder,
	firstToken *bool,
	streamStart time.Time,
	ch chan<- string,
) {
	for {
		chunk, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			log.Printf("[agentdef] stream recv error session=%s: %v", params.sessionID, recvErr)
			select {
			case ch <- "[error] internal error occurred":
			case <-ctx.Done():
			}
			break
		}
		if chunk != nil && chunk.Content != "" {
			if *firstToken {
				modelrouter.RecordModelLatency(params.modelID, params.provider, time.Since(streamStart))
				*firstToken = false
			}
			fullResponse.WriteString(chunk.Content)
			select {
			case ch <- chunk.Content:
			case <-ctx.Done():
				return
			}
		}
	}
}

// buildMessageHistory constructs the LLM message slice from system prompt and
// memory history using the framework-agnostic llm.Message type.
func buildMessageHistory(ctx context.Context, systemPrompt, sessionID string, memBackend memory.Backend) []*llm.Message {
	msgs := make([]*llm.Message, 0, 8)
	if systemPrompt != "" {
		msgs = append(msgs, llm.SystemMessage(systemPrompt))
	}

	if memBackend != nil && sessionID != "" {
		history, err := memBackend.GetMessages(ctx, sessionID, memory.GetMessagesOpts{Limit: 20})
		if err == nil {
			for _, m := range history {
				switch m.Role {
				case "user":
					msgs = append(msgs, llm.UserMessage(m.Content))
				case "assistant":
					msgs = append(msgs, llm.AssistantMessage(m.Content, nil))
				}
			}
		}
	}
	return msgs
}

// persistUserMessage saves the user message to the memory backend.
func persistUserMessage(ctx context.Context, sessionID, message string, memBackend memory.Backend) {
	if memBackend != nil && sessionID != "" {
		_ = memBackend.AddMessage(ctx, sessionID, memory.Message{
			Role:      "user",
			Content:   message,
			Timestamp: time.Now().Unix(),
		})
	}
}
