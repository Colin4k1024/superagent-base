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
	"encoding/json"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/sse"

	"github.com/superagent-ai/superagent-base/backend/pkg/a2ui"
	"github.com/superagent-ai/superagent-base/backend/pkg/agentdef"
	"github.com/superagent-ai/superagent-base/backend/pkg/observe"
)

// ChatSSEHandler provides HTTP endpoints for direct agent interaction
// without going through the full Coze conversation pipeline.
type ChatSSEHandler struct {
	runtime   *agentdef.AgentRuntime
	turnLoops *agentdef.TurnLoopManager
}

// NewChatSSEHandler creates a ChatSSEHandler.
func NewChatSSEHandler(rt *agentdef.AgentRuntime) *ChatSSEHandler {
	return &ChatSSEHandler{
		runtime:   rt,
		turnLoops: agentdef.NewTurnLoopManager(),
	}
}

// chatStreamRequest is the JSON body for POST /api/v1/chat/stream.
type chatStreamRequest struct {
	AgentID   string `json:"agent_id"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// HandleChatStream streams agent response tokens as Server-Sent Events.
//
// POST /api/v1/chat/stream
// Body: {"agent_id": "research-agent", "session_id": "s1", "message": "hello"}
//
// Legacy mode (default):
//
//	data: <token>\n\n  (one per streaming token)
//	data: [DONE]\n\n   (signals end of stream)
//
// A2UI mode (header X-A2UI: true or query param a2ui=true):
//
//	event: <type>\ndata: <json>\n\n
func (h *ChatSSEHandler) HandleChatStream(ctx context.Context, c *app.RequestContext) {
	var req chatStreamRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}
	if req.Message == "" {
		c.JSON(400, map[string]string{"error": "message is required"})
		return
	}
	if req.AgentID == "" {
		req.AgentID = "research-agent"
	}
	if req.SessionID == "" {
		req.SessionID = "default"
	}

	if h.runtime == nil {
		c.JSON(503, map[string]string{"error": "agent runtime not available"})
		return
	}

	agent, ok := h.runtime.GetAgent(req.AgentID)
	if !ok {
		c.JSON(404, map[string]string{"error": "agent not found: " + req.AgentID})
		return
	}

	// OBS-002: Track active sessions.
	observe.ActiveSessions.Inc()
	defer observe.ActiveSessions.Dec()

	// OBS-004: Record request metrics (count + latency).
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		observe.AgentRequestDuration.WithLabelValues(req.AgentID).Observe(duration)
	}()

	// Detect A2UI mode from request header or query parameter.
	useA2UI := string(c.GetHeader("X-A2UI")) == "true" ||
		string(c.QueryArgs().Peek("a2ui")) == "true"

	// OBS-004: Track request count by mode.
	mode := "legacy"
	if useA2UI {
		mode = "a2ui"
	}
	observe.AgentRequestsByMode.WithLabelValues(req.AgentID, mode).Inc()

	// OBS-003: Create OTel Agent span (no-op when OTEL_ENABLED=false).
	ctx, span := observe.StartAgentSpan(ctx, req.AgentID, "chat")
	defer span.End()

	// Set SSE-specific headers before writing the first byte.
	c.Response.Header.Set("Cache-Control", "no-cache")
	c.Response.Header.Set("Connection", "keep-alive")
	c.Response.Header.Set("X-Accel-Buffering", "no")
	c.Response.Header.Set("Access-Control-Allow-Origin", "*")

	w := sse.NewWriter(c)
	defer func() { _ = w.Close() }()

	if useA2UI {
		stream := a2ui.NewEventStream(200)
		eventCtx := agentdef.ContextWithA2UIStream(ctx, stream)
		ch, handled, err := h.turnLoops.Chat(eventCtx, req.AgentID, agent, req.SessionID, req.Message)
		if !handled {
			ch, err = agent.Chat(eventCtx, req.SessionID, req.Message)
		}
		stream = agentdef.TokenStreamToEventStream(eventCtx, ch, err, stream)
		for evt := range stream.Chan() {
			data, _ := json.Marshal(evt)
			if writeErr := w.WriteEvent("", string(evt.Type), data); writeErr != nil {
				go func() {
					for range stream.Chan() {
					}
				}()
				return
			}
		}
		return
	}

	// Legacy mode: plain text tokens.
	ch, handled, err := h.turnLoops.Chat(ctx, req.AgentID, agent, req.SessionID, req.Message)
	if !handled {
		ch, err = agent.Chat(ctx, req.SessionID, req.Message)
	}
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}
	for token := range ch {
		if writeErr := w.WriteEvent("", "message", []byte(token)); writeErr != nil {
			// Client disconnected; drain the channel and return.
			go func() {
				for range ch {
				}
			}()
			return
		}
	}
	// Signal stream completion.
	_ = w.WriteEvent("", "message", []byte("[DONE]"))
}

// chatAbortRequest is the JSON body for POST /api/v1/chat/abort.
type chatAbortRequest struct {
	AgentID   string `json:"agent_id"`
	SessionID string `json:"session_id"`
}

// HandleChatAbort stops an active TurnLoop-backed chat stream.
func (h *ChatSSEHandler) HandleChatAbort(_ context.Context, c *app.RequestContext) {
	var req chatAbortRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}
	if req.AgentID == "" {
		req.AgentID = "research-agent"
	}
	if req.SessionID == "" {
		c.JSON(400, map[string]string{"error": "session_id is required"})
		return
	}
	if h.turnLoops == nil {
		c.JSON(200, map[string]string{"status": "no_active_loop"})
		return
	}

	if h.turnLoops.Abort(req.AgentID, req.SessionID) {
		c.JSON(200, map[string]string{"status": "aborted"})
		return
	}
	c.JSON(200, map[string]string{"status": "no_active_loop"})
}

// chatResumeRequest is the JSON body for POST /api/v1/chat/resume.
type chatResumeRequest struct {
	AgentID   string         `json:"agent_id"`
	SessionID string         `json:"session_id"`
	Input     map[string]any `json:"input"`
}

// HandleChatResume resumes an interrupted agent conversation.
//
// POST /api/v1/chat/resume
// Body: {"agent_id": "approval-agent", "session_id": "s1", "input": {"confirm": true}}
//
// Streams the resumed response as plain SSE tokens followed by [DONE].
// Returns 404 if the agent is not found or not interruptable.
// Returns 409 if there is no pending interrupt for the given session.
func (h *ChatSSEHandler) HandleChatResume(ctx context.Context, c *app.RequestContext) {
	var req chatResumeRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]string{"error": "invalid request body: " + err.Error()})
		return
	}
	if req.AgentID == "" {
		c.JSON(400, map[string]string{"error": "agent_id is required"})
		return
	}
	if req.SessionID == "" {
		c.JSON(400, map[string]string{"error": "session_id is required"})
		return
	}

	if h.runtime == nil {
		c.JSON(503, map[string]string{"error": "agent runtime not available"})
		return
	}

	rawAgent, ok := h.runtime.GetAgent(req.AgentID)
	if !ok {
		c.JSON(404, map[string]string{"error": "agent not found: " + req.AgentID})
		return
	}

	interruptable, ok := rawAgent.(agentdef.Interruptable)
	if !ok {
		c.JSON(404, map[string]string{"error": "agent does not support interrupt/resume: " + req.AgentID})
		return
	}

	// Verify a pending interrupt exists before streaming.
	if _, hasPending := interruptable.GetInterruptState(ctx, req.SessionID); !hasPending {
		c.JSON(409, map[string]string{"error": "no pending interrupt for session: " + req.SessionID})
		return
	}

	ch, err := interruptable.Resume(ctx, req.SessionID, req.Input)
	if err != nil {
		c.JSON(500, map[string]string{"error": err.Error()})
		return
	}

	c.Response.Header.Set("Cache-Control", "no-cache")
	c.Response.Header.Set("Connection", "keep-alive")
	c.Response.Header.Set("X-Accel-Buffering", "no")
	c.Response.Header.Set("Access-Control-Allow-Origin", "*")

	w := sse.NewWriter(c)
	defer func() { _ = w.Close() }()

	for token := range ch {
		if writeErr := w.WriteEvent("", "message", []byte(token)); writeErr != nil {
			go func() {
				for range ch {
				}
			}()
			return
		}
	}
	_ = w.WriteEvent("", "message", []byte("[DONE]"))
}

// HandleGetInterruptState returns the current interrupt state for a session.
//
// GET /api/v1/chat/interrupt_state?agent_id=<id>&session_id=<id>
func (h *ChatSSEHandler) HandleGetInterruptState(ctx context.Context, c *app.RequestContext) {
	agentID := string(c.QueryArgs().Peek("agent_id"))
	sessionID := string(c.QueryArgs().Peek("session_id"))

	if agentID == "" || sessionID == "" {
		c.JSON(400, map[string]string{"error": "agent_id and session_id are required"})
		return
	}
	if h.runtime == nil {
		c.JSON(503, map[string]string{"error": "agent runtime not available"})
		return
	}

	rawAgent, ok := h.runtime.GetAgent(agentID)
	if !ok {
		c.JSON(404, map[string]string{"error": "agent not found: " + agentID})
		return
	}

	interruptable, ok := rawAgent.(agentdef.Interruptable)
	if !ok {
		c.JSON(200, map[string]any{"interrupted": false})
		return
	}

	state, hasPending := interruptable.GetInterruptState(ctx, sessionID)
	if !hasPending {
		c.JSON(200, map[string]any{"interrupted": false})
		return
	}

	data, _ := json.Marshal(state)
	c.JSON(200, map[string]any{"interrupted": true, "state": json.RawMessage(data)})
}

// HandleListAgents returns JSON describing all agents known to the runtime.
//
// GET /api/v1/agents
func (h *ChatSSEHandler) HandleListAgents(_ context.Context, c *app.RequestContext) {
	if h.runtime == nil {
		c.JSON(503, map[string]string{"error": "agent runtime not available"})
		return
	}

	names := h.runtime.ListAgents()
	agents := make([]map[string]string, 0, len(names))
	for _, name := range names {
		a, ok := h.runtime.GetAgent(name)
		if !ok {
			continue
		}
		agents = append(agents, map[string]string{
			"name":        name,
			"description": a.Description(),
		})
	}
	c.JSON(200, map[string]any{"agents": agents})
}
