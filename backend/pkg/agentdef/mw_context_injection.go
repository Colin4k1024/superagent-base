package agentdef

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

var _ adk.ChatModelAgentMiddleware = (*contextInjectionMiddleware)(nil)

type contextInjectionMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	injectTimestamp       bool
	injectSessionMetadata bool
	staticContext         string
}

func (m *contextInjectionMiddleware) BeforeModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	var injections []string

	if m.injectTimestamp {
		injections = append(injections, fmt.Sprintf("[current_time: %s]", time.Now().Format(time.RFC3339)))
	}
	if m.staticContext != "" {
		injections = append(injections, m.staticContext)
	}

	if len(injections) == 0 {
		return ctx, state, nil
	}

	contextMsg := schema.SystemMessage("[context_injection]\n" + joinStrings(injections, "\n"))
	state.Messages = append([]*schema.Message{contextMsg}, state.Messages...)
	return ctx, state, nil
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += sep + s
	}
	return result
}

func buildContextInjectionHandler(_ context.Context, cfg map[string]any) (adk.ChatModelAgentMiddleware, error) {
	m := &contextInjectionMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
	}
	if v, ok := cfg["inject_timestamp"]; ok {
		if b, ok := v.(bool); ok {
			m.injectTimestamp = b
		}
	}
	if v, ok := cfg["inject_session_metadata"]; ok {
		if b, ok := v.(bool); ok {
			m.injectSessionMetadata = b
		}
	}
	if v, ok := cfg["static_context"]; ok {
		if s, ok := v.(string); ok {
			m.staticContext = s
		}
	}
	return m, nil
}

func init() { RegisterMiddleware("context_injection", buildContextInjectionHandler) }
