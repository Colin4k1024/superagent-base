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

	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity/vo"
	wfschema "github.com/superagent-ai/superagent-base/backend/domain/workflow/internal/schema"
	"github.com/superagent-ai/superagent-base/backend/pkg/dag"
)

// BuildGraph converts a WorkflowSchema into a dag.Graph. Each node in
// the schema is instantiated via its NodeBuilder.Configs and wrapped as
// a dag.NodeExecutor. Edges are created from the schema's Connections.
//
// This is the entry point for using the self-built DAG engine with
// existing workflow definitions. It does NOT replace eino/compose yet —
// it provides a parallel path that can be used alongside the current
// engine for testing and incremental migration.
func BuildGraph(ctx context.Context, sc *wfschema.WorkflowSchema) (*dag.Graph, error) {
	if sc == nil {
		return nil, fmt.Errorf("dagbridge: workflow schema is nil")
	}
	sc.Init()

	builder := dag.NewGraphBuilder()

	// Build nodes.
	for _, ns := range sc.Nodes {
		exec, err := buildNodeExecutor(ctx, ns, sc)
		if err != nil {
			return nil, fmt.Errorf("dagbridge: failed to build node %q: %w", ns.Name, err)
		}
		if exec == nil {
			// Skip nodes that don't produce an executor (e.g. structural nodes).
			continue
		}

		builder.AddNode(&dag.Node{
			Meta: dag.NodeMeta{
				Key:         dag.NodeKey(ns.Key),
				Name:        ns.Name,
				Type:        string(ns.Type),
				InputTypes:  convertTypes(ns.InputTypes),
				OutputTypes: convertTypes(ns.OutputTypes),
			},
			Executor: exec,
		})
	}

	// Build edges from connections.
	for _, conn := range sc.Connections {
		edge := dag.Edge{
			From: dag.NodeKey(conn.FromNode),
			To:   dag.NodeKey(conn.ToNode),
		}
		if conn.FromPort != nil && *conn.FromPort != "" {
			edge.Port = dag.PortLabel(*conn.FromPort)
		}
		builder.AddEdge(edge)
	}

	// Set entry/exit nodes from the schema.
	entryKey, exitKey := findEntryExit(sc)
	builder.SetEntry(dag.NodeKey(entryKey))
	if exitKey != "" {
		builder.SetExit(dag.NodeKey(exitKey))
	}

	return builder.Build()
}

// buildNodeExecutor instantiates the node implementation from its
// NodeSchema.Configs (which implements NodeBuilder) and wraps it via
// Wrap(). Returns nil for structural nodes (entry, exit, etc.) that
// are handled by the graph itself.
func buildNodeExecutor(ctx context.Context, ns *wfschema.NodeSchema, sc *wfschema.WorkflowSchema) (dag.NodeExecutor, error) {
	// Entry and Exit are structural — they just pass through.
	if ns.Type == entity.NodeTypeEntry || ns.Type == entity.NodeTypeExit {
		return &passthroughNode{}, nil
	}

	// If the NodeSchema's Configs implements NodeBuilder, build the node.
	nb, ok := ns.Configs.(wfschema.NodeBuilder)
	if !ok {
		return nil, fmt.Errorf("dagbridge: node %q configs does not implement NodeBuilder", ns.Name)
	}

	opts := []wfschema.BuildOption{
		wfschema.WithWorkflowSchema(sc),
	}

	executable, err := nb.Build(ctx, ns, opts...)
	if err != nil {
		return nil, err
	}

	wrapped := Wrap(executable)
	if wrapped == nil {
		return nil, fmt.Errorf("dagbridge: node %q does not implement any known node interface", ns.Name)
	}

	// Wrap with NodeRunner for timeout/retry/error handling.
	runnerOpts := []RunnerOption{}
	if ns.ExceptionConfigs != nil {
		if ns.ExceptionConfigs.TimeoutMS > 0 {
			runnerOpts = append(runnerOpts, WithTimeout(ns.ExceptionConfigs.TimeoutMS))
		}
		runnerOpts = append(runnerOpts, WithMaxRetry(ns.ExceptionConfigs.MaxRetry))
		if ns.ExceptionConfigs.ProcessType != nil {
			runnerOpts = append(runnerOpts, WithErrorProcessType(*ns.ExceptionConfigs.ProcessType))
		}
	}

	return NewNodeRunner(wrapped, runnerOpts...), nil
}

// findEntryExit scans the schema for entry and exit node keys.
func findEntryExit(sc *wfschema.WorkflowSchema) (entry, exit vo.NodeKey) {
	for _, ns := range sc.Nodes {
		if ns.Type == entity.NodeTypeEntry {
			entry = ns.Key
		}
		if ns.Type == entity.NodeTypeExit {
			exit = ns.Key
		}
	}
	return entry, exit
}

// convertTypes converts vo.TypeInfo to dag.TypeInfo.
func convertTypes(in map[string]*vo.TypeInfo) map[string]*dag.TypeInfo {
	if in == nil {
		return nil
	}
	out := make(map[string]*dag.TypeInfo, len(in))
	for k, v := range in {
		out[k] = convertTypeInfo(v)
	}
	return out
}

func convertTypeInfo(v *vo.TypeInfo) *dag.TypeInfo {
	if v == nil {
		return nil
	}
	return &dag.TypeInfo{
		Kind:       string(v.Type),
		Required:   v.Required,
		ItemType:   convertTypeInfo(v.ElemTypeInfo),
		Properties: convertTypes(v.Properties),
	}
}

// passthroughNode just returns its input unchanged. Used for entry/exit.
type passthroughNode struct{}

func (p *passthroughNode) Execute(_ context.Context, input map[string]any) (map[string]any, error) {
	return input, nil
}
