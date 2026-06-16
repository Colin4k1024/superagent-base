package agentdef

import (
	"context"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

var _ adk.ChatModelAgentMiddleware = (*modelFailoverMiddleware)(nil)

type modelFailoverMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
	fallbackModel string
	maxRetries    int
}

func (m *modelFailoverMiddleware) WrapModel(ctx context.Context, baseModel model.BaseModel[*schema.Message], mc *adk.ModelContext) (model.BaseModel[*schema.Message], error) {
	return &failoverModelWrapper{
		primary:    baseModel,
		maxRetries: m.maxRetries,
		modelName:  m.fallbackModel,
	}, nil
}

type failoverModelWrapper struct {
	primary    model.BaseModel[*schema.Message]
	maxRetries int
	modelName  string
}

func (w *failoverModelWrapper) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	var lastErr error
	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		result, err := w.primary.Generate(ctx, input, opts...)
		if err == nil {
			return result, nil
		}
		lastErr = err
		log.Printf("[model_failover] attempt %d/%d failed: %v", attempt+1, w.maxRetries+1, err)
	}
	return nil, lastErr
}

func (w *failoverModelWrapper) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	var lastErr error
	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		result, err := w.primary.Stream(ctx, input, opts...)
		if err == nil {
			return result, nil
		}
		lastErr = err
		log.Printf("[model_failover] stream attempt %d/%d failed: %v", attempt+1, w.maxRetries+1, err)
	}
	return nil, lastErr
}

func (w *failoverModelWrapper) BindTools(tools []*schema.ToolInfo) error {
	if binder, ok := w.primary.(interface{ BindTools([]*schema.ToolInfo) error }); ok {
		return binder.BindTools(tools)
	}
	return nil
}

func buildModelFailoverHandler(_ context.Context, cfg map[string]any) (adk.ChatModelAgentMiddleware, error) {
	m := &modelFailoverMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		maxRetries:                   1,
	}
	if v, ok := cfg["fallback_model"]; ok {
		if s, ok := v.(string); ok {
			m.fallbackModel = s
		}
	}
	if v, ok := cfg["max_retries"]; ok {
		m.maxRetries = toInt(v)
	}
	return m, nil
}

func init() { RegisterMiddleware("model_failover", buildModelFailoverHandler) }
