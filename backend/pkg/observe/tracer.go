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
	"fmt"

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
	Langfuse    LangfuseConfig
}

// InitTracer sets up exporters and registers a global TracerProvider.
// Supports dual-export: OTel Collector (gRPC) and/or Langfuse (HTTP/protobuf).
// Returns a shutdown function that must be called on application exit to flush spans.
func InitTracer(ctx context.Context, cfg TracerConfig) (func(context.Context) error, error) {
	if !cfg.Enabled && !cfg.Langfuse.Enabled {
		return func(_ context.Context) error { return nil }, nil
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
		return nil, fmt.Errorf("observe: resource creation failed: %w", err)
	}

	var opts []sdktrace.TracerProviderOption
	opts = append(opts, sdktrace.WithResource(res))

	var shutdowns []func(context.Context) error

	// OTel Collector gRPC exporter.
	if cfg.Enabled {
		grpcExp, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(cfg.Endpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			return nil, fmt.Errorf("observe: gRPC exporter failed: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(grpcExp))
		shutdowns = append(shutdowns, grpcExp.Shutdown)
	}

	// Langfuse OTLP HTTP exporter.
	if cfg.Langfuse.Enabled {
		lfExp, err := newLangfuseExporter(ctx, cfg.Langfuse)
		if err != nil {
			return nil, fmt.Errorf("observe: langfuse exporter failed: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(lfExp))
		shutdowns = append(shutdowns, lfExp.Shutdown)
	}

	// Use the more restrictive sampler when Langfuse has a custom rate.
	sampler := sdktrace.AlwaysSample()
	if cfg.Langfuse.Enabled && cfg.Langfuse.SampleRate < 1.0 {
		sampler = newLangfuseSampler(cfg.Langfuse.SampleRate)
	}
	opts = append(opts, sdktrace.WithSampler(sampler))

	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)

	shutdown := func(ctx context.Context) error {
		var firstErr error
		if err := tp.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
		for _, fn := range shutdowns {
			if err := fn(ctx); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	}

	return shutdown, nil
}

// tracer returns the package-level named tracer from the global provider.
func tracer() trace.Tracer {
	return otel.Tracer(tracerName)
}

// StartAgentSpan starts a span for an agent operation.
func StartAgentSpan(ctx context.Context, agentID string, operation string) (context.Context, trace.Span) {
	return tracer().Start(ctx, "agent."+operation,
		trace.WithAttributes(
			attribute.String("agent.id", agentID),
			attribute.String("agent.operation", operation),
		),
	)
}

// StartModelSpan starts a span for a model API call with gen_ai semantic conventions.
func StartModelSpan(ctx context.Context, modelID string, provider string) (context.Context, trace.Span) {
	return tracer().Start(ctx, "gen_ai.chat",
		trace.WithAttributes(
			attribute.String("gen_ai.system", provider),
			attribute.String("gen_ai.request.model", modelID),
			attribute.String("gen_ai.operation.name", "chat"),
			// Keep legacy attributes for backward compatibility.
			attribute.String("model.id", modelID),
			attribute.String("model.provider", provider),
		),
	)
}

// StartToolSpan starts a span for a tool invocation.
func StartToolSpan(ctx context.Context, toolName string) (context.Context, trace.Span) {
	return tracer().Start(ctx, "tool.invoke",
		trace.WithAttributes(
			attribute.String("tool.name", toolName),
		),
	)
}
