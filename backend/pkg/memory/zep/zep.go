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

package zep

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
)

const (
	backendName = "zep"

	// metaStatePrefix is the key prefix used inside session metadata to store
	// agent state key-value pairs, e.g. "agent_state.<key>".
	metaStatePrefix = "agent_state."
)

func init() {
	memory.Register(backendName, func() memory.Backend {
		return &zepBackend{}
	})
}

// zepBackend implements memory.Backend backed by the Zep REST API.
//
// Scope mapping:
//   - ShortTermMemory  → Zep session messages (session_id = sessionID)
//   - LongTermMemory   → Zep session facts + semantic search (session_id = userID)
//   - AgentStateMemory → Zep session metadata (session_id = agentID)
type zepBackend struct {
	client *APIClient
}

// Name returns the backend identifier.
func (b *zepBackend) Name() string { return backendName }

// Init creates the API client.
func (b *zepBackend) Init(_ context.Context, config memory.BackendConfig) error {
	if config.Endpoint == "" {
		return fmt.Errorf("zep: endpoint is required")
	}
	if config.APIKey == "" {
		return fmt.Errorf("zep: api_key is required")
	}
	b.client = NewAPIClient(config.Endpoint, config.APIKey)
	return nil
}

// NewBackendForTest creates a ready-to-use zepBackend wired to the provided
// http.Client and base URL. Use this in tests to inject an httptest server.
func NewBackendForTest(hc *http.Client, baseURL string) memory.Backend {
	return &zepBackend{
		client: NewAPIClientWithHTTP(baseURL, "test-key", hc),
	}
}

// Close is a no-op.
func (b *zepBackend) Close() error { return nil }

// ── ShortTermMemory ───────────────────────────────────────────────────────────

// AddMessage appends a single message to the Zep session.
func (b *zepBackend) AddMessage(ctx context.Context, sessionID string, msg memory.Message) error {
	// Ensure session exists before writing.
	if err := b.client.EnsureSession(ctx, sessionID); err != nil {
		return fmt.Errorf("zep AddMessage ensure session: %w", err)
	}
	zm := toZepMessage(msg)
	if err := b.client.AddMemory(ctx, sessionID, []Message{zm}); err != nil {
		return fmt.Errorf("zep AddMessage: %w", err)
	}
	return nil
}

