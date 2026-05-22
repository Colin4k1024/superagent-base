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
	"fmt"
	"os"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"github.com/superagent-ai/superagent-base/backend/pkg/hiagent"
)

var _ tool.InvokableTool = (*HiAgentRAGTool)(nil)

// HiAgentRAGTool queries HiAgent knowledge base via its RAG API.
type HiAgentRAGTool struct {
	client     *hiagent.Client
	sessionMgr *hiagent.SessionManager
}

// newHiAgentRAGTool returns nil if required env vars are missing (conditional registration).
func newHiAgentRAGTool() *HiAgentRAGTool {
	apiURL := os.Getenv("HIAGENT_API_URL")
	apiKey := os.Getenv("HIAGENT_API_KEY")
	if apiURL == "" || apiKey == "" {
		return nil
	}

	appKey := os.Getenv("HIAGENT_APP_KEY")
	if appKey == "" {
		appKey = apiKey
	}

	cfg := hiagent.Config{
		BaseURL: apiURL,
		APIKey:  apiKey,
		AppKey:  appKey,
	}
	client := hiagent.NewClient(cfg)
	return &HiAgentRAGTool{
		client:     client,
		sessionMgr: hiagent.NewSessionManager(client),
	}
}

func (t *HiAgentRAGTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "hiagent_rag",
		Desc: "Query HiAgent knowledge base for information retrieval. Returns relevant answers from the RAG knowledge base.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Desc:     "The question to search in the knowledge base.",
				Type:     schema.String,
				Required: true,
			},
			"user_id": {
				Desc:     "User identifier for conversation session management. Defaults to 'default'.",
				Type:     schema.String,
				Required: false,
			},
		}),
	}, nil
}

func (t *HiAgentRAGTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args struct {
		Query  string `json:"query"`
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("hiagent_rag: parse arguments: %w", err)
	}
	if args.Query == "" {
		return "", fmt.Errorf("hiagent_rag: query is required")
	}
	if args.UserID == "" {
		args.UserID = "default"
	}

	resp, err := t.queryWithRetry(ctx, args.Query, args.UserID)
	if err != nil {
		return "", err
	}

	result := map[string]any{
		"answer":         resp.Answer,
		"think_messages": resp.ThinkMessages,
		"tool_messages":  resp.ToolMessages,
	}
	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("hiagent_rag: marshal result: %w", err)
	}
	return string(out), nil
}

func (t *HiAgentRAGTool) queryWithRetry(ctx context.Context, query, userID string) (*hiagent.ChatQueryV2Response, error) {
	convID, err := t.sessionMgr.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("hiagent_rag: get conversation: %w", err)
	}

	chatReq := &hiagent.ChatQueryV2Request{
		AppConversationID: convID,
		Query:             query,
		UserID:            userID,
	}

	resp, err := t.client.ChatQueryBlocking(ctx, chatReq)
	if err != nil {
		if hiagent.IsSessionExpired(err) {
			t.sessionMgr.Invalidate(userID)
			convID, err = t.sessionMgr.GetOrCreate(ctx, userID)
			if err != nil {
				return nil, fmt.Errorf("hiagent_rag: recreate conversation: %w", err)
			}
			chatReq.AppConversationID = convID
			resp, err = t.client.ChatQueryBlocking(ctx, chatReq)
			if err != nil {
				return nil, fmt.Errorf("hiagent_rag: query after retry: %w", err)
			}
			return resp, nil
		}
		return nil, fmt.Errorf("hiagent_rag: query: %w", err)
	}
	return resp, nil
}
