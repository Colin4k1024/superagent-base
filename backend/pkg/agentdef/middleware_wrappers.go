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

package agentdef

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/superagent-ai/superagent-base/backend/pkg/observe"
)

// observedAgent wraps an Agent with Prometheus metrics recording.
type observedAgent struct {
	inner         Agent
	enableMetrics bool
}

func (a *observedAgent) Name() string                    { return a.inner.Name() }
func (a *observedAgent) Description() string             { return a.inner.Description() }
func (a *observedAgent) GetDefinition() *AgentDefinition { return a.inner.GetDefinition() }
func (a *observedAgent) UnwrapAgent() Agent              { return a.inner }

func (a *observedAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	start := time.Now()
	ch, err := a.inner.Chat(ctx, sessionID, message)
	if err != nil {
		if a.enableMetrics {
			observe.AgentRequestsTotal.WithLabelValues(a.inner.Name(), "error").Inc()
			observe.AgentRequestDuration.WithLabelValues(a.inner.Name()).Observe(time.Since(start).Seconds())
		}
		return nil, err
	}

	out := make(chan string, 64)
	go func() {
		defer close(out)
		hadError := false
		for token := range ch {
			if strings.HasPrefix(token, "[error]") {
				hadError = true
			}
			select {
			case out <- token:
			case <-ctx.Done():
				if a.enableMetrics {
					observe.AgentRequestsTotal.WithLabelValues(a.inner.Name(), "cancelled").Inc()
					observe.AgentRequestDuration.WithLabelValues(a.inner.Name()).Observe(time.Since(start).Seconds())
				}
				return
			}
		}
		if a.enableMetrics {
			status := "success"
			if hadError {
				status = "error"
			}
			observe.AgentRequestsTotal.WithLabelValues(a.inner.Name(), status).Inc()
			observe.AgentRequestDuration.WithLabelValues(a.inner.Name()).Observe(time.Since(start).Seconds())
		}
	}()
	return out, nil
}

// timeoutAgent wraps an Agent with a per-request timeout.
type timeoutAgent struct {
	inner   Agent
	timeout time.Duration
}

func (a *timeoutAgent) Name() string                    { return a.inner.Name() }
func (a *timeoutAgent) Description() string             { return a.inner.Description() }
func (a *timeoutAgent) GetDefinition() *AgentDefinition { return a.inner.GetDefinition() }
func (a *timeoutAgent) UnwrapAgent() Agent              { return a.inner }

