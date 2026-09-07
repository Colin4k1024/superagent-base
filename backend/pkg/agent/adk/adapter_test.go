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

	aclagent "github.com/superagent-ai/superagent-base/backend/pkg/agent"
	aclllm "github.com/superagent-ai/superagent-base/backend/pkg/llm"
	adkllm "github.com/superagent-ai/superagent-base/backend/pkg/llm/adk"
)

func agentModelEndpointAvailable() (baseURL, apiKey, modelID string, ok bool) {
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
	return baseURL, apiKey, modelID, resp.StatusCode < 500
}

// TestADKAgentAdapter_Run verifies the agent runtime adapter can execute
// a simple chat conversation through ADK Go's runner.
func TestADKAgentAdapter_Run(t *testing.T) {
	baseURL, apiKey, modelID, ok := agentModelEndpointAvailable()
	if !ok {
		t.Skip("model endpoint not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create ADK Go LLM via the provider registry.
	reg := adkllm.NewDefaultRegistry()
	chatModel, err := reg.Create(ctx, aclllm.ModelConfig{
		Protocol: "openai",
		BaseURL:  baseURL,
		APIKey:   apiKey,
		ModelID:  modelID,
	})
	if err != nil {
		t.Fatalf("Create model: %v", err)
	}

	// Unwrap to get the raw ADK Go model.LLM for the agent adapter.
	adkAdapter, ok := chatModel.(*adkllm.ChatModelAdapter)
	if !ok {
		t.Fatalf("expected *adkllm.ChatModelAdapter, got %T", chatModel)
	}
	adkLLM := adkAdapter.UnwrapADKModel()

	// Create agent runtime adapter (no tools, simple chat).
	agentRT, err := NewAgentAdapter(
		"test-agent",
		"A test assistant",
		adkLLM,
		nil, // no tools
		10,  // max iterations
	)
	if err != nil {
		t.Fatalf("NewAgentAdapter: %v", err)
	}

	// Run a simple conversation.
	input := &aclagent.AgentInput{
		Messages: []*aclllm.Message{
			aclllm.SystemMessage("You are a helpful assistant. Answer concisely."),
			aclllm.UserMessage("What is the capital of France? One word answer."),
		},
		EnableStreaming: true,
	}

	iter, err := agentRT.Run(ctx, input)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	var fullResponse strings.Builder
	eventCount := 0
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		eventCount++
		if event.Err != nil {
			t.Fatalf("event error: %v", event.Err)
		}
		if event.MessageOutput != nil {
			if event.MessageOutput.IsStreaming && event.MessageOutput.MessageStream != nil {
				for {
					chunk, recvErr := event.MessageOutput.MessageStream.Recv()
					if recvErr == io.EOF {
						break
					}
					if recvErr != nil {
						t.Fatalf("stream recv: %v", recvErr)
					}
					if chunk != nil && chunk.Content != "" {
						fullResponse.WriteString(chunk.Content)
					}
				}
			} else if event.MessageOutput.Message != nil && event.MessageOutput.Message.Content != "" {
				fullResponse.WriteString(event.MessageOutput.Message.Content)
			}
		}
	}

	if eventCount == 0 {
		t.Fatalf("expected at least 1 event, got 0")
	}
	if fullResponse.Len() == 0 {
		t.Fatalf("expected non-empty response from agent")
	}
	t.Logf("Agent response (%d events): %s", eventCount, fullResponse.String())
	if !strings.Contains(strings.ToLower(fullResponse.String()), "paris") {
		t.Logf("warning: expected 'Paris' in response, got %q", fullResponse.String())
	}
}
