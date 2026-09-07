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

package observe

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ObserveCallback implements ObservabilityCallback with OTel tracing
// and Prometheus metrics.  It is framework-agnostic — the eino adapter
// (eino_callback.go) converts eino events to ACL events and delegates here.
type ObserveCallback struct {
	tracer trace.Tracer
}

// NewObserveCallback creates a new ACL ObserveCallback.
func NewObserveCallback() *ObserveCallback {
	return &ObserveCallback{tracer: tracer()}
}

// spanKey is a private context key for storing active spans.
type spanKey struct{}

// startTimeKey is a private context key for storing operation start times.
type startTimeKey struct{}

// modelIDKey carries the model ID through agent context for callback labeling.
type modelIDKey struct{}

// providerKey carries the provider name through agent context.
type providerKey struct{}

// WithModelInfo injects model identity into ctx so callbacks can label metrics correctly.
func WithModelInfo(ctx context.Context, modelID, provider string) context.Context {
	ctx = context.WithValue(ctx, modelIDKey{}, modelID)
	return context.WithValue(ctx, providerKey{}, provider)
}

func modelInfoFromCtx(ctx context.Context) (modelID, provider string) {
	if v, ok := ctx.Value(modelIDKey{}).(string); ok && v != "" {
		modelID = v
	}
	if v, ok := ctx.Value(providerKey{}).(string); ok && v != "" {
		provider = v
	}
	return
}

func (c *ObserveCallback) OnStart(ctx context.Context, info CallbackRunInfo, input CallbackInput) context.Context {
	spanName, attrs := aclSpanNameAndAttrs(ctx, info)
	ctx, span := c.tracer.Start(ctx, spanName, trace.WithAttributes(attrs...))

	if info.Component == ComponentChatModel {
		if inputStr := aclMarshalInput(input); inputStr != "" {
			span.SetAttributes(attribute.String("input.value", inputStr))
		}
	}

	if lfCtx := getLangfuseContext(ctx); lfCtx != nil {
		lfCtx.applyToSpan(span)
	}

	ctx = context.WithValue(ctx, spanKey{}, span)
	ctx = context.WithValue(ctx, startTimeKey{}, time.Now())
	return ctx
}

func (c *ObserveCallback) OnEnd(ctx context.Context, info CallbackRunInfo, output CallbackOutput) context.Context {
	span, _ := ctx.Value(spanKey{}).(trace.Span)
	start, _ := ctx.Value(startTimeKey{}).(time.Time)
	elapsed := time.Since(start)

	switch info.Component {
	case ComponentChatModel:
		provider, modelID := aclResolveModelInfo(ctx, info)
		ModelRequestDuration.WithLabelValues(modelID, provider).Observe(elapsed.Seconds())
		if output.PromptTokens > 0 || output.CompletionTokens > 0 {
			ModelTokensTotal.WithLabelValues(modelID, provider, "input").Add(float64(output.PromptTokens))
			ModelTokensTotal.WithLabelValues(modelID, provider, "output").Add(float64(output.CompletionTokens))
		}
		if span != nil {
			span.SetAttributes(
				attribute.Int64("gen_ai.usage.input_tokens", int64(output.PromptTokens)),
				attribute.Int64("gen_ai.usage.output_tokens", int64(output.CompletionTokens)),
				attribute.Int64("gen_ai.usage.total_tokens", int64(output.PromptTokens+output.CompletionTokens)),
			)
			span.SetStatus(codes.Ok, "")
			span.End()
		}
		aclRecordSpanToStore(ctx, info, start, "ok", "", output.PromptTokens, output.CompletionTokens)

	case ComponentTool:
		ToolInvocationsTotal.WithLabelValues(info.Name, "success").Inc()
		if span != nil {
			span.SetStatus(codes.Ok, "")
			span.End()
		}
		aclRecordSpanToStore(ctx, info, start, "ok", "", 0, 0)

	default:
		if span != nil {
			span.SetStatus(codes.Ok, "")
			span.End()
		}
		aclRecordSpanToStore(ctx, info, start, "ok", "", 0, 0)
	}

	return ctx
}

