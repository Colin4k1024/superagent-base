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

// SessionHandler exposes conversation session history via the v2 API.
// It reads from and clears the shared short-term memory backend so callers
// can inspect or reset any session without going through an agent.
type SessionHandler struct {
	mem memory.ShortTermMemory
}

// NewSessionHandler creates a SessionHandler backed by the supplied memory store.
func NewSessionHandler(mem memory.ShortTermMemory) *SessionHandler {
	return &SessionHandler{mem: mem}
}

// HandleGetMessages returns the message history for a session.
//
// GET /api/v2/sessions/:session_id/messages
// Query params:
//
//	limit  – max number of messages to return (default 50, max 200)
//	before – unix timestamp upper bound (optional)
func (h *SessionHandler) HandleGetMessages(ctx context.Context, c *app.RequestContext) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(400, map[string]any{"error": "session_id is required"})
		return
	}

	limit := 50
	if v := string(c.QueryArgs().Peek("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > 200 {
				n = 200
			}
			limit = n
		}
	}

	var before int64
	if v := string(c.QueryArgs().Peek("before")); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			before = ts
		}
	}

	msgs, err := h.mem.GetMessages(ctx, sessionID, memory.GetMessagesOpts{
		Limit:  limit,
		Before: before,
	})
	if err != nil {
		c.JSON(500, map[string]any{"error": err.Error()})
		return
	}

	if msgs == nil {
		msgs = []memory.Message{}
	}

	c.JSON(200, map[string]any{
		"session_id": sessionID,
		"messages":   msgs,
		"count":      len(msgs),
	})
}

// HandleClearSession deletes all messages for a session.
//
// DELETE /api/v2/sessions/:session_id
func (h *SessionHandler) HandleClearSession(ctx context.Context, c *app.RequestContext) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		c.JSON(400, map[string]any{"error": "session_id is required"})
		return
	}

	if err := h.mem.ClearSession(ctx, sessionID); err != nil {
		c.JSON(500, map[string]any{"error": err.Error()})
		return
	}

	c.JSON(200, map[string]any{
		"status":     "cleared",
		"session_id": sessionID,
	})
}
