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

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const langfuseIngestionVersion = "4"

// newLangfuseExporter creates an OTLP HTTP exporter configured for the Langfuse endpoint.
func newLangfuseExporter(ctx context.Context, cfg LangfuseConfig) (sdktrace.SpanExporter, error) {
	if cfg.PublicKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("langfuse: LANGFUSE_PUBLIC_KEY and LANGFUSE_SECRET_KEY are required")
	}

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(cfg.OTLPEndpoint()),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization":                cfg.AuthHeader(),
			"x-langfuse-ingestion-version": langfuseIngestionVersion,
		}),
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("langfuse: failed to create OTLP HTTP exporter: %w", err)
	}

	return exporter, nil
}

// newLangfuseSampler returns a sampler that respects the configured sample rate.
func newLangfuseSampler(rate float64) sdktrace.Sampler {
	if rate >= 1.0 {
		return sdktrace.AlwaysSample()
	}
	if rate <= 0.0 {
		return sdktrace.NeverSample()
	}
	return sdktrace.TraceIDRatioBased(rate)
}
