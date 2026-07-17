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

	"github.com/cloudwego/eino/components/tool"
	"github.com/superagent-ai/superagent-base/backend/pkg/tool/builtin"
)

// Compile-time assertion.
var _ builtin.SkillBridge = (*BridgeAdapter)(nil)

// BridgeAdapter wraps *Manager to satisfy the tool/builtin.SkillBridge interface.
// This allows builtin tools (invoke_skill, list_skills, install_skill) to
// interact with the skill system at runtime without creating a circular import.
type BridgeAdapter struct {
	mgr *Manager
}

// NewBridgeAdapter creates a BridgeAdapter wrapping the given Manager.
func NewBridgeAdapter(mgr *Manager) *BridgeAdapter {
	return &BridgeAdapter{mgr: mgr}
}

func (a *BridgeAdapter) GetTool(name string) (tool.InvokableTool, bool) {
	return a.mgr.GetTool(name)
}

func (a *BridgeAdapter) ListInstalled() []builtin.SkillInstanceInfo {
	instances := a.mgr.ListInstalled()
	out := make([]builtin.SkillInstanceInfo, 0, len(instances))
	for _, inst := range instances {
		out = append(out, builtin.SkillInstanceInfo{
			Name:        inst.Meta.Name,
			Version:     inst.Meta.Version,
			Description: inst.Meta.Description,
			Status:      inst.Status,
		})
	}
	return out
}

func (a *BridgeAdapter) Install(ctx context.Context, name, version string) error {
	return a.mgr.Install(ctx, name, version)
}
