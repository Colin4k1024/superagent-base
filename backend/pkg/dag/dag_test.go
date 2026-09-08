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

package dag

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// funcNode wraps a function as a NodeExecutor.
type funcNode func(ctx context.Context, input map[string]any) (map[string]any, error)

func (f funcNode) Execute(ctx context.Context, input map[string]any) (map[string]any, error) {
	return f(ctx, input)
}

func TestLinearGraph(t *testing.T) {
	g, err := NewGraphBuilder().
		AddNode(&Node{Meta: NodeMeta{Key: "entry", Name: "Entry", Type: "entry"}, Executor: funcNode(func(_ context.Context, input map[string]any) (map[string]any, error) {
			return input, nil
		})}).
		AddNode(&Node{Meta: NodeMeta{Key: "mid", Name: "Middle", Type: "process"}, Executor: funcNode(func(_ context.Context, input map[string]any) (map[string]any, error) {
			return map[string]any{"processed": input["data"]}, nil
		})}).
		AddNode(&Node{Meta: NodeMeta{Key: "exit", Name: "Exit", Type: "exit"}, Executor: funcNode(func(_ context.Context, input map[string]any) (map[string]any, error) {
			return map[string]any{"result": input["processed"]}, nil
		})}).
		AddEdge(Edge{From: "entry", To: "mid", FieldMapping: map[string]string{"data": "data"}}).
		AddEdge(Edge{From: "mid", To: "exit"}).
		SetEntry("entry").
		SetExit("exit").
		Build()
	require.NoError(t, err)

	ex := NewExecutor(g)
	result, err := ex.Execute(context.Background(), map[string]any{"data": "hello"})
	require.NoError(t, err)
	assert.Equal(t, "hello", result.Output["result"])
}

func TestConditionalBranch(t *testing.T) {
	executed := make(map[string]bool)
	var mu sync.Mutex

	mkNode := func(key string) *Node {
		return &Node{
			Meta: NodeMeta{Key: NodeKey(key), Name: key, Type: "process"},
			Executor: funcNode(func(_ context.Context, input map[string]any) (map[string]any, error) {
				mu.Lock()
				executed[key] = true
				mu.Unlock()
				return map[string]any{"out": key}, nil
			}),
		}
	}

	g, err := NewGraphBuilder().
		AddNode(&Node{Meta: NodeMeta{Key: "entry", Name: "Entry", Type: "entry"}, Executor: funcNode(func(_ context.Context, input map[string]any) (map[string]any, error) {
			return map[string]any{"_selected_port": "A"}, nil
		})}).
		AddNode(mkNode("branchA")).
		AddNode(mkNode("branchB")).
		AddNode(&Node{Meta: NodeMeta{Key: "exit", Name: "Exit", Type: "exit"}, Executor: funcNode(func(_ context.Context, input map[string]any) (map[string]any, error) {
			return input, nil
		})}).
		AddEdge(Edge{From: "entry", To: "branchA", Port: "A"}).
		AddEdge(Edge{From: "entry", To: "branchB", Port: "B"}).
		AddEdge(Edge{From: "branchA", To: "exit"}).
		AddEdge(Edge{From: "branchB", To: "exit"}).
		SetEntry("entry").
		SetExit("exit").
		Build()
	require.NoError(t, err)

	ex := NewExecutor(g)
	result, err := ex.Execute(context.Background(), map[string]any{})
	require.NoError(t, err)
	assert.True(t, executed["branchA"])
	assert.False(t, executed["branchB"])
	_ = result
}

func TestCycleDetection(t *testing.T) {
	_, err := NewGraphBuilder().
		AddNode(&Node{Meta: NodeMeta{Key: "a"}, Executor: funcNode(func(_ context.Context, _ map[string]any) (map[string]any, error) { return nil, nil })}).
		AddNode(&Node{Meta: NodeMeta{Key: "b"}, Executor: funcNode(func(_ context.Context, _ map[string]any) (map[string]any, error) { return nil, nil })}).
		AddNode(&Node{Meta: NodeMeta{Key: "c"}, Executor: funcNode(func(_ context.Context, _ map[string]any) (map[string]any, error) { return nil, nil })}).
		AddEdge(Edge{From: "a", To: "b"}).
		AddEdge(Edge{From: "b", To: "c"}).
		AddEdge(Edge{From: "c", To: "a"}).
		SetEntry("a").
		Build()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle")
}

func TestTopologicalSort(t *testing.T) {
	g, err := NewGraphBuilder().
		AddNode(&Node{Meta: NodeMeta{Key: "a"}, Executor: funcNode(func(_ context.Context, _ map[string]any) (map[string]any, error) { return nil, nil })}).
		AddNode(&Node{Meta: NodeMeta{Key: "b"}, Executor: funcNode(func(_ context.Context, _ map[string]any) (map[string]any, error) { return nil, nil })}).
		AddNode(&Node{Meta: NodeMeta{Key: "c"}, Executor: funcNode(func(_ context.Context, _ map[string]any) (map[string]any, error) { return nil, nil })}).
		AddNode(&Node{Meta: NodeMeta{Key: "d"}, Executor: funcNode(func(_ context.Context, _ map[string]any) (map[string]any, error) { return nil, nil })}).
		AddEdge(Edge{From: "a", To: "b"}).
		AddEdge(Edge{From: "a", To: "c"}).
		AddEdge(Edge{From: "b", To: "d"}).
		AddEdge(Edge{From: "c", To: "d"}).
		SetEntry("a").
		Build()
	require.NoError(t, err)

	order, err := g.TopologicalSort()
	require.NoError(t, err)
	assert.Equal(t, NodeKey("a"), order[0])
	// d must come after b and c
	idxB := indexOf(order, "b")
	idxC := indexOf(order, "c")
	idxD := indexOf(order, "d")
	assert.Less(t, idxB, idxD)
	assert.Less(t, idxC, idxD)
}

