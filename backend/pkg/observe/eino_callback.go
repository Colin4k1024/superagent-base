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
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// spanKey is a private context key type for storing active spans.
type spanKey struct{}

// startTimeKey is a private context key type for storing operation start times.
type startTimeKey struct{}

// EinoObserveCallback implements Eino's callback interface to feed both
// OpenTelemetry tracing and Prometheus metrics from component lifecycle events.
// Spans are enriched with gen_ai.* semantic conventions for Langfuse compatibility.
type EinoObserveCallback struct {
	tracer trace.Tracer
}

// NewEinoObserveCallback creates a new EinoObserveCallback using the global TracerProvider.
func NewEinoObserveCallback() callbacks.Handler {
	cb := &EinoObserveCallback{
		tracer: tracer(),
	}

	return callbacks.NewHandlerBuilder().
		OnStartFn(cb.OnStart).
		OnEndFn(cb.OnEnd).
		OnErrorFn(cb.OnError).
		Build()
}

// OnStart creates a span and records the operation start time for the component.
func (c *EinoObserveCallback) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	spanName, attrs := spanNameAndAttrs(info)

	ctx, span := c.tracer.Start(ctx, spanName, trace.WithAttributes(attrs...))

	// Record input content for LLM observability.
	if info.Component == components.ComponentOfChatModel {
		if inputStr := marshalInput(input); inputStr != "" {
			span.SetAttributes(attribute.String("input.value", inputStr))
		}
	}

	// Propagate Langfuse context attributes from parent context.
	if lfCtx := getLangfuseContext(ctx); lfCtx != nil {
		lfCtx.applyToSpan(span)
	}

	ctx = context.WithValue(ctx, spanKey{}, span)
	ctx = context.WithValue(ctx, startTimeKey{}, time.Now())

	return ctx
}

// OnEnd finishes the span, records latency, and increments success counters.
func (c *EinoObserveCallback) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	span, _ := ctx.Value(spanKey{}).(trace.Span)
	start, _ := ctx.Value(startTimeKey{}).(time.Time)

	elapsed := time.Since(start).Seconds()

	switch info.Component {
	case components.ComponentOfChatModel:
		provider, modelID := resolveModelInfo(info)
		ModelRequestDuration.WithLabelValues(modelID, provider).Observe(elapsed)

		if cbOut := model.ConvCallbackOutput(output); cbOut != nil && cbOut.Message != nil {
			// Record output content.
			if span != nil {
				if outputStr := marshalMessage(cbOut.Message); outputStr != "" {
					span.SetAttributes(attribute.String("output.value", outputStr))
				}
			}

			// Record token usage with gen_ai semantic conventions.
			if cbOut.Message.ResponseMeta != nil {
				usage := cbOut.Message.ResponseMeta.Usage
				if usage != nil {
					ModelTokensTotal.WithLabelValues(modelID, provider, "input").Add(float64(usage.PromptTokens))
					ModelTokensTotal.WithLabelValues(modelID, provider, "output").Add(float64(usage.CompletionTokens))

					if span != nil {
						span.SetAttributes(
							attribute.Int64("gen_ai.usage.input_tokens", int64(usage.PromptTokens)),
							attribute.Int64("gen_ai.usage.output_tokens", int64(usage.CompletionTokens)),
							attribute.Int64("gen_ai.usage.total_tokens", int64(usage.PromptTokens+usage.CompletionTokens)),
						)
					}
				}
			}
		}

	case components.ComponentOfTool:
		ToolInvocationsTotal.WithLabelValues(info.Name, "success").Inc()
		ToolInvocationDuration.WithLabelValues(info.Name).Observe(elapsed)

		// Record tool output.
		if span != nil {
			if outputStr := marshalOutput(output); outputStr != "" {
				span.SetAttributes(attribute.String("output.value", outputStr))
			}
		}
	}

	if span != nil {
		span.SetStatus(codes.Ok, "")
		span.End()
	}

	return ctx
}

// OnError records the error in the active span and increments error counters.
func (c *EinoObserveCallback) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	span, _ := ctx.Value(spanKey{}).(trace.Span)

	switch info.Component {
	case components.ComponentOfChatModel:
		provider, modelID := resolveModelInfo(info)
		ModelErrorsTotal.WithLabelValues(modelID, provider, errorType(err)).Inc()

	case components.ComponentOfTool:
		ToolInvocationsTotal.WithLabelValues(info.Name, "error").Inc()
	}

	if span != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		span.End()
	}

	return ctx
}

// spanNameAndAttrs returns an OTel span name and attribute set for a given RunInfo.
func spanNameAndAttrs(info *callbacks.RunInfo) (string, []attribute.KeyValue) {
	switch info.Component {
	case components.ComponentOfChatModel:
		provider, modelID := resolveModelInfo(info)
		return "gen_ai.chat", []attribute.KeyValue{
			attribute.String("gen_ai.system", provider),
			attribute.String("gen_ai.request.model", modelID),
			attribute.String("gen_ai.operation.name", "chat"),
			// Legacy attributes for backward compatibility.
			attribute.String("model.id", modelID),
			attribute.String("model.provider", provider),
			attribute.String("eino.component", string(info.Component)),
		}

	case components.ComponentOfTool:
		return "tool.invoke", []attribute.KeyValue{
			attribute.String("tool.name", info.Name),
			attribute.String("eino.component", string(info.Component)),
		}

	default:
		name := string(info.Component)
		if info.Name != "" {
			name = name + "." + info.Name
		}
		return name, []attribute.KeyValue{
			attribute.String("eino.component", string(info.Component)),
			attribute.String("eino.name", info.Name),
		}
	}
}

// resolveModelInfo extracts provider and model ID from RunInfo.
func resolveModelInfo(info *callbacks.RunInfo) (provider, modelID string) {
	modelID = info.Name
	provider = "unknown"
	return provider, modelID
}

// errorType classifies an error into a short string label for Prometheus.
func errorType(err error) string {
	if err == nil {
		return "none"
	}
	return "error"
}

// marshalInput serializes callback input to a JSON string for tracing.
func marshalInput(input callbacks.CallbackInput) string {
	if input == nil {
		return ""
	}
	if cbIn := model.ConvCallbackInput(input); cbIn != nil {
		if cbIn.Messages != nil {
			data, err := json.Marshal(cbIn.Messages)
			if err == nil {
				return truncate(string(data), 32000)
			}
		}
	}
	return ""
}

// marshalMessage serializes a model message to JSON for tracing.
func marshalMessage(msg interface{}) string {
	if msg == nil {
		return ""
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return ""
	}
	return truncate(string(data), 32000)
}

// marshalOutput serializes callback output to a string for tracing.
func marshalOutput(output callbacks.CallbackOutput) string {
	if output == nil {
		return ""
	}
	data, err := json.Marshal(output)
	if err != nil {
		return ""
	}
	return truncate(string(data), 32000)
}

// truncate limits string length to avoid oversized span attributes.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...[truncated]"
}
