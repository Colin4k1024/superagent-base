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

package coze

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/superagent-ai/superagent-base/backend/pkg/memory"
)

// MemoryHandler exposes long-term memory and agent state APIs.
type MemoryHandler struct {
	mem memory.Backend
}

// NewMemoryHandler creates a MemoryHandler backed by the supplied memory store.
// If mem is nil every request returns 503.
func NewMemoryHandler(mem memory.Backend) *MemoryHandler {
	return &MemoryHandler{mem: mem}
}

// ---------------------------------------------------------------------------
// Long-term memory endpoints
// ---------------------------------------------------------------------------

// HandleLTMList returns all long-term memory entries for a user.
//
// GET /api/v2/memory/long-term?user_id=<id>&limit=<n>&offset=<n>
func (h *MemoryHandler) HandleLTMList(ctx context.Context, c *app.RequestContext) {
	if h.mem == nil {
		c.JSON(503, map[string]any{"error": "memory backend not available"})
		return
	}

	userID := string(c.QueryArgs().Peek("user_id"))
	if userID == "" {
		c.JSON(400, map[string]any{"error": "user_id is required"})
		return
	}

	limit := 50
	if v := string(c.QueryArgs().Peek("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 500 {
				n = 500
			}
			limit = n
		}
	}

	offset := 0
	if v := string(c.QueryArgs().Peek("offset")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	entries, err := h.mem.GetAll(ctx, userID, memory.ListOpts{Limit: limit, Offset: offset})
	if err != nil {
		c.JSON(500, map[string]any{"error": err.Error()})
		return
	}
	if entries == nil {
		entries = []memory.MemoryEntry{}
	}
	c.JSON(200, map[string]any{"user_id": userID, "memories": entries, "count": len(entries)})
}

