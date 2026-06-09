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

package sandbox

import "context"

// Backend is the interface for sandbox execution backends.
// Implementations provide isolated environments for tool execution.
type Backend interface {
	// Init prepares the sandbox environment (e.g. pull images, create namespaces).
	Init(ctx context.Context) error
	// Execute runs a tool invocation inside the sandbox.
	Execute(ctx context.Context, req *ExecRequest) (*ExecResult, error)
	// Cleanup releases sandbox resources.
	Cleanup(ctx context.Context) error
}

// ExecRequest carries the tool invocation payload into the sandbox.
type ExecRequest struct {
	ToolName string
	Args     string
	Policy   *Policy
}

// ExecResult carries the tool output from the sandbox.
type ExecResult struct {
	Output string
	Error  string
}

// Policy defines resource and access constraints for a sandboxed execution.
type Policy struct {
	TimeoutSeconds int
	MemoryLimitMB  int64
	AllowNet       []string
	AllowRead      []string
	AllowWrite     []string
	AllowEnv       []string
}

// NewBackend creates a sandbox backend by name with automatic fallback.
// When backendName is "docker" and Docker is unavailable, falls back to "process".
func NewBackend(ctx context.Context, backendName string, defaultPolicy *Policy) (Backend, error) {
	switch backendName {
	case "docker":
		b := NewDockerBackend(defaultPolicy)
		if err := b.Init(ctx); err != nil {
			// Fallback to process sandbox if Docker is unavailable.
			fb := NewProcessBackend(defaultPolicy)
			return fb, fb.Init(ctx)
		}
		return b, nil
	case "process":
		b := NewProcessBackend(defaultPolicy)
		return b, b.Init(ctx)
	default:
		// Default: try docker, fallback to process.
		b := NewDockerBackend(defaultPolicy)
		if err := b.Init(ctx); err != nil {
			fb := NewProcessBackend(defaultPolicy)
			return fb, fb.Init(ctx)
		}
		return b, nil
	}
}