// GetMessages retrieves messages for a session.
func (b *zepBackend) GetMessages(ctx context.Context, sessionID string, opts memory.GetMessagesOpts) ([]memory.Message, error) {
	mem, err := b.client.GetMemory(ctx, sessionID, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("zep GetMessages: %w", err)
	}

	msgs := make([]memory.Message, 0, len(mem.Messages))
	for _, zm := range mem.Messages {
		m := fromZepMessage(zm)
		if opts.Before > 0 && m.Timestamp >= opts.Before {
			continue
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

// ClearSession deletes the Zep session and all its messages.
func (b *zepBackend) ClearSession(ctx context.Context, sessionID string) error {
	if err := b.client.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("zep ClearSession: %w", err)
	}
	return nil
}

// ── LongTermMemory ────────────────────────────────────────────────────────────
//
// We model long-term memory by using a per-user Zep session (session_id = userID).
// Zep automatically extracts facts from messages stored in that session.
// "Adding" a memory means adding a message to that session; Zep handles
// fact extraction in the background.

// Add stores a long-term memory entry for the user.
func (b *zepBackend) Add(ctx context.Context, userID string, content string, metadata map[string]any) (string, error) {
	sessionID := "lt-" + userID
	if err := b.client.EnsureSession(ctx, sessionID); err != nil {
		return "", fmt.Errorf("zep Add ensure session: %w", err)
	}
	zm := Message{
		Role:     "user",
		RoleType: "user",
		Content:  content,
		Metadata: metadata,
	}
	if err := b.client.AddMemory(ctx, sessionID, []Message{zm}); err != nil {
		return "", fmt.Errorf("zep Add: %w", err)
	}
	// Zep doesn't return an ID for individual messages via this endpoint;
	// we return a deterministic composite key.
	return sessionID + "/" + fmt.Sprintf("%d", time.Now().UnixNano()), nil
}

// Search performs semantic search over the user's long-term memory session.
func (b *zepBackend) Search(ctx context.Context, userID string, query string, opts memory.SearchOpts) ([]memory.MemoryEntry, error) {
	sessionID := "lt-" + userID
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	results, err := b.client.SearchMemory(ctx, sessionID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("zep Search: %w", err)
	}

	entries := make([]memory.MemoryEntry, 0, len(results))
	for _, r := range results {
		if opts.Threshold > 0 && r.Score < opts.Threshold {
			continue
		}
		content := ""
		var meta map[string]any
		if r.Message != nil {
			content = r.Message.Content
			meta = r.Message.Metadata
		} else {
			content = r.Fact
		}
		entries = append(entries, memory.MemoryEntry{
			ID:      zepResultID(r),
			Content: content,
			Score:   r.Score,
			Metadata: meta,
		})
	}
	return entries, nil
}

// Update is not directly supported by Zep's add-only message model.
// We add a corrective message that supersedes the old content.
func (b *zepBackend) Update(ctx context.Context, memoryID string, content string) error {
	// memoryID format: "lt-<userID>/<seq>" — extract sessionID.
	sessionID := extractSessionID(memoryID)
	zm := Message{
		Role:     "system",
		RoleType: "system",
		Content:  "[UPDATE] " + content,
	}
	if err := b.client.AddMemory(ctx, sessionID, []Message{zm}); err != nil {
		return fmt.Errorf("zep Update: %w", err)
	}
	return nil
}

// Delete is approximated by adding a retraction message (Zep has no delete-message API).
func (b *zepBackend) Delete(ctx context.Context, memoryID string) error {
	sessionID := extractSessionID(memoryID)
	zm := Message{
		Role:     "system",
		RoleType: "system",
		Content:  "[RETRACTED]",
		Metadata: map[string]any{"retracted_id": memoryID},
	}
	if err := b.client.AddMemory(ctx, sessionID, []Message{zm}); err != nil {
		return fmt.Errorf("zep Delete: %w", err)
	}
	return nil
}

// GetAll retrieves all messages from the user's long-term session.
func (b *zepBackend) GetAll(ctx context.Context, userID string, opts memory.ListOpts) ([]memory.MemoryEntry, error) {
	sessionID := "lt-" + userID
	mem, err := b.client.GetMemory(ctx, sessionID, 0)
	if err != nil {
		return nil, fmt.Errorf("zep GetAll: %w", err)
	}

	all := make([]memory.MemoryEntry, 0, len(mem.Messages))
	for _, zm := range mem.Messages {
		m := fromZepMessage(zm)
		all = append(all, memory.MemoryEntry{
			ID:        zm.UUID,
			Content:   m.Content,
			Metadata:  m.Metadata,
			CreatedAt: m.Timestamp,
			UpdatedAt: m.Timestamp,
		})
	}

	// Apply pagination client-side.
	if opts.Offset > 0 {
		if opts.Offset >= len(all) {
			return []memory.MemoryEntry{}, nil
		}
		all = all[opts.Offset:]
	}
	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}
	return all, nil
}

// ── AgentStateMemory ──────────────────────────────────────────────────────────
//
// Agent state is stored as Zep session metadata on a dedicated session per
// agent (session_id = "as-<agentID>").  Each state key maps to a metadata
// field prefixed with metaStatePrefix.

// SetState persists a key-value pair for an agent.
func (b *zepBackend) SetState(ctx context.Context, agentID string, key string, value any) error {
	sessionID := "as-" + agentID
	if err := b.client.EnsureSession(ctx, sessionID); err != nil {
		return fmt.Errorf("zep SetState ensure session: %w", err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("zep SetState marshal: %w", err)
	}
	meta := map[string]any{metaStatePrefix + key: string(encoded)}
	if err := b.client.UpdateSessionMetadata(ctx, sessionID, meta); err != nil {
		return fmt.Errorf("zep SetState: %w", err)
	}
	return nil
}

// GetState retrieves a single state value for an agent.
func (b *zepBackend) GetState(ctx context.Context, agentID string, key string) (any, bool, error) {
	all, err := b.GetAllState(ctx, agentID)
	if err != nil {
		return nil, false, err
	}
	v, ok := all[key]
	return v, ok, nil
}

// DeleteState removes a state key by setting its metadata value to null.
func (b *zepBackend) DeleteState(ctx context.Context, agentID string, key string) error {
	sessionID := "as-" + agentID
	meta := map[string]any{metaStatePrefix + key: nil}
	if err := b.client.UpdateSessionMetadata(ctx, sessionID, meta); err != nil {
		return fmt.Errorf("zep DeleteState: %w", err)
	}
	return nil
}

// GetAllState returns all state keys for an agent by reading session metadata.
func (b *zepBackend) GetAllState(ctx context.Context, agentID string) (map[string]any, error) {
	sessionID := "as-" + agentID
	session, err := b.client.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("zep GetAllState: %w", err)
	}

	prefix := metaStatePrefix
	state := make(map[string]any)
	for k, v := range session.Metadata {
		if len(k) > len(prefix) && k[:len(prefix)] == prefix {
			stateKey := k[len(prefix):]
			// Decode JSON-encoded value.
			if s, ok := v.(string); ok {
				var decoded any
				if err := json.Unmarshal([]byte(s), &decoded); err == nil {
					state[stateKey] = decoded
					continue
				}
			}
			state[stateKey] = v
		}
	}
	return state, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func toZepMessage(m memory.Message) Message {
	ts := m.Timestamp
	if ts == 0 {
		ts = time.Now().UnixMilli()
	}
	meta := make(map[string]any, len(m.Metadata)+1)
	for k, v := range m.Metadata {
		meta[k] = v
	}
	meta["_ts"] = ts
	return Message{
		Role:     m.Role,
		RoleType: m.Role,
		Content:  m.Content,
		Metadata: meta,
	}
}

func fromZepMessage(zm Message) memory.Message {
	ts := int64(0)
	if zm.Metadata != nil {
		switch v := zm.Metadata["_ts"].(type) {
		case float64:
			ts = int64(v)
		case int64:
			ts = v
		}
	}
	if ts == 0 {
		ts = parseISO8601Millis(zm.CreatedAt)
	}
	return memory.Message{
		Role:      zm.Role,
		Content:   zm.Content,
		Metadata:  zm.Metadata,
		Timestamp: ts,
	}
}

func zepResultID(r SearchResult) string {
	if r.Message != nil && r.Message.UUID != "" {
		return r.Message.UUID
	}
	return fmt.Sprintf("fact:%s", r.Fact)
}

func extractSessionID(memoryID string) string {
	for i := len(memoryID) - 1; i >= 0; i-- {
		if memoryID[i] == '/' {
			return memoryID[:i]
		}
	}
	return memoryID
}

func parseISO8601Millis(s string) int64 {
	if s == "" {
		return 0
	}
	formats := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05"}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UnixMilli()
		}
	}
	return 0
}
