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

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
	"github.com/superagent-ai/superagent-base/backend/pkg/modelrouter"
)

// streamConsumerParams holds the common parameters for consuming an ADK iterator.
type streamConsumerParams struct {
	sessionID    string
	modelID      string
	provider     string
	memBackend   memory.Backend
}

// interruptHandler is called when an interrupt event is detected during iteration.
// It should emit the interrupt data to the channel and return true to signal early exit.
type interruptHandler func(ctx context.Context, event *adk.AgentEvent, ch chan<- string) bool

// consumeADKIterator drains an ADK AsyncIterator, streaming text content to ch.
// It handles both streaming and non-streaming message outputs, records latency
// metrics, persists the full assistant response to memory, and drains the
// iterator on context cancellation to prevent goroutine leaks.
func consumeADKIterator(
	ctx context.Context,
	params streamConsumerParams,
	iter *adk.AsyncIterator[*adk.AgentEvent],
	ch chan<- string,
	onInterrupt interruptHandler,
) {
	defer close(ch)

	// earlyExit tracks whether we left the loop before the iterator was
	// naturally exhausted. If so, drainIterator must be called to prevent
	// the ADK's internal goroutines from blocking forever.
	earlyExit := true
	defer func() {
		if earlyExit {
			drainIterator(iter)
		}
	}()

	streamStart := time.Now()
	firstToken := true
	var fullResponse strings.Builder

	for {
		event, ok := iter.Next()
		if !ok {
			earlyExit = false
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

		if onInterrupt != nil && event.Action != nil && event.Action.Interrupted != nil {
			if onInterrupt(ctx, event, ch) {
				return
			}
		}

		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		mv := event.Output.MessageOutput

		if mv.IsStreaming && mv.MessageStream != nil {
			consumeMessageStream(ctx, mv.MessageStream, params, &fullResponse, &firstToken, streamStart, ch)
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

// consumeMessageStream reads chunks from a streaming message and forwards them.
func consumeMessageStream(
	ctx context.Context,
	stream *schema.StreamReader[*schema.Message],
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

// drainIterator consumes remaining events from an iterator to allow its
// internal goroutines to exit cleanly. Called when the consumer exits early.
func drainIterator(iter *adk.AsyncIterator[*adk.AgentEvent]) {
	go func() {
		for {
			if _, ok := iter.Next(); !ok {
				break
			}
		}
	}()
}

// buildMessageHistory constructs the LLM message slice from system prompt and
// memory history. The current user message is NOT included — the caller appends
// it separately after this call. persistUserMessage should be called AFTER this
// function to avoid the current message appearing twice (once from history, once
// from the explicit append).
func buildMessageHistory(ctx context.Context, systemPrompt, sessionID string, memBackend memory.Backend) []*schema.Message {
	msgs := make([]*schema.Message, 0, 8)
	if systemPrompt != "" {
		msgs = append(msgs, schema.SystemMessage(systemPrompt))
	}

	if memBackend != nil && sessionID != "" {
		history, err := memBackend.GetMessages(ctx, sessionID, memory.GetMessagesOpts{Limit: 20})
		if err == nil {
			for _, m := range history {
				switch m.Role {
				case "user":
					msgs = append(msgs, schema.UserMessage(m.Content))
				case "assistant":
					msgs = append(msgs, schema.AssistantMessage(m.Content, nil))
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

