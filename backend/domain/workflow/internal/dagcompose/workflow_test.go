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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity"
	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity/vo"
	wfschema "github.com/superagent-ai/superagent-base/backend/domain/workflow/internal/schema"
)

// upperCaseConfig builds a node that uppercases text.
type upperCaseConfig struct{}

func (c *upperCaseConfig) Build(_ context.Context, _ *wfschema.NodeSchema, _ ...wfschema.BuildOption) (any, error) {
	return &upperNode{}, nil
}

type upperNode struct{}

func (n *upperNode) Invoke(_ context.Context, input map[string]any) (map[string]any, error) {
	t, _ := input["text"].(string)
	return map[string]any{"text": strings.ToUpper(t)}, nil
}

// prefixConfig builds a node that adds a prefix.
type prefixConfig struct{}

func (c *prefixConfig) Build(_ context.Context, _ *wfschema.NodeSchema, _ ...wfschema.BuildOption) (any, error) {
	return &prefixNode{}, nil
}

type prefixNode struct{}

func (n *prefixNode) Invoke(_ context.Context, input map[string]any) (map[string]any, error) {
	t, _ := input["text"].(string)
	return map[string]any{"text": "PREFIX:" + t}, nil
}

func buildTestSchema() *wfschema.WorkflowSchema {
	return &wfschema.WorkflowSchema{
		Nodes: []*wfschema.NodeSchema{
			{Key: "entry", Name: "Entry", Type: entity.NodeTypeEntry},
			{Key: "upper", Name: "Upper", Type: entity.NodeTypeTextProcessor, Configs: &upperCaseConfig{}},
			{Key: "prefix", Name: "Prefix", Type: entity.NodeTypeTextProcessor, Configs: &prefixConfig{}},
			{Key: "exit", Name: "Exit", Type: entity.NodeTypeExit},
		},
		Connections: []*wfschema.Connection{
			{FromNode: "entry", ToNode: "upper"},
			{FromNode: "upper", ToNode: "prefix"},
			{FromNode: "prefix", ToNode: "exit"},
		},
	}
}

func TestNewWorkflow(t *testing.T) {
	ctx := context.Background()
	wf, err := NewWorkflow(ctx, buildTestSchema(), WithIDAsName(123))
	require.NoError(t, err)
	require.NotNil(t, wf)
	assert.Equal(t, vo.ReturnVariables, wf.TerminatePlan())
}

func TestSyncRun(t *testing.T) {
	ctx := context.Background()
	wf, err := NewWorkflow(ctx, buildTestSchema())
	require.NoError(t, err)

	out, err := wf.SyncRun(ctx, map[string]any{"text": "hello"})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "PREFIX:HELLO", out["text"])
}

func TestInterruptError(t *testing.T) {
	err := &InterruptError{NodeKey: "test", Reason: "need user input"}
	assert.True(t, IsInterrupt(err))
	assert.False(t, IsInterrupt(nil))
	assert.False(t, IsInterrupt(errNotInterrupt))
}

type errString string

func (e errString) Error() string { return string(e) }

var errNotInterrupt error = errString("not interrupt")

func TestPrepareResume(t *testing.T) {
	req := &entity.ResumeRequest{ExecuteID: 1, EventID: 42}
	opt := PrepareResume(req)
	require.NotNil(t, opt)
	assert.Equal(t, int64(1), opt.ExecuteID)
	assert.Equal(t, "42", opt.CheckpointID)

	assert.Nil(t, PrepareResume(nil))
}
