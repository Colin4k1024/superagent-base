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
	"bytes"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/ut"
	"github.com/stretchr/testify/require"

	"github.com/superagent-ai/superagent-base/backend/pkg/agentdef"
)

func TestChatSSEHandler_HandleChatAbortNoActiveLoop(t *testing.T) {
	h := server.Default()
	handler := &ChatSSEHandler{turnLoops: agentdef.NewTurnLoopManager()}
	h.POST("/api/v2/chat/abort", handler.HandleChatAbort)

	body, err := sonic.Marshal(chatAbortRequest{AgentID: "research-agent", SessionID: "s1"})
	require.NoError(t, err)

	w := ut.PerformRequest(
		h.Engine,
		"POST",
		"/api/v2/chat/abort",
		&ut.Body{Body: bytes.NewBuffer(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, 200, w.Code)
	require.JSONEq(t, `{"status":"no_active_loop"}`, string(w.Result().Body()))
}

func TestChatSSEHandler_HandleChatAbortRequiresSession(t *testing.T) {
	h := server.Default()
	handler := &ChatSSEHandler{turnLoops: agentdef.NewTurnLoopManager()}
	h.POST("/api/v1/chat/abort", handler.HandleChatAbort)

	body, err := sonic.Marshal(chatAbortRequest{AgentID: "research-agent"})
	require.NoError(t, err)

	w := ut.PerformRequest(
		h.Engine,
		"POST",
		"/api/v1/chat/abort",
		&ut.Body{Body: bytes.NewBuffer(body), Len: len(body)},
		ut.Header{Key: "Content-Type", Value: "application/json"},
	)

	require.Equal(t, 400, w.Code)
	require.JSONEq(t, `{"error":"session_id is required"}`, string(w.Result().Body()))
}
