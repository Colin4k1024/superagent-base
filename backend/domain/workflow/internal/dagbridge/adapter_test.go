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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superagent-ai/superagent-base/backend/domain/workflow/entity/vo"
)

// mockInvokable implements nodes.InvokableNode.
type mockInvokable struct {
	result map[string]any
}

func (m *mockInvokable) Invoke(_ context.Context, input map[string]any) (map[string]any, error) {
	out := make(map[string]any)
	for k, v := range m.result {
		out[k] = v
	}
	out["echo"] = input
	return out, nil
}

func TestWrapInvokableNode(t *testing.T) {
	node := &mockInvokable{result: map[string]any{"key": "value"}}
	exec := Wrap(node)
	require.NotNil(t, exec)

	out, err := exec.Execute(context.Background(), map[string]any{"input": "test"})
	require.NoError(t, err)
	assert.Equal(t, "value", out["key"])
	assert.Equal(t, "test", out["echo"].(map[string]any)["input"])
}

func TestNodeRunnerTimeout(t *testing.T) {
	exec := Wrap(&mockInvokable{result: map[string]any{"ok": true}})
	runner := NewNodeRunner(exec, WithTimeout(1)) // 1ms timeout

	out, err := runner.Execute(context.Background(), map[string]any{"x": 1})
	// The mock returns instantly, so this should succeed.
	require.NoError(t, err)
	assert.Equal(t, true, out["ok"])
}

func TestNodeRunnerRetry(t *testing.T) {
	// A node that always fails.
	failingExec := &failingNode{}
	runner := NewNodeRunner(failingExec, WithMaxRetry(2), WithTimeout(0))

	_, err := runner.Execute(context.Background(), nil)
	assert.Error(t, err)
}

func TestNodeRunnerDataOnErr(t *testing.T) {
	failingExec := &failingNode{}
	runner := NewNodeRunner(failingExec,
		WithMaxRetry(0),
		WithTimeout(0),
		WithErrorProcessType(vo.ErrorProcessTypeReturnDefaultData),
		WithDataOnErr(func(_ context.Context) map[string]any {
			return map[string]any{"fallback": true}
		}),
	)

	out, err := runner.Execute(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, true, out["fallback"])
}

func TestWrapRawDagExecutor(t *testing.T) {
	raw := &passthroughNode{}
	exec := Wrap(raw)
	require.NotNil(t, exec)

	out, err := exec.Execute(context.Background(), map[string]any{"k": "v"})
	require.NoError(t, err)
	assert.Equal(t, "v", out["k"])
}

func TestPassthroughNode(t *testing.T) {
	p := &passthroughNode{}
	out, err := p.Execute(context.Background(), map[string]any{"x": 1})
	require.NoError(t, err)
	assert.Equal(t, 1, out["x"])
}

// failingNode always returns an error.
type failingNode struct{}

func (f *failingNode) Execute(_ context.Context, _ map[string]any) (map[string]any, error) {
	return nil, assertError("always fails")
}

// assertError is a simple error type for testing.
type assertError string

func (e assertError) Error() string { return string(e) }
