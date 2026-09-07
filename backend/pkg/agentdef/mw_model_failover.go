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
	"log"

	aclagent "github.com/superagent-ai/superagent-base/backend/pkg/agent"
	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

type modelFailoverMiddleware struct {
	aclagent.BaseMiddleware
	fallbackModel string
	maxRetries    int
}

func (m *modelFailoverMiddleware) WrapModel(ctx context.Context, base llm.ChatModel) (llm.ChatModel, error) {
	return &failoverChatModel{
		primary:    base,
		maxRetries: m.maxRetries,
		modelName:  m.fallbackModel,
	}, nil
}

type failoverChatModel struct {
	primary    llm.ChatModel
	maxRetries int
	modelName  string
}

func (w *failoverChatModel) Generate(ctx context.Context, msgs []*llm.Message) (*llm.Message, error) {
	var lastErr error
	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		result, err := w.primary.Generate(ctx, msgs)
		if err == nil {
			return result, nil
		}
		lastErr = err
		log.Printf("[model_failover] attempt %d/%d failed: %v", attempt+1, w.maxRetries+1, err)
	}
	return nil, lastErr
}

func (w *failoverChatModel) Stream(ctx context.Context, msgs []*llm.Message) (*llm.StreamReader, error) {
	var lastErr error
	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		result, err := w.primary.Stream(ctx, msgs)
		if err == nil {
			return result, nil
		}
		lastErr = err
		log.Printf("[model_failover] stream attempt %d/%d failed: %v", attempt+1, w.maxRetries+1, err)
	}
	return nil, lastErr
}

func buildModelFailoverHandler(_ context.Context, cfg map[string]any) (aclagent.Middleware, error) {
	m := &modelFailoverMiddleware{maxRetries: 1}
	if v, ok := cfg["fallback_model"]; ok {
		if s, ok := v.(string); ok {
			m.fallbackModel = s
		}
	}
	if v, ok := cfg["max_retries"]; ok {
		m.maxRetries = toInt(v)
	}
	return m, nil
}

func init() { RegisterMiddleware("model_failover", buildModelFailoverHandler) }

func (w *failoverChatModel) ID() string { return w.modelName }

func (w *failoverChatModel) BindTools(tools []llm.Tool) (llm.ChatModel, error) {
	bound, err := w.primary.BindTools(tools)
	if err != nil {
		return nil, err
	}
	return &failoverChatModel{
		primary:    bound,
		maxRetries: w.maxRetries,
		modelName:  w.modelName,
	}, nil
}
