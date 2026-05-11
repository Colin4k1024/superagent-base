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

package mem0_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
	"github.com/superagent-ai/superagent-base/backend/pkg/memory/mem0"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

// mockServer wires up a simple httptest server. The caller provides a handler
// per path that returns a JSON body and optional status code.
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
			// fallback: try path-only match
			h, ok = ms.handlers[r.URL.Path]
		}
		if !ok {
			http.Error(w, "no handler for "+key, http.StatusNotFound)
			return
		}
		h(w, r)
	}))
	t.Cleanup(ms.ts.Close)
	return ms
}

// on registers a handler for "METHOD /path".
func (ms *mockServer) on(method, path string, h http.HandlerFunc) {
	ms.handlers[method+" "+path] = h
}

// jsonResponse writes v as JSON with the given status.
func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// newBackend builds a mem0Backend wired to the mock server.
func newBackend(t *testing.T, ms *mockServer) memory.Backend {
	t.Helper()
	b := mem0.NewBackendForTest(ms.ts.Client(), ms.ts.URL)
	return b
}

// ── Factory / Init ────────────────────────────────────────────────────────────

func TestFactory_Mem0Registered(t *testing.T) {
	b, err := memory.New(memory.BackendConfig{Type: "mem0"})
	require.NoError(t, err)
	assert.Equal(t, "mem0", b.Name())
}

func TestInit_MissingEndpoint(t *testing.T) {
	b, _ := memory.New(memory.BackendConfig{Type: "mem0"})
	err := b.Init(context.Background(), memory.BackendConfig{APIKey: "key"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint")
}

func TestInit_MissingAPIKey(t *testing.T) {
	b, _ := memory.New(memory.BackendConfig{Type: "mem0"})
	err := b.Init(context.Background(), memory.BackendConfig{Endpoint: "http://x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api_key")
}

// ── ShortTermMemory ───────────────────────────────────────────────────────────

func TestAddMessage_RequestFormat(t *testing.T) {
	ms := newMockServer(t)
	var capturedBody AddMemoryRequestCapture
	ms.on(http.MethodPost, "/v1/memories", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))
		jsonResponse(w, 201, map[string]any{"id": "m1"})
	})

	b := newBackend(t, ms)
	err := b.AddMessage(context.Background(), "session-42", memory.Message{
		Role:    "user",
		Content: "hello world",
	})
	require.NoError(t, err)

	require.Len(t, capturedBody.Messages, 1)
	assert.Equal(t, "user", capturedBody.Messages[0].Role)
	assert.Equal(t, "hello world", capturedBody.Messages[0].Content)
	assert.Equal(t, "session-42", capturedBody.RunID)
}

func TestGetMessages_ResponseParsing(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodPost, "/v1/memories/search", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, []map[string]any{
			{
				"id":     "m1",
				"memory": "hello",
				"metadata": map[string]any{
					"_role": "user",
					"_ts":   float64(1000),
				},
			},
			{
				"id":     "m2",
				"memory": "hi there",
				"metadata": map[string]any{
					"_role": "assistant",
					"_ts":   float64(2000),
				},
			},
		})
	})

	b := newBackend(t, ms)
	msgs, err := b.GetMessages(context.Background(), "session-1", memory.GetMessagesOpts{})
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	// Should be sorted ascending by timestamp.
	assert.Equal(t, "user", msgs[0].Role)
	assert.Equal(t, "assistant", msgs[1].Role)
}

func TestGetMessages_BeforeFilter(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodPost, "/v1/memories/search", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, []map[string]any{
			{"id": "m1", "memory": "old", "metadata": map[string]any{"_ts": float64(500), "_role": "user"}},
			{"id": "m2", "memory": "new", "metadata": map[string]any{"_ts": float64(1500), "_role": "user"}},
		})
	})

	b := newBackend(t, ms)
	msgs, err := b.GetMessages(context.Background(), "s", memory.GetMessagesOpts{Before: 1000})
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "old", msgs[0].Content)
}

func TestClearSession(t *testing.T) {
	ms := newMockServer(t)
	deleted := []string{}

	ms.on(http.MethodPost, "/v1/memories/search", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, []map[string]any{
			{"id": "m1", "memory": "msg1", "metadata": map[string]any{}},
			{"id": "m2", "memory": "msg2", "metadata": map[string]any{}},
		})
	})
	ms.on(http.MethodDelete, "/v1/memories/m1", func(w http.ResponseWriter, r *http.Request) {
		deleted = append(deleted, "m1")
		w.WriteHeader(204)
	})
	ms.on(http.MethodDelete, "/v1/memories/m2", func(w http.ResponseWriter, r *http.Request) {
		deleted = append(deleted, "m2")
		w.WriteHeader(204)
	})

	b := newBackend(t, ms)
	err := b.ClearSession(context.Background(), "session-x")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"m1", "m2"}, deleted)
}

// ── LongTermMemory ────────────────────────────────────────────────────────────

func TestAdd_ReturnsID(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodPost, "/v1/memories", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 201, map[string]any{"id": "entry-99"})
	})

	b := newBackend(t, ms)
	id, err := b.Add(context.Background(), "user-1", "I prefer Go", nil)
	require.NoError(t, err)
	assert.Equal(t, "entry-99", id)
}

