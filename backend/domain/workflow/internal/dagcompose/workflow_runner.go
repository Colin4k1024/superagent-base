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
	"context"
	"fmt"

	workflowmodel "github.com/superagent-ai/superagent-base/backend/crossdomain/workflow/model"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity"
	wfschema "github.com/superagent-ai/superagent-base/backend/domain/workflow/internal/schema"
	"github.com/superagent-ai/superagent-base/backend/pkg/dag"
)

// WorkflowRunner mirrors compose.WorkflowRunner but uses the DAG engine.
// It handles execution context, event channels, and interrupt detection.
type WorkflowRunner struct {
	basic  *entity.WorkflowBasic
	input  string
	schema *wfschema.WorkflowSchema
	config workflowmodel.ExecuteConfig

	executeID int64
	cpStore   dag.CheckpointStore
}

// WorkflowRunnerOption configures the WorkflowRunner.
type WorkflowRunnerOption func(*workflowRunnerOptions)

type workflowRunnerOptions struct {
	input   string
	cpStore dag.CheckpointStore
}

// WithInput sets the serialized input string.
func WithInput(input string) WorkflowRunnerOption {
	return func(o *workflowRunnerOptions) { o.input = input }
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
	var dagOpts []dag.ExecutorOption

	if r.config.InputFailFast {
		dagOpts = append(dagOpts, dag.WithFailFast())
	}

	// Create event channel (currently unused — events are handled inline).
	eventChan := make(chan *Event, 10)

	return ctx, executeID, dagOpts, eventChan, nil
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
