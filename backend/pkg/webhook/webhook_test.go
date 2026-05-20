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

package webhook_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/superagent-ai/superagent-base/backend/pkg/webhook"
)

// ─── Store CRUD ─────────────────────────────────────────────────────────────

func TestMemoryStore_CRUD(t *testing.T) {
	s := webhook.NewMemoryStore()

	sub := &webhook.Subscription{
		ID:     "sub-1",
		URL:    "https://example.com/hook",
		Events: []webhook.EventType{webhook.EventAgentRunCompleted},
		Secret: "s3cr3t",
		Active: true,
	}

	// Create
	if err := s.CreateSubscription(sub); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Get
	got, err := s.GetSubscription("sub-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.URL != sub.URL {
		t.Errorf("URL mismatch: want %q got %q", sub.URL, got.URL)
	}

	// List
	list := s.ListSubscriptions()
	if len(list) != 1 {
		t.Errorf("expected 1 subscription, got %d", len(list))
	}

	// Update
	sub.URL = "https://example.com/hook2"
	if err := s.UpdateSubscription("sub-1", sub); err != nil {
		t.Fatalf("update: %v", err)
	}
	updated, _ := s.GetSubscription("sub-1")
	if updated.URL != "https://example.com/hook2" {
		t.Errorf("update did not persist: got %q", updated.URL)
	}

	// Delete
	if err := s.DeleteSubscription("sub-1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.GetSubscription("sub-1"); !errors.Is(err, webhook.ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMemoryStore_NotFound(t *testing.T) {
	s := webhook.NewMemoryStore()
	if _, err := s.GetSubscription("nonexistent"); !errors.Is(err, webhook.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if err := s.UpdateSubscription("nonexistent", &webhook.Subscription{}); !errors.Is(err, webhook.ErrNotFound) {
		t.Errorf("expected ErrNotFound on update, got %v", err)
	}
	if err := s.DeleteSubscription("nonexistent"); !errors.Is(err, webhook.ErrNotFound) {
		t.Errorf("expected ErrNotFound on delete, got %v", err)
	}
}

func TestMemoryStore_DeliveryLogs(t *testing.T) {
	s := webhook.NewMemoryStore()
	for i := 0; i < 5; i++ {
		_ = s.AddDeliveryLog(&webhook.DeliveryLog{
			ID:             "log",
			SubscriptionID: "sub-x",
			EventType:      webhook.EventAgentRunCompleted,
			StatusCode:     200,
			Attempt:        i + 1,
		})
	}

	all := s.GetDeliveryLogs("sub-x", 0)
	if len(all) != 5 {
		t.Errorf("expected 5 logs, got %d", len(all))
	}
	recent := s.GetDeliveryLogs("sub-x", 2)
	if len(recent) != 2 {
		t.Errorf("expected 2 logs with limit=2, got %d", len(recent))
	}
}

// ─── Signature ───────────────────────────────────────────────────────────────

func TestSign_Verify(t *testing.T) {
	secret := "my-secret"
	ts := int64(1700000000000)
	body := []byte(`{"type":"agent.run.completed"}`)

	sig := webhook.Sign(secret, ts, body)
	if sig == "" {
		t.Fatal("Sign returned empty string")
	}
	if !webhook.Verify(secret, ts, body, sig) {
		t.Error("Verify returned false for valid signature")
	}
	if webhook.Verify(secret, ts+1, body, sig) {
		t.Error("Verify should fail with different timestamp")
	}
	if webhook.Verify("wrong-secret", ts, body, sig) {
		t.Error("Verify should fail with wrong secret")
	}
	if webhook.Verify(secret, ts, []byte(`tampered`), sig) {
		t.Error("Verify should fail with tampered body")
	}
}

// ─── Dispatcher Emit + delivery ──────────────────────────────────────────────

func TestDispatcher_Emit_Delivery(t *testing.T) {
	var received atomic.Int32

	// httptest server that counts received requests.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var evt webhook.Event
		if err := json.Unmarshal(body, &evt); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := webhook.NewMemoryStore()
	_ = store.CreateSubscription(&webhook.Subscription{
		ID:     "sub-deliver",
		URL:    srv.URL,
		Events: []webhook.EventType{webhook.EventAgentRunCompleted},
		Active: true,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := webhook.NewDispatcher(store, 2)
	d.Start(ctx)

	d.Emit(ctx, webhook.Event{
		ID:   "evt-1",
		Type: webhook.EventAgentRunCompleted,
		Data: map[string]string{"agent": "test-agent"},
	})

	// Wait up to 2 s for delivery.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if received.Load() >= 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if received.Load() < 1 {
		t.Error("expected at least 1 delivery, got 0")
	}

	d.Stop()
}

// ─── Retry logic ─────────────────────────────────────────────────────────────

func TestDispatcher_Retry_ThenSuccess(t *testing.T) {
	var attempts atomic.Int32

	// Fail the first two calls, succeed on the third.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := webhook.NewMemoryStore()
	_ = store.CreateSubscription(&webhook.Subscription{
		ID:     "sub-retry",
		URL:    srv.URL,
		Events: []webhook.EventType{webhook.EventAgentRunFailed},
		Active: true,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := webhook.NewDispatcher(store, 1)
	d.Start(ctx)

	d.Emit(ctx, webhook.Event{
		ID:   "evt-retry",
		Type: webhook.EventAgentRunFailed,
		Data: nil,
	})

	// Allow enough time for all 3 attempts (1s + 5s back-off = ~6s).
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if attempts.Load() >= 3 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if attempts.Load() < 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}

	// Delivery logs should contain 3 entries (2 failures + 1 success).
	logs := store.GetDeliveryLogs("sub-retry", 0)
	if len(logs) < 3 {
		t.Errorf("expected at least 3 delivery log entries, got %d", len(logs))
	}

	d.Stop()
}

// ─── Event type filtering ─────────────────────────────────────────────────────

func TestDispatcher_Emit_Filtering(t *testing.T) {
	var received atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store := webhook.NewMemoryStore()
	// Subscribed only to agent.created — should NOT receive tool.call.completed.
	_ = store.CreateSubscription(&webhook.Subscription{
		ID:     "sub-filter",
		URL:    srv.URL,
		Events: []webhook.EventType{webhook.EventAgentCreated},
		Active: true,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := webhook.NewDispatcher(store, 1)
	d.Start(ctx)

	d.Emit(ctx, webhook.Event{Type: webhook.EventToolCallCompleted})

	// Wait briefly to confirm no delivery.
	time.Sleep(200 * time.Millisecond)
	if received.Load() != 0 {
		t.Errorf("expected 0 deliveries for non-subscribed event, got %d", received.Load())
	}

	d.Stop()
}
