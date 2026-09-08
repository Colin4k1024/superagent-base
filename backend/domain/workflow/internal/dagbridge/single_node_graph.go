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

package dagbridge

import (
	"context"
	"fmt"

	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity/vo"
	wfschema "github.com/superagent-ai/superagent-base/backend/domain/workflow/internal/schema"
	"github.com/superagent-ai/superagent-base/backend/pkg/dag"
)

// BuildSingleNodeGraph creates a dag.Graph that runs only the specified
// node. The graph topology is: entry → node → exit, where entry and
// exit are passthrough nodes and the target node is built from its
// NodeSchema.Configs (same wrapping logic as BuildGraph).
//
// This is the dagcompose equivalent of compose.NewWorkflowFromNode,
// used for single-node debug execution.
func BuildSingleNodeGraph(ctx context.Context, sc *wfschema.WorkflowSchema, nodeKey vo.NodeKey) (*dag.Graph, error) {
	if sc == nil {
		return nil, fmt.Errorf("dagbridge: workflow schema is nil")
	}
	sc.Init()

	ns := sc.GetNode(nodeKey)
	if ns == nil {
		return nil, fmt.Errorf("dagbridge: node %q not found in schema", nodeKey)
	}

	builder := dag.NewGraphBuilder()

	// Entry node — passes input through.
	entryKey := dag.NodeKey("__single_entry")
	builder.AddNode(&dag.Node{
		Meta:     dag.NodeMeta{Key: entryKey, Name: "Entry", Type: "entry"},
		Executor: &passthroughNode{},
	})

	// Target node — built from its NodeSchema.
	exec, err := buildNodeExecutor(ctx, ns, sc)
	if err != nil {
		return nil, fmt.Errorf("dagbridge: failed to build node %q: %w", ns.Name, err)
	}
	if exec == nil {
		return nil, fmt.Errorf("dagbridge: node %q produced nil executor", ns.Name)
	}

	targetKey := dag.NodeKey(ns.Key)
	builder.AddNode(&dag.Node{
		Meta: dag.NodeMeta{
			Key:         targetKey,
			Name:        ns.Name,
			Type:        string(ns.Type),
			InputTypes:  convertTypes(ns.InputTypes),
			OutputTypes: convertTypes(ns.OutputTypes),
		},
		Executor: exec,
	})

	// Exit node — passes output through.
	exitKey := dag.NodeKey("__single_exit")
	builder.AddNode(&dag.Node{
		Meta:     dag.NodeMeta{Key: exitKey, Name: "Exit", Type: "exit"},
		Executor: &passthroughNode{},
	})

	builder.AddEdge(dag.Edge{From: entryKey, To: targetKey})
	builder.AddEdge(dag.Edge{From: targetKey, To: exitKey})

	builder.SetEntry(entryKey)
	builder.SetExit(exitKey)

	return builder.Build()
}
