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

package memory

import "context"

// ShortTermMemory handles conversation-level context (session scope).
type ShortTermMemory interface {
	AddMessage(ctx context.Context, sessionID string, msg Message) error
	GetMessages(ctx context.Context, sessionID string, opts GetMessagesOpts) ([]Message, error)
	ClearSession(ctx context.Context, sessionID string) error
}

// LongTermMemory handles persistent knowledge across sessions.
type LongTermMemory interface {
	Add(ctx context.Context, userID string, content string, metadata map[string]any) (string, error)
	Search(ctx context.Context, userID string, query string, opts SearchOpts) ([]MemoryEntry, error)
	Update(ctx context.Context, memoryID string, content string) error
	Delete(ctx context.Context, memoryID string) error
	GetAll(ctx context.Context, userID string, opts ListOpts) ([]MemoryEntry, error)
}

// AgentStateMemory handles agent-level persistent state.
type AgentStateMemory interface {
	SetState(ctx context.Context, agentID string, key string, value any) error
	GetState(ctx context.Context, agentID string, key string) (any, bool, error)
	DeleteState(ctx context.Context, agentID string, key string) error
	GetAllState(ctx context.Context, agentID string) (map[string]any, error)
}

// Backend combines all memory capabilities.
type Backend interface {
	ShortTermMemory
	LongTermMemory
	AgentStateMemory

	// Lifecycle
	Init(ctx context.Context, config BackendConfig) error
	Close() error
	Name() string
}

// Message represents a single conversation message.
type Message struct {
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Timestamp int64          `json:"timestamp"`
}

// MemoryEntry represents a stored long-term memory item.
type MemoryEntry struct {
	ID        string         `json:"id"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Score     float64        `json:"score,omitempty"`
	CreatedAt int64          `json:"created_at"`
	UpdatedAt int64          `json:"updated_at"`
}

// GetMessagesOpts configures message retrieval.
type GetMessagesOpts struct {
	Limit  int
	Before int64 // unix timestamp upper bound
}

// SearchOpts configures semantic/keyword search.
type SearchOpts struct {
	Limit     int
	Threshold float64
	Filter    map[string]any
}

// ListOpts configures paginated listing.
type ListOpts struct {
	Limit  int
	Offset int
}

// BackendConfig holds the configuration for a memory backend.
type BackendConfig struct {
	Type     string         `yaml:"type"`    // builtin, mem0, zep, letta
	Endpoint string         `yaml:"endpoint"`
	APIKey   string         `yaml:"api_key"`
	Options  map[string]any `yaml:"options"`
}
