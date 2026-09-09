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

package agentflow

import (
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
	"context"


	"github.com/superagent-ai/superagent-base/backend/domain/agent/singleagent/entity"
	"github.com/superagent-ai/superagent-base/backend/pkg/dag"
	"github.com/superagent-ai/superagent-base/backend/pkg/logs"
)

// DAGAgentRunner wraps the eino AgentRunner behind a DAG-engine facade.
//
// During the incremental migration from eino/compose to the self-built
// DAG engine, the inner execution (ReAct loop, LLM calls, streaming
// callbacks, interrupt/resume) stays in eino while the outer orchestration
// layer progressively moves to dag.Graph.
//
// StreamExecute currently delegates to the eino runner so that streaming
// behaviour remains unchanged. Subsequent steps will replace the
// delegation with a native dag.Executor driven by the graph built in
// BuildDAGAgent, replacing eino callbacks with DAG stream handlers.
type DAGAgentRunner struct {
	einoRunner *AgentRunner
	graph      *dag.Graph
}

// StreamExecute runs the agent flow. During the incremental migration
// it delegates to the eino runner's StreamExecute so that streaming,
// callbacks, and interrupt/resume behaviour remain unchanged.
func (r *DAGAgentRunner) StreamExecute(ctx context.Context, req *AgentRequest) (
	*wfcompose.StreamReader[*entity.AgentEvent], error,
) {
	logs.CtxInfof(ctx, "[DAGAgentRunner] StreamExecute delegating to eino runner (incremental)")
	return r.einoRunner.StreamExecute(ctx, req)
}

// PreHandlerReq applies the same input preprocessing as AgentRunner.
func (r *DAGAgentRunner) PreHandlerReq(ctx context.Context, req *AgentRequest) *AgentRequest {
	return r.einoRunner.PreHandlerReq(ctx, req)
}

// Runner is the common interface implemented by AgentRunner and
// DAGAgentRunner. The single agent service uses it to switch between
// the eino and DAG-engine implementations behind a feature flag.
type Runner interface {
	StreamExecute(ctx context.Context, req *AgentRequest) (*wfcompose.StreamReader[*entity.AgentEvent], error)
	PreHandlerReq(ctx context.Context, req *AgentRequest) *AgentRequest
}
