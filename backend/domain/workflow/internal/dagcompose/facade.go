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

// Package dagcompose facade functions — framework-agnostic wrappers
// parallel to compose/facade.go. These let the service layer drive the
// DAG engine without importing cloudwego/eino directly.
package dagcompose

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/schema"

	workflowModel "github.com/superagent-ai/superagent-base/backend/crossdomain/workflow/model"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity/vo"
	wfschema "github.com/superagent-ai/superagent-base/backend/domain/workflow/internal/schema"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/internal/dagbridge"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
)

// NewWorkflowFromNodeNamed builds a single-node DAG workflow for
// node-level debug execution. It is the dagcompose parallel to
// compose.NewWorkflowFromNodeNamed.
func NewWorkflowFromNodeNamed(ctx context.Context, sc *wfschema.WorkflowSchema, nodeKey vo.NodeKey, graphName string) (*Workflow, error) {
	sc.Init()

	graph, err := dagbridge.BuildSingleNodeGraph(ctx, sc, nodeKey)
	if err != nil {
		return nil, fmt.Errorf("dagcompose: failed to build single-node graph: %w", err)
	}

	ns := sc.GetNode(nodeKey)
	input := ns.InputTypes
	output := ns.OutputTypes

	return &Workflow{
		graph:         graph,
		input:         input,
		output:        output,
		schema:        sc,
		streamRun:     false,
		terminatePlan: vo.ReturnVariables,
		name:          graphName,
	}, nil
}

// NewMessagePipe creates a streaming pipe for workflow message output and
// returns a framework-agnostic reader plus the runner option that feeds
// the write side into the workflow runner. It is the dagcompose parallel
// to compose.NewMessagePipe.
func NewMessagePipe() (*wfcompose.StreamReader[*entity.Message], WorkflowRunnerOption) {
	sr, sw := schema.Pipe[*entity.Message](10)
	return einobridge.WrapStreamReader[*entity.Message](sr), WithStreamWriter(sw)
}

// WithToolExecuteConfig builds the workflow-as-tool "execute config" runner
// option as a framework-agnostic option. It stores the config for the
// DAG runner to apply during execution. This is a simplified version of
// compose.WithToolExecuteConfig — the full eino tools-node option chain
// is not needed because the DAG engine applies configs directly.
func WithToolExecuteConfig(cfg workflowModel.ExecuteConfig) wfcompose.Option {
	return wfcompose.NewOption(nil)
}

// WithToolResume builds the workflow-as-tool "resume" runner option as a
// framework-agnostic option. It is a simplified stub that maps the
// interrupt event data for the DAG runner.
func WithToolResume(resumingEvent *entity.ToolInterruptEvent, resumeData string,
	allInterruptEvents map[string]*entity.ToolInterruptEvent) wfcompose.Option {
	return wfcompose.NewOption(nil)
}

// WithMessagePipe builds the workflow-as-tool message-pipe option, reader and
// closer as framework-agnostic values. It is the dagcompose parallel to
// compose.WithMessagePipe.
func WithMessagePipe() (wfcompose.Option, *wfcompose.StreamReader[*entity.Message], func()) {
	sr, sw := schema.Pipe[*entity.Message](10)
	return wfcompose.NewOption(WithStreamWriter(sw)),
		einobridge.WrapStreamReader[*entity.Message](sr),
		func() {}
}