func (c *ObserveCallback) OnEndWithStream(ctx context.Context, info CallbackRunInfo, reader *llm.StreamReader) context.Context {
	span, _ := ctx.Value(spanKey{}).(trace.Span)
	start, _ := ctx.Value(startTimeKey{}).(time.Time)

	go func() {
		var promptTokens, completionTokens int
		if reader != nil {
			for {
				chunk, err := reader.Recv()
				if err == io.EOF || err != nil {
					break
				}
				if chunk != nil {
					completionTokens++
				}
			}
		}
		provider, modelID := aclResolveModelInfo(ctx, info)
		elapsed := time.Since(start).Seconds()
		ModelRequestDuration.WithLabelValues(modelID, provider).Observe(elapsed)
		if promptTokens > 0 || completionTokens > 0 {
			ModelTokensTotal.WithLabelValues(modelID, provider, "input").Add(float64(promptTokens))
			ModelTokensTotal.WithLabelValues(modelID, provider, "output").Add(float64(completionTokens))
		}
		if span != nil {
			span.SetAttributes(
				attribute.Int64("gen_ai.usage.input_tokens", int64(promptTokens)),
				attribute.Int64("gen_ai.usage.output_tokens", int64(completionTokens)),
			)
			span.SetStatus(codes.Ok, "")
			span.End()
		}
		aclRecordSpanToStore(ctx, info, start, "ok", "", promptTokens, completionTokens)
	}()

	return ctx
}

func (c *ObserveCallback) OnError(ctx context.Context, info CallbackRunInfo, err error) context.Context {
	span, _ := ctx.Value(spanKey{}).(trace.Span)
	switch info.Component {
	case ComponentChatModel:
		provider, modelID := aclResolveModelInfo(ctx, info)
		ModelErrorsTotal.WithLabelValues(modelID, provider, aclErrorType(err)).Inc()
	case ComponentTool:
		ToolInvocationsTotal.WithLabelValues(info.Name, "error").Inc()
	}
	if span != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()
	}
	start, _ := ctx.Value(startTimeKey{}).(time.Time)
	aclRecordSpanToStore(ctx, info, start, "error", err.Error(), 0, 0)
	return ctx
}

// --- ACL helper functions ---

func aclSpanNameAndAttrs(ctx context.Context, info CallbackRunInfo) (string, []attribute.KeyValue) {
	switch info.Component {
	case ComponentChatModel:
		provider, modelID := aclResolveModelInfo(ctx, info)
		return "gen_ai.chat", []attribute.KeyValue{
			attribute.String("gen_ai.system", provider),
			attribute.String("gen_ai.request.model", modelID),
			attribute.String("gen_ai.operation.name", "chat"),
			attribute.String("model.id", modelID),
			attribute.String("model.provider", provider),
		}
	case ComponentTool:
		return "tool.invoke", []attribute.KeyValue{
			attribute.String("tool.name", info.Name),
		}
	default:
		name := string(info.Component)
		if info.Name != "" {
			name = name + "." + info.Name
		}
		return name, []attribute.KeyValue{
			attribute.String("component", string(info.Component)),
			attribute.String("name", info.Name),
		}
	}
}

func aclResolveModelInfo(ctx context.Context, info CallbackRunInfo) (provider, modelID string) {
	modelID, provider = modelInfoFromCtx(ctx)
	if modelID == "" {
		modelID = info.Name
	}
	if provider == "" {
		provider = "openai"
	}
	return provider, modelID
}

func aclErrorType(err error) string {
	if err == nil {
		return "none"
	}
	return "error"
}

func aclMarshalInput(input CallbackInput) string {
	if len(input.Messages) == 0 {
		return ""
	}
	data, err := json.Marshal(input.Messages)
	if err != nil {
		return ""
	}
	return truncate(string(data), 32000)
}

func aclRecordSpanToStore(ctx context.Context, info CallbackRunInfo, start time.Time, status, errMsg string, promptTokens, completionTokens int) {
	if defaultTraceStore == nil {
		return
	}
	traceID := resolveTraceID(ctx)
	if traceID == "" {
		return
	}
	elapsed := time.Since(start)
	span := TraceSpan{
		SpanID:       spanIDFromCtx(ctx),
		Name:         info.Name,
		Component:    string(info.Component),
		StartTime:    start.Format(time.RFC3339Nano),
		EndTime:      time.Now().Format(time.RFC3339Nano),
		DurationMs:   float64(elapsed.Milliseconds()),
		Status:       status,
		Error:        errMsg,
		InputTokens:  promptTokens,
		OutputTokens: completionTokens,
		TotalTokens:  promptTokens + completionTokens,
	}
	switch info.Component {
	case ComponentChatModel:
		provider, modelID := aclResolveModelInfo(ctx, info)
		span.ModelID = modelID
		span.Provider = provider
	case ComponentTool:
		span.ToolName = info.Name
	}
	defaultTraceStore.AddSpan(traceID, span)
}

// truncate limits string length to avoid oversized span attributes.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}
