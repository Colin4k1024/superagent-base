package agentdef

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

var _ adk.ChatModelAgentMiddleware = (*postProcessingMiddleware)(nil)

type postProcessingMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	maxOutputLength int
	stripPII        bool
	piiPatterns     []string
}

func (m *postProcessingMiddleware) AfterModelRewriteState(ctx context.Context, state *adk.ChatModelAgentState, mc *adk.ModelContext) (context.Context, *adk.ChatModelAgentState, error) {
	if len(state.Messages) == 0 {
		return ctx, state, nil
	}
	last := state.Messages[len(state.Messages)-1]
	if last.Role != schema.Assistant {
		return ctx, state, nil
	}

	content := last.Content

	if m.maxOutputLength > 0 && len(content) > m.maxOutputLength {
		content = content[:m.maxOutputLength] + "\n[truncated by post_processing middleware]"
	}

	if m.stripPII && len(m.piiPatterns) > 0 {
		for _, pattern := range m.piiPatterns {
			content = strings.ReplaceAll(content, pattern, "[REDACTED]")
		}
	}

	if content != last.Content {
		last.Content = content
	}
	return ctx, state, nil
}

func buildPostProcessingHandler(_ context.Context, cfg map[string]any) (adk.ChatModelAgentMiddleware, error) {
	m := &postProcessingMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
	}
	if v, ok := cfg["max_output_length"]; ok {
		m.maxOutputLength = toInt(v)
	}
	if v, ok := cfg["strip_pii"]; ok {
		if b, ok := v.(bool); ok {
			m.stripPII = b
		}
	}
	if v, ok := cfg["pii_patterns"]; ok {
		m.piiPatterns = toStringSlice(v)
	}
	return m, nil
}

func init() { RegisterMiddleware("post_processing", buildPostProcessingHandler) }
