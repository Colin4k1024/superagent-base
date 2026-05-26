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
//
// Tool inventory:
//
//	Original tools
//	  web_search    — web search via configured provider
//	  http_request  — generic HTTP client
//	  code_execute  — execute Python / Bash code blocks
//
//	File-system tools (opencode-inspired)
//	  file_read     — read file contents with optional line-range
//	  file_write    — create or overwrite a file
//	  file_edit     — exact string replacement inside a file
//	  file_glob     — find files matching a glob pattern
//
//	Shell tool
//	  shell_execute — run an arbitrary shell command
//
//	Code-search tools (opencode-inspired)
//	  grep_search   — regex content search (ripgrep or stdlib)
//	  find_files    — find files by name / extension
//	  repo_overview — directory tree + statistics
//
//	Git tools (read-only, opencode-inspired)
//	  git_status    — working tree status
//	  git_diff      — staged / unstaged diff
//	  git_log       — recent commit history
func GetAllBuiltinTools() []tool.InvokableTool {
	return []tool.InvokableTool{
		// ── original tools ────────────────────────────────────────────────
		newWebSearchTool(),
		newHTTPRequestTool(),
		newCodeExecTool(),

		// ── file-system tools ─────────────────────────────────────────────
		// file_write and file_edit are registered with AllowWrite=true so that
		// agents can use them when the YAML includes the corresponding tool refs.
		// Agents that should be read-only can simply omit those refs.
		newFileReadTool(FileOpsConfig{}),
		newFileWriteTool(FileOpsConfig{AllowWrite: true}),
		newFileEditTool(FileOpsConfig{AllowWrite: true}),
		newFileGlobTool(FileOpsConfig{}),

		// ── shell tool ────────────────────────────────────────────────────
		newShellExecuteTool(),

		// ── code-search tools ─────────────────────────────────────────────
		newGrepSearchTool(),
		newFindFilesTool(),
		newRepoOverviewTool(),

		// ── git tools (read-only) ─────────────────────────────────────────
		newGitStatusTool(),
		newGitDiffTool(),
		newGitLogTool(),
	}
}
