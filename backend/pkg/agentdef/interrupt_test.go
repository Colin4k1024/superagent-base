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

package agentdef_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/superagent-ai/superagent-base/backend/pkg/agentdef"
)

// ─── stub agent helpers ───────────────────────────────────────────────────────

// stubAgent returns a fixed response for every Chat call.
type stubAgent struct {
	name     string
	response string
	def      *agentdef.AgentDefinition
}

func newStubAgent(name, response string) *stubAgent {
	return &stubAgent{
		name:     name,
		response: response,
		def: &agentdef.AgentDefinition{
			Metadata: agentdef.Metadata{Name: name},
			Spec:     agentdef.AgentSpec{Type: "chat_model_agent"},
		},
	}
}

func (s *stubAgent) Name() string                           { return s.name }
func (s *stubAgent) Description() string                    { return "stub" }
func (s *stubAgent) GetDefinition() *agentdef.AgentDefinition { return s.def }
func (s *stubAgent) Chat(_ context.Context, _ string, _ string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- s.response
	close(ch)
	return ch, nil
}

// memStore is a minimal in-memory CheckpointStore used by tests.
type memStore struct {
	data map[string][]byte
}

func newMemStore() *memStore { return &memStore{data: make(map[string][]byte)} }

func (m *memStore) Get(_ context.Context, id string) ([]byte, bool, error) {
	v, ok := m.data[id]
	return v, ok, nil
}
func (m *memStore) Set(_ context.Context, id string, data []byte) error {
	m.data[id] = data
	return nil
}

// ─── tests ────────────────────────────────────────────────────────────────────

// TestInterruptableAgent_NoInterrupt verifies that ordinary responses pass through unchanged.
func TestInterruptableAgent_NoInterrupt(t *testing.T) {
	inner := newStubAgent("test-agent", "Hello, world!")
	store := newMemStore()
	a := agentdef.NewInterruptableAgent(inner, store, 5*time.Minute)

	ctx := context.Background()
	ch, err := a.Chat(ctx, "s1", "hi")
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	var buf strings.Builder
	for tok := range ch {
		buf.WriteString(tok)
	}
	got := buf.String()
	if got != "Hello, world!" {
		t.Errorf("expected %q, got %q", "Hello, world!", got)
	}

	// No interrupt state should have been recorded.
	_, interrupted := a.GetInterruptState(ctx, "s1")
	if interrupted {
		t.Error("GetInterruptState: expected no interrupt state for s1")
	}
}

// TestInterruptableAgent_DetectsInterrupt verifies that confirmation-seeking
// responses trigger an interrupt signal.
func TestInterruptableAgent_DetectsInterrupt(t *testing.T) {
	inner := newStubAgent("approval-agent", "Please confirm: do you want me to delete all files?")
	store := newMemStore()
	a := agentdef.NewInterruptableAgent(inner, store, 5*time.Minute)

	ctx := context.Background()
	ch, err := a.Chat(ctx, "s1", "delete all files")
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	var tokens []string
	for tok := range ch {
		tokens = append(tokens, tok)
	}
	if len(tokens) == 0 {
		t.Fatal("expected at least one token")
	}
	last := tokens[len(tokens)-1]
	if !strings.HasPrefix(last, "\x00INTERRUPT:") {
		t.Errorf("expected interrupt signal token, got %q", last)
	}

	// Interrupt state should be stored.
	state, interrupted := a.GetInterruptState(ctx, "s1")
	if !interrupted {
		t.Fatal("GetInterruptState: expected interrupt state for s1")
	}
	if state.SessionID != "s1" {
		t.Errorf("state.SessionID = %q, want s1", state.SessionID)
	}
	if state.AgentName != "approval-agent" {
		t.Errorf("state.AgentName = %q, want approval-agent", state.AgentName)
	}
}

// TestInterruptableAgent_Resume verifies that Resume clears the interrupt state
// and forwards the user response to the inner agent.
func TestInterruptableAgent_Resume(t *testing.T) {
	callLog := make([]string, 0)
	callCount := 0

	// First call returns a confirmation request; subsequent calls return the action result.
	inner := &recordingAgent{
		name: "approval-agent",
		fn: func(msg string) string {
			callLog = append(callLog, msg)
			callCount++
			if callCount == 1 {
				return "Please confirm: do you want me to delete all files?"
			}
			return "Action completed."
		},
	}
	store := newMemStore()
	a := agentdef.NewInterruptableAgent(inner, store, 5*time.Minute)

	ctx := context.Background()

	// First call should trigger interrupt.
	ch, err := a.Chat(ctx, "s1", "delete all files")
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	for range ch {
	} // drain

	// Interrupt state must exist.
	_, ok := a.GetInterruptState(ctx, "s1")
	if !ok {
		t.Fatal("expected interrupt state before Resume")
	}

	// Resume the conversation with user confirmation.
	resumeCh, err := a.Resume(ctx, "s1", map[string]any{"confirm": true})
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}

	var resp strings.Builder
	for tok := range resumeCh {
		resp.WriteString(tok)
	}
	if resp.String() != "Action completed." {
		t.Errorf("resume response = %q, want %q", resp.String(), "Action completed.")
	}

	// Interrupt state must have been cleared after Resume.
	_, ok = a.GetInterruptState(ctx, "s1")
	if ok {
		t.Error("expected interrupt state to be cleared after Resume")
	}

	// The resume message passed to the inner agent should contain the user input.
	if len(callLog) == 0 {
		t.Fatal("inner agent was never called during Resume")
	}
	resumeMsg := callLog[len(callLog)-1]
	if !strings.Contains(strings.ToLower(resumeMsg), "confirm") {
		t.Errorf("resume message %q does not mention confirm", resumeMsg)
	}
}

