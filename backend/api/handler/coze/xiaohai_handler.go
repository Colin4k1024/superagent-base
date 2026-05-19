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
	"os"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	hsse "github.com/cloudwego/hertz/pkg/protocol/sse"

	"github.com/superagent-ai/superagent-base/backend/pkg/a2ui"
	"github.com/superagent-ai/superagent-base/backend/pkg/agentdef"
)

// XiaohaiHandler implements the 集团IT智能体对接规范 (普通API类) endpoints.
// It translates internal A2UI events into the 集团IT智能体输出规范 v1.0.0 format.
type XiaohaiHandler struct {
	runtime *agentdef.AgentRuntime
}

// NewXiaohaiHandler creates a XiaohaiHandler.
func NewXiaohaiHandler(rt *agentdef.AgentRuntime) *XiaohaiHandler {
	return &XiaohaiHandler{runtime: rt}
}

// xiaohaiRequest is the JSON body conforming to the 普通API类入参规范.
type xiaohaiRequest struct {
	UserQuery    string             `json:"userQuery"`
	UserToken    string             `json:"userToken"`
	TerminalType string             `json:"terminalType"`
	SessionID    string             `json:"sessionId"`
	HisMsg       []xiaohaiHisMsg    `json:"hisMsg"`
	FileList     []xiaohaiFile      `json:"fileList"`
	ExtParam     map[string]any     `json:"ext_param"`
}

type xiaohaiHisMsg struct {
	Human string `json:"human"`
	AI    string `json:"ai"`
}

type xiaohaiFile struct {
	FileName string `json:"fileName"`
	FileURL  string `json:"fileUrl"`
}

// HandleStream handles streaming requests from the 小海 primary agent.
// POST /api/v1/xiaohai/stream/:agent_id
// Headers: Api-Key, Access-Token
func (h *XiaohaiHandler) HandleStream(ctx context.Context, c *app.RequestContext) {
	// Validate Api-Key.
	apiKey := string(c.GetHeader("Api-Key"))
	expectedKey := os.Getenv("ADMIN_API_KEY")
	if expectedKey != "" && apiKey != expectedKey {
		c.JSON(401, map[string]any{"code": 1, "message": "invalid Api-Key"})
		return
	}

	// Parse agent_id from path.
	agentID := c.Param("agent_id")
	if agentID == "" {
		c.JSON(400, map[string]any{"code": 1, "message": "agent_id is required in path"})
		return
	}

	// Parse request body.
	var req xiaohaiRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]any{"code": 1, "message": "invalid request body: " + err.Error()})
		return
	}

	if req.UserQuery == "" {
		c.JSON(400, map[string]any{"code": 1, "message": "userQuery is required"})
		return
	}

	// Generate sessionId if not provided.
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = "xiaohai-" + agentID + "-" + req.UserToken
	}

	// Resolve agent.
	agent, ok := h.runtime.GetAgent(agentID)
	if !ok {
		c.JSON(404, map[string]any{"code": 1, "message": "agent not found: " + agentID})
		return
	}

	// Set SSE headers.
	c.Response.Header.Set("Content-Type", "text/event-stream")
	c.Response.Header.Set("Cache-Control", "no-cache")
	c.Response.Header.Set("Connection", "keep-alive")
	c.Response.Header.Set("X-Accel-Buffering", "no")
	c.Response.Header.Set("Access-Control-Allow-Origin", "*")

	w := hsse.NewWriter(c)
	defer func() { _ = w.Close() }()

	// Use A2UI event agent for structured events.
	eventAgent := agentdef.NewEventAgent(agent)
	stream, _ := eventAgent.ChatWithEvents(ctx, sessionID, req.UserQuery)

	for evt := range stream.Chan() {
		xiaohaiEvents := a2ui.ConvertA2UIToXiaohai(evt)
		for _, xe := range xiaohaiEvents {
			data, _ := json.Marshal(xe)
			if writeErr := w.WriteEvent("", "", data); writeErr != nil {
				// Client disconnected.
				go func() {
					for range stream.Chan() {
					}
				}()
				return
			}
		}
	}
}

// HandleNonStream handles non-streaming requests from the 小海 primary agent.
// POST /api/v1/xiaohai/chat/:agent_id
// Headers: Api-Key, Access-Token
func (h *XiaohaiHandler) HandleNonStream(ctx context.Context, c *app.RequestContext) {
	// Validate Api-Key.
	apiKey := string(c.GetHeader("Api-Key"))
	expectedKey := os.Getenv("ADMIN_API_KEY")
	if expectedKey != "" && apiKey != expectedKey {
		c.JSON(401, map[string]any{"code": 1, "message": "invalid Api-Key"})
		return
	}

	// Parse agent_id from path.
	agentID := c.Param("agent_id")
	if agentID == "" {
		c.JSON(400, map[string]any{"code": 1, "message": "agent_id is required in path"})
		return
	}

	// Parse request body.
	var req xiaohaiRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]any{"code": 1, "message": "invalid request body: " + err.Error()})
		return
	}

	if req.UserQuery == "" {
		c.JSON(400, map[string]any{"code": 1, "message": "userQuery is required"})
		return
	}

	// Generate sessionId if not provided.
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = "xiaohai-" + agentID + "-" + req.UserToken
	}

	// Resolve agent.
	agent, ok := h.runtime.GetAgent(agentID)
	if !ok {
		c.JSON(404, map[string]any{"code": 1, "message": "agent not found: " + agentID})
		return
	}

	// Call agent and collect full response.
	ch, err := agent.Chat(ctx, sessionID, req.UserQuery)
	if err != nil {
		c.JSON(500, map[string]any{"code": 1, "message": err.Error()})
		return
	}

	var sb strings.Builder
	for token := range ch {
		sb.WriteString(token)
	}

	// Return in 集团规范 non-streaming format.
	c.JSON(200, map[string]any{
		"code": 0,
		"data": &a2ui.XiaohaiEvent{
			Type: "answer",
			Data: &a2ui.XiaohaiData{
				ContentType: "markdown",
				Content:     sb.String(),
			},
			Version: "1.0.0",
		},
	})
}
