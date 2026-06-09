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

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"

	"github.com/superagent-ai/superagent-base/backend/pkg/tool/sandbox"
)

// Compile-time assertion.
var _ adk.ChatModelAgentMiddleware = (*sandboxMiddleware)(nil)

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
	*adk.BaseChatModelAgentMiddleware
	backend       sandbox.Backend
	defaultPolicy *sandbox.Policy
	perToolPolicy map[string]*sandbox.Policy
}

// newSandboxMiddleware creates a sandbox middleware with the given backend and policy.
func newSandboxMiddleware(backend sandbox.Backend, defaultPolicy *sandbox.Policy, perToolPolicy map[string]*sandbox.Policy) *sandboxMiddleware {
	return &sandboxMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
		backend:                      backend,
		defaultPolicy:                defaultPolicy,
		perToolPolicy:                perToolPolicy,
	}
}

func (m *sandboxMiddleware) WrapInvokableToolCall(
	ctx context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		policy := m.effectivePolicy(tCtx.Name)

		// Tools requiring full isolation are delegated to the sandbox backend.
		if sandboxToolsRequiringFullIsolation[tCtx.Name] {
			return m.executeInSandbox(ctx, tCtx.Name, argumentsInJSON, policy)
		}

		// Other tools: wrap original endpoint with sandbox timeout + error boundary.
		return m.executeWrapped(ctx, endpoint, tCtx.Name, argumentsInJSON, policy, opts...)
	}, nil
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

// executeWrapped runs the original tool endpoint with sandbox constraints:
// enforced timeout, panic recovery, and output size limits.
func (m *sandboxMiddleware) executeWrapped(
	ctx context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	toolName, args string,
	policy *sandbox.Policy,
	opts ...tool.Option,
) (string, error) {
	timeout := time.Duration(policy.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute original endpoint with timeout context.
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
		out, err := endpoint(execCtx, args, opts...)
		ch <- result{output: out, err: err}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			return "", fmt.Errorf("[sandbox] %s: %w", toolName, r.err)
		}
		// Enforce output size limit (1MB).
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
		MemoryLimitMB:  spec.MemoryLimitMB,
		AllowNet:       spec.AllowNet,
		AllowRead:      spec.AllowRead,
		AllowWrite:     spec.AllowWrite,
		AllowEnv:       spec.AllowEnv,
	}
	if p.TimeoutSeconds == 0 {
		p.TimeoutSeconds = 30
	}
	if p.MemoryLimitMB == 0 {
		p.MemoryLimitMB = 256
	}
	return p
}
