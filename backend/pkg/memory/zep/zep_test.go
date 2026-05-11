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

package zep_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
	"github.com/superagent-ai/superagent-base/backend/pkg/memory/zep"
)

// ── Test helpers ──────────────────────────────────────────────────────────────

type mockServer struct {
	ts       *httptest.Server
	handlers map[string]http.HandlerFunc
}

func newMockServer(t *testing.T) *mockServer {
	t.Helper()
	ms := &mockServer{handlers: map[string]http.HandlerFunc{}}
	ms.ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		h, ok := ms.handlers[key]
		if !ok {
			http.Error(w, "no handler for "+key, http.StatusNotFound)
			return
		}
		h(w, r)
	}))
	t.Cleanup(ms.ts.Close)
	return ms
}

func (ms *mockServer) on(method, path string, h http.HandlerFunc) {
	ms.handlers[method+" "+path] = h
}

func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func newBackend(t *testing.T, ms *mockServer) memory.Backend {
	t.Helper()
	return zep.NewBackendForTest(ms.ts.Client(), ms.ts.URL)
}

// ── Factory / Init ────────────────────────────────────────────────────────────

func TestFactory_ZepRegistered(t *testing.T) {
	b, err := memory.New(memory.BackendConfig{Type: "zep"})
	require.NoError(t, err)
	assert.Equal(t, "zep", b.Name())
}

func TestInit_MissingEndpoint(t *testing.T) {
	b, _ := memory.New(memory.BackendConfig{Type: "zep"})
	err := b.Init(context.Background(), memory.BackendConfig{APIKey: "k"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint")
}

func TestInit_MissingAPIKey(t *testing.T) {
	b, _ := memory.New(memory.BackendConfig{Type: "zep"})
	err := b.Init(context.Background(), memory.BackendConfig{Endpoint: "http://x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_key")
}

// ── ShortTermMemory ───────────────────────────────────────────────────────────

func TestAddMessage_CreatesSessionAndAddsMemory(t *testing.T) {
	ms := newMockServer(t)
	sessionCreated := false
	memorySent := false

	ms.on(http.MethodPost, "/api/v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		sessionCreated = true
		jsonResponse(w, 200, map[string]any{"session_id": "s1"})
	})
	ms.on(http.MethodPost, "/api/v2/sessions/s1/memory", func(w http.ResponseWriter, r *http.Request) {
		memorySent = true
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(200)
	})

	b := newBackend(t, ms)
	err := b.AddMessage(context.Background(), "s1", memory.Message{Role: "user", Content: "hello"})
	require.NoError(t, err)
	assert.True(t, sessionCreated)
	assert.True(t, memorySent)
}

func TestAddMessage_409ConflictIgnored(t *testing.T) {
	ms := newMockServer(t)
	// Simulate session already existing (409).
	ms.on(http.MethodPost, "/api/v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"session already exists"}`, http.StatusConflict)
	})
	ms.on(http.MethodPost, "/api/v2/sessions/s2/memory", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	b := newBackend(t, ms)
	err := b.AddMessage(context.Background(), "s2", memory.Message{Role: "user", Content: "hi"})
	require.NoError(t, err)
}

func TestGetMessages_ParsesResponse(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodGet, "/api/v2/sessions/s1/memory", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]any{
			"messages": []map[string]any{
				{"uuid": "m1", "role": "user", "content": "hello", "metadata": map[string]any{"_ts": float64(1000)}},
				{"uuid": "m2", "role": "assistant", "content": "hi", "metadata": map[string]any{"_ts": float64(2000)}},
			},
		})
	})

	b := newBackend(t, ms)
	msgs, err := b.GetMessages(context.Background(), "s1", memory.GetMessagesOpts{})
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	assert.Equal(t, "user", msgs[0].Role)
	assert.Equal(t, "hello", msgs[0].Content)
}

func TestGetMessages_BeforeFilter(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodGet, "/api/v2/sessions/s1/memory", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]any{
			"messages": []map[string]any{
				{"role": "user", "content": "old", "metadata": map[string]any{"_ts": float64(500)}},
				{"role": "user", "content": "new", "metadata": map[string]any{"_ts": float64(1500)}},
			},
		})
	})

	b := newBackend(t, ms)
	msgs, err := b.GetMessages(context.Background(), "s1", memory.GetMessagesOpts{Before: 1000})
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "old", msgs[0].Content)
}

func TestClearSession_DeletesSession(t *testing.T) {
	ms := newMockServer(t)
	deleted := false
	ms.on(http.MethodDelete, "/api/v2/sessions/s1", func(w http.ResponseWriter, r *http.Request) {
		deleted = true
		w.WriteHeader(200)
	})

	b := newBackend(t, ms)
	err := b.ClearSession(context.Background(), "s1")
	require.NoError(t, err)
	assert.True(t, deleted)
}

