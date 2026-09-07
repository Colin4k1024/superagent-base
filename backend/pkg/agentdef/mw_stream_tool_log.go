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
	"time"

	aclagent "github.com/superagent-ai/superagent-base/backend/pkg/agent"
	"github.com/superagent-ai/superagent-base/backend/pkg/llm"
)

type streamToolLogMiddleware struct {
	aclagent.BaseMiddleware
	logChunks bool
}

func (m *streamToolLogMiddleware) WrapTool(ctx context.Context, tool llm.Tool) (llm.Tool, error) {
	return &loggingToolWrapper{
		inner:     tool,
		logChunks: m.logChunks,
	}, nil
}

type loggingToolWrapper struct {
	inner     llm.Tool
	logChunks bool
}

func (w *loggingToolWrapper) Info(ctx context.Context) (*llm.ToolInfo, error) {
	return w.inner.Info(ctx)
}

func (w *loggingToolWrapper) Run(ctx context.Context, args string, opts ...llm.ToolOption) (string, error) {
	start := time.Now()
	info, _ := w.inner.Info(ctx)
	name := ""
	if info != nil {
		name = info.Name
	}
	log.Printf("[stream_tool_log] tool=%s started", name)
	result, err := w.inner.Run(ctx, args, opts...)
	if err != nil {
		log.Printf("[stream_tool_log] tool=%s error=%v duration=%s", name, err, time.Since(start))
	} else {
		log.Printf("[stream_tool_log] tool=%s completed duration=%s", name, time.Since(start))
	}
	return result, err
}

func buildStreamToolLogHandler(_ context.Context, cfg map[string]any) (aclagent.Middleware, error) {
	m := &streamToolLogMiddleware{}
	if v, ok := cfg["log_chunks"]; ok {
		if b, ok := v.(bool); ok {
			m.logChunks = b
		}
	}
	return m, nil
}

func init() { RegisterMiddleware("stream_tool_log", buildStreamToolLogHandler) }
