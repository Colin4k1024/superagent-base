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
	"sync"
)

// MultiHubClient aggregates multiple HubClient implementations into one.
// Search queries all clients concurrently and merges results.
// Get/Install try each client sequentially until one succeeds.
type MultiHubClient struct {
	clients []HubClient
}

// NewMultiHubClient creates an aggregating hub client from multiple sources.
// Nil clients in the variadic args are silently skipped.
func NewMultiHubClient(clients ...HubClient) *MultiHubClient {
	filtered := make([]HubClient, 0, len(clients))
	for _, c := range clients {
		if c != nil {
			filtered = append(filtered, c)
		}
	}
	return &MultiHubClient{clients: filtered}
}

// Search queries all underlying clients concurrently and merges results.
// Results are deduplicated by Name (first occurrence wins).
func (m *MultiHubClient) Search(ctx context.Context, query string, opts SearchOpts) ([]SkillMeta, error) {
	if len(m.clients) == 0 {
		return nil, nil
	}

	type result struct {
		skills []SkillMeta
		err    error
	}

	results := make([]result, len(m.clients))
	var wg sync.WaitGroup

	for i, client := range m.clients {
		wg.Add(1)
		go func(idx int, c HubClient) {
			defer wg.Done()
			skills, err := c.Search(ctx, query, opts)
			results[idx] = result{skills: skills, err: err}
		}(i, client)
	}
	wg.Wait()

	// Merge and deduplicate (first occurrence wins).
	seen := make(map[string]bool)
	var merged []SkillMeta

	for _, r := range results {
		if r.err != nil || r.skills == nil {
			continue
		}
		for _, s := range r.skills {
			key := s.Name
			if s.Author != "" {
				key = s.Author + "/" + s.Name
			}
			if !seen[key] {
				seen[key] = true
				merged = append(merged, s)
			}
		}
	}

	// Apply limit if specified.
	if opts.Limit > 0 && len(merged) > opts.Limit {
		merged = merged[:opts.Limit]
	}

	return merged, nil
}

// Get tries each client sequentially and returns the first successful result.
func (m *MultiHubClient) Get(ctx context.Context, name string, version string) (*SkillMeta, error) {
	var lastErr error
	for _, c := range m.clients {
		meta, err := c.Get(ctx, name, version)
		if err == nil && meta != nil {
			return meta, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, nil
}

// Install tries each client sequentially and returns the first successful result.
func (m *MultiHubClient) Install(ctx context.Context, name string, version string) (*SkillInstance, error) {
	var lastErr error
	for _, c := range m.clients {
		inst, err := c.Install(ctx, name, version)
		if err == nil && inst != nil {
			return inst, nil
		}
		if err != nil {
			lastErr = err
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, nil
}

// Uninstall delegates to all clients (best-effort).
func (m *MultiHubClient) Uninstall(ctx context.Context, name string) error {
	for _, c := range m.clients {
		_ = c.Uninstall(ctx, name)
	}
	return nil
}

// List aggregates installed skills from all clients.
func (m *MultiHubClient) List(ctx context.Context) ([]SkillInstance, error) {
	var all []SkillInstance
	for _, c := range m.clients {
		items, err := c.List(ctx)
		if err == nil {
			all = append(all, items...)
		}
	}
	return all, nil
}

// CheckHealth returns true if any client reports the skill as healthy.
func (m *MultiHubClient) CheckHealth(ctx context.Context, name string) (bool, error) {
	for _, c := range m.clients {
		healthy, err := c.CheckHealth(ctx, name)
		if err == nil && healthy {
			return true, nil
		}
	}
	return false, nil
}
