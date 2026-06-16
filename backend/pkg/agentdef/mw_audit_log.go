package agentdef

import (
	"context"
	"log"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

var _ adk.ChatModelAgentMiddleware = (*auditLogMiddleware)(nil)

type auditLogMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	level           string
	includeMessages bool
}

type auditCtxKey struct{}

type auditState struct {
	startTime  time.Time
	iterations int
	toolCalls  int
}

func (m *auditLogMiddleware) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	state := &auditState{startTime: time.Now()}
	ctx = context.WithValue(ctx, auditCtxKey{}, state)
	log.Printf("[audit_log] agent run started, tools=%d", len(runCtx.Tools))
	return ctx, runCtx, nil
}

func (m *auditLogMiddleware) AfterAgent(ctx context.Context, state *adk.ChatModelAgentState) (context.Context, error) {
	as, _ := ctx.Value(auditCtxKey{}).(*auditState)
	if as == nil {
		return ctx, nil
	}
	elapsed := time.Since(as.startTime)
	msgCount := len(state.Messages)
	log.Printf("[audit_log] agent run completed: duration=%s iterations=%d tool_calls=%d messages=%d",
		elapsed, as.iterations, as.toolCalls, msgCount)
	if m.includeMessages && msgCount > 0 {
		last := state.Messages[msgCount-1]
		log.Printf("[audit_log] final_message: role=%s content_len=%d", last.Role, len(last.Content))
	}
	return ctx, nil
}

func (m *auditLogMiddleware) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	if as, ok := ctx.Value(auditCtxKey{}).(*auditState); ok {
		as.iterations++
	}
	return ctx, state, nil
}

func (m *auditLogMiddleware) AfterModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	if as, ok := ctx.Value(auditCtxKey{}).(*auditState); ok && len(state.Messages) > 0 {
		last := state.Messages[len(state.Messages)-1]
		if last.Role == schema.Assistant && len(last.ToolCalls) > 0 {
			as.toolCalls += len(last.ToolCalls)
		}
	}
	return ctx, state, nil
}

func buildAuditLogHandler(_ context.Context, cfg map[string]any) (adk.ChatModelAgentMiddleware, error) {
	m := &auditLogMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		level:                        "info",
	}
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
