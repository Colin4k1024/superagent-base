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
	"sync"
	"testing"

	"github.com/cloudwego/eino/schema"

	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
)

// inMemoryBackend is a test memory backend that records messages in order.
type inMemoryBackend struct {
	mu       sync.Mutex
	messages map[string][]memory.Message
}

func newInMemoryBackend() *inMemoryBackend {
	return &inMemoryBackend{messages: make(map[string][]memory.Message)}
}

func (b *inMemoryBackend) AddMessage(_ context.Context, sessionID string, msg memory.Message) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.messages[sessionID] = append(b.messages[sessionID], msg)
	return nil
}

func (b *inMemoryBackend) GetMessages(_ context.Context, sessionID string, opts memory.GetMessagesOpts) ([]memory.Message, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	msgs := b.messages[sessionID]
	if opts.Limit > 0 && len(msgs) > opts.Limit {
		msgs = msgs[len(msgs)-opts.Limit:]
	}
	result := make([]memory.Message, len(msgs))
	copy(result, msgs)
	return result, nil
}

func (b *inMemoryBackend) ClearSession(_ context.Context, _ string) error { return nil }

// LongTermMemory stubs
func (b *inMemoryBackend) Add(_ context.Context, _ string, _ string, _ map[string]any) (string, error) {
	return "", nil
}
func (b *inMemoryBackend) Search(_ context.Context, _ string, _ string, _ memory.SearchOpts) ([]memory.MemoryEntry, error) {
	return nil, nil
}
func (b *inMemoryBackend) Update(_ context.Context, _ string, _ string) error { return nil }
func (b *inMemoryBackend) Delete(_ context.Context, _ string) error           { return nil }
func (b *inMemoryBackend) GetAll(_ context.Context, _ string, _ memory.ListOpts) ([]memory.MemoryEntry, error) {
	return nil, nil
}

// AgentStateMemory stubs
func (b *inMemoryBackend) SetState(_ context.Context, _ string, _ string, _ any) error {
	return nil
}
func (b *inMemoryBackend) GetState(_ context.Context, _ string, _ string) (any, bool, error) {
	return nil, false, nil
}
func (b *inMemoryBackend) DeleteState(_ context.Context, _ string, _ string) error { return nil }
func (b *inMemoryBackend) GetAllState(_ context.Context, _ string) (map[string]any, error) {
	return nil, nil
}

// Lifecycle stubs
func (b *inMemoryBackend) Init(_ context.Context, _ memory.BackendConfig) error { return nil }
func (b *inMemoryBackend) Close() error                                         { return nil }
func (b *inMemoryBackend) Name() string                                         { return "in-memory-test" }

func TestBuildMessageHistory_DoesNotIncludeCurrentMessage(t *testing.T) {
	backend := newInMemoryBackend()
	ctx := context.Background()
	sessionID := "session-1"

	// Simulate a prior turn: user said "hello", assistant replied "hi".
	_ = backend.AddMessage(ctx, sessionID, memory.Message{Role: "user", Content: "hello", Timestamp: 1})
	_ = backend.AddMessage(ctx, sessionID, memory.Message{Role: "assistant", Content: "hi", Timestamp: 2})

	// Now simulate the correct ordering: build history BEFORE persisting the new message.
	msgs := buildMessageHistory(ctx, "You are helpful.", sessionID, backend)

	// At this point, history should have: system + user("hello") + assistant("hi") = 3 messages.
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages (system + 1 user + 1 assistant), got %d", len(msgs))
	}

	// Now persist the new user message (this is what Chat() does after buildMessageHistory).
	persistUserMessage(ctx, sessionID, "what's 2+2?", backend)

	// Append the new message to the slice (as Chat() does).
	msgs = append(msgs, schema.UserMessage("what's 2+2?"))

	// Verify: the new message appears exactly once in the slice.
	count := 0
	for _, m := range msgs {
		if m.Role == schema.User && m.Content == "what's 2+2?" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected current message to appear exactly once, appeared %d times", count)
	}
}

func TestBuildMessageHistory_NoDuplicationOnNextCall(t *testing.T) {
	backend := newInMemoryBackend()
	ctx := context.Background()
	sessionID := "session-2"

	// First turn: persist user + assistant.
	persistUserMessage(ctx, sessionID, "first question", backend)
	_ = backend.AddMessage(ctx, sessionID, memory.Message{Role: "assistant", Content: "first answer", Timestamp: 2})

	// Second turn: build history, then persist. Simulates the fixed Chat() flow.
	msgs := buildMessageHistory(ctx, "", sessionID, backend)
	persistUserMessage(ctx, sessionID, "second question", backend)
	msgs = append(msgs, schema.UserMessage("second question"))

	// History should have: user("first question") + assistant("first answer") = 2
	// Then appended: user("second question") = 1
	// Total: 3
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}

	// Verify "second question" appears exactly once.
	count := 0
	for _, m := range msgs {
		if m.Role == schema.User && m.Content == "second question" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 'second question' once, got %d", count)
	}
}

func TestBuildMessageHistory_EmptySession(t *testing.T) {
	backend := newInMemoryBackend()
	ctx := context.Background()

	msgs := buildMessageHistory(ctx, "system prompt", "new-session", backend)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message (system only), got %d", len(msgs))
	}
	if msgs[0].Role != schema.System {
		t.Errorf("expected system message, got role=%v", msgs[0].Role)
	}
}

func TestBuildMessageHistory_NoSystemPrompt(t *testing.T) {
	backend := newInMemoryBackend()
	ctx := context.Background()
	sessionID := "session-3"

	_ = backend.AddMessage(ctx, sessionID, memory.Message{Role: "user", Content: "hi", Timestamp: 1})

	msgs := buildMessageHistory(ctx, "", sessionID, backend)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message (user only, no system), got %d", len(msgs))
	}
	if msgs[0].Role != schema.User {
		t.Errorf("expected user message, got role=%v", msgs[0].Role)
	}
}

func TestPersistUserMessage_NilBackend(t *testing.T) {
	// Should not panic with nil backend.
	persistUserMessage(context.Background(), "sess", "msg", nil)
}

func TestPersistUserMessage_EmptySession(t *testing.T) {
	backend := newInMemoryBackend()
	// Should not persist with empty session ID.
	persistUserMessage(context.Background(), "", "msg", backend)
	if len(backend.messages) != 0 {
		t.Error("expected no messages persisted with empty session ID")
	}
}
