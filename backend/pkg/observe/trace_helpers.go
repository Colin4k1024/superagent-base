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
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"go.opentelemetry.io/otel/trace"
)

// recordSpanToStore writes a completed span to the local TraceStore.
func recordSpanToStore(ctx context.Context, info *callbacks.RunInfo, start time.Time, status, errMsg string) {
	if defaultTraceStore == nil {
		return
	}

	traceID := resolveTraceID(ctx)
	if traceID == "" {
		return
	}

	elapsed := time.Since(start)
	span := TraceSpan{
		SpanID:    spanIDFromCtx(ctx),
		Name:      info.Name,
		Component: string(info.Component),
		StartTime: start.Format(time.RFC3339Nano),
		EndTime:   time.Now().Format(time.RFC3339Nano),
		DurationMs: float64(elapsed.Milliseconds()),
		Status:    status,
		Error:     errMsg,
	}

	switch info.Component {
	case components.ComponentOfChatModel:
		provider, modelID := resolveModelInfo(ctx, info)
		span.ModelID = modelID
		span.Provider = provider
	case components.ComponentOfTool:
		span.ToolName = info.Name
	}

	defaultTraceStore.AddSpan(traceID, span)
}

// recordStreamSpanToStore writes a streaming model span with token counts.
func recordStreamSpanToStore(ctx context.Context, info *callbacks.RunInfo, start time.Time, modelID, provider string, promptTokens, completionTokens int) {
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
		Status:       "ok",
		ModelID:      modelID,
		Provider:     provider,
		InputTokens:  promptTokens,
		OutputTokens: completionTokens,
		TotalTokens:  promptTokens + completionTokens,
	}

	defaultTraceStore.AddSpan(traceID, span)
}

// resolveTraceID extracts a trace ID, preferring our custom context key,
// falling back to the OTel span context.
func resolveTraceID(ctx context.Context) string {
	if id := TraceIDFromCtx(ctx); id != "" {
		return id
	}
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.HasTraceID() {
		return sc.TraceID().String()
	}
	return ""
}

// spanIDFromCtx extracts the current OTel span ID for correlation.
func spanIDFromCtx(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.HasSpanID() {
		return sc.SpanID().String()
	}
	return ""
}
