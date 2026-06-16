package agentdef

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var _ adk.ChatModelAgentMiddleware = (*toolPermissionMiddleware)(nil)

type toolPermissionMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	allow []string
	deny  []string
}

func (m *toolPermissionMiddleware) isAllowed(toolName string) bool {
	if len(m.deny) > 0 {
		for _, pattern := range m.deny {
			if matchPattern(pattern, toolName) {
				return false
			}
		}
	}
	if len(m.allow) > 0 {
		for _, pattern := range m.allow {
			if matchPattern(pattern, toolName) {
				return true
			}
		}
		return false
	}
	return true
}

func matchPattern(pattern, name string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(name, pattern[:len(pattern)-1])
	}
	return pattern == name
}

func (m *toolPermissionMiddleware) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	if m.isAllowed(tCtx.Name) {
		return endpoint, nil
	}
	return func(_ context.Context, _ string, _ ...tool.Option) (string, error) {
		return "", fmt.Errorf("tool %q denied by tool_permission middleware", tCtx.Name)
	}, nil
}

func (m *toolPermissionMiddleware) WrapStreamableToolCall(ctx context.Context, endpoint adk.StreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.StreamableToolCallEndpoint, error) {
	if m.isAllowed(tCtx.Name) {
		return endpoint, nil
	}
	return func(_ context.Context, _ string, _ ...tool.Option) (*schema.StreamReader[string], error) {
		return nil, fmt.Errorf("tool %q denied by tool_permission middleware", tCtx.Name)
	}, nil
}

func (m *toolPermissionMiddleware) WrapEnhancedInvokableToolCall(ctx context.Context, endpoint adk.EnhancedInvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.EnhancedInvokableToolCallEndpoint, error) {
	if m.isAllowed(tCtx.Name) {
		return endpoint, nil
	}
	return func(_ context.Context, _ *schema.ToolArgument, _ ...tool.Option) (*schema.ToolResult, error) {
		return nil, fmt.Errorf("tool %q denied by tool_permission middleware", tCtx.Name)
	}, nil
}

func (m *toolPermissionMiddleware) WrapEnhancedStreamableToolCall(ctx context.Context, endpoint adk.EnhancedStreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.EnhancedStreamableToolCallEndpoint, error) {
	if m.isAllowed(tCtx.Name) {
		return endpoint, nil
	}
	return func(_ context.Context, _ *schema.ToolArgument, _ ...tool.Option) (*schema.StreamReader[*schema.ToolResult], error) {
		return nil, fmt.Errorf("tool %q denied by tool_permission middleware", tCtx.Name)
	}, nil
}

func buildToolPermissionHandler(_ context.Context, cfg map[string]any) (adk.ChatModelAgentMiddleware, error) {
	m := &toolPermissionMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
	}
	if v, ok := cfg["allow"]; ok {
		m.allow = toStringSlice(v)
	}
	if v, ok := cfg["deny"]; ok {
		m.deny = toStringSlice(v)
	}
	return m, nil
}

func toStringSlice(v any) []string {
	switch t := v.(type) {
	case []any:
		result := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return t
	}
	return nil
}

func init() { RegisterMiddleware("tool_permission", buildToolPermissionHandler) }
