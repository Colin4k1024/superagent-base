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

package observe

import (
	"context"
	"sync"
	"time"
)

// traceStoreKey is a context key for carrying the active trace ID through request scope.
type traceStoreKey struct{}

// WithTraceID injects a trace ID into the context for span collection.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceStoreKey{}, traceID)
}

// TraceIDFromCtx extracts the trace ID from context.
func TraceIDFromCtx(ctx context.Context) string {
	if v, ok := ctx.Value(traceStoreKey{}).(string); ok {
		return v
	}
	return ""
}

// TraceSpan represents a single operation within a trace.
type TraceSpan struct {
	SpanID       string  `json:"id"`
	ParentID     string  `json:"parent_id,omitempty"`
	Name         string  `json:"name"`
	Component    string  `json:"level"` // maps to frontend "level" field
	StartTime    string  `json:"startTime,omitempty"`
	EndTime      string  `json:"endTime,omitempty"`
	DurationMs   float64 `json:"duration_ms"`
	Status       string  `json:"status"`
	ModelID      string  `json:"model,omitempty"`
	Provider     string  `json:"provider,omitempty"`
	InputTokens  int     `json:"promptTokens,omitempty"`
	OutputTokens int     `json:"completionTokens,omitempty"`
	TotalTokens  int     `json:"totalTokens,omitempty"`
	ToolName     string  `json:"tool_name,omitempty"`
	Error        string  `json:"error,omitempty"`
	Input        any     `json:"input,omitempty"`
	Output       any     `json:"output,omitempty"`
}

// TraceRecord represents a complete trace (one agent request lifecycle).
type TraceRecord struct {
	TraceID           string      `json:"id"`
	Name              string      `json:"name,omitempty"`
	AgentID           string      `json:"agent_id,omitempty"`
	SessionID         string      `json:"session_id,omitempty"`
	StartTime         time.Time   `json:"timestamp"`
	EndTime           time.Time   `json:"end_time,omitempty"`
	DurationMs        float64     `json:"latency"`
	Status            string      `json:"status"`
	Spans             []TraceSpan `json:"observations"`
	TotalInputTokens  int         `json:"total_input_tokens"`
	TotalOutputTokens int         `json:"total_output_tokens"`
	TotalCost         float64     `json:"totalCost,omitempty"`
}

// TraceQueryParams holds parameters for listing traces.
type TraceQueryParams struct {
	Page  int
	Limit int
	From  string
	To    string
	Name  string
}

// TraceStore is a thread-safe, fixed-capacity ring buffer for recent traces.
type TraceStore struct {
	mu       sync.RWMutex
	buf      []*TraceRecord
	head     int
	count    int
	capacity int
	active   map[string]*TraceRecord
}

// NewTraceStore creates a TraceStore with the given capacity.
func NewTraceStore(capacity int) *TraceStore {
	if capacity <= 0 {
		capacity = 500
	}
	return &TraceStore{
		buf:      make([]*TraceRecord, capacity),
		capacity: capacity,
		active:   make(map[string]*TraceRecord),
	}
}

// StartTrace begins tracking a new trace.
func (ts *TraceStore) StartTrace(traceID, name, agentID, sessionID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.active[traceID] = &TraceRecord{
		TraceID:   traceID,
		Name:      name,
		AgentID:   agentID,
		SessionID: sessionID,
		StartTime: time.Now(),
		Status:    "ok",
		Spans:     make([]TraceSpan, 0, 8),
	}
}

// AddSpan appends a span to an active trace.
func (ts *TraceStore) AddSpan(traceID string, span TraceSpan) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	tr, ok := ts.active[traceID]
	if !ok {
		return
	}
	tr.Spans = append(tr.Spans, span)
	tr.TotalInputTokens += span.InputTokens
	tr.TotalOutputTokens += span.OutputTokens
}

// EndTrace finalizes and commits the trace to the ring buffer.
func (ts *TraceStore) EndTrace(traceID, status string) *TraceRecord {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	tr, ok := ts.active[traceID]
	if !ok {
		return nil
	}
	delete(ts.active, traceID)

	tr.EndTime = time.Now()
	tr.DurationMs = float64(tr.EndTime.Sub(tr.StartTime).Milliseconds())
	if status != "" {
		tr.Status = status
	}

	// Insert into ring buffer
	ts.buf[ts.head] = tr
	ts.head = (ts.head + 1) % ts.capacity
	if ts.count < ts.capacity {
		ts.count++
	}

	return tr
}

// List returns traces matching the query, ordered newest-first.
func (ts *TraceStore) List(params TraceQueryParams) ([]*TraceRecord, int) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	var fromTime, toTime time.Time
	if params.From != "" {
		fromTime, _ = time.Parse(time.RFC3339, params.From)
	}
	if params.To != "" {
		toTime, _ = time.Parse(time.RFC3339, params.To)
	}

	// Collect matching traces newest-first
	matching := make([]*TraceRecord, 0, ts.count)
	for i := 0; i < ts.count; i++ {
		idx := (ts.head - 1 - i + ts.capacity) % ts.capacity
		tr := ts.buf[idx]
		if tr == nil {
			continue
		}
		if !fromTime.IsZero() && tr.StartTime.Before(fromTime) {
			continue
		}
		if !toTime.IsZero() && tr.StartTime.After(toTime) {
			continue
		}
		if params.Name != "" && tr.Name != params.Name && tr.AgentID != params.Name {
			continue
		}
		matching = append(matching, tr)
	}

	total := len(matching)
	start := (params.Page - 1) * params.Limit
	if start >= total {
		return nil, total
	}
	end := start + params.Limit
	if end > total {
		end = total
	}

	return matching[start:end], total
}

// Get retrieves a single trace by ID.
func (ts *TraceStore) Get(traceID string) *TraceRecord {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	// Check active traces first
	if tr, ok := ts.active[traceID]; ok {
		return tr
	}

	// Search ring buffer
	for i := 0; i < ts.count; i++ {
		idx := (ts.head - 1 - i + ts.capacity) % ts.capacity
		if ts.buf[idx] != nil && ts.buf[idx].TraceID == traceID {
			return ts.buf[idx]
		}
	}
	return nil
}

// Sessions returns distinct session IDs from stored traces.
func (ts *TraceStore) Sessions(page, limit int) ([]string, int) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	seen := make(map[string]struct{})
	sessions := make([]string, 0)
	for i := 0; i < ts.count; i++ {
		idx := (ts.head - 1 - i + ts.capacity) % ts.capacity
		tr := ts.buf[idx]
		if tr == nil || tr.SessionID == "" {
			continue
		}
		if _, ok := seen[tr.SessionID]; !ok {
			seen[tr.SessionID] = struct{}{}
			sessions = append(sessions, tr.SessionID)
		}
	}

	total := len(sessions)
	start := (page - 1) * limit
	if start >= total {
		return nil, total
	}
	end := start + limit
	if end > total {
		end = total
	}
	return sessions[start:end], total
}
