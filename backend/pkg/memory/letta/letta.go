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

package letta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
)

const backendName = "letta"

func init() {
	memory.Register(backendName, func() memory.Backend {
		return &lettaBackend{}
	})
}

// lettaBackend implements memory.Backend backed by the Letta REST API.
//
// Scope mapping:
//   - ShortTermMemory  → Letta agent messages (agentID = sessionID)
//   - LongTermMemory   → Letta archival memory (agentID = userID)
//   - AgentStateMemory → Letta core memory blocks (agentID = agentID)
type lettaBackend struct {
	client *APIClient
}

// Name returns the backend identifier.
func (b *lettaBackend) Name() string { return backendName }

// Init creates the API client from config.
func (b *lettaBackend) Init(_ context.Context, config memory.BackendConfig) error {
	if config.Endpoint == "" {
		return fmt.Errorf("letta: endpoint is required")
	}
	if config.APIKey == "" {
		return fmt.Errorf("letta: api_key is required")
	}
	b.client = NewAPIClient(config.Endpoint, config.APIKey)
	return nil
}

// NewBackendForTest creates a ready-to-use lettaBackend wired to the provided
// http.Client and base URL. Use this in tests to inject an httptest server.
func NewBackendForTest(hc *http.Client, baseURL string) memory.Backend {
	return &lettaBackend{
		client: NewAPIClientWithHTTP(baseURL, "test-key", hc),
	}
}

// Close is a no-op.
func (b *lettaBackend) Close() error { return nil }

// ── ShortTermMemory ───────────────────────────────────────────────────────────

// AddMessage sends a message to the Letta agent for this session.
// sessionID is used as the Letta agentID.
func (b *lettaBackend) AddMessage(ctx context.Context, sessionID string, msg memory.Message) error {
	if _, err := b.client.SendMessage(ctx, sessionID, msg.Content); err != nil {
		return fmt.Errorf("letta AddMessage: %w", err)
	}
	return nil
}

// GetMessages is not directly exposed by Letta's REST API (history is managed
// internally). Returns an empty slice; use archival search for persistent recall.
func (b *lettaBackend) GetMessages(_ context.Context, _ string, _ memory.GetMessagesOpts) ([]memory.Message, error) {
	return []memory.Message{}, nil
}

// ClearSession asks the agent to clear its context via a system message.
func (b *lettaBackend) ClearSession(ctx context.Context, sessionID string) error {
	if _, err := b.client.SendMessage(ctx, sessionID, "[SYSTEM: clear conversation context]"); err != nil {
		return fmt.Errorf("letta ClearSession: %w", err)
	}
	return nil
}

// ── LongTermMemory ────────────────────────────────────────────────────────────

// Add inserts a passage into the user's archival memory.
func (b *lettaBackend) Add(ctx context.Context, userID string, content string, _ map[string]any) (string, error) {
	entry, err := b.client.InsertArchival(ctx, userID, content)
	if err != nil {
		return "", fmt.Errorf("letta Add: %w", err)
	}
	if entry != nil && entry.ID != "" {
		return entry.ID, nil
	}
	return userID + "/" + fmt.Sprintf("%d", time.Now().UnixNano()), nil
}

// Search performs archival memory search.
func (b *lettaBackend) Search(ctx context.Context, userID string, query string, opts memory.SearchOpts) ([]memory.MemoryEntry, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 10
	}
	entries, err := b.client.SearchArchival(ctx, userID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("letta Search: %w", err)
	}

	results := make([]memory.MemoryEntry, 0, len(entries))
	for _, e := range entries {
		results = append(results, memory.MemoryEntry{
			ID:        e.ID,
			Content:   e.Text,
			CreatedAt: parseISO8601Millis(e.Timestamp),
			UpdatedAt: parseISO8601Millis(e.Timestamp),
		})
	}
	return results, nil
}

// Update deletes the old passage and inserts new content.
// memoryID format: "<agentID>/<passageID>"
func (b *lettaBackend) Update(ctx context.Context, memoryID string, content string) error {
	agentID, passageID := splitMemoryID(memoryID)
	if passageID != "" {
		if err := b.client.DeleteArchival(ctx, agentID, passageID); err != nil {
			return fmt.Errorf("letta Update delete: %w", err)
		}
	}
	if _, err := b.client.InsertArchival(ctx, agentID, content); err != nil {
		return fmt.Errorf("letta Update insert: %w", err)
	}
	return nil
}

// Delete removes an archival passage by ID.
func (b *lettaBackend) Delete(ctx context.Context, memoryID string) error {
	agentID, passageID := splitMemoryID(memoryID)
	if passageID == "" {
		return fmt.Errorf("letta Delete: invalid memoryID %q", memoryID)
	}
	if err := b.client.DeleteArchival(ctx, agentID, passageID); err != nil {
		return fmt.Errorf("letta Delete: %w", err)
	}
	return nil
}

// GetAll returns archival passages via an empty-query search.
func (b *lettaBackend) GetAll(ctx context.Context, userID string, opts memory.ListOpts) ([]memory.MemoryEntry, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}
	entries, err := b.client.SearchArchival(ctx, userID, "", limit)
	if err != nil {
		return nil, fmt.Errorf("letta GetAll: %w", err)
	}

	all := make([]memory.MemoryEntry, 0, len(entries))
	for _, e := range entries {
		all = append(all, memory.MemoryEntry{
			ID:        e.ID,
			Content:   e.Text,
			CreatedAt: parseISO8601Millis(e.Timestamp),
			UpdatedAt: parseISO8601Millis(e.Timestamp),
		})
	}

	if opts.Offset > 0 {
		if opts.Offset >= len(all) {
			return []memory.MemoryEntry{}, nil
		}
		all = all[opts.Offset:]
	}
	return all, nil
}

// ── AgentStateMemory ──────────────────────────────────────────────────────────

// SetState writes a JSON-encoded value to a named core-memory block.
func (b *lettaBackend) SetState(ctx context.Context, agentID string, key string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("letta SetState marshal: %w", err)
	}
	if err := b.client.UpdateCoreMemory(ctx, agentID, key, string(encoded)); err != nil {
		return fmt.Errorf("letta SetState: %w", err)
	}
	return nil
}

// GetState retrieves and JSON-decodes a single core-memory block.
func (b *lettaBackend) GetState(ctx context.Context, agentID string, key string) (any, bool, error) {
	all, err := b.GetAllState(ctx, agentID)
	if err != nil {
		return nil, false, err
	}
	v, ok := all[key]
	return v, ok, nil
}

// DeleteState clears a core-memory block.
func (b *lettaBackend) DeleteState(ctx context.Context, agentID string, key string) error {
	if err := b.client.UpdateCoreMemory(ctx, agentID, key, ""); err != nil {
		return fmt.Errorf("letta DeleteState: %w", err)
	}
	return nil
}

// GetAllState returns all core-memory blocks as key-value pairs.
func (b *lettaBackend) GetAllState(ctx context.Context, agentID string) (map[string]any, error) {
	cm, err := b.client.GetCoreMemory(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("letta GetAllState: %w", err)
	}

	state := make(map[string]any, len(cm.Memory))
	for k, block := range cm.Memory {
		if block.Value == "" {
			continue
		}
		var decoded any
		if err := json.Unmarshal([]byte(block.Value), &decoded); err == nil {
			state[k] = decoded
		} else {
			state[k] = block.Value
		}
	}
	return state, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func splitMemoryID(id string) (agentID, passageID string) {
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] == '/' {
			return id[:i], id[i+1:]
		}
	}
	return id, ""
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
