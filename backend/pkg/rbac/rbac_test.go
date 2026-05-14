package rbac

import (
	"context"
	"testing"
	"time"
)

func TestAdminHasAllPermissions(t *testing.T) {
	resources := []string{"agents", "models", "skills", "config", "monitor"}
	actions := []string{"read", "write", "delete", "execute"}
	for _, r := range resources {
		for _, a := range actions {
			if !HasPermission(RoleAdmin, r, a) {
				t.Errorf("admin should have permission %s:%s", r, a)
			}
		}
	}
}

func TestEditorPermissions(t *testing.T) {
	// Editor can write agents
	if !HasPermission(RoleEditor, "agents", "write") {
		t.Error("editor should be able to write agents")
	}
	// Editor can read monitor but not write it
	if !HasPermission(RoleEditor, "monitor", "read") {
		t.Error("editor should be able to read monitor")
	}
	if HasPermission(RoleEditor, "monitor", "write") {
		t.Error("editor should NOT be able to write monitor")
	}
}

func TestViewerPermissions(t *testing.T) {
	// Viewer cannot write agents
	if HasPermission(RoleViewer, "agents", "write") {
		t.Error("viewer should NOT be able to write agents")
	}
	// Viewer can read agents
	if !HasPermission(RoleViewer, "agents", "read") {
		t.Error("viewer should be able to read agents")
	}
	// Viewer cannot write config
	if HasPermission(RoleViewer, "config", "write") {
		t.Error("viewer should NOT be able to write config")
	}
}

func TestOperatorPermissions(t *testing.T) {
	// Operator can write monitor
	if !HasPermission(RoleOperator, "monitor", "write") {
		t.Error("operator should be able to write monitor")
	}
	// Operator cannot write agents
	if HasPermission(RoleOperator, "agents", "write") {
		t.Error("operator should NOT be able to write agents")
	}
	// Operator cannot write config
	if HasPermission(RoleOperator, "config", "write") {
		t.Error("operator should NOT be able to write config")
	}
}

func TestDisabledUserReturnsFalse(t *testing.T) {
	user := &User{
		ID:        "u1",
		Name:      "Disabled User",
		Role:      RoleAdmin,
		APIKey:    "key-disabled",
		CreatedAt: time.Now(),
		Disabled:  true,
	}
	ctx := WithUser(context.Background(), user)
	if CheckPermission(ctx, "agents", "write") {
		t.Error("disabled user should not have any permission")
	}
}

func TestUnknownRoleReturnsFalse(t *testing.T) {
	if HasPermission(Role("unknown"), "agents", "read") {
		t.Error("unknown role should return false")
	}
}

func TestWithUserGetUserRoundTrip(t *testing.T) {
	user := &User{
		ID:        "u2",
		Name:      "Test User",
		Role:      RoleViewer,
		APIKey:    "key-test",
		CreatedAt: time.Now(),
	}
	ctx := WithUser(context.Background(), user)
	got := GetUser(ctx)
	if got == nil {
		t.Fatal("GetUser returned nil after WithUser")
	}
	if got.ID != user.ID {
		t.Errorf("expected ID %q, got %q", user.ID, got.ID)
	}
	if got.Role != user.Role {
		t.Errorf("expected role %q, got %q", user.Role, got.Role)
	}
}

func TestGetUserOnEmptyContextReturnsNil(t *testing.T) {
	got := GetUser(context.Background())
	if got != nil {
		t.Error("GetUser on empty context should return nil")
	}
}

func TestCheckPermissionNoUser(t *testing.T) {
	if CheckPermission(context.Background(), "agents", "read") {
		t.Error("CheckPermission with no user in context should return false")
	}
}
