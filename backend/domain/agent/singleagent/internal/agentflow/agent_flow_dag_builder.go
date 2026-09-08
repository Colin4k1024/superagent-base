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
	"context"
	"fmt"

	"github.com/superagent-ai/superagent-base/backend/pkg/dag"
)

// BuildDAGAgent creates a DAGAgentRunner that uses the self-built DAG
// engine for outer orchestration while keeping eino for inner execution.
//
// It reuses BuildAgent to obtain the compiled eino graph (model, tools,
// knowledge, ReAct agent) and wraps it in a minimal dag.Graph structure
// (entry → agent → exit). As the migration progresses, the eino runner
// will be replaced node-by-node with DAG-native executors until the
// dag.Executor can drive the entire flow without eino.
func BuildDAGAgent(ctx context.Context, conf *Config) (*DAGAgentRunner, error) {
	einoRunner, err := BuildAgent(ctx, conf)
	if err != nil {
		return nil, err
	}

	builder := dag.NewGraphBuilder()

	builder.AddNode(&dag.Node{
		Meta:     dag.NodeMeta{Key: "entry", Name: "Entry", Type: "entry"},
		Executor: &passthroughExecutor{},
	})

	// The agent node wraps the compiled eino runner. Its executor is a
	// placeholder for now — the real execution happens inside
	// DAGAgentRunner.StreamExecute via the eino runner. Future steps will
	// replace this with a streaming executor that drives dag.Executor.
	builder.AddNode(&dag.Node{
		Meta:     dag.NodeMeta{Key: "agent", Name: "Agent", Type: "agent"},
		Executor: &passthroughExecutor{},
	})

	builder.AddNode(&dag.Node{
		Meta:     dag.NodeMeta{Key: "exit", Name: "Exit", Type: "exit"},
		Executor: &passthroughExecutor{},
	})

	builder.AddEdge(dag.Edge{From: "entry", To: "agent"})
	builder.AddEdge(dag.Edge{From: "agent", To: "exit"})

	builder.SetEntry("entry")
	builder.SetExit("exit")

	graph, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("dag agent: failed to build graph: %w", err)
	}

	return &DAGAgentRunner{
		einoRunner: einoRunner,
		graph:      graph,
	}, nil
}

// passthroughExecutor returns its input unchanged. It is a placeholder
// for nodes whose real implementation still lives in the eino runner.
type passthroughExecutor struct{}

func (p *passthroughExecutor) Execute(_ context.Context, input map[string]any) (map[string]any, error) {
	return input, nil
}
