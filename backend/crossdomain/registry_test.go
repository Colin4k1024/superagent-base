package crossdomain

import (
	"testing"
)

func TestServiceRegistry_Accessors(t *testing.T) {
	// Test that a registry with no services returns nil for all accessors
	r := NewServiceRegistry()

	if r.Agent() != nil {
		t.Error("Agent() should return nil when not set")
	}
	if r.AgentRun() != nil {
		t.Error("AgentRun() should return nil when not set")
	}
	if r.App() != nil {
		t.Error("App() should return nil when not set")
	}
	if r.Conversation() != nil {
		t.Error("Conversation() should return nil when not set")
	}
	if r.Database() != nil {
		t.Error("Database() should return nil when not set")
	}
	if r.Knowledge() != nil {
		t.Error("Knowledge() should return nil when not set")
	}
	if r.Message() != nil {
		t.Error("Message() should return nil when not set")
	}
	if r.Permission() != nil {
		t.Error("Permission() should return nil when not set")
	}
	if r.Plugin() != nil {
		t.Error("Plugin() should return nil when not set")
	}
	if r.Upload() != nil {
		t.Error("Upload() should return nil when not set")
	}
	if r.User() != nil {
		t.Error("User() should return nil when not set")
	}
	if r.Variables() != nil {
		t.Error("Variables() should return nil when not set")
	}
	if r.Workflow() != nil {
		t.Error("Workflow() should return nil when not set")
	}
}

func TestSetDefaultRegistry_And_Default(t *testing.T) {
	// Reset global state after test
	defer SetDefaultRegistry(nil)

	// Initially nil
	if Default() != nil {
		t.Error("Default() should return nil before SetDefaultRegistry")
	}

	// Set and retrieve
	r := NewServiceRegistry()
	SetDefaultRegistry(r)

	if Default() != r {
		t.Error("Default() should return the registry set by SetDefaultRegistry")
	}
}

func TestMustDefault_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustDefault() should panic when registry not set")
		}
	}()
	defer SetDefaultRegistry(nil)

	SetDefaultRegistry(nil)
	MustDefault()
}

func TestMustDefault_Success(t *testing.T) {
	defer SetDefaultRegistry(nil)

	r := NewServiceRegistry()
	SetDefaultRegistry(r)

	got := MustDefault()
	if got != r {
		t.Error("MustDefault() should return the registry")
	}
}
