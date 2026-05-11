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

// Package builtin provides a Redis-backed memory backend that serves as the
// default fallback when no external memory system is configured.
package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/superagent-ai/superagent-base/backend/infra/cache"
	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
)

const (
	backendName = "builtin"

	// Key prefixes
	prefixShortTerm  = "mem:st:"   // short-term: session messages list
	prefixLongTerm   = "mem:lt:"   // long-term: user memory hash
	prefixAgentState = "mem:as:"   // agent state hash

	// Default TTL for short-term session data (24 hours).
	defaultSessionTTL = 24 * time.Hour
)

func init() {
	memory.Register(backendName, func() memory.Backend {
		return &builtinBackend{}
	})
}

// NewWithCache creates a ready-to-use builtin backend with the supplied cache
// client. The caller must still call Init before using the backend.
func NewWithCache(c cache.Cmdable) *builtinBackend {
	return &builtinBackend{cache: c}
}

// builtinBackend implements memory.Backend using Redis.
type builtinBackend struct {
	cache cache.Cmdable
}

// Name returns the backend identifier.
func (b *builtinBackend) Name() string { return backendName }

// Init initialises the backend. For the builtin backend the cache client must
// be injected via SetCache; Init only validates that it has been set.
func (b *builtinBackend) Init(_ context.Context, _ memory.BackendConfig) error {
	if b.cache == nil {
		return fmt.Errorf("builtin memory: cache client not set; call SetCache before Init")
	}
	return nil
}

// SetCache injects a cache.Cmdable dependency. Call this before Init.
func (b *builtinBackend) SetCache(c cache.Cmdable) {
	b.cache = c
}

// Close is a no-op for the builtin backend (connection lifecycle is managed
// by the caller who provided the cache.Cmdable).
func (b *builtinBackend) Close() error { return nil }

// ── ShortTermMemory ──────────────────────────────────────────────────────────

// AddMessage appends a message to the session's Redis list.
func (b *builtinBackend) AddMessage(ctx context.Context, sessionID string, msg memory.Message) error {
	if msg.Timestamp == 0 {
		msg.Timestamp = time.Now().UnixMilli()
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("builtin memory AddMessage: marshal: %w", err)
	}
	key := prefixShortTerm + sessionID
	if err := b.cache.RPush(ctx, key, string(data)).Err(); err != nil {
		return fmt.Errorf("builtin memory AddMessage: rpush: %w", err)
	}
	// Refresh TTL on every write so active sessions stay alive.
	return b.cache.Expire(ctx, key, defaultSessionTTL).Err()
}

// GetMessages returns messages for a session, newest-first when opts.Before is
// set, or all messages (oldest-first) otherwise.
func (b *builtinBackend) GetMessages(ctx context.Context, sessionID string, opts memory.GetMessagesOpts) ([]memory.Message, error) {
	key := prefixShortTerm + sessionID
	raw, err := b.cache.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("builtin memory GetMessages: lrange: %w", err)
	}

	msgs := make([]memory.Message, 0, len(raw))
	for _, s := range raw {
		var m memory.Message
		if err := json.Unmarshal([]byte(s), &m); err != nil {
			continue // skip malformed entries
		}
		if opts.Before > 0 && m.Timestamp >= opts.Before {
			continue
		}
		msgs = append(msgs, m)
	}

	if opts.Limit > 0 && len(msgs) > opts.Limit {
		// Return the most recent `Limit` messages.
		msgs = msgs[len(msgs)-opts.Limit:]
	}
	return msgs, nil
}

// ClearSession removes all messages for a session.
func (b *builtinBackend) ClearSession(ctx context.Context, sessionID string) error {
	key := prefixShortTerm + sessionID
	if err := b.cache.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("builtin memory ClearSession: %w", err)
	}
	return nil
}

// ── LongTermMemory ───────────────────────────────────────────────────────────

// Add stores a new memory entry for the user and returns its generated ID.
func (b *builtinBackend) Add(ctx context.Context, userID string, content string, metadata map[string]any) (string, error) {
	id := uuid.New().String()
	now := time.Now().UnixMilli()
	entry := memory.MemoryEntry{
		ID:        id,
		Content:   content,
		Metadata:  metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return "", fmt.Errorf("builtin memory Add: marshal: %w", err)
	}
	key := prefixLongTerm + userID
	if err := b.cache.HSet(ctx, key, id, string(data)).Err(); err != nil {
		return "", fmt.Errorf("builtin memory Add: hset: %w", err)
	}
	return id, nil
}