func (a *timeoutAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	ch, err := a.inner.Chat(ctx, sessionID, message)
	if err != nil {
		cancel()
		return nil, err
	}
	out := make(chan string, 64)
	go func() {
		defer close(out)
		defer cancel()
		for token := range ch {
			select {
			case out <- token:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

// retryAgent wraps an Agent with retry-on-failure logic.
type retryAgent struct {
	inner       Agent
	maxAttempts int
}

func (a *retryAgent) Name() string                    { return a.inner.Name() }
func (a *retryAgent) Description() string             { return a.inner.Description() }
func (a *retryAgent) GetDefinition() *AgentDefinition { return a.inner.GetDefinition() }
func (a *retryAgent) UnwrapAgent() Agent              { return a.inner }

func (a *retryAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	var lastErr error
	for attempt := 0; attempt < a.maxAttempts; attempt++ {
		ch, err := a.inner.Chat(ctx, sessionID, message)
		if err == nil {
			return ch, nil
		}
		lastErr = err
		backoff := time.Duration(100<<uint(attempt)) * time.Millisecond
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return nil, fmt.Errorf("agentdef: retry exhausted after %d attempts: %w", a.maxAttempts, lastErr)
}

// fallbackAgent wraps a primary Agent with a fallback Agent.
type fallbackAgent struct {
	primary  Agent
	fallback Agent
}

func (a *fallbackAgent) Name() string                    { return a.primary.Name() }
func (a *fallbackAgent) Description() string             { return a.primary.Description() }
func (a *fallbackAgent) GetDefinition() *AgentDefinition { return a.primary.GetDefinition() }
func (a *fallbackAgent) UnwrapAgent() Agent              { return a.primary }

func (a *fallbackAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	ch, err := a.primary.Chat(ctx, sessionID, message)
	if err != nil {
		return a.fallback.Chat(ctx, sessionID, message)
	}
	return ch, nil
}

// rateLimitAgent wraps an Agent with a sliding-window rate limiter.
type rateLimitAgent struct {
	inner Agent
	rpm   int
	mu    sync.Mutex
	times []time.Time
}

func (a *rateLimitAgent) Name() string                    { return a.inner.Name() }
func (a *rateLimitAgent) Description() string             { return a.inner.Description() }
func (a *rateLimitAgent) GetDefinition() *AgentDefinition { return a.inner.GetDefinition() }
func (a *rateLimitAgent) UnwrapAgent() Agent              { return a.inner }

func (a *rateLimitAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	a.mu.Lock()
	now := time.Now()
	windowStart := now.Add(-time.Minute)

	valid := a.times[:0]
	for _, t := range a.times {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}
	a.times = valid

	if len(a.times) >= a.rpm {
		a.mu.Unlock()
		return nil, fmt.Errorf("agentdef: rate limit exceeded (%d requests/minute)", a.rpm)
	}
	a.times = append(a.times, now)
	a.mu.Unlock()

	return a.inner.Chat(ctx, sessionID, message)
}

// cacheEntry holds a cached response with its expiration time.
type cacheEntry struct {
	tokens    []string
	expiresAt time.Time
}

const maxCacheEntries = 1000

// cacheAgent wraps an Agent with an in-memory response cache.
type cacheAgent struct {
	inner Agent
	ttl   time.Duration
	mu    sync.RWMutex
	cache map[string]cacheEntry
}

func (a *cacheAgent) Name() string                    { return a.inner.Name() }
func (a *cacheAgent) Description() string             { return a.inner.Description() }
func (a *cacheAgent) GetDefinition() *AgentDefinition { return a.inner.GetDefinition() }
func (a *cacheAgent) UnwrapAgent() Agent              { return a.inner }

func (a *cacheAgent) Chat(ctx context.Context, sessionID string, message string) (<-chan string, error) {
	key := cacheKey(a.inner.Name(), sessionID, message)

	a.mu.RLock()
	if entry, ok := a.cache[key]; ok && time.Now().Before(entry.expiresAt) {
		a.mu.RUnlock()
		ch := make(chan string, len(entry.tokens))
		for _, tok := range entry.tokens {
			ch <- tok
		}
		close(ch)
		return ch, nil
	}
	a.mu.RUnlock()

	innerCh, err := a.inner.Chat(ctx, sessionID, message)
	if err != nil {
		return nil, err
	}

	out := make(chan string, 64)
	go func() {
		defer close(out)
		var tokens []string
		for tok := range innerCh {
			select {
			case out <- tok:
				tokens = append(tokens, tok)
			case <-ctx.Done():
				return
			}
		}
		hasError := false
		for _, tok := range tokens {
			if strings.HasPrefix(tok, "[error]") {
				hasError = true
				break
			}
		}
		if !hasError {
			a.mu.Lock()
			now := time.Now()
			for k, e := range a.cache {
				if now.After(e.expiresAt) {
					delete(a.cache, k)
				}
			}
			if len(a.cache) >= maxCacheEntries {
				var oldestKey string
				var oldestTime time.Time
				for k, e := range a.cache {
					if oldestKey == "" || e.expiresAt.Before(oldestTime) {
						oldestKey = k
						oldestTime = e.expiresAt
					}
				}
				if oldestKey != "" {
					delete(a.cache, oldestKey)
				}
			}
			a.cache[key] = cacheEntry{
				tokens:    tokens,
				expiresAt: time.Now().Add(a.ttl),
			}
			a.mu.Unlock()
		}
	}()
	return out, nil
}

// cacheKey produces a SHA256 hex digest from agent name, session ID, and message.
func cacheKey(agentName, sessionID, message string) string {
	h := sha256.New()
	h.Write([]byte(agentName))
	h.Write([]byte{0})
	h.Write([]byte(sessionID))
	h.Write([]byte{0})
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
