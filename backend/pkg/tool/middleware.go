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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ToolInvoker is the core function type that executes a tool by name with the given arguments.
type ToolInvoker func(ctx context.Context, name string, args map[string]any) (map[string]any, error)

// Middleware wraps a ToolInvoker and returns a new one with additional behaviour.
type Middleware func(next ToolInvoker) ToolInvoker

// Chain composes multiple middlewares into one. Middlewares are applied
// left-to-right: the first middleware in the list is outermost.
func Chain(middlewares ...Middleware) Middleware {
	return func(next ToolInvoker) ToolInvoker {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// Logger is the logging interface expected by LogMiddleware.
// It is intentionally narrow so callers can pass any logger that satisfies it.
type Logger interface {
	Infof(format string, v ...any)
	Warnf(format string, v ...any)
	Errorf(format string, v ...any)
}

// RetryMiddleware retries the invocation up to maxRetries times on error,
// waiting backoff between attempts.
func RetryMiddleware(maxRetries int, backoff time.Duration) Middleware {
	return func(next ToolInvoker) ToolInvoker {
		return func(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
			var (
				result map[string]any
				err    error
			)
			for attempt := 0; attempt <= maxRetries; attempt++ {
				result, err = next(ctx, name, args)
				if err == nil {
					return result, nil
				}
				if attempt < maxRetries {
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(backoff):
					}
				}
			}
			return nil, fmt.Errorf("tool %s failed after %d retries: %w", name, maxRetries, err)
		}
	}
}

// TimeoutMiddleware enforces a per-invocation deadline. If the tool does not
// return before timeout, the context is cancelled and an error is returned.
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next ToolInvoker) ToolInvoker {
		return func(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			type result struct {
				out map[string]any
				err error
			}
			ch := make(chan result, 1)
			go func() {
				out, err := next(ctx, name, args)
				ch <- result{out, err}
			}()

			select {
			case r := <-ch:
				return r.out, r.err
			case <-ctx.Done():
				return nil, fmt.Errorf("tool %s timed out after %s", name, timeout)
			}
		}
	}
}

// RateLimitMiddleware limits invocations to rpm calls per minute using a token
// bucket that refills once per minute. Callers block until a token is available
// or the context is cancelled.
func RateLimitMiddleware(rpm int) Middleware {
	if rpm <= 0 {
		rpm = 1
	}
	// tokens is a buffered channel acting as a token bucket.
	tokens := make(chan struct{}, rpm)
	for i := 0; i < rpm; i++ {
		tokens <- struct{}{}
	}
	// Refill the bucket every minute.
	go func() {
		ticker := time.NewTicker(time.Minute)
		for range ticker.C {
			for i := 0; i < rpm; i++ {
				select {
				case tokens <- struct{}{}:
				default:
				}
			}
		}
	}()

	return func(next ToolInvoker) ToolInvoker {
		return func(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
			select {
			case <-tokens:
			case <-ctx.Done():
				return nil, fmt.Errorf("tool %s rate-limit wait cancelled: %w", name, ctx.Err())
			}
			return next(ctx, name, args)
		}
	}
}

// cacheEntry holds a cached result and its expiry time.
type cacheEntry struct {
	result  map[string]any
	expiry  time.Time
}

// CacheMiddleware caches tool results keyed by (name, args-hash) for ttl duration.
// Only use this for idempotent tools.
func CacheMiddleware(ttl time.Duration) Middleware {
	var mu sync.RWMutex
	cache := make(map[string]cacheEntry)

	return func(next ToolInvoker) ToolInvoker {
		return func(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
			key := cacheKey(name, args)

			mu.RLock()
			if entry, ok := cache[key]; ok && time.Now().Before(entry.expiry) {
				mu.RUnlock()
				return entry.result, nil
			}
			mu.RUnlock()

			result, err := next(ctx, name, args)
			if err != nil {
				return nil, err
			}

			mu.Lock()
			cache[key] = cacheEntry{result: result, expiry: time.Now().Add(ttl)}
			mu.Unlock()

			return result, nil
		}
	}
}

// LogMiddleware logs each tool invocation: start, finish, and any error.
func LogMiddleware(logger Logger) Middleware {
	return func(next ToolInvoker) ToolInvoker {
		return func(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
			start := time.Now()
			logger.Infof("tool %s: starting invocation", name)

			result, err := next(ctx, name, args)
			elapsed := time.Since(start)

			if err != nil {
				logger.Errorf("tool %s: failed in %s: %v", name, elapsed, err)
				return nil, err
			}
			logger.Infof("tool %s: completed in %s", name, elapsed)
			return result, nil
		}
	}
}

// cacheKey returns a stable hash of the tool name and its serialised arguments.
func cacheKey(name string, args map[string]any) string {
	data, _ := json.Marshal(args)
	h := sha256.Sum256(append([]byte(name+":"), data...))
	return fmt.Sprintf("%x", h)
}
