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

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// langfuseCtxKey is the context key for Langfuse trace metadata.
type langfuseCtxKey struct{}

// LangfuseTraceContext carries metadata that should be propagated to all child spans
// for Langfuse trace correlation.
type LangfuseTraceContext struct {
	SessionID string
	UserID    string
	TraceName string
	Tags      []string
	Metadata  map[string]string
}

// applyToSpan sets Langfuse-specific attributes on the given span.
func (lc *LangfuseTraceContext) applyToSpan(span trace.Span) {
	if lc.SessionID != "" {
		span.SetAttributes(attribute.String("langfuse.session.id", lc.SessionID))
	}
	if lc.UserID != "" {
		span.SetAttributes(attribute.String("langfuse.user.id", lc.UserID))
	}
	if lc.TraceName != "" {
		span.SetAttributes(attribute.String("langfuse.trace.name", lc.TraceName))
	}
	if len(lc.Tags) > 0 {
		span.SetAttributes(attribute.StringSlice("langfuse.trace.tags", lc.Tags))
	}
	for k, v := range lc.Metadata {
		span.SetAttributes(attribute.String("langfuse.trace.metadata."+k, v))
	}
}

// WithLangfuseContext injects Langfuse trace metadata into the context.
// All descendant spans created through Eino callbacks will inherit these attributes.
func WithLangfuseContext(ctx context.Context, lc *LangfuseTraceContext) context.Context {
	return context.WithValue(ctx, langfuseCtxKey{}, lc)
}

// getLangfuseContext retrieves Langfuse trace context from the given context.
func getLangfuseContext(ctx context.Context) *LangfuseTraceContext {
	if v := ctx.Value(langfuseCtxKey{}); v != nil {
		return v.(*LangfuseTraceContext)
	}
	return nil
}

// SetLangfuseSpanAttrs is a convenience function to set Langfuse attributes
// on the current active span directly.
func SetLangfuseSpanAttrs(ctx context.Context, sessionID, userID, traceName string) {
	span := trace.SpanFromContext(ctx)
	if span == nil || !span.IsRecording() {
		return
	}
	if sessionID != "" {
		span.SetAttributes(attribute.String("langfuse.session.id", sessionID))
	}
	if userID != "" {
		span.SetAttributes(attribute.String("langfuse.user.id", userID))
	}
	if traceName != "" {
		span.SetAttributes(attribute.String("langfuse.trace.name", traceName))
	}
}
