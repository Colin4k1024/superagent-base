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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var _ tool.InvokableTool = (*HaierKnowledgeQATool)(nil)

// HaierKnowledgeQATool queries Haier enterprise RAG knowledge base via SSE streaming API.
type HaierKnowledgeQATool struct {
	baseURL     string
	accessToken string
	appToken    string
	kCode       string
	httpClient  *http.Client
}

// newHaierKnowledgeQATool returns nil if required env vars are missing.
func newHaierKnowledgeQATool() *HaierKnowledgeQATool {
	baseURL := os.Getenv("HAIER_RAG_BASE_URL")
	accessToken := os.Getenv("HAIER_RAG_ACCESS_TOKEN")
	appToken := os.Getenv("HAIER_RAG_APP_TOKEN")
	kCode := os.Getenv("HAIER_RAG_K_CODE")
	if baseURL == "" || accessToken == "" || appToken == "" || kCode == "" {
		return nil
	}

	return &HaierKnowledgeQATool{
		baseURL:     strings.TrimRight(baseURL, "/"),
		accessToken: accessToken,
		appToken:    appToken,
		kCode:       kCode,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (t *HaierKnowledgeQATool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "haier_knowledge_qa",
		Desc: "Query Haier enterprise knowledge base for information retrieval. Searches internal documents, policies, technical guides and returns relevant answers with source references.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Desc:     "The question to search in the enterprise knowledge base.",
				Type:     schema.String,
				Required: true,
			},
			"top_k": {
				Desc: "Number of top relevant document chunks to retrieve. Default 8.",
				Type: schema.Integer,
			},
			"score_threshold": {
				Desc: "Minimum relevance score threshold (0-1). Default 0.8.",
				Type: schema.Number,
			},
			"kb_type": {
				Desc: "Knowledge base type: 0=all, 1=unstructured, 2=structured, 3=QA, 4=tabular. Default 0.",
				Type: schema.Integer,
			},
			"model": {
				Desc: "LLM model for answer generation: qwen, DeepSeek-R1, deepseek-v3, Qwen3-32B. Default qwen.",
				Type: schema.String,
			},
			"enable_multi_turn": {
				Desc: "Enable multi-turn conversation mode. Default false.",
				Type: schema.Boolean,
			},
			"session_id": {
				Desc: "Session ID for multi-turn conversation continuity.",
				Type: schema.String,
			},
		}),
	}, nil
}

func (t *HaierKnowledgeQATool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args haierRAGArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("haier_knowledge_qa: parse arguments: %w", err)
	}
	if args.Query == "" {
		return "", fmt.Errorf("haier_knowledge_qa: query is required")
	}

	reqBody := t.buildRequestBody(args)
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("haier_knowledge_qa: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+"/v1/chat/rag_chat", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("haier_knowledge_qa: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Access-Token", t.accessToken)
	req.Header.Set("App-Access-Token", t.appToken)
	req.Header.Set("K-Code", t.kCode)
	req.Header.Set("VISIT-K-Code", t.kCode)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("haier_knowledge_qa: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("haier_knowledge_qa: HTTP %d: %s", resp.StatusCode, string(body))
	}

	result, err := t.consumeSSEStream(resp.Body)
	if err != nil {
		return "", fmt.Errorf("haier_knowledge_qa: read stream: %w", err)
	}

	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("haier_knowledge_qa: marshal result: %w", err)
	}
	return string(out), nil
}

type haierRAGArgs struct {
	Query           string  `json:"query"`
	TopK            int     `json:"top_k,omitempty"`
	ScoreThreshold  float64 `json:"score_threshold,omitempty"`
	KBType          int     `json:"kb_type,omitempty"`
	Model           string  `json:"model,omitempty"`
	EnableMultiTurn bool    `json:"enable_multi_turn,omitempty"`
	SessionID       string  `json:"session_id,omitempty"`
}

type haierRAGRequest struct {
	Query          string              `json:"query"`
	TopK           int                 `json:"top_k"`
	ScoreThreshold float64             `json:"score_threshold"`
	MaxTokens      int                 `json:"max_tokens"`
	Conditions     haierRAGConditions  `json:"conditions"`
	EnableMultiTurn bool              `json:"enable_multi_turn"`
	SessionID      string              `json:"session_id,omitempty"`
}

type haierRAGConditions struct {
	Model       string `json:"model"`
	KBType      int    `json:"kb_type"`
	IsDeepThink bool   `json:"is_deepthink"`
}

type haierRAGResult struct {
	Answer       string              `json:"answer"`
	Sources      []haierRAGSource    `json:"sources,omitempty"`
	IsSufficient bool                `json:"is_sufficient"`
}

type haierRAGSource struct {
	Filename string `json:"filename"`
	KBType   int    `json:"kb_type"`
	FileURL  string `json:"file_url,omitempty"`
}

type haierSSEChunk struct {
	Type   string          `json:"type"`
	Data   haierSSEData    `json:"data"`
	Source []haierRAGSource `json:"source,omitempty"`
}

type haierSSEData struct {
	Content string `json:"content"`
}

func (t *HaierKnowledgeQATool) buildRequestBody(args haierRAGArgs) haierRAGRequest {
	topK := 8
	if args.TopK > 0 {
		topK = args.TopK
	}
	scoreThreshold := 0.8
	if args.ScoreThreshold > 0 {
		scoreThreshold = args.ScoreThreshold
	}
	model := "qwen"
	if args.Model != "" {
		model = args.Model
	}

	return haierRAGRequest{
		Query:          args.Query,
		TopK:           topK,
		ScoreThreshold: scoreThreshold,
		MaxTokens:      4096,
		Conditions: haierRAGConditions{
			Model:       model,
			KBType:      args.KBType,
			IsDeepThink: false,
		},
		EnableMultiTurn: args.EnableMultiTurn,
		SessionID:       args.SessionID,
	}
}

func (t *HaierKnowledgeQATool) consumeSSEStream(body io.Reader) (*haierRAGResult, error) {
	var answer strings.Builder
	var sources []haierRAGSource
	isSufficient := true

	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimSpace(data)
		if data == "" || data == "[DONE]" {
			continue
		}

		var chunk haierSSEChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if chunk.Type == "answer" {
			answer.WriteString(chunk.Data.Content)
		}
		if len(chunk.Source) > 0 {
			sources = chunk.Source
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	answerText := answer.String()
	if strings.HasPrefix(answerText, "~") {
		isSufficient = false
		answerText = strings.TrimPrefix(answerText, "~")
	}

	return &haierRAGResult{
		Answer:       answerText,
		Sources:      sources,
		IsSufficient: isSufficient,
	}, nil
}