// TestInterruptableAgent_ResumeNoState verifies that Resume returns an error
// when no interrupt state exists.
func TestInterruptableAgent_ResumeNoState(t *testing.T) {
	inner := newStubAgent("test-agent", "ok")
	store := newMemStore()
	a := agentdef.NewInterruptableAgent(inner, store, 5*time.Minute)

	ctx := context.Background()
	_, err := a.Resume(ctx, "nonexistent-session", nil)
	if err == nil {
		t.Error("Resume: expected error for missing interrupt state, got nil")
	}
}

// TestInterruptableAgent_Timeout verifies that expired interrupt states are
// treated as absent.
func TestInterruptableAgent_Timeout(t *testing.T) {
	inner := newStubAgent("approval-agent", "Please confirm: are you sure?")
	store := newMemStore()
	// Use a very short timeout so it expires immediately.
	a := agentdef.NewInterruptableAgent(inner, store, 1*time.Millisecond)

	ctx := context.Background()
	ch, _ := a.Chat(ctx, "s1", "do it")
	for range ch {
	}

	// Give the state time to expire.
	time.Sleep(5 * time.Millisecond)

	_, ok := a.GetInterruptState(ctx, "s1")
	if ok {
		t.Error("GetInterruptState: expected expired state to return false")
	}

	_, err := a.Resume(ctx, "s1", nil)
	if err == nil {
		t.Error("Resume: expected error for expired interrupt state, got nil")
	}
}

// TestInterruptableAgent_InterfaceCompliance verifies that *InterruptableAgent
// satisfies the Interruptable interface at compile time.
func TestInterruptableAgent_InterfaceCompliance(t *testing.T) {
	inner := newStubAgent("test", "ok")
	store := newMemStore()
	var _ agentdef.Interruptable = agentdef.NewInterruptableAgent(inner, store, time.Minute)
}

// TestInterruptableAgent_MultiSession verifies independent interrupt state
// across different sessions.
func TestInterruptableAgent_MultiSession(t *testing.T) {
	inner := newStubAgent("approval-agent", "Please confirm: proceed?")
	store := newMemStore()
	a := agentdef.NewInterruptableAgent(inner, store, 5*time.Minute)

	ctx := context.Background()

	// Trigger interrupts on two different sessions.
	for _, sid := range []string{"sess-A", "sess-B"} {
		ch, err := a.Chat(ctx, sid, "do something risky")
		if err != nil {
			t.Fatalf("[%s] Chat: %v", sid, err)
		}
		for range ch {
		}
	}

	// Both sessions should have independent interrupt state.
	for _, sid := range []string{"sess-A", "sess-B"} {
		state, ok := a.GetInterruptState(ctx, sid)
		if !ok {
			t.Errorf("[%s] expected interrupt state", sid)
		}
		if state != nil && state.SessionID != sid {
			t.Errorf("[%s] state.SessionID = %q", sid, state.SessionID)
		}
	}

	// Resuming sess-A must not affect sess-B.
	if _, err := a.Resume(ctx, "sess-A", nil); err != nil {
		t.Fatalf("Resume sess-A: %v", err)
	}
	if _, ok := a.GetInterruptState(ctx, "sess-B"); !ok {
		t.Error("sess-B interrupt state was unexpectedly cleared by sess-A Resume")
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// recordingAgent calls fn with each incoming message and returns its result.
type recordingAgent struct {
	name string
	fn   func(msg string) string
}

func (r *recordingAgent) Name() string { return r.name }
func (r *recordingAgent) Description() string { return "recording" }
func (r *recordingAgent) GetDefinition() *agentdef.AgentDefinition {
	return &agentdef.AgentDefinition{Metadata: agentdef.Metadata{Name: r.name}}
}
func (r *recordingAgent) Chat(_ context.Context, _ string, msg string) (<-chan string, error) {
	ch := make(chan string, 1)
	ch <- r.fn(msg)
	close(ch)
	return ch, nil
}

// TestBuilderInterruptWrap verifies that the builder wraps the agent with
// InterruptableAgent when spec.interrupt.enabled = true.
func TestBuilderInterruptWrap(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "approval-agent.yaml", fmt.Sprintf(`
apiVersion: superagent/v1
kind: Agent
metadata:
  name: approval-agent
  version: "1.0.0"
spec:
  type: chat_model_agent
  model:
    primary: stub
  system_prompt: "Safety agent."
  interrupt:
    enabled: true
    checkpoint_backend: memory
    timeout_seconds: 300
`))

	builder := agentdef.NewAgentBuilder() // stub mode
	rt := agentdef.NewRuntime(agentdef.RuntimeConfig{ConfigDir: dir}, builder)

	ctx := context.Background()
	if err := rt.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer rt.Stop()

	a, ok := rt.GetAgent("approval-agent")
	if !ok {
		t.Fatal("approval-agent not found")
	}

	_, isInterruptable := a.(agentdef.Interruptable)
	if !isInterruptable {
		t.Error("expected approval-agent to implement Interruptable, but it does not")
	}
}