// HandleLTMAdd stores a new long-term memory entry for a user.
//
// POST /api/v2/memory/long-term
// Body: {"user_id": "...", "content": "...", "metadata": {...}}
func (h *MemoryHandler) HandleLTMAdd(ctx context.Context, c *app.RequestContext) {
	if h.mem == nil {
		c.JSON(503, map[string]any{"error": "memory backend not available"})
		return
	}

	var req struct {
		UserID   string         `json:"user_id"`
		Content  string         `json:"content"`
		Metadata map[string]any `json:"metadata"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]any{"error": "invalid request body: " + err.Error()})
		return
	}
	if req.UserID == "" {
		c.JSON(400, map[string]any{"error": "user_id is required"})
		return
	}
	if req.Content == "" {
		c.JSON(400, map[string]any{"error": "content is required"})
		return
	}

	id, err := h.mem.Add(ctx, req.UserID, req.Content, req.Metadata)
	if err != nil {
		c.JSON(500, map[string]any{"error": err.Error()})
		return
	}
	c.JSON(201, map[string]any{"id": id})
}

// HandleLTMSearch performs semantic or keyword search over long-term memory.
//
// GET /api/v2/memory/long-term/search?user_id=<id>&q=<query>&limit=<n>&threshold=<f>
func (h *MemoryHandler) HandleLTMSearch(ctx context.Context, c *app.RequestContext) {
	if h.mem == nil {
		c.JSON(503, map[string]any{"error": "memory backend not available"})
		return
	}

	userID := string(c.QueryArgs().Peek("user_id"))
	query := string(c.QueryArgs().Peek("q"))
	if userID == "" {
		c.JSON(400, map[string]any{"error": "user_id is required"})
		return
	}
	if query == "" {
		c.JSON(400, map[string]any{"error": "q is required"})
		return
	}

	limit := 10
	if v := string(c.QueryArgs().Peek("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 100 {
				n = 100
			}
			limit = n
		}
	}

	threshold := 0.0
	if v := string(c.QueryArgs().Peek("threshold")); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			threshold = f
		}
	}

	results, err := h.mem.Search(ctx, userID, query, memory.SearchOpts{
		Limit:     limit,
		Threshold: threshold,
	})
	if err != nil {
		c.JSON(500, map[string]any{"error": err.Error()})
		return
	}
	if results == nil {
		results = []memory.MemoryEntry{}
	}
	c.JSON(200, map[string]any{"user_id": userID, "query": query, "results": results, "count": len(results)})
}

// HandleLTMUpdate replaces the content of an existing long-term memory entry.
//
// PUT /api/v2/memory/long-term/:id
// Body: {"content": "..."}
func (h *MemoryHandler) HandleLTMUpdate(ctx context.Context, c *app.RequestContext) {
	if h.mem == nil {
		c.JSON(503, map[string]any{"error": "memory backend not available"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(400, map[string]any{"error": "id is required"})
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]any{"error": "invalid request body: " + err.Error()})
		return
	}
	if req.Content == "" {
		c.JSON(400, map[string]any{"error": "content is required"})
		return
	}

	if err := h.mem.Update(ctx, id, req.Content); err != nil {
		c.JSON(500, map[string]any{"error": err.Error()})
		return
	}
	c.JSON(200, map[string]any{"status": "updated", "id": id})
}

// HandleLTMDelete removes a long-term memory entry.
//
// DELETE /api/v2/memory/long-term/:id
func (h *MemoryHandler) HandleLTMDelete(ctx context.Context, c *app.RequestContext) {
	if h.mem == nil {
		c.JSON(503, map[string]any{"error": "memory backend not available"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(400, map[string]any{"error": "id is required"})
		return
	}

	if err := h.mem.Delete(ctx, id); err != nil {
		c.JSON(500, map[string]any{"error": err.Error()})
		return
	}
	c.JSON(200, map[string]any{"status": "deleted", "id": id})
}

// ---------------------------------------------------------------------------
// Agent state endpoints
// ---------------------------------------------------------------------------

// HandleAgentStateGetAll returns all persisted state keys for an agent.
//
// GET /api/v2/agents/:agent_id/state
func (h *MemoryHandler) HandleAgentStateGetAll(ctx context.Context, c *app.RequestContext) {
	if h.mem == nil {
		c.JSON(503, map[string]any{"error": "memory backend not available"})
		return
	}

	agentID := c.Param("agent_id")
	if agentID == "" {
		c.JSON(400, map[string]any{"error": "agent_id is required"})
		return
	}

	state, err := h.mem.GetAllState(ctx, agentID)
	if err != nil {
		c.JSON(500, map[string]any{"error": err.Error()})
		return
	}
	if state == nil {
		state = map[string]any{}
	}
	c.JSON(200, map[string]any{"agent_id": agentID, "state": state})
}

// HandleAgentStateGet returns a single state key for an agent.
//
// GET /api/v2/agents/:agent_id/state/:key
func (h *MemoryHandler) HandleAgentStateGet(ctx context.Context, c *app.RequestContext) {
	if h.mem == nil {
		c.JSON(503, map[string]any{"error": "memory backend not available"})
		return
	}

	agentID := c.Param("agent_id")
	key := c.Param("key")
	if agentID == "" || key == "" {
		c.JSON(400, map[string]any{"error": "agent_id and key are required"})
		return
	}

	value, found, err := h.mem.GetState(ctx, agentID, key)
	if err != nil {
		c.JSON(500, map[string]any{"error": err.Error()})
		return
	}
	if !found {
		c.JSON(404, map[string]any{"error": "key not found"})
		return
	}
	c.JSON(200, map[string]any{"agent_id": agentID, "key": key, "value": value})
}

// HandleAgentStateSet creates or updates a state key for an agent.
//
// POST /api/v2/agents/:agent_id/state
// Body: {"key": "...", "value": <any>}
func (h *MemoryHandler) HandleAgentStateSet(ctx context.Context, c *app.RequestContext) {
	if h.mem == nil {
		c.JSON(503, map[string]any{"error": "memory backend not available"})
		return
	}

	agentID := c.Param("agent_id")
	if agentID == "" {
		c.JSON(400, map[string]any{"error": "agent_id is required"})
		return
	}

	var req struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]any{"error": "invalid request body: " + err.Error()})
		return
	}
	if req.Key == "" {
		c.JSON(400, map[string]any{"error": "key is required"})
		return
	}

	if err := h.mem.SetState(ctx, agentID, req.Key, req.Value); err != nil {
		c.JSON(500, map[string]any{"error": err.Error()})
		return
	}
	c.JSON(200, map[string]any{"status": "ok", "agent_id": agentID, "key": req.Key})
}

// HandleAgentStateDelete removes a state key for an agent.
//
// DELETE /api/v2/agents/:agent_id/state/:key
func (h *MemoryHandler) HandleAgentStateDelete(ctx context.Context, c *app.RequestContext) {
	if h.mem == nil {
		c.JSON(503, map[string]any{"error": "memory backend not available"})
		return
	}

	agentID := c.Param("agent_id")
	key := c.Param("key")
	if agentID == "" || key == "" {
		c.JSON(400, map[string]any{"error": "agent_id and key are required"})
		return
	}

	if err := h.mem.DeleteState(ctx, agentID, key); err != nil {
		c.JSON(500, map[string]any{"error": err.Error()})
		return
	}
	c.JSON(200, map[string]any{"status": "deleted", "agent_id": agentID, "key": key})
}
