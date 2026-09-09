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

package dagcompose

import (
	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose/einobridge"
	"context"
	"fmt"


	workflowmodel "github.com/superagent-ai/superagent-base/backend/crossdomain/workflow/model"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity"
	wfschema "github.com/superagent-ai/superagent-base/backend/domain/workflow/internal/schema"
	"github.com/superagent-ai/superagent-base/backend/pkg/dag"
	"github.com/superagent-ai/superagent-base/backend/pkg/logs"
)

// WorkflowRunner mirrors compose.WorkflowRunner but uses the DAG engine.
// It handles execution context, event channels, and interrupt detection.
type WorkflowRunner struct {
	basic     *entity.WorkflowBasic
	input     string
	resumeReq *entity.ResumeRequest
	sw        *einobridge.StreamWriter[*entity.Message]
	schema    *wfschema.WorkflowSchema
	config    workflowmodel.ExecuteConfig

	executeID int64
	cpStore   dag.CheckpointStore
}

// WorkflowRunnerOption configures the WorkflowRunner.
type WorkflowRunnerOption func(*workflowRunnerOptions)

type workflowRunnerOptions struct {
	input     string
	cpStore   dag.CheckpointStore
	resumeReq *entity.ResumeRequest
	sw        *einobridge.StreamWriter[*entity.Message]
}

// WithInput sets the serialized input string.
func WithInput(input string) WorkflowRunnerOption {
	return func(o *workflowRunnerOptions) { o.input = input }
}

// WithResumeReq sets the resume request for interrupt/resume flows.
func WithResumeReq(req *entity.ResumeRequest) WorkflowRunnerOption {
	return func(o *workflowRunnerOptions) { o.resumeReq = req }
}

// WithStreamWriter sets the eino schema StreamWriter for streaming output.
// This is used by NewMessagePipe to feed streaming results to the caller.
func WithStreamWriter(sw *einobridge.StreamWriter[*entity.Message]) WorkflowRunnerOption {
	return func(o *workflowRunnerOptions) { o.sw = sw }
}

// WithCheckpointStore sets the checkpoint store for interrupt/resume.
func WithCheckpointStore(store dag.CheckpointStore) WorkflowRunnerOption {
	return func(o *workflowRunnerOptions) { o.cpStore = store }
}

// NewWorkflowRunner creates a DAG-engine-backed workflow runner.
func NewWorkflowRunner(b *entity.WorkflowBasic, sc *wfschema.WorkflowSchema, config workflowmodel.ExecuteConfig, opts ...WorkflowRunnerOption) *WorkflowRunner {
	options := &workflowRunnerOptions{}
	for _, o := range opts {
		o(options)
	}
	return &WorkflowRunner{
		basic:     b,
		input:     options.input,
		resumeReq: options.resumeReq,
		sw:        options.sw,
		schema:    sc,
		config:    config,
		executeID: 0,
		cpStore:   options.cpStore,
	}
}

// Prepare initializes the execution context. It mirrors the interface
// of compose.WorkflowRunner.Prepare but returns DAG-specific types
// instead of eino compose.Option.
//
// Returns:
// - context.Context: the prepared execution context
// - int64: the execute ID (0 if not persisted)
// - dag.ExecutorOption: DAG-specific execution options
// - <-chan *Event: event channel (currently nil — events handled inline)
// - error: preparation error
func (r *WorkflowRunner) Prepare(ctx context.Context) (context.Context, int64, []dag.ExecutorOption, <-chan *Event, error) {
	// For now, this is a simplified version that doesn't do DB persistence.
	// The full implementation would mirror compose.WorkflowRunner.Prepare
	// with workflow execution records, node status tracking, etc.

	executeID := r.executeID
	if r.resumeReq != nil {
		executeID = r.resumeReq.ExecuteID
	}
	var dagOpts []dag.ExecutorOption

	if r.config.InputFailFast {
		dagOpts = append(dagOpts, dag.WithFailFast())
	}

	// Create event channel (currently unused — events are handled inline).
	eventChan := make(chan *Event, 10)

	if r.resumeReq != nil {
		logs.CtxInfof(ctx, "dagcompose: resuming execution %d with event %d",
			r.resumeReq.ExecuteID, r.resumeReq.EventID)
	}

	return ctx, executeID, dagOpts, eventChan, nil
}

// StreamWriter returns the stream writer, if set.
func (r *WorkflowRunner) StreamWriter() *einobridge.StreamWriter[*entity.Message] {
	return r.sw
}

// Event is a simplified workflow execution event.
type Event struct {
	Type    EventType
	NodeKey string
	Data    map[string]any
	Err     error
}

// EventType enumerates workflow execution event types.
type EventType int

const (
	EventTypeNodeStart EventType = iota
	EventTypeNodeEnd
	EventTypeNodeError
	EventTypeWorkflowComplete
)

// IsInterrupt checks if an error is a workflow interrupt (not a real error).
// This mirrors compose.IsInterrupt but uses the DAG engine's interrupt type.
func IsInterrupt(err error) bool {
	if err == nil {
		return false
	}
	// Check for DAG interrupt error type.
	_, ok := err.(*InterruptError)
	return ok
}

// InterruptError represents a workflow interrupt (e.g. user input needed).
type InterruptError struct {
	NodeKey string
	Reason  string
	Data    map[string]any
}

func (e *InterruptError) Error() string {
	return fmt.Sprintf("dagcompose: interrupt at node %q: %s", e.NodeKey, e.Reason)
}

// Compile-time check.
var _ error = (*InterruptError)(nil)

// ResumeOption carries data needed to resume an interrupted workflow.
type ResumeOption struct {
	ExecuteID    int64
	CheckpointID string
	ResumeData   map[string]any
}

// PrepareResume creates resume options from a ResumeRequest.
func PrepareResume(req *entity.ResumeRequest) *ResumeOption {
	if req == nil {
		return nil
	}
	return &ResumeOption{
		ExecuteID:    req.ExecuteID,
		CheckpointID: fmt.Sprintf("%d", req.EventID),
	}
}