// ── LongTermMemory ────────────────────────────────────────────────────────────

func TestAdd_EnsuresSessionAndAddsMessage(t *testing.T) {
	ms := newMockServer(t)
	ensureCalled := false
	addCalled := false

	ms.on(http.MethodPost, "/api/v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		ensureCalled = true
		jsonResponse(w, 200, map[string]any{})
	})
	ms.on(http.MethodPost, "/api/v2/sessions/lt-user1/memory", func(w http.ResponseWriter, r *http.Request) {
		addCalled = true
		w.WriteHeader(200)
	})

	b := newBackend(t, ms)
	id, err := b.Add(context.Background(), "user1", "I like Go", nil)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.True(t, ensureCalled)
	assert.True(t, addCalled)
}

func TestSearch_ScoreFiltering(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodPost, "/api/v2/sessions/lt-u1/search", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, []map[string]any{
			{"message": map[string]any{"uuid": "a", "content": "Go is great"}, "score": 0.9},
			{"message": map[string]any{"uuid": "b", "content": "Python too"}, "score": 0.2},
		})
	})

	b := newBackend(t, ms)
	results, err := b.Search(context.Background(), "u1", "Go", memory.SearchOpts{Threshold: 0.5})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Go is great", results[0].Content)
}

func TestGetAll_Pagination(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodGet, "/api/v2/sessions/lt-u1/memory", func(w http.ResponseWriter, r *http.Request) {
		msgs := make([]map[string]any, 5)
		for i := range msgs {
			msgs[i] = map[string]any{"uuid": fmt.Sprintf("m%d", i), "role": "user", "content": "data"}
		}
		jsonResponse(w, 200, map[string]any{"messages": msgs})
	})

	b := newBackend(t, ms)
	entries, err := b.GetAll(context.Background(), "u1", memory.ListOpts{Limit: 2, Offset: 2})
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

// ── AgentStateMemory ──────────────────────────────────────────────────────────

func TestSetState_UpdatesMetadata(t *testing.T) {
	ms := newMockServer(t)
	ensureCalled := false
	var patchBody map[string]any

	ms.on(http.MethodPost, "/api/v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		ensureCalled = true
		jsonResponse(w, 200, map[string]any{})
	})
	ms.on(http.MethodPatch, "/api/v2/sessions/as-agent1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&patchBody)
		w.WriteHeader(200)
	})

	b := newBackend(t, ms)
	err := b.SetState(context.Background(), "agent1", "score", 42)
	require.NoError(t, err)
	assert.True(t, ensureCalled)
	meta, _ := patchBody["metadata"].(map[string]any)
	require.NotNil(t, meta)
	assert.Contains(t, meta, "agent_state.score")
}

func TestGetState_Found(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodGet, "/api/v2/sessions/as-agent1", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]any{
			"session_id": "as-agent1",
			"metadata": map[string]any{
				"agent_state.score": "42",
			},
		})
	})

	b := newBackend(t, ms)
	val, ok, err := b.GetState(context.Background(), "agent1", "score")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, float64(42), val)
}

func TestGetState_Missing(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodGet, "/api/v2/sessions/as-agent1", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]any{
			"session_id": "as-agent1",
			"metadata":   map[string]any{},
		})
	})

	b := newBackend(t, ms)
	_, ok, err := b.GetState(context.Background(), "agent1", "missing")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestDeleteState_NullsMetadataKey(t *testing.T) {
	ms := newMockServer(t)
	var patchBody map[string]any
	ms.on(http.MethodPatch, "/api/v2/sessions/as-agent1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&patchBody)
		w.WriteHeader(200)
	})

	b := newBackend(t, ms)
	err := b.DeleteState(context.Background(), "agent1", "score")
	require.NoError(t, err)
	meta, _ := patchBody["metadata"].(map[string]any)
	require.NotNil(t, meta)
	assert.Nil(t, meta["agent_state.score"])
}

func TestGetAllState_MultipleKeys(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodGet, "/api/v2/sessions/as-agent2", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]any{
			"session_id": "as-agent2",
			"metadata": map[string]any{
				"agent_state.name":  `"Alice"`,
				"agent_state.count": "7",
				"other_key":         "ignored",
			},
		})
	})

	b := newBackend(t, ms)
	state, err := b.GetAllState(context.Background(), "agent2")
	require.NoError(t, err)
	assert.Len(t, state, 2)
	assert.Equal(t, "Alice", state["name"])
	assert.Equal(t, float64(7), state["count"])
}

// ── Error handling ────────────────────────────────────────────────────────────

func TestAPIError_Propagates(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodPost, "/api/v2/sessions", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"server error"}`, http.StatusInternalServerError)
	})

	b := newBackend(t, ms)
	err := b.AddMessage(context.Background(), "s1", memory.Message{Role: "user", Content: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}
