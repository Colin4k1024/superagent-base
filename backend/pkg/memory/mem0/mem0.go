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

// Package mem0 provides a Mem0 (https://mem0.ai) memory backend that maps the
// three-tier memory interface to the Mem0 REST API.
//
// Scope mapping:
//   - ShortTermMemory  → Mem0 run_id  (session)
//   - LongTermMemory   → Mem0 user_id (user)
//   - AgentStateMemory → Mem0 agent_id + metadata tag "state_key"
package mem0

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
)

const (
	backendName = "mem0"

	// metaKeyStateKey is the metadata key used to mark agent-state entries.
	metaKeyStateKey = "_state_key"
	// metaKeyTimestamp is stored in metadata so short-term messages have ordering.
	metaKeyTimestamp = "_ts"
	// metaKeyRole carries the message role for short-term messages.
	metaKeyRole = "_role"
)

func init() {
	memory.Register(backendName, func() memory.Backend {
		return &mem0Backend{}
	})
}

// mem0Backend implements memory.Backend backed by the Mem0 REST API.
type mem0Backend struct {
	client *APIClient
}

// Name returns the backend identifier.
func (b *mem0Backend) Name() string { return backendName }

// Init creates an API client from the supplied config.
func (b *mem0Backend) Init(_ context.Context, config memory.BackendConfig) error {
	if config.Endpoint == "" {
		return fmt.Errorf("mem0: endpoint is required")
	}
	if config.APIKey == "" {
		return fmt.Errorf("mem0: api_key is required")
	}
	b.client = NewAPIClient(config.Endpoint, config.APIKey)
	return nil
}

// NewBackendForTest creates a ready-to-use mem0Backend wired to the provided
// http.Client and base URL. Use this in tests to inject an httptest server.
func NewBackendForTest(hc *http.Client, baseURL string) memory.Backend {
	return &mem0Backend{
		client: NewAPIClientWithHTTP(baseURL, "test-key", hc),
	}
}

// Close is a no-op (HTTP connections are managed by the http.Client pool).
func (b *mem0Backend) Close() error { return nil }

// ── ShortTermMemory ──────────────────────────────────────────────────────────
//
// Mem0 does not have a dedicated "session message list" endpoint. We model it
// by storing each message as a Mem0 memory record scoped to run_id, with the
// role and timestamp embedded in metadata. GetMessages then retrieves all
// memories for that run and re-sorts them by timestamp.

// AddMessage stores a single message as a Mem0 memory record for the given
// session (run_id).
func (b *mem0Backend) AddMessage(ctx context.Context, sessionID string, msg memory.Message) error {
	ts := msg.Timestamp
	if ts == 0 {
		ts = time.Now().UnixMilli()
	}

	meta := make(map[string]any, len(msg.Metadata)+2)
	for k, v := range msg.Metadata {
		meta[k] = v
	}
	meta[metaKeyRole] = msg.Role
	meta[metaKeyTimestamp] = ts

	_, err := b.client.AddMemory(ctx, AddMemoryRequest{
		Messages: []APIMessage{{Role: msg.Role, Content: msg.Content}},
		RunID:    sessionID,
		Metadata: meta,
	})
	if err != nil {
		return fmt.Errorf("mem0 AddMessage: %w", err)
	}
	return nil
}

