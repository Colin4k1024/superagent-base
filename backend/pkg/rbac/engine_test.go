package rbac

import (
	"context"
	"testing"
)

func TestEngine_CheckPermission_AdminHasFullAccess(t *testing.T) {
	store := NewMemoryStore()
	engine := NewEngine(store, "viewer")

	user := &User{ID: "u1", Role: RoleAdmin}
	_ = store.AssignRole(context.Background(), "u1", "", "admin")

	tests := []ResourcePermission{
		{ResourceType: "agents", Action: "read"},
		{ResourceType: "agents", Action: "write"},
		{ResourceType: "agents", Action: "delete"},
		{ResourceType: "models", Action: "write"},
		{ResourceType: "config", Action: "write"},
		{ResourceType: "anything", Action: "anything"},
	}

	for _, req := range tests {
		if !engine.CheckPermission(context.Background(), user, req) {
			t.Errorf("admin should have access to %s:%s", req.ResourceType, req.Action)
		}
	}
}

func TestEngine_CheckPermission_ViewerHasLimitedAccess(t *testing.T) {
	store := NewMemoryStore()
	engine := NewEngine(store, "viewer")

	user := &User{ID: "u1", Role: RoleViewer}
	_ = store.AssignRole(context.Background(), "u1", "", "viewer")

	// Should have read access
	if !engine.CheckPermission(context.Background(), user, ResourcePermission{ResourceType: "agents", Action: "read"}) {
		t.Error("viewer should have read access to agents")
	}

	// Should NOT have write access
	if engine.CheckPermission(context.Background(), user, ResourcePermission{ResourceType: "agents", Action: "write"}) {
		t.Error("viewer should NOT have write access to agents")
	}

	// Should NOT have delete access
	if engine.CheckPermission(context.Background(), user, ResourcePermission{ResourceType: "agents", Action: "delete"}) {
		t.Error("viewer should NOT have delete access to agents")
	}
}

func TestEngine_CheckPermission_DisabledUser(t *testing.T) {
	store := NewMemoryStore()
	engine := NewEngine(store, "viewer")

	user := &User{ID: "u1", Role: RoleAdmin, Disabled: true}

	if engine.CheckPermission(context.Background(), user, ResourcePermission{ResourceType: "agents", Action: "read"}) {
		t.Error("disabled user should have no access")
	}
}

func TestEngine_CheckPermission_NilUser(t *testing.T) {
	store := NewMemoryStore()
	engine := NewEngine(store, "viewer")

	if engine.CheckPermission(context.Background(), nil, ResourcePermission{ResourceType: "agents", Action: "read"}) {
		t.Error("nil user should have no access")
	}
}

func TestEngine_DynamicRole(t *testing.T) {
	store := NewMemoryStore()
	engine := NewEngine(store, "viewer")

	// Create a custom role
	customRole := &RoleSpec{
		ID:       "data-analyst",
		Name:     "Data Analyst",
		IsSystem: false,
		Permissions: []PermSpec{
			{ID: "data:read", Resource: "data", Action: "read", Scope: "tenant"},
			{ID: "data:query", Resource: "data", Action: "query", Scope: "tenant"},
		},
	}
	if err := store.CreateRole(context.Background(), customRole); err != nil {
		t.Fatalf("failed to create custom role: %v", err)
	}

	// Assign to user
	user := &User{ID: "u1", Role: RoleViewer}
	if err := engine.AssignRoleToUser(context.Background(), "u1", "t1", "data-analyst"); err != nil {
		t.Fatalf("failed to assign role: %v", err)
	}

	// Should have data:read access
	if !engine.CheckPermission(context.Background(), user, ResourcePermission{ResourceType: "data", Action: "read"}) {
		t.Error("data-analyst should have read access to data")
	}

	// Should NOT have agents:write access
	if engine.CheckPermission(context.Background(), user, ResourcePermission{ResourceType: "agents", Action: "write"}) {
		t.Error("data-analyst should NOT have write access to agents")
	}
}

func TestEngine_MultipleRoles(t *testing.T) {
	store := NewMemoryStore()
	engine := NewEngine(store, "viewer")

	user := &User{ID: "u1", Role: RoleViewer}
	_ = store.AssignRole(context.Background(), "u1", "", "viewer")
	_ = store.AssignRole(context.Background(), "u1", "", "operator")

	// Should have monitor:write from operator role
	if !engine.CheckPermission(context.Background(), user, ResourcePermission{ResourceType: "monitor", Action: "write"}) {
		t.Error("user with operator role should have monitor:write access")
	}

	// Should have agents:read from either role
	if !engine.CheckPermission(context.Background(), user, ResourcePermission{ResourceType: "agents", Action: "read"}) {
		t.Error("user should have agents:read access")
	}
}

func TestEngine_ProtectedSystemRoles(t *testing.T) {
	store := NewMemoryStore()
	engine := NewEngine(store, "viewer")
	_ = engine // suppress unused warning

	// Try to delete a system role
	err := store.DeleteRole(context.Background(), "admin")
	if err == nil {
		t.Error("should not be able to delete system role")
	}

	// Try to update a system role
	err = store.UpdateRole(context.Background(), &RoleSpec{ID: "admin", Name: "Hacked Admin"})
	if err == nil {
		t.Error("should not be able to update system role")
	}
}

func TestEngine_GetUserPermissions(t *testing.T) {
	store := NewMemoryStore()
	engine := NewEngine(store, "viewer")

	user := &User{ID: "u1", Role: RoleEditor}
	_ = store.AssignRole(context.Background(), "u1", "", "editor")

	perms, err := engine.GetUserPermissions(context.Background(), user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(perms) == 0 {
		t.Error("editor should have permissions")
	}

	// Check that agents:write is in the list
	found := false
	for _, p := range perms {
		if p.Resource == "agents" && p.Action == "write" {
			found = true
			break
		}
	}
	if !found {
		t.Error("editor permissions should include agents:write")
	}
}
