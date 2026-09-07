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

package agentdef

import (
	"context"
	"log"
	"time"

	aclagent "github.com/superagent-ai/superagent-base/backend/pkg/agent"
	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

type auditLogMiddleware struct {
	aclagent.BaseMiddleware
	level           string
	includeMessages bool
}

type auditCtxKey struct{}

type auditState struct {
	startTime   time.Time
	iterations  int
	toolCalls   int
}

func (m *auditLogMiddleware) BeforeAgent(ctx context.Context, mCtx *aclagent.MiddlewareContext) (context.Context, error) {
	state := &auditState{startTime: time.Now()}
	ctx = context.WithValue(ctx, auditCtxKey{}, state)
	log.Printf("[audit_log] agent run started, tools=%d", len(mCtx.Tools))
	return ctx, nil
}

func (m *auditLogMiddleware) BeforeModel(ctx context.Context, mState *aclagent.MiddlewareState) (context.Context, error) {
	if as, ok := ctx.Value(auditCtxKey{}).(*auditState); ok {
		as.iterations++
	}
	return ctx, nil
}

func (m *auditLogMiddleware) AfterModel(ctx context.Context, mState *aclagent.MiddlewareState) error {
	if as, ok := ctx.Value(auditCtxKey{}).(*auditState); ok && len(mState.Messages) > 0 {
		last := mState.Messages[len(mState.Messages)-1]
		if last.Role == llm.RoleAssistant && len(last.ToolCalls) > 0 {
			as.toolCalls += len(last.ToolCalls)
		}
	}
	return nil
}

func (m *auditLogMiddleware) AfterAgent(ctx context.Context, mCtx *aclagent.MiddlewareContext) error {
	as, _ := ctx.Value(auditCtxKey{}).(*auditState)
	if as == nil {
		return nil
	}
	elapsed := time.Since(as.startTime)
	msgCount := len(mCtx.Messages)
	log.Printf("[audit_log] agent run completed: duration=%s iterations=%d tool_calls=%d messages=%d",
		elapsed, as.iterations, as.toolCalls, msgCount)
	if m.includeMessages && msgCount > 0 {
		last := mCtx.Messages[msgCount-1]
		log.Printf("[audit_log] final_message: role=%s content_len=%d", last.Role, len(last.Content))
	}
	return nil
}

func buildAuditLogHandler(_ context.Context, cfg map[string]any) (aclagent.Middleware, error) {
	m := &auditLogMiddleware{level: "info"}
	if v, ok := cfg["level"]; ok {
		if s, ok := v.(string); ok {
			m.level = s
		}
	}
	if v, ok := cfg["include_messages"]; ok {
		if b, ok := v.(bool); ok {
			m.includeMessages = b
		}
	}
	return m, nil
}

func init() { RegisterMiddleware("audit_log", buildAuditLogHandler) }
