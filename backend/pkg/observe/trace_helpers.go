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

	"go.opentelemetry.io/otel/trace"
)

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

// resolveModelInfoFromCtx extracts provider and model ID from context.
// This is the framework-agnostic version used by trace_helpers.
func resolveModelInfoFromCtx(ctx context.Context) (provider, modelID string) {
	return modelInfoFromCtx(ctx)
}

// recordSpanToTraceStore writes a completed span to the local TraceStore
// using ACL types.  This replaces the old eino-dependent recordSpanToStore.
func recordSpanToTraceStore(ctx context.Context, info CallbackRunInfo, start time.Time, status, errMsg string) {
	aclRecordSpanToStore(ctx, info, start, status, errMsg, 0, 0)
}
