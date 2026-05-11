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

package skill_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/superagent-ai/superagent-base/backend/pkg/skill"
)

// ─── LocalInvoker ─────────────────────────────────────────────────────────────

func TestLocalInvoker_RegisterAndInvoke(t *testing.T) {
	inv := skill.NewLocalInvoker()
	inv.Register("echo", func(_ context.Context, input map[string]any) (map[string]any, error) {
		return map[string]any{"echo": input["msg"]}, nil
	})

	got, err := inv.Invoke(context.Background(), "echo", map[string]any{"msg": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["echo"] != "hello" {
		t.Errorf("echo mismatch: got %v", got["echo"])
	}
}

func TestLocalInvoker_UnknownSkill(t *testing.T) {
	inv := skill.NewLocalInvoker()
	_, err := inv.Invoke(context.Background(), "missing", nil)
	if err == nil {
		t.Fatal("expected error for unregistered skill")
	}
}

// ─── HTTPInvoker ──────────────────────────────────────────────────────────────

func TestHTTPInvoker_RegisterAndInvoke(t *testing.T) {
	// Mock HTTP server that echoes the "msg" field from the request body.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"echo": body["msg"]})
	}))
	defer ts.Close()

	inv := skill.NewHTTPInvoker()
	inv.Register("echo", ts.URL)

	got, err := inv.Invoke(context.Background(), "echo", map[string]any{"msg": "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["echo"] != "world" {
		t.Errorf("echo mismatch: got %v", got["echo"])
	}
}

func TestHTTPInvoker_UnknownSkill(t *testing.T) {
	inv := skill.NewHTTPInvoker()
	_, err := inv.Invoke(context.Background(), "missing", nil)
	if err == nil {
		t.Fatal("expected error for unregistered skill")
	}
}

func TestHTTPInvoker_NonOKStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer ts.Close()

	inv := skill.NewHTTPInvoker()
	inv.Register("bad", ts.URL)

	_, err := inv.Invoke(context.Background(), "bad", nil)
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

// ─── CompositeInvoker ─────────────────────────────────────────────────────────

func TestCompositeInvoker_LocalFirst(t *testing.T) {
	local := skill.NewLocalInvoker()
	local.Register("greet", func(_ context.Context, input map[string]any) (map[string]any, error) {
		return map[string]any{"from": "local"}, nil
	})

	// HTTP server should not be reached.
	reached := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		json.NewEncoder(w).Encode(map[string]any{"from": "http"})
	}))
	defer ts.Close()

	httpInv := skill.NewHTTPInvoker()
	httpInv.Register("greet", ts.URL)

	comp := skill.NewCompositeInvoker(local, httpInv)
	got, err := comp.Invoke(context.Background(), "greet", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["from"] != "local" {
		t.Errorf("expected local, got %v", got["from"])
	}
	if reached {
		t.Error("HTTP invoker should not have been called")
	}
}

func TestCompositeInvoker_FallsBackToHTTP(t *testing.T) {
	local := skill.NewLocalInvoker() // no "remote" skill registered

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"from": "http"})
	}))
	defer ts.Close()

	httpInv := skill.NewHTTPInvoker()
	httpInv.Register("remote", ts.URL)

	comp := skill.NewCompositeInvoker(local, httpInv)
	got, err := comp.Invoke(context.Background(), "remote", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["from"] != "http" {
		t.Errorf("expected http fallback, got %v", got["from"])
	}
}