func TestSearch_ScoreFiltering(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodPost, "/v1/memories/search", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, []map[string]any{
			{"id": "a", "memory": "Go is great", "score": 0.9, "metadata": map[string]any{}},
			{"id": "b", "memory": "Python too", "score": 0.3, "metadata": map[string]any{}},
		})
	})

	b := newBackend(t, ms)
	results, err := b.Search(context.Background(), "u1", "Go", memory.SearchOpts{Threshold: 0.5})
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "a", results[0].ID)
}

func TestGetAll_Pagination(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodGet, "/v1/memories", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "u1", r.URL.Query().Get("user_id"))
		assert.Equal(t, "5", r.URL.Query().Get("limit"))
		assert.Equal(t, "10", r.URL.Query().Get("offset"))
		jsonResponse(w, 200, []map[string]any{
			{"id": "x", "memory": "data", "metadata": map[string]any{}},
		})
	})

	b := newBackend(t, ms)
	entries, err := b.GetAll(context.Background(), "u1", memory.ListOpts{Limit: 5, Offset: 10})
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func TestUpdate(t *testing.T) {
	ms := newMockServer(t)
	var body map[string]string
	ms.on(http.MethodPut, "/v1/memories/m42", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(200)
	})

	b := newBackend(t, ms)
	err := b.Update(context.Background(), "m42", "new content")
	require.NoError(t, err)
	assert.Equal(t, "new content", body["memory"])
}

func TestDelete(t *testing.T) {
	ms := newMockServer(t)
	called := false
	ms.on(http.MethodDelete, "/v1/memories/m7", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(204)
	})

	b := newBackend(t, ms)
	err := b.Delete(context.Background(), "m7")
	require.NoError(t, err)
	assert.True(t, called)
}

// ── AgentStateMemory ─────────────────────────────────────────────────────────

func TestSetState_UpsertSemantics(t *testing.T) {
	ms := newMockServer(t)
	searchCalls := 0
	deleteCalled := false

	ms.on(http.MethodPost, "/v1/memories/search", func(w http.ResponseWriter, r *http.Request) {
		searchCalls++
		if searchCalls == 1 {
			// First call: return existing entry with the same key.
			jsonResponse(w, 200, []map[string]any{
				{"id": "old-1", "memory": `"old_value"`,
					"metadata": map[string]any{"_state_key": "counter"}},
			})
		} else {
			jsonResponse(w, 200, []map[string]any{})
		}
	})
	ms.on(http.MethodDelete, "/v1/memories/old-1", func(w http.ResponseWriter, r *http.Request) {
		deleteCalled = true
		w.WriteHeader(204)
	})
	ms.on(http.MethodPost, "/v1/memories", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 201, map[string]any{"id": "new-1"})
	})

	b := newBackend(t, ms)
	err := b.SetState(context.Background(), "agent-1", "counter", 42)
	require.NoError(t, err)
	assert.True(t, deleteCalled, "old entry should have been deleted")
}

func TestGetState_Found(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodPost, "/v1/memories/search", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, []map[string]any{
			{"id": "s1", "memory": `42`, "metadata": map[string]any{"_state_key": "score"}},
		})
	})

	b := newBackend(t, ms)
	val, ok, err := b.GetState(context.Background(), "agent-1", "score")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, float64(42), val)
}

func TestGetState_Missing(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodPost, "/v1/memories/search", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, []map[string]any{})
	})

	b := newBackend(t, ms)
	_, ok, err := b.GetState(context.Background(), "agent-1", "missing")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestDeleteState(t *testing.T) {
	ms := newMockServer(t)
	deleted := false
	ms.on(http.MethodPost, "/v1/memories/search", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, []map[string]any{
			{"id": "s99", "memory": `"v"`, "metadata": map[string]any{"_state_key": "k"}},
		})
	})
	ms.on(http.MethodDelete, "/v1/memories/s99", func(w http.ResponseWriter, r *http.Request) {
		deleted = true
		w.WriteHeader(204)
	})

	b := newBackend(t, ms)
	err := b.DeleteState(context.Background(), "agent-1", "k")
	require.NoError(t, err)
	assert.True(t, deleted)
}

func TestGetAllState(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodPost, "/v1/memories/search", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, []map[string]any{
			{"id": "1", "memory": `"hello"`, "metadata": map[string]any{"_state_key": "name"}},
			{"id": "2", "memory": `99`, "metadata": map[string]any{"_state_key": "count"}},
		})
	})

	b := newBackend(t, ms)
	state, err := b.GetAllState(context.Background(), "agent-2")
	require.NoError(t, err)
	assert.Equal(t, "hello", state["name"])
	assert.Equal(t, float64(99), state["count"])
}

// ── API Error Handling ────────────────────────────────────────────────────────

func TestAPIError_PropagatesUpward(t *testing.T) {
	ms := newMockServer(t)
	ms.on(http.MethodPost, "/v1/memories", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"unauthorized"}`, http.StatusUnauthorized)
	})

	b := newBackend(t, ms)
	_, err := b.Add(context.Background(), "u1", "content", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

// ── Helper types ──────────────────────────────────────────────────────────────

// AddMemoryRequestCapture matches the wire format of AddMemoryRequest.
type AddMemoryRequestCapture struct {
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	UserID  string         `json:"user_id"`
	AgentID string         `json:"agent_id"`
	RunID   string         `json:"run_id"`
	Meta    map[string]any `json:"metadata"`
}
