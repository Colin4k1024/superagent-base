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

package hiagent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateConversation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/create_conversation" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Apikey") != "test-key" {
			t.Errorf("missing or wrong Apikey header: %s", r.Header.Get("Apikey"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("wrong content-type: %s", r.Header.Get("Content-Type"))
		}

		var req CreateConversationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.UserID != "user-123" {
			t.Errorf("unexpected UserID: %s", req.UserID)
		}

		resp := CreateConversationResponse{}
		resp.Conversation.AppConversationID = "conv-abc"
		resp.Conversation.ConversationName = "test conv"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		APIKey:  "test-key",
		AppKey:  "test-key",
	})

	convID, err := client.CreateConversation(context.Background(), "user-123")
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if convID != "conv-abc" {
		t.Errorf("got convID=%q, want %q", convID, "conv-abc")
	}
}

func TestChatQueryBlocking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat_query_v2" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req ChatQueryV2Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if req.ResponseMode != "blocking" {
			t.Errorf("expected blocking mode, got %q", req.ResponseMode)
		}
		if req.Query != "什么是RAG?" {
			t.Errorf("unexpected query: %s", req.Query)
		}

		resp := ChatQueryV2Response{
			Event:  "message",
			Answer: "RAG 是检索增强生成技术。",
			TaskID: "task-1",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		APIKey:  "test-key",
		AppKey:  "test-key",
	})

	resp, err := client.ChatQueryBlocking(context.Background(), &ChatQueryV2Request{
		AppConversationID: "conv-abc",
		Query:             "什么是RAG?",
		UserID:            "user-123",
	})
	if err != nil {
		t.Fatalf("ChatQueryBlocking: %v", err)
	}
	if resp.Answer != "RAG 是检索增强生成技术。" {
		t.Errorf("unexpected answer: %s", resp.Answer)
	}
}

func TestChatQueryBlocking_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"conversation not found"}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		BaseURL: server.URL,
		APIKey:  "test-key",
		AppKey:  "test-key",
	})

	_, err := client.ChatQueryBlocking(context.Background(), &ChatQueryV2Request{
		AppConversationID: "conv-expired",
		Query:             "test",
		UserID:            "user-123",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsSessionExpired(err) {
		t.Errorf("expected session expired, got: %v", err)
	}
}

func TestSessionManager_GetOrCreate(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := CreateConversationResponse{}
		resp.Conversation.AppConversationID = "conv-cached"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "k", AppKey: "k"})
	sm := NewSessionManager(client)

	id1, err := sm.GetOrCreate(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("first GetOrCreate: %v", err)
	}
	if id1 != "conv-cached" {
		t.Errorf("unexpected id: %s", id1)
	}

	id2, err := sm.GetOrCreate(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("second GetOrCreate: %v", err)
	}
	if id2 != "conv-cached" {
		t.Errorf("unexpected id: %s", id2)
	}

	if callCount != 1 {
		t.Errorf("expected 1 API call (cached), got %d", callCount)
	}
}

func TestSessionManager_Invalidate(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := CreateConversationResponse{}
		resp.Conversation.AppConversationID = "conv-new"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, APIKey: "k", AppKey: "k"})
	sm := NewSessionManager(client)

	sm.GetOrCreate(context.Background(), "user-1")
	sm.Invalidate("user-1")
	sm.GetOrCreate(context.Background(), "user-1")

	if callCount != 2 {
		t.Errorf("expected 2 API calls after invalidation, got %d", callCount)
	}
}
