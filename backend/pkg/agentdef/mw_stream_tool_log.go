package agentdef

import (
	"context"
	"log"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var _ adk.ChatModelAgentMiddleware = (*streamToolLogMiddleware)(nil)

type streamToolLogMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	logChunks bool
}

func (m *streamToolLogMiddleware) WrapStreamableToolCall(ctx context.Context, endpoint adk.StreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.StreamableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		start := time.Now()
		log.Printf("[stream_tool_log] tool=%s call_id=%s started", tCtx.Name, tCtx.CallID)

		reader, err := endpoint(ctx, args, opts...)
		if err != nil {
			log.Printf("[stream_tool_log] tool=%s call_id=%s error=%v duration=%s", tCtx.Name, tCtx.CallID, err, time.Since(start))
			return nil, err
		}

		log.Printf("[stream_tool_log] tool=%s call_id=%s stream_opened duration=%s", tCtx.Name, tCtx.CallID, time.Since(start))
		return reader, nil
	}, nil
}

func (m *streamToolLogMiddleware) WrapEnhancedStreamableToolCall(ctx context.Context, endpoint adk.EnhancedStreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.EnhancedStreamableToolCallEndpoint, error) {
	return func(ctx context.Context, arg *schema.ToolArgument, opts ...tool.Option) (*schema.StreamReader[*schema.ToolResult], error) {
		start := time.Now()
		log.Printf("[stream_tool_log] enhanced tool=%s call_id=%s started", tCtx.Name, tCtx.CallID)

		reader, err := endpoint(ctx, arg, opts...)
		if err != nil {
			log.Printf("[stream_tool_log] enhanced tool=%s call_id=%s error=%v duration=%s", tCtx.Name, tCtx.CallID, err, time.Since(start))
			return nil, err
		}

		log.Printf("[stream_tool_log] enhanced tool=%s call_id=%s stream_opened duration=%s", tCtx.Name, tCtx.CallID, time.Since(start))
		return reader, nil
	}, nil
}

func buildStreamToolLogHandler(_ context.Context, cfg map[string]any) (adk.ChatModelAgentMiddleware, error) {
	m := &streamToolLogMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
	}
	if v, ok := cfg["log_chunks"]; ok {
		if b, ok := v.(bool); ok {
			m.logChunks = b
		}
	}
	return m, nil
}

func init() { RegisterMiddleware("stream_tool_log", buildStreamToolLogHandler) }