// Search performs a simple substring match over stored entries. It does not
// perform vector search; that requires an external backend such as Mem0.
func (b *builtinBackend) Search(ctx context.Context, userID string, query string, opts memory.SearchOpts) ([]memory.MemoryEntry, error) {
	all, err := b.GetAll(ctx, userID, memory.ListOpts{})
	if err != nil {
		return nil, err
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = len(all)
	}

	results := make([]memory.MemoryEntry, 0, limit)
	for _, e := range all {
		if containsSubstring(e.Content, query) {
			results = append(results, e)
			if len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

// Update replaces the content of an existing memory entry.
func (b *builtinBackend) Update(ctx context.Context, memoryID string, content string) error {
	// memoryID format: <userID>/<entryID> — the callers know both parts.
	// For simplicity the builtin backend stores memoryID as-is in a global hash.
	return b.updateEntry(ctx, memoryID, content)
}

// Delete removes a memory entry by ID.
func (b *builtinBackend) Delete(ctx context.Context, memoryID string) error {
	userID, entryID := splitMemoryID(memoryID)
	key := prefixLongTerm + userID
	if err := b.cache.Del(ctx, key+":"+entryID).Err(); err != nil {
		return fmt.Errorf("builtin memory Delete: %w", err)
	}
	return nil
}

// GetAll returns all memory entries for a user with optional pagination.
func (b *builtinBackend) GetAll(ctx context.Context, userID string, opts memory.ListOpts) ([]memory.MemoryEntry, error) {
	key := prefixLongTerm + userID
	raw, err := b.cache.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("builtin memory GetAll: hgetall: %w", err)
	}

	entries := make([]memory.MemoryEntry, 0, len(raw))
	for _, v := range raw {
		var e memory.MemoryEntry
		if err := json.Unmarshal([]byte(v), &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}

	if opts.Offset > 0 && opts.Offset < len(entries) {
		entries = entries[opts.Offset:]
	} else if opts.Offset >= len(entries) {
		return []memory.MemoryEntry{}, nil
	}
	if opts.Limit > 0 && len(entries) > opts.Limit {
		entries = entries[:opts.Limit]
	}
	return entries, nil
}

// ── AgentStateMemory ─────────────────────────────────────────────────────────

// SetState persists a key-value pair for an agent.
func (b *builtinBackend) SetState(ctx context.Context, agentID string, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("builtin memory SetState: marshal: %w", err)
	}
	hashKey := prefixAgentState + agentID
	if err := b.cache.HSet(ctx, hashKey, key, string(data)).Err(); err != nil {
		return fmt.Errorf("builtin memory SetState: hset: %w", err)
	}
	return nil
}

// GetState retrieves a single state value. Returns (nil, false, nil) when the
// key does not exist.
func (b *builtinBackend) GetState(ctx context.Context, agentID string, key string) (any, bool, error) {
	all, err := b.GetAllState(ctx, agentID)
	if err != nil {
		return nil, false, err
	}
	v, ok := all[key]
	return v, ok, nil
}

// DeleteState removes a state key from an agent's hash.
// HDel is not exposed by cache.Cmdable, so we read-modify-write the hash.
func (b *builtinBackend) DeleteState(ctx context.Context, agentID string, key string) error {
	all, err := b.GetAllState(ctx, agentID)
	if err != nil {
		return err
	}
	delete(all, key)
	return b.replaceAgentState(ctx, agentID, all)
}

// GetAllState returns all state keys for an agent.
func (b *builtinBackend) GetAllState(ctx context.Context, agentID string) (map[string]any, error) {
	hashKey := prefixAgentState + agentID
	raw, err := b.cache.HGetAll(ctx, hashKey).Result()
	if err != nil {
		return nil, fmt.Errorf("builtin memory GetAllState: hgetall: %w", err)
	}
	result := make(map[string]any, len(raw))
	for k, v := range raw {
		var val any
		if err := json.Unmarshal([]byte(v), &val); err != nil {
			result[k] = v // store raw string if unmarshal fails
			continue
		}
		result[k] = val
	}
	return result, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func (b *builtinBackend) updateEntry(ctx context.Context, memoryID string, content string) error {
	userID, entryID := splitMemoryID(memoryID)
	key := prefixLongTerm + userID
	raw, err := b.cache.HGetAll(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("builtin memory Update: hgetall: %w", err)
	}
	v, ok := raw[entryID]
	if !ok {
		return fmt.Errorf("builtin memory Update: entry %q not found", memoryID)
	}
	var entry memory.MemoryEntry
	if err := json.Unmarshal([]byte(v), &entry); err != nil {
		return fmt.Errorf("builtin memory Update: unmarshal: %w", err)
	}
	entry.Content = content
	entry.UpdatedAt = time.Now().UnixMilli()
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("builtin memory Update: marshal: %w", err)
	}
	return b.cache.HSet(ctx, key, entryID, string(data)).Err()
}

func (b *builtinBackend) replaceAgentState(ctx context.Context, agentID string, state map[string]any) error {
	hashKey := prefixAgentState + agentID
	if err := b.cache.Del(ctx, hashKey).Err(); err != nil {
		return fmt.Errorf("builtin memory replaceAgentState: del: %w", err)
	}
	if len(state) == 0 {
		return nil
	}
	args := make([]interface{}, 0, len(state)*2)
	for k, v := range state {
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		args = append(args, k, string(data))
	}
	return b.cache.HSet(ctx, hashKey, args...).Err()
}

// splitMemoryID splits "userID/entryID" into its parts. If the separator is
// absent the entire string is treated as entryID with an empty userID.
func splitMemoryID(id string) (userID, entryID string) {
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] == '/' {
			return id[:i], id[i+1:]
		}
	}
	return "", id
}

func containsSubstring(s, sub string) bool {
	if sub == "" {
		return true
	}
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstringLinear(s, sub))
}

func containsSubstringLinear(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
