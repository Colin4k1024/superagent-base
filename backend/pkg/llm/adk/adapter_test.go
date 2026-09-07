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

package adk

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	aclllm "github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

// modelEndpointAvailable probes the local model endpoint to determine whether
// integration tests that require a live model should run.
func modelEndpointAvailable() (baseURL, apiKey, modelID string, ok bool) {
	baseURL = os.Getenv("MODEL_BASE_URL_0")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8000/v1"
	}
	apiKey = os.Getenv("MODEL_API_KEY_0")
	if apiKey == "" {
		apiKey = "123456"
	}
	modelID = os.Getenv("MODEL_ID_0")
	if modelID == "" {
		modelID = "Qwen3-Coder-Next-4bit"
	}

	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest("GET", baseURL+"/models", nil)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return baseURL, apiKey, modelID, false
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return baseURL, apiKey, modelID, false
	}
	return baseURL, apiKey, modelID, true
}

// TestADKChatModel_Generate verifies non-streaming generation via the ADK Go adapter.
func TestADKChatModel_Generate(t *testing.T) {
	baseURL, apiKey, modelID, ok := modelEndpointAvailable()
	if !ok {
		t.Skip("model endpoint not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reg := NewDefaultRegistry()
	chatModel, err := reg.Create(ctx, aclllm.ModelConfig{
		Protocol: "openai",
		BaseURL:  baseURL,
		APIKey:   apiKey,
		ModelID:  modelID,
	})
	if err != nil {
		t.Fatalf("Create model: %v", err)
	}

	msgs := []*aclllm.Message{
		aclllm.SystemMessage("You are a helpful assistant. Answer concisely."),
		aclllm.UserMessage("What is 2+2? Answer with just the number."),
	}

	resp, err := chatModel.Generate(ctx, msgs)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if resp == nil || resp.Content == "" {
		t.Fatalf("expected non-empty response, got %+v", resp)
	}
	t.Logf("Generate response: %s", resp.Content)
	if !strings.Contains(resp.Content, "4") {
		t.Logf("warning: expected '4' in response, got %q", resp.Content)
	}
}

// TestADKChatModel_Stream verifies streaming generation via the ADK Go adapter.
func TestADKChatModel_Stream(t *testing.T) {
	baseURL, apiKey, modelID, ok := modelEndpointAvailable()
	if !ok {
		t.Skip("model endpoint not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reg := NewDefaultRegistry()
	chatModel, err := reg.Create(ctx, aclllm.ModelConfig{
		Protocol: "openai",
		BaseURL:  baseURL,
		APIKey:   apiKey,
		ModelID:  modelID,
	})
	if err != nil {
		t.Fatalf("Create model: %v", err)
	}

	msgs := []*aclllm.Message{
		aclllm.SystemMessage("You are a helpful assistant."),
		aclllm.UserMessage("Say hello in one sentence."),
	}

	reader, err := chatModel.Stream(ctx, msgs)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	var fullText strings.Builder
	chunkCount := 0
	for {
		chunk, err := reader.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		if chunk != nil && chunk.Content != "" {
			fullText.WriteString(chunk.Content)
			chunkCount++
		}
	}

	if chunkCount == 0 {
		t.Fatalf("expected at least 1 stream chunk, got 0")
	}
	if fullText.Len() == 0 {
		t.Fatalf("expected non-empty streamed text")
	}
	t.Logf("Stream response (%d chunks): %s", chunkCount, fullText.String())
}
