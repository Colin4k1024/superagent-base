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
	"sync"
)

// SessionManager caches HiAgent conversation IDs per user, thread-safe.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]string // userID → AppConversationID
	client   *Client
}

func NewSessionManager(client *Client) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]string),
		client:   client,
	}
}

// GetOrCreate returns a cached conversation ID or creates a new one.
func (sm *SessionManager) GetOrCreate(ctx context.Context, userID string) (string, error) {
	sm.mu.RLock()
	if id, ok := sm.sessions[userID]; ok {
		sm.mu.RUnlock()
		return id, nil
	}
	sm.mu.RUnlock()

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Double-check after acquiring write lock.
	if id, ok := sm.sessions[userID]; ok {
		return id, nil
	}

	id, err := sm.client.CreateConversation(ctx, userID)
	if err != nil {
		return "", err
	}

	sm.sessions[userID] = id
	return id, nil
}

// Invalidate removes the cached conversation for a user, forcing re-creation on next call.
func (sm *SessionManager) Invalidate(userID string) {
	sm.mu.Lock()
	delete(sm.sessions, userID)
	sm.mu.Unlock()
}
