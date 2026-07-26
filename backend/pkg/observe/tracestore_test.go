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

package observe

import (
	"testing"
	"time"
)

func TestTraceStore_BasicLifecycle(t *testing.T) {
	ts := NewTraceStore(10)

	ts.StartTrace("t1", "test-agent", "agent-1", "session-1")
	ts.AddSpan("t1", TraceSpan{
		SpanID:       "s1",
		Name:         "gpt-4",
		Component:    "ChatModel",
		DurationMs:   150,
		Status:       "ok",
		InputTokens:  100,
		OutputTokens: 50,
	})
	tr := ts.EndTrace("t1", "ok")

	if tr == nil {
		t.Fatal("expected trace record, got nil")
	}
	if tr.TraceID != "t1" {
		t.Errorf("expected trace ID t1, got %s", tr.TraceID)
	}
	if tr.TotalInputTokens != 100 {
		t.Errorf("expected 100 input tokens, got %d", tr.TotalInputTokens)
	}
	if tr.TotalOutputTokens != 50 {
		t.Errorf("expected 50 output tokens, got %d", tr.TotalOutputTokens)
	}
	if len(tr.Spans) != 1 {
		t.Errorf("expected 1 span, got %d", len(tr.Spans))
	}
}

func TestTraceStore_List(t *testing.T) {
	ts := NewTraceStore(100)

	for i := 0; i < 5; i++ {
		id := "trace-" + string(rune('a'+i))
		ts.StartTrace(id, "agent", "a1", "s1")
		ts.EndTrace(id, "ok")
		time.Sleep(time.Millisecond)
	}

	traces, total := ts.List(TraceQueryParams{Page: 1, Limit: 3})
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(traces) != 3 {
		t.Errorf("expected 3 results, got %d", len(traces))
	}
	// Newest first
	if traces[0].TraceID != "trace-e" {
		t.Errorf("expected newest trace first, got %s", traces[0].TraceID)
	}
}

func TestTraceStore_RingBufferOverflow(t *testing.T) {
	ts := NewTraceStore(3)

	for i := 0; i < 5; i++ {
		id := "t" + string(rune('0'+i))
		ts.StartTrace(id, "agent", "a1", "s1")
		ts.EndTrace(id, "ok")
	}

	// Only 3 most recent should be stored
	traces, total := ts.List(TraceQueryParams{Page: 1, Limit: 10})
	if total != 3 {
		t.Errorf("expected 3 (capacity), got %d", total)
	}
	if len(traces) != 3 {
		t.Errorf("expected 3 traces, got %d", len(traces))
	}
}

func TestTraceStore_Get(t *testing.T) {
	ts := NewTraceStore(10)

	ts.StartTrace("find-me", "agent", "a1", "s1")
	ts.EndTrace("find-me", "ok")

	tr := ts.Get("find-me")
	if tr == nil {
		t.Fatal("expected to find trace")
	}
	if tr.TraceID != "find-me" {
		t.Errorf("expected find-me, got %s", tr.TraceID)
	}

	if ts.Get("nonexistent") != nil {
		t.Error("expected nil for nonexistent trace")
	}
}

func TestTraceStore_Sessions(t *testing.T) {
	ts := NewTraceStore(10)

	ts.StartTrace("t1", "agent", "a1", "sess-a")
	ts.EndTrace("t1", "ok")
	ts.StartTrace("t2", "agent", "a1", "sess-b")
	ts.EndTrace("t2", "ok")
	ts.StartTrace("t3", "agent", "a1", "sess-a")
	ts.EndTrace("t3", "ok")

	sessions, total := ts.Sessions(1, 10)
	if total != 2 {
		t.Errorf("expected 2 distinct sessions, got %d", total)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestMetricsBucketer_Record(t *testing.T) {
	mb := NewMetricsBucketer(30)

	tr := &TraceRecord{
		TraceID:           "t1",
		StartTime:         time.Now(),
		Status:            "ok",
		Spans:             []TraceSpan{{SpanID: "s1"}, {SpanID: "s2"}},
		TotalInputTokens:  200,
		TotalOutputTokens: 100,
		DurationMs:        500,
	}

	mb.Record(tr)

	buckets := mb.Query("", "")
	if len(buckets) != 1 {
		t.Fatalf("expected 1 bucket, got %d", len(buckets))
	}
	b := buckets[0]
	if b.TraceCount != 1 {
		t.Errorf("expected 1 trace, got %d", b.TraceCount)
	}
	if b.ObservationCount != 2 {
		t.Errorf("expected 2 observations, got %d", b.ObservationCount)
	}
	if b.InputTokens != 200 {
		t.Errorf("expected 200 input tokens, got %d", b.InputTokens)
	}
}
