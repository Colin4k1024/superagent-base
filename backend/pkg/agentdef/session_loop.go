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
	"os"
	"strconv"
	"sync"
	"time"
)

// defaultTurnTimeout is the per-turn timeout when TURN_TIMEOUT_SECONDS is not set.
// Zero means no additional timeout (the parent/HTTP context still applies).
var defaultTurnTimeout = func() time.Duration {
	if s := os.Getenv("TURN_TIMEOUT_SECONDS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 0
}()

// SessionLoop manages per-session turn lifecycle.
//
// It implements two TurnLoop-inspired capabilities:
//   - Preempt: when a new message arrives for an active session, the current
//     turn is cancelled so the agent stops at its next context-check boundary.
//   - Abort: the user explicitly requests the active turn to stop immediately.
//
// Per-turn timeout is configurable via the TURN_TIMEOUT_SECONDS env var.
// When set, each turn is bounded by that duration regardless of the parent context.
//
// All methods are safe for concurrent use.
type SessionLoop struct {
	sessions sync.Map // sessionID → *turnEntry
}

type turnEntry struct {
	mu     sync.Mutex
	cancel context.CancelFunc // nil when no active turn
}

// NewSessionLoop creates a SessionLoop.
func NewSessionLoop() *SessionLoop { return &SessionLoop{} }

// StartTurn begins a new turn for sessionID.
// If there is already an active turn it is cancelled first (preempt).
// The returned context is derived from parent, so it is also cancelled when
// the parent (e.g. the HTTP request context) is cancelled.
// When TURN_TIMEOUT_SECONDS is set, an additional per-turn deadline is applied.
// The caller must call EndTurn when the turn completes.
func (s *SessionLoop) StartTurn(sessionID string, parent context.Context) context.Context {
	e := s.getOrCreate(sessionID)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cancel != nil {
		e.cancel() // preempt the current turn
	}
	var ctx context.Context
	var cancel context.CancelFunc
	if defaultTurnTimeout > 0 {
		ctx, cancel = context.WithTimeout(parent, defaultTurnTimeout)
	} else {
		ctx, cancel = context.WithCancel(parent)
	}
	e.cancel = cancel
	return ctx
}

// EndTurn marks the current turn as finished, releasing its cancel function.
func (s *SessionLoop) EndTurn(sessionID string) {
	v, ok := s.sessions.Load(sessionID)
	if !ok {
		return
	}
	e := v.(*turnEntry)
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cancel = nil
}

// Abort cancels the active turn for sessionID immediately.
// Returns true if there was an active turn, false otherwise.
func (s *SessionLoop) Abort(sessionID string) bool {
	v, ok := s.sessions.Load(sessionID)
	if !ok {
		return false
	}
	e := v.(*turnEntry)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cancel == nil {
		return false
	}
	e.cancel()
	e.cancel = nil
	return true
}

func (s *SessionLoop) getOrCreate(sessionID string) *turnEntry {
	v, _ := s.sessions.LoadOrStore(sessionID, &turnEntry{})
	return v.(*turnEntry)
}
