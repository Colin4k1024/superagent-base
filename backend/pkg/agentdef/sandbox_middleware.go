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
	"fmt"
	"time"

	aclagent "github.com/superagent-ai/superagent-base/backend/pkg/agent"
	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
	"github.com/superagent-ai/superagent-base/backend/pkg/tool/sandbox"
)

// sandboxToolsRequiringFullIsolation lists tools whose execution is fully
// delegated to the sandbox backend (code runs inside container/process).
var sandboxToolsRequiringFullIsolation = map[string]bool{
	"code_execute": true,
}

// sandboxMiddleware wraps all tool calls with sandbox constraints.
// - For code-execution tools: fully delegates to the sandbox backend.
// - For other tools: wraps the original endpoint with timeout enforcement
//   and error boundary, logging the sandbox policy constraints.
type sandboxMiddleware struct {
	aclagent.BaseMiddleware
	backend       sandbox.Backend
	defaultPolicy *sandbox.Policy
	perToolPolicy map[string]*sandbox.Policy
}

// newSandboxMiddleware creates a sandbox middleware with the given backend and policy.
func newSandboxMiddleware(backend sandbox.Backend, defaultPolicy *sandbox.Policy, perToolPolicy map[string]*sandbox.Policy) *sandboxMiddleware {
	return &sandboxMiddleware{
		backend:       backend,
		defaultPolicy: defaultPolicy,
		perToolPolicy: perToolPolicy,
	}
}

// WrapTool implements agent.Middleware.WrapTool.
func (m *sandboxMiddleware) WrapTool(ctx context.Context, t llm.Tool) (llm.Tool, error) {
	info, err := t.Info(ctx)
	if err != nil {
		return nil, err
	}
	policy := m.effectivePolicy(info.Name)
	if sandboxToolsRequiringFullIsolation[info.Name] {
		return &sandboxedTool{
			inner:      t,
			name:       info.Name,
			mw:         m,
			policy:     policy,
			fullIsolation: true,
		}, nil
	}
	return &sandboxedTool{
		inner:      t,
		name:       info.Name,
		mw:         m,
		policy:     policy,
		fullIsolation: false,
	}, nil
}

type sandboxedTool struct {
	inner          llm.Tool
	name           string
	mw             *sandboxMiddleware
	policy         *sandbox.Policy
	fullIsolation  bool
}

func (t *sandboxedTool) Info(ctx context.Context) (*llm.ToolInfo, error) {
	return t.inner.Info(ctx)
}

func (t *sandboxedTool) Run(ctx context.Context, args string, opts ...llm.ToolOption) (string, error) {
	if t.fullIsolation {
		return t.mw.executeInSandbox(ctx, t.name, args, t.policy)
	}
	return t.mw.executeWrapped(ctx, t.inner, t.name, args, t.policy, opts...)
}

// executeInSandbox fully delegates tool execution to the sandbox backend.
func (m *sandboxMiddleware) executeInSandbox(ctx context.Context, toolName, args string, policy *sandbox.Policy) (string, error) {
	result, err := m.backend.Execute(ctx, &sandbox.ExecRequest{
		ToolName: toolName,
		Args:     args,
		Policy:   policy,
	})
	if err != nil {
		return "", fmt.Errorf("[sandbox] %s: %w", toolName, err)
	}
	if result.Error != "" {
		return fmt.Sprintf("[sandbox error] %s: %s", toolName, result.Error), nil
	}
	return result.Output, nil
}

// executeWrapped runs the original tool with sandbox constraints:
// enforced timeout, panic recovery, and output size limits.
func (m *sandboxMiddleware) executeWrapped(
	ctx context.Context,
	inner llm.Tool,
	toolName, args string,
	policy *sandbox.Policy,
	opts ...llm.ToolOption,
) (string, error) {
	timeout := time.Duration(policy.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	type result struct {
		output string
		err    error
	}
	ch := make(chan result, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				ch <- result{err: fmt.Errorf("[sandbox] %s: panic: %v", toolName, r)}
			}
		}()
		out, err := inner.Run(execCtx, args, opts...)
		ch <- result{output: out, err: err}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			return "", fmt.Errorf("[sandbox] %s: %w", toolName, r.err)
		}
		const maxOutputBytes = 1 << 20
		if len(r.output) > maxOutputBytes {
			return r.output[:maxOutputBytes] + "\n...[truncated by sandbox]", nil
		}
		return r.output, nil
	case <-execCtx.Done():
		return "", fmt.Errorf("[sandbox] %s: execution timed out after %s", toolName, timeout)
	}
}

// effectivePolicy returns the per-tool policy if configured, otherwise the default.
func (m *sandboxMiddleware) effectivePolicy(toolName string) *sandbox.Policy {
	if tp, ok := m.perToolPolicy[toolName]; ok {
		return tp
	}
	return m.defaultPolicy
}

// policyFromSandboxSpec converts a SandboxSpec into a sandbox.Policy.
func policyFromSandboxSpec(spec *SandboxSpec) *sandbox.Policy {
	if spec == nil {
		return &sandbox.Policy{TimeoutSeconds: 30, MemoryLimitMB: 256}
	}
	p := &sandbox.Policy{
		TimeoutSeconds: spec.TimeoutSeconds,
		MemoryLimitMB: spec.MemoryLimitMB,
		AllowNet:      spec.AllowNet,
		AllowRead:     spec.AllowRead,
		AllowWrite:    spec.AllowWrite,
		AllowEnv:      spec.AllowEnv,
	}
	if p.TimeoutSeconds == 0 {
		p.TimeoutSeconds = 30
	}
	if p.MemoryLimitMB == 0 {
		p.MemoryLimitMB = 256
	}
	return p
}
