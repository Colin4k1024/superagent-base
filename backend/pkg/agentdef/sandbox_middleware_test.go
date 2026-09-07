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
	"testing"
	"time"

	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
	"github.com/superagent-ai/superagent-base/backend/pkg/tool/sandbox"
)

// mockBackend implements sandbox.Backend for testing.
type mockBackend struct {
	initErr    error
	execResult *sandbox.ExecResult
	execErr    error
	execCalled bool
	lastReq    *sandbox.ExecRequest
}

func (m *mockBackend) Init(_ context.Context) error { return m.initErr }
func (m *mockBackend) Cleanup(_ context.Context) error { return nil }
func (m *mockBackend) Execute(_ context.Context, req *sandbox.ExecRequest) (*sandbox.ExecResult, error) {
	m.execCalled = true
	m.lastReq = req
	return m.execResult, m.execErr
}

// mockTool implements llm.Tool for testing.
type mockTool struct {
	name     string
	runFunc  func(ctx context.Context, args string) (string, error)
}

func (t *mockTool) Info(_ context.Context) (*llm.ToolInfo, error) {
	return &llm.ToolInfo{Name: t.name}, nil
}

func (t *mockTool) Run(ctx context.Context, args string, _ ...llm.ToolOption) (string, error) {
	if t.runFunc != nil {
		return t.runFunc(ctx, args)
	}
	return "original", nil
}

func TestSandboxMiddleware_CodeExecute_DelegatesToBackend(t *testing.T) {
	backend := &mockBackend{
		execResult: &sandbox.ExecResult{Output: `{"stdout":"hello","stderr":"","exit_code":0}`},
	}
	policy := &sandbox.Policy{TimeoutSeconds: 10, MemoryLimitMB: 128}
	mw := newSandboxMiddleware(backend, policy, nil)

	ctx := context.Background()
	original := &mockTool{name: "code_execute"}

	wrapped, err := mw.WrapTool(ctx, original)
	if err != nil {
		t.Fatalf("WrapTool error: %v", err)
	}

	result, err := wrapped.Run(ctx, `{"language":"python","code":"print('hello')"}`)
	if err != nil {
		t.Fatalf("wrapped call error: %v", err)
	}

	if !backend.execCalled {
		t.Error("expected backend.Execute to be called")
	}
	if result != `{"stdout":"hello","stderr":"","exit_code":0}` {
		t.Errorf("unexpected result: %s", result)
	}
	if backend.lastReq.ToolName != "code_execute" {
		t.Errorf("unexpected tool name: %s", backend.lastReq.ToolName)
	}
}

