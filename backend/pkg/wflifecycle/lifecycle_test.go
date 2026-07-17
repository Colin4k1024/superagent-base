package wflifecycle

import (
	"context"
	"errors"
	"testing"
)

// mockExecutor is a test double for NodeExecutor.
type mockExecutor struct {
	nodeType   NodeType
	prepared   bool
	executed   bool
	finalized  bool
	execErr    error
	prepErr    error
}

func (m *mockExecutor) Type() NodeType { return m.nodeType }

func (m *mockExecutor) Prepare(_ context.Context, _ *NodeContext) error {
	m.prepared = true
	return m.prepErr
}

func (m *mockExecutor) Execute(_ context.Context, _ *NodeContext) (*NodeResult, error) {
	m.executed = true
	if m.execErr != nil {
		return nil, m.execErr
	}
	return &NodeResult{
		State:   StateSuccess,
		Outputs: map[string]any{"result": "ok"},
	}, nil
}

func (m *mockExecutor) Finalize(_ context.Context, _ *NodeContext, _ *NodeResult) {
	m.finalized = true
}

func TestNodeRegistry_Register_And_Get(t *testing.T) {
	r := NewNodeRegistry()
	exec := &mockExecutor{nodeType: NodeTypeLLM}
	r.Register(exec)

	if !r.Has(NodeTypeLLM) {
		t.Error("Has should return true for registered type")
	}
	if r.Get(NodeTypeLLM) != exec {
		t.Error("Get should return the registered executor")
	}
	if r.Has(NodeTypeCode) {
		t.Error("Has should return false for unregistered type")
	}
}

func TestNodeRegistry_Register_Duplicate_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Register should panic on duplicate type")
		}
	}()
	r := NewNodeRegistry()
	r.Register(&mockExecutor{nodeType: NodeTypeLLM})
	r.Register(&mockExecutor{nodeType: NodeTypeLLM})
}

func TestNodeRegistry_Types(t *testing.T) {
	r := NewNodeRegistry()
	r.Register(&mockExecutor{nodeType: NodeTypeLLM})
	r.Register(&mockExecutor{nodeType: NodeTypeCode})

	types := r.Types()
	if len(types) != 2 {
		t.Errorf("Types() returned %d types; want 2", len(types))
	}
}

func TestExecuteNode_Success(t *testing.T) {
	r := NewNodeRegistry()
	exec := &mockExecutor{nodeType: NodeTypeLLM}
	r.Register(exec)

	nodeCtx := &NodeContext{NodeID: "n1", NodeType: NodeTypeLLM}
	result := r.ExecuteNode(context.Background(), nodeCtx)

	if result.State != StateSuccess {
		t.Errorf("State = %q; want %q", result.State, StateSuccess)
	}
	if !exec.prepared || !exec.executed || !exec.finalized {
		t.Errorf("Lifecycle not complete: prepared=%v executed=%v finalized=%v", exec.prepared, exec.executed, exec.finalized)
	}
}

func TestExecuteNode_PrepareFails(t *testing.T) {
	r := NewNodeRegistry()
	exec := &mockExecutor{nodeType: NodeTypeLLM, prepErr: errors.New("bad config")}
	r.Register(exec)

	nodeCtx := &NodeContext{NodeID: "n1", NodeType: NodeTypeLLM}
	result := r.ExecuteNode(context.Background(), nodeCtx)

	if result.State != StateFailed {
		t.Errorf("State = %q; want %q", result.State, StateFailed)
	}
	if !exec.prepared {
		t.Error("Prepare should have been called")
	}
	if exec.executed {
		t.Error("Execute should NOT have been called when Prepare fails")
	}
	// Finalize is always called
	if !exec.finalized {
		t.Error("Finalize should always be called")
	}
}

func TestExecuteNode_ExecuteFails(t *testing.T) {
	r := NewNodeRegistry()
	exec := &mockExecutor{nodeType: NodeTypeCode, execErr: errors.New("runtime error")}
	r.Register(exec)

	nodeCtx := &NodeContext{NodeID: "n1", NodeType: NodeTypeCode}
	result := r.ExecuteNode(context.Background(), nodeCtx)

	if result.State != StateFailed {
		t.Errorf("State = %q; want %q", result.State, StateFailed)
	}
	if !exec.finalized {
		t.Error("Finalize should be called even when Execute fails")
	}
}

func TestExecuteNode_UnregisteredType(t *testing.T) {
	r := NewNodeRegistry()
	nodeCtx := &NodeContext{NodeID: "n1", NodeType: NodeTypeLLM}
	result := r.ExecuteNode(context.Background(), nodeCtx)

	if result.State != StateFailed {
		t.Errorf("State = %q; want %q", result.State, StateFailed)
	}
}
