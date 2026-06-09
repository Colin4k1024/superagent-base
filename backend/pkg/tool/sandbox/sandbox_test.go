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

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

func TestProcessBackend_Init(t *testing.T) {
	b := NewProcessBackend(nil)
	if err := b.Init(context.Background()); err != nil {
		t.Fatalf("Init should not error: %v", err)
	}
}

func TestProcessBackend_Execute_Python(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}

	b := NewProcessBackend(&Policy{TimeoutSeconds: 10, MemoryLimitMB: 128})
	_ = b.Init(context.Background())

	result, err := b.Execute(context.Background(), &ExecRequest{
		ToolName: "code_execute",
		Args:     `{"language":"python","code":"print('sandbox-test')"}`,
		Policy:   nil,
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.Error != "" {
		t.Fatalf("unexpected sandbox error: %s", result.Error)
	}

	var output struct {
		Stdout   string `json:"stdout"`
		Stderr   string `json:"stderr"`
		ExitCode int    `json:"exit_code"`
	}
	if err := json.Unmarshal([]byte(result.Output), &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if strings.TrimSpace(output.Stdout) != "sandbox-test" {
		t.Errorf("expected 'sandbox-test', got %q", output.Stdout)
	}
	if output.ExitCode != 0 {
		t.Errorf("expected exit_code 0, got %d", output.ExitCode)
	}
}

func TestProcessBackend_Execute_Bash(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}

	b := NewProcessBackend(&Policy{TimeoutSeconds: 10, MemoryLimitMB: 128})
	_ = b.Init(context.Background())

	result, err := b.Execute(context.Background(), &ExecRequest{
		ToolName: "code_execute",
		Args:     `{"language":"bash","code":"echo hello-bash"}`,
		Policy:   nil,
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.Error != "" {
		t.Fatalf("unexpected sandbox error: %s", result.Error)
	}

	var output struct {
		Stdout   string `json:"stdout"`
		ExitCode int    `json:"exit_code"`
	}
	if err := json.Unmarshal([]byte(result.Output), &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if strings.TrimSpace(output.Stdout) != "hello-bash" {
		t.Errorf("expected 'hello-bash', got %q", output.Stdout)
	}
}

func TestProcessBackend_Execute_Timeout(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}

	b := NewProcessBackend(nil)
	_ = b.Init(context.Background())

	result, err := b.Execute(context.Background(), &ExecRequest{
		ToolName: "code_execute",
		Args:     `{"language":"python","code":"import time; time.sleep(10)"}`,
		Policy:   &Policy{TimeoutSeconds: 1, MemoryLimitMB: 128},
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.Error == "" {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(result.Error, "timeout") {
		t.Errorf("expected timeout in error, got: %s", result.Error)
	}
}

func TestProcessBackend_Execute_NonZeroExit(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}

	b := NewProcessBackend(&Policy{TimeoutSeconds: 10, MemoryLimitMB: 128})
	_ = b.Init(context.Background())

	result, err := b.Execute(context.Background(), &ExecRequest{
		ToolName: "code_execute",
		Args:     `{"language":"python","code":"import sys; sys.exit(42)"}`,
		Policy:   nil,
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.Error != "" {
		t.Fatalf("unexpected sandbox error: %s", result.Error)
	}

	var output struct {
		ExitCode int `json:"exit_code"`
	}
	if err := json.Unmarshal([]byte(result.Output), &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if output.ExitCode != 42 {
		t.Errorf("expected exit_code 42, got %d", output.ExitCode)
	}
}

func TestProcessBackend_Execute_RestrictedEnv(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}

	b := NewProcessBackend(nil)
	_ = b.Init(context.Background())

	// HOME should not be passed through when AllowEnv is restricted.
	result, err := b.Execute(context.Background(), &ExecRequest{
		ToolName: "code_execute",
		Args:     `{"language":"python","code":"import os; print(os.environ.get('HOME', 'NONE'))"}`,
		Policy:   &Policy{TimeoutSeconds: 5, MemoryLimitMB: 128, AllowEnv: []string{}},
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	var output struct {
		Stdout string `json:"stdout"`
	}
	if err := json.Unmarshal([]byte(result.Output), &output); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if strings.TrimSpace(output.Stdout) != "NONE" {
		t.Errorf("expected HOME to be hidden, got %q", output.Stdout)
	}
}

func TestProcessBackend_Execute_NoCode(t *testing.T) {
	b := NewProcessBackend(&Policy{TimeoutSeconds: 5, MemoryLimitMB: 128})
	_ = b.Init(context.Background())

	result, err := b.Execute(context.Background(), &ExecRequest{
		ToolName: "code_execute",
		Args:     `{"language":"python"}`,
		Policy:   nil,
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.Error == "" || !strings.Contains(result.Error, "no code") {
		t.Errorf("expected 'no code' error, got: %q", result.Error)
	}
}

func TestProcessBackend_Execute_InvalidArgs(t *testing.T) {
	b := NewProcessBackend(&Policy{TimeoutSeconds: 5, MemoryLimitMB: 128})
	_ = b.Init(context.Background())

	result, err := b.Execute(context.Background(), &ExecRequest{
		ToolName: "code_execute",
		Args:     `not-json`,
		Policy:   nil,
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.Error == "" || !strings.Contains(result.Error, "parse") {
		t.Errorf("expected parse error, got: %q", result.Error)
	}
}

func TestNewBackend_ProcessFallback(t *testing.T) {
	// When Docker is unavailable, NewBackend should fallback to process.
	ctx := context.Background()
	b, err := NewBackend(ctx, "process", &Policy{TimeoutSeconds: 10})
	if err != nil {
		t.Fatalf("NewBackend error: %v", err)
	}
	if b == nil {
		t.Fatal("expected non-nil backend")
	}
	// Verify it's a ProcessBackend.
	if _, ok := b.(*ProcessBackend); !ok {
		t.Errorf("expected ProcessBackend, got %T", b)
	}
}

func TestBuildRestrictedEnv(t *testing.T) {
	env := buildRestrictedEnv([]string{"PATH", "NONEXIST_VAR"})
	if len(env) < 1 {
		t.Fatal("expected at least PATH entry")
	}
	foundPath := false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			foundPath = true
		}
	}
	if !foundPath {
		t.Error("expected PATH in restricted env")
	}
}