func TestSandboxMiddleware_RegularTool_WrapsOriginalEndpoint(t *testing.T) {
	backend := &mockBackend{}
	policy := &sandbox.Policy{TimeoutSeconds: 5, MemoryLimitMB: 128}
	mw := newSandboxMiddleware(backend, policy, nil)

	ctx := context.Background()
	original := &mockTool{name: "web_search"}

	wrapped, err := mw.WrapTool(ctx, original)
	if err != nil {
		t.Fatalf("WrapTool error: %v", err)
	}

	result, err := wrapped.Run(ctx, `{"query":"test"}`)
	if err != nil {
		t.Fatalf("wrapped call error: %v", err)
	}

	if backend.execCalled {
		t.Error("expected backend.Execute NOT to be called for web_search")
	}
	if result != "original" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestSandboxMiddleware_PerToolPolicyOverride(t *testing.T) {
	backend := &mockBackend{
		execResult: &sandbox.ExecResult{Output: "ok"},
	}
	defaultPolicy := &sandbox.Policy{TimeoutSeconds: 30, MemoryLimitMB: 256}
	perTool := map[string]*sandbox.Policy{
		"code_execute": {TimeoutSeconds: 5, MemoryLimitMB: 64},
	}
	mw := newSandboxMiddleware(backend, defaultPolicy, perTool)

	ctx := context.Background()
	original := &mockTool{name: "code_execute"}

	wrapped, _ := mw.WrapTool(ctx, original)
	_, _ = wrapped.Run(ctx, `{"code":"x"}`)

	if backend.lastReq.Policy.TimeoutSeconds != 5 {
		t.Errorf("expected per-tool timeout 5, got %d", backend.lastReq.Policy.TimeoutSeconds)
	}
	if backend.lastReq.Policy.MemoryLimitMB != 64 {
		t.Errorf("expected per-tool memory 64, got %d", backend.lastReq.Policy.MemoryLimitMB)
	}
}

func TestSandboxMiddleware_Timeout(t *testing.T) {
	backend := &mockBackend{}
	policy := &sandbox.Policy{TimeoutSeconds: 1, MemoryLimitMB: 128}
	mw := newSandboxMiddleware(backend, policy, nil)

	ctx := context.Background()
	slowTool := &mockTool{name: "http_request", runFunc: func(ctx context.Context, _ string) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(5 * time.Second):
			return "late", nil
		}
	}}

	wrapped, _ := mw.WrapTool(ctx, slowTool)
	_, err := wrapped.Run(ctx, `{}`)

	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestSandboxMiddleware_BackendError(t *testing.T) {
	backend := &mockBackend{execErr: fmt.Errorf("container crashed")}
	policy := &sandbox.Policy{TimeoutSeconds: 10, MemoryLimitMB: 128}
	mw := newSandboxMiddleware(backend, policy, nil)

	ctx := context.Background()
	original := &mockTool{name: "code_execute"}

	wrapped, _ := mw.WrapTool(ctx, original)
	_, err := wrapped.Run(ctx, `{"code":"x"}`)

	if err == nil {
		t.Fatal("expected error from backend")
	}
}

func TestSandboxMiddleware_BackendSandboxError(t *testing.T) {
	backend := &mockBackend{
		execResult: &sandbox.ExecResult{Error: "memory limit exceeded"},
	}
	policy := &sandbox.Policy{TimeoutSeconds: 10, MemoryLimitMB: 128}
	mw := newSandboxMiddleware(backend, policy, nil)

	ctx := context.Background()
	original := &mockTool{name: "code_execute"}

	wrapped, _ := mw.WrapTool(ctx, original)
	result, err := wrapped.Run(ctx, `{"code":"x"}`)

	if err != nil {
		t.Fatalf("unexpected hard error: %v", err)
	}
	expected := "[sandbox error] code_execute: memory limit exceeded"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestPolicyFromSandboxSpec(t *testing.T) {
	spec := &SandboxSpec{
		Enabled:        true,
		TimeoutSeconds: 60,
		MemoryLimitMB:  512,
		AllowNet:       []string{"*.example.com"},
		AllowRead:      []string{"/data"},
		AllowWrite:     []string{"/tmp"},
		AllowEnv:       []string{"TOKEN"},
	}
	p := policyFromSandboxSpec(spec)
	if p.TimeoutSeconds != 60 {
		t.Errorf("expected timeout 60, got %d", p.TimeoutSeconds)
	}
	if p.MemoryLimitMB != 512 {
		t.Errorf("expected memory 512, got %d", p.MemoryLimitMB)
	}
}

func TestPolicyFromSandboxSpec_Defaults(t *testing.T) {
	p := policyFromSandboxSpec(nil)
	if p.TimeoutSeconds != 30 {
		t.Errorf("expected default timeout 30, got %d", p.TimeoutSeconds)
	}
	if p.MemoryLimitMB != 256 {
		t.Errorf("expected default memory 256, got %d", p.MemoryLimitMB)
	}
}

func TestExtractToolName(t *testing.T) {
	tests := []struct {
		ref  string
		want string
	}{
		{"builtin/web_search", "web_search"},
		{"builtin/code_execute", "code_execute"},
		{"mcp://server/tool_name", "tool_name"},
		{"skill://my_skill", "my_skill"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		got := extractToolName(tt.ref)
		if got != tt.want {
			t.Errorf("extractToolName(%q) = %q, want %q", tt.ref, got, tt.want)
		}
	}
}
