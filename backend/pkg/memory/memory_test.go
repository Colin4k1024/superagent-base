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

package memory_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superagent-ai/superagent-base/backend/infra/cache/impl/redis"
	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
	"github.com/superagent-ai/superagent-base/backend/pkg/memory/builtin"

	// Register mem0 backend so factory knows about it.
	_ "github.com/superagent-ai/superagent-base/backend/pkg/memory/mem0"
)

// newBuiltinBackend creates a builtin Backend wired to an in-process miniredis.
func newBuiltinBackend(t *testing.T) memory.Backend {
	t.Helper()
	mr := miniredis.RunT(t)
	c := redis.NewWithAddrAndPassword(mr.Addr(), "")

	b := builtin.NewWithCache(c)
	err := b.Init(context.Background(), memory.BackendConfig{Type: "builtin"})
	require.NoError(t, err)
	return b
}

// ── Factory ──────────────────────────────────────────────────────────────────

func TestFactory_UnknownType(t *testing.T) {
	_, err := memory.New(memory.BackendConfig{Type: "nonexistent"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestFactory_BuiltinRegistered(t *testing.T) {
	b, err := memory.New(memory.BackendConfig{Type: "builtin"})
	require.NoError(t, err)
	assert.Equal(t, "builtin", b.Name())
}

func TestFactory_Mem0Registered(t *testing.T) {
	b, err := memory.New(memory.BackendConfig{Type: "mem0"})
	require.NoError(t, err)
	assert.Equal(t, "mem0", b.Name())
}

// ── Builtin: ShortTermMemory ─────────────────────────────────────────────────

func TestBuiltin_AddAndGetMessages(t *testing.T) {
	ctx := context.Background()
	b := newBuiltinBackend(t)

	err := b.AddMessage(ctx, "session-1", memory.Message{Role: "user", Content: "hello"})
	require.NoError(t, err)

	err = b.AddMessage(ctx, "session-1", memory.Message{Role: "assistant", Content: "hi there"})
	require.NoError(t, err)

	msgs, err := b.GetMessages(ctx, "session-1", memory.GetMessagesOpts{})
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	assert.Equal(t, "user", msgs[0].Role)
	assert.Equal(t, "assistant", msgs[1].Role)
}

func TestBuiltin_GetMessages_Limit(t *testing.T) {
	ctx := context.Background()
	b := newBuiltinBackend(t)

	for i := 0; i < 5; i++ {
		require.NoError(t, b.AddMessage(ctx, "s", memory.Message{Role: "user", Content: "msg"}))
	}

	msgs, err := b.GetMessages(ctx, "s", memory.GetMessagesOpts{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, msgs, 3)
}

func TestBuiltin_ClearSession(t *testing.T) {
	ctx := context.Background()
	b := newBuiltinBackend(t)

	require.NoError(t, b.AddMessage(ctx, "s", memory.Message{Role: "user", Content: "hello"}))
	require.NoError(t, b.ClearSession(ctx, "s"))

	msgs, err := b.GetMessages(ctx, "s", memory.GetMessagesOpts{})
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

// ── Builtin: LongTermMemory ──────────────────────────────────────────────────

func TestBuiltin_AddAndGetAll(t *testing.T) {
	ctx := context.Background()
	b := newBuiltinBackend(t)

	id, err := b.Add(ctx, "user-1", "I love Go", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	entries, err := b.GetAll(ctx, "user-1", memory.ListOpts{})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "I love Go", entries[0].Content)
}

func TestBuiltin_Search(t *testing.T) {
	ctx := context.Background()
	b := newBuiltinBackend(t)

	_, err := b.Add(ctx, "user-1", "I love Go programming", nil)
	require.NoError(t, err)
	_, err = b.Add(ctx, "user-1", "Python is great too", nil)
	require.NoError(t, err)

	results, err := b.Search(ctx, "user-1", "Go", memory.SearchOpts{})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Content, "Go")
}

func TestBuiltin_GetAll_Pagination(t *testing.T) {
	ctx := context.Background()
	b := newBuiltinBackend(t)

	for i := 0; i < 5; i++ {
		_, err := b.Add(ctx, "user-p", "entry", nil)
		require.NoError(t, err)
	}

	page, err := b.GetAll(ctx, "user-p", memory.ListOpts{Limit: 2, Offset: 2})
	require.NoError(t, err)
	assert.Len(t, page, 2)
}

// ── Builtin: AgentStateMemory ────────────────────────────────────────────────

func TestBuiltin_SetAndGetState(t *testing.T) {
	ctx := context.Background()
	b := newBuiltinBackend(t)

	require.NoError(t, b.SetState(ctx, "agent-1", "counter", float64(42)))

	val, ok, err := b.GetState(ctx, "agent-1", "counter")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, float64(42), val)
}

func TestBuiltin_GetState_Missing(t *testing.T) {
	ctx := context.Background()
	b := newBuiltinBackend(t)

	_, ok, err := b.GetState(ctx, "agent-x", "missing")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestBuiltin_DeleteState(t *testing.T) {
	ctx := context.Background()
	b := newBuiltinBackend(t)

	require.NoError(t, b.SetState(ctx, "agent-1", "k", "v"))
	require.NoError(t, b.DeleteState(ctx, "agent-1", "k"))

	_, ok, err := b.GetState(ctx, "agent-1", "k")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestBuiltin_GetAllState(t *testing.T) {
	ctx := context.Background()
	b := newBuiltinBackend(t)

	require.NoError(t, b.SetState(ctx, "agent-2", "a", 1))
	require.NoError(t, b.SetState(ctx, "agent-2", "b", "hello"))

	state, err := b.GetAllState(ctx, "agent-2")
	require.NoError(t, err)
	assert.Len(t, state, 2)
}
