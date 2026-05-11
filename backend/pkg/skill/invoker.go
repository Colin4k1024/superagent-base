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

package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// SkillFunc is the signature for locally registered skill functions.
type SkillFunc func(ctx context.Context, input map[string]any) (map[string]any, error)

// ─── LocalInvoker ─────────────────────────────────────────────────────────────

// LocalInvoker calls locally registered SkillFunc implementations.
type LocalInvoker struct {
	skills map[string]SkillFunc
	mu     sync.RWMutex
}

// NewLocalInvoker creates an empty LocalInvoker.
func NewLocalInvoker() *LocalInvoker {
	return &LocalInvoker{skills: make(map[string]SkillFunc)}
}

// Register adds or replaces a named skill function.
func (l *LocalInvoker) Register(name string, fn SkillFunc) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.skills[name] = fn
}

// Invoke dispatches to the registered function for skillName.
func (l *LocalInvoker) Invoke(ctx context.Context, skillName string, input map[string]any) (map[string]any, error) {
	l.mu.RLock()
	fn, ok := l.skills[skillName]
	l.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("local skill %q not registered", skillName)
	}
	return fn(ctx, input)
}

// ─── HTTPInvoker ──────────────────────────────────────────────────────────────

// HTTPInvoker calls skills via HTTP REST endpoints.
// Each skill name maps to a base endpoint URL; requests are POSTed to
// <endpoint>/invoke with a JSON body.
type HTTPInvoker struct {
	httpClient *http.Client
	endpoints  map[string]string // skill name → base endpoint URL
	mu         sync.RWMutex
}

// NewHTTPInvoker creates an HTTPInvoker with a 30-second default timeout.
func NewHTTPInvoker() *HTTPInvoker {
	return &HTTPInvoker{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		endpoints:  make(map[string]string),
	}
}

// Register maps a skill name to its HTTP endpoint base URL.
func (h *HTTPInvoker) Register(skillName, endpoint string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.endpoints[skillName] = endpoint
}

// Invoke POSTs the input as JSON to <endpoint>/invoke and returns the parsed response.
func (h *HTTPInvoker) Invoke(ctx context.Context, skillName string, input map[string]any) (map[string]any, error) {
	h.mu.RLock()
	endpoint, ok := h.endpoints[skillName]
	h.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("skill %q not registered", skillName)
	}

	body, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("skill %q: marshal input: %w", skillName, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/invoke", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("skill %q: create request: %w", skillName, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("skill %q: http invoke: %w", skillName, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("skill %q: read response: %w", skillName, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("skill %q returned HTTP %d: %s", skillName, resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("skill %q: decode response: %w", skillName, err)
	}
	return result, nil
}

// ─── CompositeInvoker ─────────────────────────────────────────────────────────

// CompositeInvoker attempts local invocation first; if the skill is not
// registered locally it falls back to the HTTP invoker.
type CompositeInvoker struct {
	local *LocalInvoker
	http  *HTTPInvoker
}

// NewCompositeInvoker creates a CompositeInvoker that tries local before HTTP.
func NewCompositeInvoker(local *LocalInvoker, httpInv *HTTPInvoker) *CompositeInvoker {
	return &CompositeInvoker{local: local, http: httpInv}
}

// Invoke tries the LocalInvoker first; on failure it delegates to HTTPInvoker.
func (c *CompositeInvoker) Invoke(ctx context.Context, skillName string, input map[string]any) (map[string]any, error) {
	result, err := c.local.Invoke(ctx, skillName, input)
	if err == nil {
		return result, nil
	}
	return c.http.Invoke(ctx, skillName, input)
}
