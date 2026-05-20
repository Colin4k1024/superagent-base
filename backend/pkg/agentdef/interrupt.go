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
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// InterruptState holds the state of an interrupted conversation.
type InterruptState struct {
	SessionID    string       `json:"session_id"`
	AgentName    string       `json:"agent_name"`
	Reason       string       `json:"reason"`
	Fields       []InputField `json:"fields,omitempty"`
	Timestamp    int64        `json:"timestamp"`
	CheckpointID string       `json:"checkpoint_id"`
}

// InputField describes what input is needed to resume.
type InputField struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`     // text, confirm, select
	Label    string   `json:"label"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
}

// CheckpointStore is the storage interface for interrupt state persistence.
// It matches compose.CheckPointStore to allow direct use of
// infra/checkpoint/redis.go or infra/checkpoint/mem.go.
type CheckpointStore interface {
	Get(ctx context.Context, id string) ([]byte, bool, error)
	Set(ctx context.Context, id string, data []byte) error
}

// Interruptable extends Agent with interrupt/resume capability.
type Interruptable interface {
	Agent
	// Resume continues execution from an interrupted state with the user-provided input.
	Resume(ctx context.Context, sessionID string, input map[string]any) (<-chan string, error)
	// GetInterruptState returns the current interrupt state for a session, if any.
	GetInterruptState(ctx context.Context, sessionID string) (*InterruptState, bool)
}

// interruptPrefix is a sentinel prefix embedded in streamed tokens to signal an interrupt.
// The byte \x00 is used because it never appears in valid UTF-8 model output.
const interruptPrefix = "\x00INTERRUPT:"

// confirmationKeywords are phrases in model output that indicate the model is requesting
// confirmation before proceeding. Pattern-based detection is used in v1; full Eino
// compose.ExtractInterruptInfo() integration can replace this later.
var confirmationKeywords = []string{
	"please confirm",
	"do you confirm",
	"are you sure",
	"shall i proceed",
	"would you like me to proceed",
	"do you want to proceed",
	"do you want me to",
	"should i go ahead",
	"confirm before",
	"confirmation required",
	"waiting for your confirmation",
	"need your approval",
}

// maxInterruptEntries is the maximum number of interrupt states retained in memory.
// When exceeded, the oldest entries are evicted (LRU by expiry time).
const maxInterruptEntries = 1000

// InterruptableAgent wraps an Agent with interrupt/resume capability.
type InterruptableAgent struct {
	inner      Agent
	store      CheckpointStore
	timeout    time.Duration
	mu         sync.RWMutex
	interrupts map[string]*interruptEntry
}

// interruptEntry stores an interrupt state along with an expiry timestamp.
type interruptEntry struct {
	state     *InterruptState
	expiresAt time.Time
}

// NewInterruptableAgent wraps inner with interrupt/resume support.
// store is used to persist interrupt state; timeout controls how long state is retained.
func NewInterruptableAgent(inner Agent, store CheckpointStore, timeout time.Duration) *InterruptableAgent {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	return &InterruptableAgent{
		inner:      inner,
		store:      store,
		timeout:    timeout,
		interrupts: make(map[string]*interruptEntry),
	}
}

// Name delegates to the inner agent.
func (a *InterruptableAgent) Name() string { return a.inner.Name() }

// Description delegates to the inner agent.
func (a *InterruptableAgent) Description() string { return a.inner.Description() }

// GetDefinition delegates to the inner agent.
func (a *InterruptableAgent) GetDefinition() *AgentDefinition { return a.inner.GetDefinition() }

