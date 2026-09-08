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

// Package compose provides the eino-backed workflow orchestration engine.
//
// This file hosts framework-agnostic facade functions that wrap the remaining
// eino compose/schema touch-points (interrupt detection, streaming pipe, the
// workflow-as-tool runner options, and single-node workflow construction) so
// that the service layer can drive the engine without importing cloudwego/eino
// directly. eino usage stays confined to this package (and the einobridge).
package compose

import (
	"context"

	einoCompose "github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	workflowModel "github.com/superagent-ai/superagent-base/backend/crossdomain/workflow/model"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity/vo"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/internal/execute"
	wfSchema "github.com/superagent-ai/superagent-base/backend/domain/workflow/internal/schema"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
)

// IsInterrupt reports whether err carries an eino interrupt signal. It wraps
// einoCompose.ExtractInterruptInfo for callers that only need the boolean.
func IsInterrupt(err error) bool {
	_, ok := einoCompose.ExtractInterruptInfo(err)
	return ok
}

// NewMessagePipe creates a streaming pipe for workflow message output and
// returns a framework-agnostic reader plus the runner option that feeds the
// write side into the workflow runner. It replaces direct eino schema.Pipe
// usage in the service layer so callers never import eino/schema for streaming.
func NewMessagePipe() (*wfcompose.StreamReader[*entity.Message], WorkflowRunnerOption) {
	sr, sw := schema.Pipe[*entity.Message](10)
	return einobridge.WrapStreamReader[*entity.Message](sr), WithStreamWriter(sw)
}

// WithToolExecuteConfig builds the workflow-as-tool "execute config" runner
// option as a framework-agnostic option. It wraps eino's
// WithToolsNodeOption(WithToolOption(...)) chain used when a workflow is
// invoked as a tool within an agent run.
func WithToolExecuteConfig(cfg workflowModel.ExecuteConfig) wfcompose.Option {
	return einobridge.WrapOption(einoCompose.WithToolsNodeOption(
		einoCompose.WithToolOption(execute.WithExecuteConfig(cfg))))
}

// WithToolResume builds the workflow-as-tool "resume" runner option as a
// framework-agnostic option. It maps each interrupt event to its execution ID
// and delegates to execute.WithResume under eino's tools-node option chain.
func WithToolResume(resumingEvent *entity.ToolInterruptEvent, resumeData string,
	allInterruptEvents map[string]*entity.ToolInterruptEvent) wfcompose.Option {
	toolCallID2ExeID := make(map[string]int64, len(allInterruptEvents))
	for callID, event := range allInterruptEvents {
		toolCallID2ExeID[callID] = event.ExecuteID
	}
	return einobridge.WrapOption(einoCompose.WithToolsNodeOption(
		einoCompose.WithToolOption(
			execute.WithResume(&entity.ResumeRequest{
				ExecuteID:  resumingEvent.ExecuteID,
				EventID:    resumingEvent.ID,
				ResumeData: resumeData,
			}, toolCallID2ExeID))))
}

// WithMessagePipe builds the workflow-as-tool message-pipe option, reader and
// closer as framework-agnostic values. It wraps execute.WithMessagePipe.
func WithMessagePipe() (wfcompose.Option, *wfcompose.StreamReader[*entity.Message], func()) {
	opt, sr, closer := execute.WithMessagePipe()
	return einobridge.WrapOption(opt), einobridge.WrapStreamReader[*entity.Message](sr), closer
}

// NewWorkflowFromNodeNamed builds a single-node workflow (for node-level debug
// execution) and applies the graph name derived from the workflow ID. It wraps
// NewWorkflowFromNode + eino's WithGraphName so callers avoid importing eino.
func NewWorkflowFromNodeNamed(ctx context.Context, sc *wfSchema.WorkflowSchema, nodeKey vo.NodeKey, graphName string) (*Workflow, error) {
	return NewWorkflowFromNode(ctx, sc, nodeKey, einoCompose.WithGraphName(graphName))
}
