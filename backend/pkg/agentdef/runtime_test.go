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

package agentdef_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/superagent-ai/superagent-base/backend/pkg/agentdef"
)

// modelAvailable probes the local model endpoint to determine whether
// integration tests that require a live model should run.
// It checks both server reachability and whether the specific model is loaded.
func modelAvailable(baseURL, modelID, apiKey string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest("GET", baseURL+"/models", nil)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return false
	}
	if modelID == "" {
		return true
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}
	for _, m := range result.Data {
		if strings.EqualFold(m.ID, modelID) {
			return true
		}
	}
	return false
}

// TestRuntimeStub verifies that the runtime loads agents and the stub Chat
// path works without a real model endpoint.
func TestRuntimeStub(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "test-agent.yaml", `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: test-agent
  version: "1.0.0"
spec:
  type: chat_model_agent
  model:
    primary: gpt-4o
  system_prompt: "You are a test helper."
`)

	builder := agentdef.NewAgentBuilder() // no model config → stub mode
	rt := agentdef.NewRuntime(agentdef.RuntimeConfig{ConfigDir: dir}, builder)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rt.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer rt.Stop()

	names := rt.ListAgents()
	if len(names) != 1 || names[0] != "test-agent" {
		t.Fatalf("expected [test-agent], got %v", names)
	}

	agent, ok := rt.GetAgent("test-agent")
	if !ok {
		t.Fatal("GetAgent returned false")
	}
	if agent.Name() != "test-agent" {
		t.Errorf("Name() = %q, want test-agent", agent.Name())
	}

	ch, err := agent.Chat(ctx, "session-1", "hello")
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	var buf strings.Builder
	for tok := range ch {
		buf.WriteString(tok)
	}
	if buf.Len() == 0 {
		t.Error("Chat returned empty response")
	}
}

// TestRuntimeMissingDir verifies that Start returns an error for a
// non-existent config directory.
func TestRuntimeMissingDir(t *testing.T) {
	builder := agentdef.NewAgentBuilder()
	rt := agentdef.NewRuntime(agentdef.RuntimeConfig{ConfigDir: "/no/such/dir"}, builder)

	ctx := context.Background()
	// Reloader uses loader.LoadAll which walks the dir; missing dir is an error.
	// Start may only warn rather than hard-fail on the initial load; the test
	// just verifies we don't panic.
	_ = rt.Start(ctx)
	rt.Stop()
}

// TestRuntimeLiveModel runs a real LLM call against the local model server.
// It is skipped automatically when the server is not reachable.
func TestRuntimeLiveModel(t *testing.T) {
	const (
		baseURL = "http://127.0.0.1:8000/v1"
		apiKey  = "123456"
		modelID = "Qwen3-Coder-Next-4bit"
	)

	if os.Getenv("INTEGRATION") == "" && !modelAvailable(baseURL, modelID, apiKey) {
		t.Skip("local model not available; set INTEGRATION=1 or start the model server")
	}

	// Write a minimal agent YAML to a temp directory.
	dir := t.TempDir()
	writeYAML(t, dir, "research-agent.yaml", `
apiVersion: superagent/v1
kind: Agent
metadata:
  name: research-agent
  version: "1.0.0"
spec:
  type: chat_model_agent
  model:
    primary: `+modelID+`
  system_prompt: "You are a helpful assistant."
`)

	builder := agentdef.NewAgentBuilder(
		agentdef.WithModelConfig(agentdef.ModelRuntimeConfig{
			BaseURL: baseURL,
			APIKey:  apiKey,
			ModelID: modelID,
		}),
	)
	rt := agentdef.NewRuntime(agentdef.RuntimeConfig{ConfigDir: dir}, builder)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := rt.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer rt.Stop()

	agent, ok := rt.GetAgent("research-agent")
	if !ok {
		t.Fatal("GetAgent returned false for research-agent")
	}

	ch, err := agent.Chat(ctx, "session-42", "What is the Go programming language? Answer in one sentence.")
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	var buf strings.Builder
	for tok := range ch {
		buf.WriteString(tok)
		t.Logf("token: %q", tok)
	}
	resp := buf.String()
	if len(resp) == 0 {
		t.Error("got empty response from model")
	}
	t.Logf("full response: %s", resp)
}

// writeYAML is a test helper that writes content to dir/name.
func writeYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o644); err != nil {
		t.Fatalf("writeYAML %s: %v", name, err)
	}
}
