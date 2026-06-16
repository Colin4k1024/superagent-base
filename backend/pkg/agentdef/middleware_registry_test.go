package agentdef

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/adk"
)

func TestRegisterMiddleware_AndGet(t *testing.T) {
	name := "test_mw_register"
	factory := func(_ context.Context, _ map[string]any) (adk.ChatModelAgentMiddleware, error) {
		return &adk.BaseChatModelAgentMiddleware{}, nil
	}

	RegisterMiddleware(name, factory)

	got, ok := GetMiddlewareFactory(name)
	if !ok {
		t.Fatalf("expected factory for %q to be registered", name)
	}
	if got == nil {
		t.Fatal("factory should not be nil")
	}

	mw, err := got(context.Background(), nil)
	if err != nil {
		t.Fatalf("factory returned error: %v", err)
	}
	if mw == nil {
		t.Fatal("factory returned nil middleware")
	}
}

func TestGetMiddlewareFactory_NotFound(t *testing.T) {
	_, ok := GetMiddlewareFactory("nonexistent_middleware_xyz")
	if ok {
		t.Error("expected not-found for unregistered middleware")
	}
}

func TestListMiddleware_ContainsRegistered(t *testing.T) {
	name := "test_mw_list"
	RegisterMiddleware(name, func(_ context.Context, _ map[string]any) (adk.ChatModelAgentMiddleware, error) {
		return &adk.BaseChatModelAgentMiddleware{}, nil
	})

	names := ListMiddleware()
	found := false
	for _, n := range names {
		if n == name {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ListMiddleware() should contain %q", name)
	}
}

func TestRegisterMiddleware_PanicsOnEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on empty name")
		}
	}()
	RegisterMiddleware("", func(_ context.Context, _ map[string]any) (adk.ChatModelAgentMiddleware, error) {
		return nil, nil
	})
}

func TestRegisterMiddleware_PanicsOnNilFactory(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on nil factory")
		}
	}()
	RegisterMiddleware("test_nil_factory", nil)
}