// GetMessages retrieves all messages for a session, ordered by timestamp.
// When opts.Before is set, only messages older than that timestamp are returned.
// When opts.Limit is set, the most-recent N messages are returned.
func (b *mem0Backend) GetMessages(ctx context.Context, sessionID string, opts memory.GetMessagesOpts) ([]memory.Message, error) {
	results, err := b.client.SearchMemory(ctx, SearchRequest{
		Query: "",
		RunID: sessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("mem0 GetMessages: %w", err)
	}

	msgs := make([]memory.Message, 0, len(results))
	for _, r := range results {
		ts := extractInt64(r.Metadata, metaKeyTimestamp)
		if opts.Before > 0 && ts >= opts.Before {
			continue
		}
		role, _ := r.Metadata[metaKeyRole].(string)
		if role == "" {
			role = "user"
		}
		msgs = append(msgs, memory.Message{
			Role:      role,
			Content:   r.Memory,
			Metadata:  r.Metadata,
			Timestamp: ts,
		})
	}

	// Sort ascending by timestamp (simple insertion sort — session sizes are small).
	sortMessagesByTimestamp(msgs)

	if opts.Limit > 0 && len(msgs) > opts.Limit {
		msgs = msgs[len(msgs)-opts.Limit:]
	}
	return msgs, nil
}

// ClearSession deletes all memory records for a given session (run_id).
func (b *mem0Backend) ClearSession(ctx context.Context, sessionID string) error {
	results, err := b.client.SearchMemory(ctx, SearchRequest{
		Query: "",
		RunID: sessionID,
	})
	if err != nil {
		return fmt.Errorf("mem0 ClearSession search: %w", err)
	}
	for _, r := range results {
		if delErr := b.client.DeleteMemory(ctx, r.ID); delErr != nil {
			return fmt.Errorf("mem0 ClearSession delete %s: %w", r.ID, delErr)
		}
	}
	return nil
}

// ── LongTermMemory ───────────────────────────────────────────────────────────

// Add stores a new long-term memory entry for the given user.
func (b *mem0Backend) Add(ctx context.Context, userID string, content string, metadata map[string]any) (string, error) {
	resp, err := b.client.AddMemory(ctx, AddMemoryRequest{
		Messages: []APIMessage{{Role: "user", Content: content}},
		UserID:   userID,
		Metadata: metadata,
	})
	if err != nil {
		return "", fmt.Errorf("mem0 Add: %w", err)
	}
	return resp.ID, nil
}

// Search performs a semantic search over a user's long-term memories.
func (b *mem0Backend) Search(ctx context.Context, userID string, query string, opts memory.SearchOpts) ([]memory.MemoryEntry, error) {
	limit := opts.Limit
	results, err := b.client.SearchMemory(ctx, SearchRequest{
		Query:  query,
		UserID: userID,
		Limit:  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("mem0 Search: %w", err)
	}

	entries := make([]memory.MemoryEntry, 0, len(results))
	for _, r := range results {
		if opts.Threshold > 0 && r.Score < opts.Threshold {
			continue
		}
		entries = append(entries, toMemoryEntry(r))
	}
	return entries, nil
}

// Update replaces the content of an existing memory entry by ID.
func (b *mem0Backend) Update(ctx context.Context, memoryID string, content string) error {
	if err := b.client.UpdateMemory(ctx, memoryID, content); err != nil {
		return fmt.Errorf("mem0 Update: %w", err)
	}
	return nil
}

// Delete removes a memory entry by ID.
func (b *mem0Backend) Delete(ctx context.Context, memoryID string) error {
	if err := b.client.DeleteMemory(ctx, memoryID); err != nil {
		return fmt.Errorf("mem0 Delete: %w", err)
	}
	return nil
}

// GetAll returns all long-term memory entries for a user.
func (b *mem0Backend) GetAll(ctx context.Context, userID string, opts memory.ListOpts) ([]memory.MemoryEntry, error) {
	results, err := b.client.GetAllMemories(ctx, userID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, fmt.Errorf("mem0 GetAll: %w", err)
	}
	entries := make([]memory.MemoryEntry, 0, len(results))
	for _, r := range results {
		entries = append(entries, toMemoryEntry(r))
	}
	return entries, nil
}

// ── AgentStateMemory ─────────────────────────────────────────────────────────
//
// Agent state is modelled as individual Mem0 memories scoped to agent_id, each
// carrying the state key in metadata under metaKeyStateKey. SetState upserts
// by deleting any existing record with the same key, then creating a new one.

// SetState persists a key-value pair for an agent.
func (b *mem0Backend) SetState(ctx context.Context, agentID string, key string, value any) error {
	// Delete any existing record for this key first (upsert semantics).
	if err := b.deleteStateByKey(ctx, agentID, key); err != nil {
		return err
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("mem0 SetState marshal: %w", err)
	}
	_, err = b.client.AddMemory(ctx, AddMemoryRequest{
		Messages: []APIMessage{{Role: "system", Content: string(encoded)}},
		AgentID:  agentID,
		Metadata: map[string]any{metaKeyStateKey: key},
	})
	if err != nil {
		return fmt.Errorf("mem0 SetState: %w", err)
	}
	return nil
}

// GetState retrieves a single state value for an agent.
// Returns (nil, false, nil) when the key does not exist.
func (b *mem0Backend) GetState(ctx context.Context, agentID string, key string) (any, bool, error) {
	all, err := b.GetAllState(ctx, agentID)
	if err != nil {
		return nil, false, err
	}
	v, ok := all[key]
	return v, ok, nil
}

// DeleteState removes a state key from an agent.
func (b *mem0Backend) DeleteState(ctx context.Context, agentID string, key string) error {
	return b.deleteStateByKey(ctx, agentID, key)
}

// GetAllState returns all state key-value pairs for an agent.
func (b *mem0Backend) GetAllState(ctx context.Context, agentID string) (map[string]any, error) {
	results, err := b.client.SearchMemory(ctx, SearchRequest{
		Query:   "",
		AgentID: agentID,
	})
	if err != nil {
		return nil, fmt.Errorf("mem0 GetAllState: %w", err)
	}

	state := make(map[string]any, len(results))
	for _, r := range results {
		k, ok := r.Metadata[metaKeyStateKey].(string)
		if !ok || k == "" {
			continue
		}
		var val any
		if err := json.Unmarshal([]byte(r.Memory), &val); err != nil {
			state[k] = r.Memory // store raw string on unmarshal failure
			continue
		}
		state[k] = val
	}
	return state, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func (b *mem0Backend) deleteStateByKey(ctx context.Context, agentID, key string) error {
	results, err := b.client.SearchMemory(ctx, SearchRequest{
		Query:   "",
		AgentID: agentID,
	})
	if err != nil {
		return fmt.Errorf("mem0 deleteStateByKey search: %w", err)
	}
	for _, r := range results {
		if k, _ := r.Metadata[metaKeyStateKey].(string); k == key {
			if delErr := b.client.DeleteMemory(ctx, r.ID); delErr != nil {
				return fmt.Errorf("mem0 deleteStateByKey delete %s: %w", r.ID, delErr)
			}
		}
	}
	return nil
}

// toMemoryEntry converts a Mem0 API result to our domain MemoryEntry type.
func toMemoryEntry(r MemoryResult) memory.MemoryEntry {
	return memory.MemoryEntry{
		ID:        r.ID,
		Content:   r.Memory,
		Metadata:  r.Metadata,
		Score:     r.Score,
		CreatedAt: parseISO8601Millis(r.CreatedAt),
		UpdatedAt: parseISO8601Millis(r.UpdatedAt),
	}
}

// parseISO8601Millis converts an ISO-8601 timestamp string to Unix milliseconds.
// Returns 0 on parse error.
func parseISO8601Millis(s string) int64 {
	if s == "" {
		return 0
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05.999999",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UnixMilli()
		}
	}
	return 0
}

// extractInt64 reads a numeric value from a metadata map as int64.
func extractInt64(meta map[string]any, key string) int64 {
	if meta == nil {
		return 0
	}
	switch v := meta[key].(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	}
	return 0
}

// sortMessagesByTimestamp sorts messages in-place, ascending by Timestamp.
func sortMessagesByTimestamp(msgs []memory.Message) {
	// Insertion sort: session message lists are short.
	for i := 1; i < len(msgs); i++ {
		for j := i; j > 0 && msgs[j].Timestamp < msgs[j-1].Timestamp; j-- {
			msgs[j], msgs[j-1] = msgs[j-1], msgs[j]
		}
	}
}
