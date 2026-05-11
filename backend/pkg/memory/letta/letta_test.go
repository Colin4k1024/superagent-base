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

package letta_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
	"github.com/superagent-ai/superagent-base/backend/pkg/memory/letta"
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
	return letta.NewBackendForTest(ms.ts.Client(), ms.ts.URL)
}

// ── Factory / Init ────────────────────────────────────────────────────────────

func TestFactory_LettaRegistered(t *testing.T) {
	b, err := memory.New(memory.BackendConfig{Type: "letta"})
	require.NoError(t, err)
	assert.Equal(t, "letta", b.Name())
}

func TestInit_MissingEndpoint(t *testing.T) {
	b, _ := memory.New(memory.BackendConfig{Type: "letta"})
	err := b.Init(context.Background(), memory.BackendConfig{APIKey: "k"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint")
}

func TestInit_MissingAPIKey(t *testing.T) {
	b, _ := memory.New(memory.BackendConfig{Type: "letta"})
	err := b.Init(context.Background(), memory.BackendConfig{Endpoint: "http://x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_key")
}

// ── ShortTermMemory ───────────────────────────────────────────────────────────

func TestAddMessage_SendsToAgent(t *testing.T) {
	ms := newMockServer(t)
	var capturedBody map[string]any
	ms.on(http.MethodPost, "/v1/agents/session-1/messages", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		jsonResponse(w, 200, map[string]any{"messages": []map[string]any{}})
	})

	b := newBackend(t, ms)
	err := b.AddMessage(context.Background(), "session-1", memory.Message{
		Role:    "user",
		Content: "What is Go?",
	})
	require.NoError(t, err)

	msgs, _ := capturedBody["messages"].([]any)
	require.Len(t, msgs, 1)
	first := msgs[0].(map[string]any)
	assert.Equal(t, "What is Go?", first["content"])
}

func TestGetMessages_ReturnsEmpty(t *testing.T) {
	ms := newMockServer(t)
	b := newBackend(t, ms)
	msgs, err := b.GetMessages(context.Background(), "s1", memory.GetMessagesOpts{})
	require.NoError(t, err)
	assert.Empty(t, msgs)
}

func TestClearSession_SendsClearMessage(t *testing.T) {
	ms := newMockServer(t)
	var capturedBody map[string]any
	ms.on(http.MethodPost, "/v1/agents/session-1/messages", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		jsonResponse(w, 200, map[string]any{"messages": []map[string]any{}})
	})

	b := newBackend(t, ms)
	err := b.ClearSession(context.Background(), "session-1")
	require.NoError(t, err)

	msgs, _ := capturedBody["messages"].([]any)
	require.Len(t, msgs, 1)
	first := msgs[0].(map[string]any)
	assert.Contains(t, first["content"], "clear")
}

// ── LongTermMemory ────────────────────────────────────────────────────────────

func TestAdd_InsertsArchival(t *testing.T) {
	ms := newMockServer(t)
	var capturedBody map[string]any
	ms.on(http.MethodPost, "/v1/agents/user-1/archival", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		jsonResponse(w, 200, map[string]any{"id": "arch-42", "text": "I love Go"})
	})

	b := newBackend(t, ms)
	id, err := b.Add(context.Background(), "user-1", "I love Go", nil)
	require.NoError(t, err)
	assert.Equal(t, "arch-42", id)
	assert.Equal(t, "I love Go", capturedBody["text"])
}

func TestSearch_ArchivalSearch(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodGet, "/v1/agents/user-1/archival", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Go", r.URL.Query().Get("query"))
		jsonResponse(w, 200, []map[string]any{
			{"id": "p1", "text": "Go is awesome"},
			{"id": "p2", "text": "Python rocks"},
		})
	})

	b := newBackend(t, ms)
	entries, err := b.Search(context.Background(), "user-1", "Go", memory.SearchOpts{Limit: 5})
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, "Go is awesome", entries[0].Content)
}

