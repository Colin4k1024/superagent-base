/*
 * Copyright 2025 coze-dev Authors
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

package llm

import "context"

// Tool is the framework-agnostic tool interface.
// It abstracts eino's components/tool.InvokableTool
// and Google ADK Go's tool.Tool into a single contract.
type Tool interface {
	// Info returns the tool's metadata (name, description, params schema).
	Info(ctx context.Context) (*ToolInfo, error)

	// Run executes the tool with JSON-encoded arguments and returns
	// the result as a string.
	Run(ctx context.Context, argsJSON string, opts ...ToolOption) (string, error)
}

// ToolRegistry manages named tools and optionally wraps invocations
// with a middleware chain.
type ToolRegistry interface {
	// Register adds a tool to the registry.
	Register(t Tool) error

	// Get returns a tool by name.
	Get(name string) (Tool, bool)

	// List returns all registered tools.
	List(ctx context.Context) ([]Tool, error)
}
