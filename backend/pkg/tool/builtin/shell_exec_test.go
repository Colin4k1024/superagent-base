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
	"strings"
	"testing"
)

func TestShellExecuteTool_Echo(t *testing.T) {
	tool := newShellExecuteTool()
	args, _ := json.Marshal(shellExecuteParams{Command: "echo hello"})
	out, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res shellExecuteResult
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !strings.Contains(res.Stdout, "hello") {
		t.Errorf("want 'hello' in stdout, got %q", res.Stdout)
	}
	if res.ExitCode != 0 {
		t.Errorf("want exit_code 0, got %d", res.ExitCode)
	}
	if res.TimedOut {
		t.Error("should not time out")
	}
}

func TestShellExecuteTool_ExitCode(t *testing.T) {
	tool := newShellExecuteTool()
	args, _ := json.Marshal(shellExecuteParams{Command: "exit 42"})
	out, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res shellExecuteResult
	json.Unmarshal([]byte(out), &res)
	if res.ExitCode != 42 {
		t.Errorf("want exit_code 42, got %d", res.ExitCode)
	}
}

func TestShellExecuteTool_Timeout(t *testing.T) {
	tool := newShellExecuteTool()
	args, _ := json.Marshal(shellExecuteParams{Command: "sleep 60", TimeoutSeconds: 1})
	out, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res shellExecuteResult
	json.Unmarshal([]byte(out), &res)
	if !res.TimedOut {
		t.Error("expected timed_out=true")
	}
}

func TestShellExecuteTool_EnvVar(t *testing.T) {
	tool := newShellExecuteTool()
	args, _ := json.Marshal(shellExecuteParams{
		Command: "echo $MY_VAR",
		Env:     map[string]string{"MY_VAR": "testvalue"},
	})
	out, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var res shellExecuteResult
	json.Unmarshal([]byte(out), &res)
	if !strings.Contains(res.Stdout, "testvalue") {
		t.Errorf("want 'testvalue' in stdout, got %q", res.Stdout)
	}
}

func TestShellExecuteTool_EmptyCommand(t *testing.T) {
	tool := newShellExecuteTool()
	args, _ := json.Marshal(shellExecuteParams{Command: ""})
	_, err := tool.InvokableRun(context.Background(), string(args))
	if err == nil {
		t.Fatal("expected error for empty command")
	}
}