func TestUpdate_DeletesAndReinserts(t *testing.T) {
	ms := newMockServer(t)
	deleteCalled := false
	insertCalled := false

	ms.on(http.MethodDelete, "/v1/agents/user-1/archival/old-id", func(w http.ResponseWriter, r *http.Request) {
		deleteCalled = true
		w.WriteHeader(200)
	})
	ms.on(http.MethodPost, "/v1/agents/user-1/archival", func(w http.ResponseWriter, r *http.Request) {
		insertCalled = true
		jsonResponse(w, 200, map[string]any{"id": "new-id", "text": "updated"})
	})

	b := newBackend(t, ms)
	err := b.Update(context.Background(), "user-1/old-id", "updated content")
	require.NoError(t, err)
	assert.True(t, deleteCalled)
	assert.True(t, insertCalled)
}

func TestDelete_RemovesPassage(t *testing.T) {
	ms := newMockServer(t)
	called := false
	ms.on(http.MethodDelete, "/v1/agents/agent-1/archival/passage-5", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(200)
	})

	b := newBackend(t, ms)
	err := b.Delete(context.Background(), "agent-1/passage-5")
	require.NoError(t, err)
	assert.True(t, called)
}

func TestGetAll_SearchWithEmptyQuery(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodGet, "/v1/agents/user-1/archival", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, []map[string]any{
			{"id": "a", "text": "one"},
			{"id": "b", "text": "two"},
			{"id": "c", "text": "three"},
		})
	})

	b := newBackend(t, ms)
	entries, err := b.GetAll(context.Background(), "user-1", memory.ListOpts{Offset: 1})
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

// ── AgentStateMemory ──────────────────────────────────────────────────────────

func TestSetState_UpdatesCoreMemory(t *testing.T) {
	ms := newMockServer(t)
	var patchBody map[string]any
	ms.on(http.MethodPatch, "/v1/agents/agent-1/core-memory/blocks/counter", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&patchBody)
		w.WriteHeader(200)
	})

	b := newBackend(t, ms)
	err := b.SetState(context.Background(), "agent-1", "counter", 99)
	require.NoError(t, err)
	assert.Equal(t, "99", patchBody["value"])
}

func TestGetState_ReadsCoreMemoryBlock(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodGet, "/v1/agents/agent-1/core-memory", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]any{
			"memory": map[string]any{
				"counter": map[string]any{"value": "42", "limit": 2000, "name": "counter", "label": "counter"},
				"name":    map[string]any{"value": `"Alice"`, "limit": 2000, "name": "name", "label": "name"},
			},
		})
	})

	b := newBackend(t, ms)
	val, ok, err := b.GetState(context.Background(), "agent-1", "counter")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, float64(42), val)
}

func TestGetState_Missing(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodGet, "/v1/agents/agent-1/core-memory", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]any{"memory": map[string]any{}})
	})

	b := newBackend(t, ms)
	_, ok, err := b.GetState(context.Background(), "agent-1", "missing")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestDeleteState_SetsEmptyString(t *testing.T) {
	ms := newMockServer(t)
	var patchBody map[string]any
	ms.on(http.MethodPatch, "/v1/agents/agent-1/core-memory/blocks/counter", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&patchBody)
		w.WriteHeader(200)
	})

	b := newBackend(t, ms)
	err := b.DeleteState(context.Background(), "agent-1", "counter")
	require.NoError(t, err)
	assert.Equal(t, "", patchBody["value"])
}

func TestGetAllState_MultipleBlocks(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodGet, "/v1/agents/agent-2/core-memory", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]any{
			"memory": map[string]any{
				"name":  map[string]any{"value": `"Bob"`, "limit": 2000, "name": "name", "label": "name"},
				"score": map[string]any{"value": "7", "limit": 2000, "name": "score", "label": "score"},
				"empty": map[string]any{"value": "", "limit": 2000, "name": "empty", "label": "empty"},
			},
		})
	})

	b := newBackend(t, ms)
	state, err := b.GetAllState(context.Background(), "agent-2")
	require.NoError(t, err)
	// empty block should be skipped
	assert.Len(t, state, 2)
	assert.Equal(t, "Bob", state["name"])
	assert.Equal(t, float64(7), state["score"])
}

// ── Error handling ────────────────────────────────────────────────────────────

func TestAPIError_Propagates(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodPost, "/v1/agents/session-1/messages", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"unauthorized"}`, http.StatusUnauthorized)
	})

	b := newBackend(t, ms)
	err := b.AddMessage(context.Background(), "session-1", memory.Message{Role: "user", Content: "hi"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}
