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

import "github.com/cloudwego/eino/components/tool"

// GetAllBuiltinTools returns one instance of every built-in tool.
// Each call produces fresh instances so callers can configure them independently.
// Tools that depend on optional env vars return nil when unconfigured and are skipped.
func GetAllBuiltinTools() []tool.InvokableTool {
	tools := []tool.InvokableTool{
		newWebSearchTool(),
		newHTTPRequestTool(),
	}
	// code_execute is disabled by default (no sandbox isolation, C-4).
	// Enable with CODE_EXECUTE_ENABLED=true in controlled environments only.
	if t := newCodeExecTool(); t != nil {
		tools = append(tools, t)
	}
	if t := newHiAgentRAGTool(); t != nil {
		tools = append(tools, t)
	}
	// Dynamic skill bridge tools (nil when InitSkillBridge not called).
	if t := newInvokeSkillTool(); t != nil {
		tools = append(tools, t)
	}
	if t := newListSkillsTool(); t != nil {
		tools = append(tools, t)
	}
	if t := newInstallSkillTool(); t != nil {
		tools = append(tools, t)
	}
	return tools
}