// Chat executes the inner agent. If the model output contains a confirmation
// request, the state is saved and a special INTERRUPT token is sent so the
// caller can pause and call Resume later.
func (a *InterruptableAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	ch := make(chan string, 100)
	go func() {
		defer close(ch)

		innerCh, err := a.inner.Chat(ctx, sessionID, message)
		if err != nil {
			select {
			case ch <- fmt.Sprintf("[error] %v", err):
			case <-ctx.Done():
			}
			return
		}

		// Buffer the full response so we can inspect it for interrupt keywords.
		var buf strings.Builder
		var tokens []string
		for token := range innerCh {
			buf.WriteString(token)
			tokens = append(tokens, token)
		}
		response := buf.String()

		if state := a.detectInterrupt(response, sessionID); state != nil {
			if saveErr := a.saveInterruptState(ctx, sessionID, state); saveErr != nil {
				// Non-fatal: emit the interrupt signal even if persistence failed.
				_ = saveErr
			}
			data, _ := json.Marshal(state)
			select {
			case ch <- interruptPrefix + string(data):
			case <-ctx.Done():
			}
			return
		}

		// No interrupt detected — stream the buffered tokens.
		for _, tok := range tokens {
			select {
			case ch <- tok:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

// Resume continues execution from an interrupted state with user-supplied input.
func (a *InterruptableAgent) Resume(ctx context.Context, sessionID string, input map[string]any) (<-chan string, error) {
	state, ok := a.getInterruptState(ctx, sessionID)
	if !ok {
		return nil, fmt.Errorf("agentdef: Resume: no interrupt state for session %q", sessionID)
	}

	a.clearInterruptState(a.interruptKey(sessionID))

	resumeMsg := formatResumeInput(state, input)
	return a.inner.Chat(ctx, sessionID, resumeMsg)
}

// GetInterruptState returns the current interrupt state for a session.
func (a *InterruptableAgent) GetInterruptState(ctx context.Context, sessionID string) (*InterruptState, bool) {
	return a.getInterruptState(ctx, sessionID)
}

// ─── internal helpers ─────────────────────────────────────────────────────────

// detectInterrupt inspects the full model response for confirmation requests.
// Returns a populated InterruptState if an interrupt is warranted.
func (a *InterruptableAgent) detectInterrupt(response, sessionID string) *InterruptState {
	lower := strings.ToLower(response)
	for _, kw := range confirmationKeywords {
		if strings.Contains(lower, kw) {
			return &InterruptState{
				SessionID:    sessionID,
				AgentName:    a.inner.Name(),
				Reason:       response,
				Fields:       []InputField{{Name: "confirm", Type: "confirm", Label: "Confirm action", Required: true}},
				Timestamp:    time.Now().Unix(),
				CheckpointID: fmt.Sprintf("%s_%s_%d", a.inner.Name(), sessionID, time.Now().UnixNano()),
			}
		}
	}
	return nil
}

// saveInterruptState persists the interrupt state both in-memory and in the store.
// Both memory and store use interruptKey(sessionID) so multi-replica resume works.
func (a *InterruptableAgent) saveInterruptState(ctx context.Context, sessionID string, state *InterruptState) error {
	key := a.interruptKey(sessionID)

	a.mu.Lock()
	// Evict expired entries and enforce maxSize (LRU by expiry).
	now := time.Now()
	for k, e := range a.interrupts {
		if now.After(e.expiresAt) {
			delete(a.interrupts, k)
		}
	}
	if len(a.interrupts) >= maxInterruptEntries {
		// Evict the oldest entry by earliest expiresAt.
		var oldestKey string
		var oldestTime time.Time
		for k, e := range a.interrupts {
			if oldestKey == "" || e.expiresAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = e.expiresAt
			}
		}
		if oldestKey != "" {
			delete(a.interrupts, oldestKey)
		}
	}
	a.interrupts[key] = &interruptEntry{
		state:     state,
		expiresAt: time.Now().Add(a.timeout),
	}
	a.mu.Unlock()

	if a.store != nil {
		data, err := json.Marshal(state)
		if err != nil {
			return fmt.Errorf("agentdef: saveInterruptState: marshal: %w", err)
		}
		if err := a.store.Set(ctx, key, data); err != nil {
			return fmt.Errorf("agentdef: saveInterruptState: store.Set: %w", err)
		}
	}
	return nil
}

// interruptKey returns the storage key for a session's interrupt state.
// Using a dedicated key helper ensures both memory and Redis use the same
// namespace, which is required for cross-pod resume in multi-replica deployments.
func (a *InterruptableAgent) interruptKey(sessionID string) string {
	return "interrupt:" + sessionID
}

// getInterruptState returns the interrupt state for a session, checking
// expiry. Returns false if not found or expired.
// On a memory miss it falls back to the CheckpointStore (Redis) so that
// resume requests routed to a different pod can still locate the state.
func (a *InterruptableAgent) getInterruptState(ctx context.Context, sessionID string) (*InterruptState, bool) {
	key := a.interruptKey(sessionID)

	// 1. Check in-memory cache first (fast path).
	a.mu.RLock()
	entry, ok := a.interrupts[key]
	a.mu.RUnlock()
	if ok {
		if time.Now().After(entry.expiresAt) {
			a.clearInterruptState(key)
			return nil, false
		}
		return entry.state, true
	}

	// 2. Memory miss — fall back to CheckpointStore (Redis in multi-replica mode).
	if a.store != nil {
		data, found, err := a.store.Get(ctx, key)
		// Empty data is a tombstone written by clearInterruptState; treat as absent.
		if err == nil && found && len(data) > 0 {
			var state InterruptState
			if jsonErr := json.Unmarshal(data, &state); jsonErr == nil {
				// Populate memory cache so subsequent calls are fast.
				a.mu.Lock()
				a.interrupts[key] = &interruptEntry{
					state:     &state,
					expiresAt: time.Now().Add(a.timeout),
				}
				a.mu.Unlock()
				return &state, true
			}
		}
	}

	return nil, false
}

// clearInterruptState removes the interrupt entry for a session from both memory
// and the CheckpointStore. The key parameter must already be the full storage key
// (i.e. the result of interruptKey).
// A zero-byte tombstone is written to the store so that a Redis-fallback lookup
// on a different pod cannot resurrect a cleared (resumed) state.
func (a *InterruptableAgent) clearInterruptState(key string) {
	a.mu.Lock()
	delete(a.interrupts, key)
	a.mu.Unlock()

	// Write a tombstone (zero bytes) so the store fallback treats this key as absent.
	if a.store != nil {
		_ = a.store.Set(context.Background(), key, []byte{})
	}
}

// formatResumeInput builds a user message from the interrupt state and the
// provided input map. The format is understood by most instruction-following
// models as a continuation of the interrupted conversation.
func formatResumeInput(state *InterruptState, input map[string]any) string {
	if len(input) == 0 {
		return "User confirmed: yes, please proceed."
	}

	var parts []string
	for _, field := range state.Fields {
		if v, ok := input[field.Name]; ok {
			parts = append(parts, fmt.Sprintf("%s: %v", field.Label, v))
		}
	}
	if len(parts) == 0 {
		// Fallback: encode the raw map.
		for k, v := range input {
			parts = append(parts, fmt.Sprintf("%s: %v", k, v))
		}
	}
	return "User response: " + strings.Join(parts, "; ") + ". Please proceed."
}
