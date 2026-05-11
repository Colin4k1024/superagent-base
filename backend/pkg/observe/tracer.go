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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/superagent-ai/superagent-base"

// TracerConfig holds configuration for OpenTelemetry tracer setup.
type TracerConfig struct {
	ServiceName string
	Endpoint    string // OTel collector gRPC endpoint, e.g. "otel-collector:4317"
	Enabled     bool
}

// InitTracer sets up the OTLP gRPC exporter and registers a global TracerProvider.
// Returns a shutdown function that must be called on application exit to flush spans.
func InitTracer(ctx context.Context, cfg TracerConfig) (func(context.Context) error, error) {
	if !cfg.Enabled {
		// Return a no-op shutdown when tracing is disabled.
		return func(_ context.Context) error { return nil }, nil
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			attribute.String("library.language", "go"),
		),
		resource.WithProcess(),
		resource.WithOS(),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}

// tracer returns the package-level named tracer from the global provider.
func tracer() trace.Tracer {
	return otel.Tracer(tracerName)
}

// StartAgentSpan starts a span for an agent operation.
// The caller must call span.End() when the operation completes.
func StartAgentSpan(ctx context.Context, agentID string, operation string) (context.Context, trace.Span) {
	return tracer().Start(ctx, "agent."+operation,
		trace.WithAttributes(
			attribute.String("agent.id", agentID),
			attribute.String("agent.operation", operation),
		),
	)
}

// StartModelSpan starts a span for a model API call.
// The caller must call span.End() when the operation completes.
func StartModelSpan(ctx context.Context, modelID string, provider string) (context.Context, trace.Span) {
	return tracer().Start(ctx, "model.invoke",
		trace.WithAttributes(
			attribute.String("model.id", modelID),
			attribute.String("model.provider", provider),
		),
	)
}

// StartToolSpan starts a span for a tool invocation.
// The caller must call span.End() when the operation completes.
func StartToolSpan(ctx context.Context, toolName string) (context.Context, trace.Span) {
	return tracer().Start(ctx, "tool.invoke",
		trace.WithAttributes(
			attribute.String("tool.name", toolName),
		),
	)
}
