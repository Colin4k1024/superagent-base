package agentdef

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

var _ adk.ChatModelAgentMiddleware = (*guardrailsMiddleware)(nil)

type guardrailsMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	maxInputLength  int
	maxOutputLength int
	blockedPatterns []string
}

func (m *guardrailsMiddleware) BeforeAgent(ctx context.Context, runCtx *adk.ChatModelAgentContext) (context.Context, *adk.ChatModelAgentContext, error) {
	if m.maxInputLength > 0 && len(runCtx.Instruction) > m.maxInputLength {
		return ctx, runCtx, fmt.Errorf("guardrails: input exceeds max length (%d > %d)", len(runCtx.Instruction), m.maxInputLength)
	}
	return ctx, runCtx, nil
}

func (m *guardrailsMiddleware) AfterAgent(ctx context.Context, state *adk.ChatModelAgentState) (context.Context, error) {
	if len(state.Messages) == 0 {
		return ctx, nil
	}
	last := state.Messages[len(state.Messages)-1]
	if last.Role != schema.Assistant {
		return ctx, nil
	}
	content := last.Content
	if m.maxOutputLength > 0 && len(content) > m.maxOutputLength {
		return ctx, fmt.Errorf("guardrails: output exceeds max length (%d > %d)", len(content), m.maxOutputLength)
	}
	for _, pattern := range m.blockedPatterns {
		if strings.Contains(strings.ToLower(content), strings.ToLower(pattern)) {
			return ctx, fmt.Errorf("guardrails: output contains blocked pattern %q", pattern)
		}
	}
	return ctx, nil
}

func buildGuardrailsHandler(_ context.Context, cfg map[string]any) (adk.ChatModelAgentMiddleware, error) {
	m := &guardrailsMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
	}
	if v, ok := cfg["max_input_length"]; ok {
		m.maxInputLength = toInt(v)
	}
	if v, ok := cfg["max_output_length"]; ok {
		m.maxOutputLength = toInt(v)
	}
	if v, ok := cfg["blocked_patterns"]; ok {
		m.blockedPatterns = toStringSlice(v)
	}
	return m, nil
}

func toInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case float64:
		return int(t)
	case int64:
		return int(t)
	}
	return 0
}

func init() { RegisterMiddleware("guardrails", buildGuardrailsHandler) }
