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

// Package dagcompose provides a DAG-engine-based implementation of the
// workflow interface, parallel to domain/workflow/internal/compose.
//
// It uses pkg/dag for graph execution and dagbridge for node adaptation,
// providing a migration path off cloudwego/eino/compose.
package dagcompose

import (
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
	"context"
	"fmt"
	"sync"


	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity/vo"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/internal/dagbridge"
	wfschema "github.com/superagent-ai/superagent-base/backend/domain/workflow/internal/schema"
	"github.com/superagent-ai/superagent-base/backend/pkg/dag"
	"github.com/superagent-ai/superagent-base/backend/pkg/logs"
)

// Workflow mirrors the compose.Workflow interface but uses the self-built
// DAG engine internally. It is a drop-in alternative for simple workflows
// that don't require eino-specific features (checkpoint, complex streaming).
type Workflow struct {
	graph         *dag.Graph
	input         map[string]*vo.TypeInfo
	output        map[string]*vo.TypeInfo
	schema        *wfschema.WorkflowSchema
	streamRun     bool
	terminatePlan vo.TerminatePlan
	mu            sync.Mutex
	name          string
}

// WorkflowOption configures the Workflow.
type WorkflowOption func(*workflowOptions)

type workflowOptions struct {
	wfID              int64
	idAsName          bool
	maxNodeCount      int
	requireCheckpoint bool
}

// WithIDAsName sets the workflow ID for naming.
func WithIDAsName(id int64) WorkflowOption {
	return func(o *workflowOptions) { o.wfID = id; o.idAsName = true }
}

// WithMaxNodeCount sets the max node count per workflow.
func WithMaxNodeCount(c int) WorkflowOption {
	return func(o *workflowOptions) { o.maxNodeCount = c }
}

// WithParentRequireCheckpoint marks that checkpoint is required.
func WithParentRequireCheckpoint() WorkflowOption {
	return func(o *workflowOptions) { o.requireCheckpoint = true }
}

// NewWorkflow builds a DAG-engine-backed workflow from a WorkflowSchema.
// This is the parallel path to compose.NewWorkflow.
func NewWorkflow(ctx context.Context, sc *wfschema.WorkflowSchema, opts ...WorkflowOption) (*Workflow, error) {
	options := &workflowOptions{}
	for _, o := range opts {
		o(options)
	}

	sc.Init()

	graph, err := dagbridge.BuildGraph(ctx, sc)
	if err != nil {
		return nil, fmt.Errorf("dagcompose: failed to build graph: %w", err)
	}

	// Determine input/output types from entry/exit nodes.
	input := collectInputTypes(sc)
	output := collectOutputTypes(sc)

	// Determine if any node requires streaming.
	streamRun := false
	for _, ns := range sc.Nodes {
		if ns.StreamConfigs != nil && ns.StreamConfigs.CanGeneratesStream {
			streamRun = true
			break
		}
	}

	return &Workflow{
		graph:         graph,
		input:         input,
		output:        output,
		schema:        sc,
		streamRun:     streamRun,
		terminatePlan: vo.ReturnVariables,
	}, nil
}

// SyncRun executes the workflow synchronously.
// opts are currently ignored (compose.Option compatibility shim);
// DAG-specific options are configured via the Executor config.
func (w *Workflow) SyncRun(ctx context.Context, input map[string]any, _ ...any) (map[string]any, error) {
	ex := dag.NewExecutor(w.graph, dag.WithFailFast())
	result, err := ex.Execute(ctx, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.Output, nil
}

// AsyncRun executes the workflow asynchronously.
func (w *Workflow) AsyncRun(ctx context.Context, input map[string]any, _ ...any) {
	go func() {
		_, err := w.SyncRun(ctx, input)
		if err != nil {
			logs.CtxErrorf(ctx, "dagcompose: AsyncRun error: %v", err)
		}
	}()
}

// StreamRun executes the workflow with streaming output via the given
// StreamWriter. Each node's output is sent as a Message to the stream.
func (w *Workflow) StreamRun(ctx context.Context, input map[string]any, sw *einobridge.StreamWriter[*entity.Message]) (map[string]any, error) {
	ex := dag.NewExecutor(w.graph,
		dag.WithFailFast(),
		dag.WithStreamHandler(func(key dag.NodeKey, data map[string]any) {
			if sw != nil && data != nil {
				sw.Send(&entity.Message{
					DataMessage: &entity.DataMessage{
						NodeType: entity.NodeTypeOutputEmitter,
						Content:  extractContent(data),
					},
				}, nil)
			}
		}),
	)
	result, err := ex.Execute(ctx, input)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.Output, nil
}

// Inputs returns the input type information.
func (w *Workflow) Inputs() map[string]*vo.TypeInfo { return w.input }

// Outputs returns the output type information.
func (w *Workflow) Outputs() map[string]*vo.TypeInfo { return w.output }

// StreamRunEnabled returns whether streaming is enabled.
func (w *Workflow) StreamRunEnabled() bool { return w.streamRun }

// TerminatePlan returns the workflow's termination plan.
func (w *Workflow) TerminatePlan() vo.TerminatePlan { return w.terminatePlan }

// Schema returns the underlying WorkflowSchema.
func (w *Workflow) Schema() *wfschema.WorkflowSchema { return w.schema }

// collectInputTypes gathers input types from the entry node.
func collectInputTypes(sc *wfschema.WorkflowSchema) map[string]*vo.TypeInfo {
	for _, ns := range sc.Nodes {
		if ns.Type == entity.NodeTypeEntry {
			return ns.InputTypes
		}
	}
	return nil
}

// collectOutputTypes gathers output types from the exit node.
func collectOutputTypes(sc *wfschema.WorkflowSchema) map[string]*vo.TypeInfo {
	for _, ns := range sc.Nodes {
		if ns.Type == entity.NodeTypeExit {
			return ns.OutputTypes
		}
	}
	return nil
}

// extractContent pulls the primary content string from a node output map.
func extractContent(data map[string]any) string {
	if data == nil {
		return ""
	}
	if content, ok := data["content"].(string); ok {
		return content
	}
	if text, ok := data["text"].(string); ok {
		return text
	}
	return ""
}
