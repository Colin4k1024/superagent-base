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

package tool

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

// --- helpers -----------------------------------------------------------------

var okResult = map[string]any{"ok": true}

func alwaysOK(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
	return okResult, nil
}

func alwaysErr(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
	return nil, errors.New("permanent failure")
}

// --- RetryMiddleware ---------------------------------------------------------

func TestRetryMiddleware_SuccessOnFirstAttempt(t *testing.T) {
	mw := RetryMiddleware(3, 0)
	invoker := mw(alwaysOK)

	got, err := invoker(context.Background(), "t", nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got["ok"] != true {
		t.Fatalf("unexpected result: %v", got)
	}
}

func TestRetryMiddleware_SuccessAfterTransientFailures(t *testing.T) {
	var calls atomic.Int32
	// Fail twice, succeed on the third call.
	transient := func(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
		n := calls.Add(1)
		if n < 3 {
			return nil, fmt.Errorf("transient error %d", n)
		}
		return okResult, nil
	}

	mw := RetryMiddleware(3, 0)
	invoker := mw(transient)

	got, err := invoker(context.Background(), "t", nil)
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if got["ok"] != true {
		t.Fatalf("unexpected result: %v", got)
	}
	if calls.Load() != 3 {
		t.Fatalf("expected 3 calls, got %d", calls.Load())
	}
}

func TestRetryMiddleware_ExhaustsRetries(t *testing.T) {
	var calls atomic.Int32
	counter := func(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
		calls.Add(1)
		return nil, errors.New("always fails")
	}

	mw := RetryMiddleware(2, 0)
	invoker := mw(counter)

	_, err := invoker(context.Background(), "t", nil)
	if err == nil {
		t.Fatal("expected error after exhausted retries")
	}
	// maxRetries=2 means 3 total attempts (initial + 2 retries).
	if calls.Load() != 3 {
		t.Fatalf("expected 3 total calls, got %d", calls.Load())
	}
}

// --- TimeoutMiddleware -------------------------------------------------------

func TestTimeoutMiddleware_CompletesBeforeTimeout(t *testing.T) {
	mw := TimeoutMiddleware(5 * time.Second)
	invoker := mw(alwaysOK)

	got, err := invoker(context.Background(), "t", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["ok"] != true {
		t.Fatalf("unexpected result: %v", got)
	}
}

func TestTimeoutMiddleware_TriggersOnSlowTool(t *testing.T) {
	slowTool := func(ctx context.Context, _ string, _ map[string]any) (map[string]any, error) {
		select {
		case <-time.After(10 * time.Second):
			return okResult, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	mw := TimeoutMiddleware(50 * time.Millisecond)
	invoker := mw(slowTool)

	_, err := invoker(context.Background(), "slow", nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

// --- Chain -------------------------------------------------------------------

func TestChain_AppliesMiddlewaresInOrder(t *testing.T) {
	var order []int

	// Each middleware records its position in the call order.
	makeRecorder := func(pos int) Middleware {
		return func(next ToolInvoker) ToolInvoker {
			return func(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
				order = append(order, pos)
				return next(ctx, name, args)
			}
		}
	}

	combined := Chain(makeRecorder(1), makeRecorder(2), makeRecorder(3))
	invoker := combined(alwaysOK)

	_, err := invoker(context.Background(), "t", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Fatalf("middlewares applied in wrong order: %v", order)
	}
}

func TestChain_EmptyIsIdentity(t *testing.T) {
	combined := Chain()
	invoker := combined(alwaysOK)

	got, err := invoker(context.Background(), "t", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["ok"] != true {
		t.Fatalf("unexpected result: %v", got)
	}
}

// --- CacheMiddleware ---------------------------------------------------------

func TestCacheMiddleware_ReturnsCachedResult(t *testing.T) {
	var calls atomic.Int32
	counter := func(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
		calls.Add(1)
		return okResult, nil
	}

	mw := CacheMiddleware(1 * time.Minute)
	invoker := mw(counter)

	args := map[string]any{"x": 1}
	for i := 0; i < 5; i++ {
		_, err := invoker(context.Background(), "t", args)
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("expected 1 upstream call, got %d", calls.Load())
	}
}

func TestCacheMiddleware_DifferentArgsMissCache(t *testing.T) {
	var calls atomic.Int32
	counter := func(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
		calls.Add(1)
		return okResult, nil
	}

	mw := CacheMiddleware(1 * time.Minute)
	invoker := mw(counter)

	invoker(context.Background(), "t", map[string]any{"x": 1}) //nolint:errcheck
	invoker(context.Background(), "t", map[string]any{"x": 2}) //nolint:errcheck

	if calls.Load() != 2 {
		t.Fatalf("expected 2 upstream calls for different args, got %d", calls.Load())
	}
}

// --- LogMiddleware -----------------------------------------------------------

func TestLogMiddleware_LogsSuccessAndError(t *testing.T) {
	type logEntry struct {
		level   string
		message string
	}
	var entries []logEntry

	infof := func(format string, v ...any) {
		entries = append(entries, logEntry{"info", fmt.Sprintf(format, v...)})
	}
	errorf := func(format string, v ...any) {
		entries = append(entries, logEntry{"error", fmt.Sprintf(format, v...)})
	}

	logger := &recordLogger{infof: infof, warnf: func(string, ...any) {}, errorf: errorf}
	mw := LogMiddleware(logger)

	// Success path.
	_, _ = mw(alwaysOK)(context.Background(), "good", nil)
	// Error path.
	_, _ = mw(alwaysErr)(context.Background(), "bad", nil)

	if len(entries) < 2 {
		t.Fatalf("expected at least 2 log entries, got %d", len(entries))
	}
}

// recordLogger satisfies the Logger interface with injectable callbacks.
type recordLogger struct {
	infof  func(string, ...any)
	warnf  func(string, ...any)
	errorf func(string, ...any)
}

func (r *recordLogger) Infof(f string, v ...any)  { r.infof(f, v...) }
func (r *recordLogger) Warnf(f string, v ...any)  { r.warnf(f, v...) }
func (r *recordLogger) Errorf(f string, v ...any) { r.errorf(f, v...) }
