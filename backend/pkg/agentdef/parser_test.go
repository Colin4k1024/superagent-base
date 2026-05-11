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
	"os"
	"path/filepath"
	"testing"
)

const validYAML = `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: test-agent
  version: "1.0.0"
  tags: [test]
spec:
  type: chat_model_agent
  model:
    primary: gpt-4o
    fallback: deepseek-r1
  system_prompt: "You are a helpful assistant."
  tools:
    - ref: builtin/web_search
  memory:
    backend: builtin
  observability:
    tracing: true
    log_level: info
`

func TestParse_ValidYAML(t *testing.T) {
	def, err := Parse([]byte(validYAML))
	if err != nil {
		t.Fatalf("Parse returned unexpected error: %v", err)
	}
	if def.Metadata.Name != "test-agent" {
		t.Errorf("expected name %q, got %q", "test-agent", def.Metadata.Name)
	}
	if def.Spec.Type != "chat_model_agent" {
		t.Errorf("expected type %q, got %q", "chat_model_agent", def.Spec.Type)
	}
	if def.Spec.Model.Primary != "gpt-4o" {
		t.Errorf("expected model.primary %q, got %q", "gpt-4o", def.Spec.Model.Primary)
	}
	if def.Spec.Model.Fallback != "deepseek-r1" {
		t.Errorf("expected model.fallback %q, got %q", "deepseek-r1", def.Spec.Model.Fallback)
	}
	if len(def.Spec.Tools) != 1 || def.Spec.Tools[0].Ref != "builtin/web_search" {
		t.Errorf("unexpected tools: %+v", def.Spec.Tools)
	}
	if def.Spec.Memory.Backend != "builtin" {
		t.Errorf("expected memory.backend %q, got %q", "builtin", def.Spec.Memory.Backend)
	}
	if !def.Spec.Observability.Tracing {
		t.Error("expected observability.tracing = true")
	}
}

func TestValidate_MissingName(t *testing.T) {
	yaml := `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: ""
spec:
  type: chat_model_agent
  model:
    primary: gpt-4o
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
}

func TestValidate_InvalidName(t *testing.T) {
	yaml := `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: "My Agent!"
spec:
  type: chat_model_agent
  model:
    primary: gpt-4o
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for invalid name, got nil")
	}
}

func TestValidate_InvalidType(t *testing.T) {
	yaml := `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: test-agent
spec:
  type: unknown_type
  model:
    primary: gpt-4o
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}
}

func TestValidate_MissingModel(t *testing.T) {
	yaml := `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: test-agent
spec:
  type: chat_model_agent
  model:
    primary: ""
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing model.primary, got nil")
	}
}

func TestValidate_WrongAPIVersion(t *testing.T) {
	yaml := `
apiVersion: superagent/v2
kind: Agent
metadata:
  name: test-agent
spec:
  type: chat_model_agent
  model:
    primary: gpt-4o
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for wrong apiVersion, got nil")
	}
}

func TestValidate_WrongKind(t *testing.T) {
	yaml := `
apiVersion: superagent/v1
kind: Workflow
metadata:
  name: test-agent
spec:
  type: chat_model_agent
  model:
    primary: gpt-4o
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for wrong kind, got nil")
	}
}

func TestParseFile_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.yaml")
	if err := os.WriteFile(path, []byte(validYAML), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	def, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile returned unexpected error: %v", err)
	}
	if def.Metadata.Name != "test-agent" {
		t.Errorf("expected name %q, got %q", "test-agent", def.Metadata.Name)
	}
}

func TestParseFile_NotFound(t *testing.T) {
	_, err := ParseFile("/nonexistent/path/agent.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestLoader_LoadAll(t *testing.T) {
	dir := t.TempDir()
	yaml2 := `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: second-agent
spec:
  type: deep_agent
  model:
    primary: deepseek-r1
`
	for _, tc := range []struct{ name, content string }{
		{"agent.yaml", validYAML},
		{"second.yaml", yaml2},
	} {
		if err := os.WriteFile(filepath.Join(dir, tc.name), []byte(tc.content), 0o644); err != nil {
			t.Fatalf("write %s: %v", tc.name, err)
		}
	}

	l := NewLoader(dir)
	loaded, err := l.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll returned unexpected error: %v", err)
	}
	if len(loaded) != 2 {
		t.Errorf("expected 2 agents, got %d", len(loaded))
	}
	if _, ok := loaded["test-agent"]; !ok {
		t.Error("expected test-agent in loaded map")
	}
	if _, ok := l.Get("second-agent"); !ok {
		t.Error("expected second-agent retrievable via Get")
	}
}

func TestLoader_SkipsBadFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte("not: valid: agent:"), 0o644); err != nil {
		t.Fatalf("write bad file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "good.yaml"), []byte(validYAML), 0o644); err != nil {
		t.Fatalf("write good file: %v", err)
	}
	l := NewLoader(dir)
	loaded, err := l.LoadAll()
	// Should return partial results + error.
	if err == nil {
		t.Fatal("expected error from bad file, got nil")
	}
	if len(loaded) != 1 {
		t.Errorf("expected 1 valid agent, got %d", len(loaded))
	}
}
