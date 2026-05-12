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
	"fmt"

	"github.com/cloudwego/eino/components/tool"
)

// Manager handles the lifecycle of skills: installation, lookup, and removal.
// It combines a HubClient for fetching metadata, an in-memory Cache, and a
// SkillInvoker that backs any tool returned via GetTool.
type Manager struct {
	hubClient HubClient
	cache     *Cache
	invoker   SkillInvoker
}

// NewManager creates a Manager backed by the provided HubClient and invoker.
func NewManager(hubClient HubClient, invoker SkillInvoker) *Manager {
	return &Manager{
		hubClient: hubClient,
		cache:     NewCache(),
		invoker:   invoker,
	}
}

// Install fetches skill metadata from the hub and registers it in the local cache.
func (m *Manager) Install(ctx context.Context, name, version string) error {
	meta, err := m.hubClient.Get(ctx, name, version)
	if err != nil {
		return fmt.Errorf("skill manager: install %s@%s: %w", name, version, err)
	}
	m.cache.Set(name, &SkillInstance{Meta: *meta, Status: "installed"})
	return nil
}

// GetTool returns the named skill wrapped as an Eino InvokableTool.
// Returns (nil, false) if the skill is not installed.
func (m *Manager) GetTool(name string) (tool.InvokableTool, bool) {
	inst, ok := m.cache.Get(name)
	if !ok {
		return nil, false
	}
	return NewSkillTool(inst.Meta, m.invoker), true
}

// ListInstalled returns a snapshot of all currently installed SkillInstances.
func (m *Manager) ListInstalled() []SkillInstance {
	return m.cache.All()
}

// RegisterLocal adds a skill directly to the local cache without fetching from the hub.
// Use this to pre-register builtin skills that don't come from SkillHub.
func (m *Manager) RegisterLocal(meta SkillMeta) {
	m.cache.Set(meta.Name, &SkillInstance{Meta: meta, Status: "builtin"})
}

// Search delegates to the HubClient to search available skills.
func (m *Manager) Search(ctx context.Context, query string, opts SearchOpts) ([]SkillMeta, error) {
	if m.hubClient == nil {
		return nil, fmt.Errorf("skill manager: no hub client configured")
	}
	return m.hubClient.Search(ctx, query, opts)
}

// Uninstall removes the named skill from the local cache.
func (m *Manager) Uninstall(name string) {
	m.cache.Delete(name)
}
