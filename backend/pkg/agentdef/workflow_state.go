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

import "sync"

// safeState is a concurrency-safe key/value store for workflow execution state.
// It is safe to call snapshot, set, and get concurrently from multiple goroutines.
type safeState struct {
	mu   sync.RWMutex
	data map[string]string
}

// newSafeState initialises a safeState seeded with the provided initial data.
// The provided map is copied so that the caller retains ownership.
func newSafeState(initial map[string]string) *safeState {
	s := &safeState{data: make(map[string]string, len(initial))}
	for k, v := range initial {
		s.data[k] = v
	}
	return s
}

// snapshot returns a read-only copy of the current state.
// The returned map is safe to pass to read-only helpers (resolveTemplate, etc.)
// without holding the lock.
func (s *safeState) snapshot() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]string, len(s.data))
	for k, v := range s.data {
		cp[k] = v
	}
	return cp
}

// set safely writes a single key/value pair into the state.
func (s *safeState) set(key, val string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = val
}

// get safely reads a single value from the state.
func (s *safeState) get(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data[key]
}
