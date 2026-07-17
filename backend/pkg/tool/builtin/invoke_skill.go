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

package builtin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// SkillBridge provides runtime access to the skill system.
// Defined here to avoid circular dependency (tool/builtin → skill).
// The skill package provides an implementation via BridgeAdapter.
type SkillBridge interface {
	GetTool(name string) (tool.InvokableTool, bool)
	ListInstalled() []SkillInstanceInfo
	Install(ctx context.Context, name, version string) error
}

// SkillInstanceInfo mirrors skill.SkillInstance for the bridge interface.
type SkillInstanceInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// skillBridge is the package-level bridge, initialized via InitSkillBridge.
var skillBridge SkillBridge

// InitSkillBridge sets the SkillBridge used by invoke_skill, list_skills,
// and install_skill builtins. Must be called before RegisterBuiltins.
func InitSkillBridge(bridge SkillBridge) {
	skillBridge = bridge
}

// ---------------------------------------------------------------------------
// invoke_skill
// ---------------------------------------------------------------------------

// Compile-time assertion.
var _ tool.InvokableTool = (*InvokeSkillTool)(nil)

// InvokeSkillTool dynamically invokes an installed skill at runtime.
type InvokeSkillTool struct{}

func newInvokeSkillTool() tool.InvokableTool {
	if skillBridge == nil {
		return nil
	}
	return &InvokeSkillTool{}
}

func (t *InvokeSkillTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "invoke_skill",
		Desc: "Invoke an installed skill by name at runtime. Use this to dynamically call skills that were discovered via find-skills or installed via install_skill. The skill must already be installed before invoking.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"skill_name": {
				Type:     "string",
				Desc:     "Name of the installed skill to invoke",
				Required: true,
			},
			"parameters": {
				Type: "object",
				Desc: "Parameters to pass to the skill as key-value pairs. Check the skill's documentation for expected parameters.",
			},
		}),
	}, nil
}

func (t *InvokeSkillTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	var input struct {
		SkillName  string         `json:"skill_name"`
		Parameters map[string]any `json:"parameters"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", fmt.Errorf("invoke_skill: invalid arguments: %w", err)
	}
	if input.SkillName == "" {
		return "", fmt.Errorf("invoke_skill: 'skill_name' is required")
	}

	// Look up the skill tool dynamically.
	skillTool, ok := skillBridge.GetTool(input.SkillName)
	if !ok {
		// Return structured error so the model can reason about it.
		result, _ := json.Marshal(map[string]any{
			"error":   true,
			"message": fmt.Sprintf("skill '%s' is not installed", input.SkillName),
			"hint":    "Use install_skill to install it first, then try again.",
		})
		return string(result), nil
	}

	// Marshal parameters to JSON for the skill's InvokableRun.
	skillArgs := "{}"
	if input.Parameters != nil {
		b, err := json.Marshal(input.Parameters)
		if err != nil {
			return "", fmt.Errorf("invoke_skill: failed to marshal parameters: %w", err)
		}
		skillArgs = string(b)
	}

	// Invoke the skill.
	out, err := skillTool.InvokableRun(ctx, skillArgs)
	if err != nil {
		result, _ := json.Marshal(map[string]any{
			"error":   true,
			"message": err.Error(),
			"skill":   input.SkillName,
			"hint":    "Check the skill's expected parameters and try again.",
		})
		return string(result), nil
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// list_skills
// ---------------------------------------------------------------------------

var _ tool.InvokableTool = (*ListSkillsTool)(nil)

// ListSkillsTool lists all currently installed skills.
type ListSkillsTool struct{}

func newListSkillsTool() tool.InvokableTool {
	if skillBridge == nil {
		return nil
	}
	return &ListSkillsTool{}
}

func (t *ListSkillsTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "list_skills",
		Desc:        "List all currently installed skills with their name, version, description, and status. Use this to check what skills are available before invoking them.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *ListSkillsTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	installed := skillBridge.ListInstalled()

	type skillEntry struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
		Status      string `json:"status"`
	}

	skills := make([]skillEntry, 0, len(installed))
	for _, s := range installed {
		skills = append(skills, skillEntry{
			Name:        s.Name,
			Version:     s.Version,
			Description: s.Description,
			Status:      s.Status,
		})
	}

	result, _ := json.Marshal(map[string]any{
		"skills": skills,
		"count":  len(skills),
		"tip":    "Use invoke_skill to call an installed skill, or find_skills to search for new ones.",
	})
	return string(result), nil
}

// ---------------------------------------------------------------------------
// install_skill
// ---------------------------------------------------------------------------

var _ tool.InvokableTool = (*InstallSkillTool)(nil)

// InstallSkillTool installs a skill from the marketplace.
type InstallSkillTool struct{}

func newInstallSkillTool() tool.InvokableTool {
	if skillBridge == nil {
		return nil
	}
	return &InstallSkillTool{}
}

func (t *InstallSkillTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "install_skill",
		Desc: "Install a skill from the skills.sh marketplace. After installation, the skill can be invoked with invoke_skill. Use find_skills first to discover available skills.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"skill_name": {
				Type:     "string",
				Desc:     "Name of the skill to install (from find-skills results)",
				Required: true,
			},
			"version": {
				Type: "string",
				Desc: "Version to install (default: 'latest')",
			},
		}),
	}, nil
}

func (t *InstallSkillTool) InvokableRun(ctx context.Context, args string, opts ...tool.Option) (string, error) {
	var input struct {
		SkillName string `json:"skill_name"`
		Version   string `json:"version"`
	}
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", fmt.Errorf("install_skill: invalid arguments: %w", err)
	}
	if input.SkillName == "" {
		return "", fmt.Errorf("install_skill: 'skill_name' is required")
	}
	if input.Version == "" {
		input.Version = "latest"
	}

	err := skillBridge.Install(ctx, input.SkillName, input.Version)
	if err != nil {
		result, _ := json.Marshal(map[string]any{
			"error":   true,
			"message": err.Error(),
			"skill":   input.SkillName,
			"hint":    "Check the skill name with find_skills and try again.",
		})
		return string(result), nil
	}

	result, _ := json.Marshal(map[string]any{
		"success": true,
		"message": fmt.Sprintf("Skill '%s@%s' installed successfully. You can now invoke it with invoke_skill.", input.SkillName, input.Version),
		"skill":   input.SkillName,
		"version": input.Version,
	})
	return string(result), nil
}