func TestParallelExecution(t *testing.T) {
	g, err := NewGraphBuilder().
		AddNode(&Node{Meta: NodeMeta{Key: "entry"}, Executor: funcNode(func(_ context.Context, input map[string]any) (map[string]any, error) {
			return input, nil
		})}).
		AddNode(&Node{Meta: NodeMeta{Key: "a"}, Executor: funcNode(func(_ context.Context, input map[string]any) (map[string]any, error) {
			return map[string]any{"val": "a"}, nil
		})}).
		AddNode(&Node{Meta: NodeMeta{Key: "b"}, Executor: funcNode(func(_ context.Context, input map[string]any) (map[string]any, error) {
			return map[string]any{"val": "b"}, nil
		})}).
		AddNode(&Node{Meta: NodeMeta{Key: "exit"}, Executor: funcNode(func(_ context.Context, input map[string]any) (map[string]any, error) {
			return input, nil
		})}).
		AddEdge(Edge{From: "entry", To: "a"}).
		AddEdge(Edge{From: "entry", To: "b"}).
		AddEdge(Edge{From: "a", To: "exit"}).
		AddEdge(Edge{From: "b", To: "exit"}).
		SetEntry("entry").
		SetExit("exit").
		Build()
	require.NoError(t, err)

	px := NewParallelExecutor(g)
	result, err := px.Execute(context.Background(), map[string]any{"data": "hello"})
	require.NoError(t, err)
	assert.NotNil(t, result.Output)
}

func TestCheckpointRestore(t *testing.T) {
	ec := NewExecutionContext(map[string]any{"initial": "value"})
	ec.SetOutput("node1", map[string]any{"out": "data1"})
	ec.SetState("custom", 42)

	executed := map[NodeKey]bool{"node1": true, "entry": true}
	data, err := ec.Snapshot(executed)
	require.NoError(t, err)

	ec2 := NewExecutionContext(nil)
	restored, err := ec2.Restore(data)
	require.NoError(t, err)
	assert.True(t, restored["node1"])
	assert.True(t, restored["entry"])
	assert.Equal(t, "data1", ec2.GetOutput("node1")["out"])
	v, ok := ec2.GetState("custom")
	assert.True(t, ok)
	assert.Equal(t, float64(42), v)
}

func TestFieldMapping(t *testing.T) {
	g, err := NewGraphBuilder().
		AddNode(&Node{Meta: NodeMeta{Key: "entry"}, Executor: funcNode(func(_ context.Context, _ map[string]any) (map[string]any, error) {
			return map[string]any{"raw_text": "hello world"}, nil
		})}).
		AddNode(&Node{Meta: NodeMeta{Key: "exit"}, Executor: funcNode(func(_ context.Context, input map[string]any) (map[string]any, error) {
			return input, nil
		})}).
		AddEdge(Edge{From: "entry", To: "exit", FieldMapping: map[string]string{"raw_text": "processed_text"}}).
		SetEntry("entry").
		SetExit("exit").
		Build()
	require.NoError(t, err)

	ex := NewExecutor(g)
	result, err := ex.Execute(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, "hello world", result.Output["processed_text"])
}

func TestStreamHandler(t *testing.T) {
	var streamed []NodeKey
	var mu sync.Mutex

	g, err := NewGraphBuilder().
		AddNode(&Node{Meta: NodeMeta{Key: "entry"}, Executor: funcNode(func(_ context.Context, _ map[string]any) (map[string]any, error) {
			return map[string]any{"v": 1}, nil
		})}).
		AddNode(&Node{Meta: NodeMeta{Key: "mid"}, Executor: funcNode(func(_ context.Context, _ map[string]any) (map[string]any, error) {
			return map[string]any{"v": 2}, nil
		})}).
		AddNode(&Node{Meta: NodeMeta{Key: "exit"}, Executor: funcNode(func(_ context.Context, _ map[string]any) (map[string]any, error) {
			return map[string]any{"v": 3}, nil
		})}).
		AddEdge(Edge{From: "entry", To: "mid"}).
		AddEdge(Edge{From: "mid", To: "exit"}).
		SetEntry("entry").
		SetExit("exit").
		Build()
	require.NoError(t, err)

	ex := NewExecutor(g, WithStreamHandler(func(key NodeKey, _ map[string]any) {
		mu.Lock()
		streamed = append(streamed, key)
		mu.Unlock()
	}))
	_, err = ex.Execute(context.Background(), nil)
	require.NoError(t, err)
	assert.Contains(t, streamed, NodeKey("mid"))
	assert.Contains(t, streamed, NodeKey("exit"))
}

func TestInMemoryCheckpointStore(t *testing.T) {
	store := NewInMemoryCheckpointStore()
	ctx := context.Background()

	err := store.Set(ctx, "cp1", []byte("data"))
	require.NoError(t, err)

	data, ok, err := store.Get(ctx, "cp1")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, []byte("data"), data)

	err = store.Delete(ctx, "cp1")
	require.NoError(t, err)

	_, ok, err = store.Get(ctx, "cp1")
	require.NoError(t, err)
	assert.False(t, ok)
}

func indexOf(slice []NodeKey, val NodeKey) int {
	for i, v := range slice {
		if v == val {
			return i
		}
	}
	return -1
}
