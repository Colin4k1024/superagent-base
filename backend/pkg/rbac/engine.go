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

package rbac

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ── Enhanced RBAC Models ────────────────────────────────────────────────────

// Permission defines a granular action on a specific resource or resource pattern.
type Permission struct {
	ID       string `json:"id"`       // Unique permission identifier
	Resource string `json:"resource"` // Resource type pattern (e.g., "agents", "agents:*", "agents:knowledge")
	Action   string `json:"action"`   // Action pattern (e.g., "read", "write", "execute", "*")
	Scope    string `json:"scope"`    // Scope: "global", "tenant", "own" (default: "tenant")
}

// Role defines a named collection of permissions.
type Role struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Permissions []Permission `json:"permissions"`
	IsSystem    bool         `json:"is_system"` // System roles cannot be deleted
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// UserWithRoles extends the basic User with role assignments.
type UserWithRoles struct {
	User
	Roles      []string `json:"roles"`       // Role IDs assigned to this user
	TenantID   string   `json:"tenant_id"`   // Tenant isolation
	AssignedAt time.Time `json:"assigned_at"`
}

// ResourcePermission checks are scoped to a specific resource instance.
type ResourcePermission struct {
	ResourceType string // "agent", "knowledge", "workflow"
	ResourceID   string // Specific resource ID or "*" for all
	Action       string // "read", "write", "delete", "execute"
}

// ── Permission Store ────────────────────────────────────────────────────────

// PermissionStore persists roles and user-role assignments.
// For production, implement with a database; for testing, use MemoryStore.
type PermissionStore interface {
	// Role CRUD
	CreateRole(ctx context.Context, role *Role) error
	GetRole(ctx context.Context, roleID string) (*Role, error)
	ListRoles(ctx context.Context) ([]*Role, error)
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, roleID string) error

	// User-Role assignments
	AssignRole(ctx context.Context, userID, tenantID, roleID string) error
	RevokeRole(ctx context.Context, userID, tenantID, roleID string) error
	GetUserRoles(ctx context.Context, userID, tenantID string) ([]string, error)
	ListUsersWithRole(ctx context.Context, tenantID, roleID string) ([]string, error)
}

// ── Memory Store (for testing and single-instance deployments) ──────────────

// MemoryStore is an in-memory implementation of PermissionStore.
// NOT safe for production multi-instance deployments.
type MemoryStore struct {
	mu    sync.RWMutex
	roles map[string]*Role
	// userAssignments maps "tenantID:userID" → set of role IDs
	userAssignments map[string]map[string]struct{}
}

// NewMemoryStore creates a new in-memory permission store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		roles:           make(map[string]*Role),
		userAssignments: make(map[string]map[string]struct{}),
	}
}

func (s *MemoryStore) CreateRole(_ context.Context, role *Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.roles[role.ID]; exists {
		return fmt.Errorf("role %q already exists", role.ID)
	}
	s.roles[role.ID] = role
	return nil
}

func (s *MemoryStore) GetRole(_ context.Context, roleID string) (*Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	role, ok := s.roles[roleID]
	if !ok {
		return nil, fmt.Errorf("role %q not found", roleID)
	}
	return role, nil
}

func (s *MemoryStore) ListRoles(_ context.Context) ([]*Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	roles := make([]*Role, 0, len(s.roles))
	for _, r := range s.roles {
		roles = append(roles, r)
	}
	return roles, nil
}

func (s *MemoryStore) UpdateRole(_ context.Context, role *Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.roles[role.ID]; !exists {
		return fmt.Errorf("role %q not found", role.ID)
	}
	if s.roles[role.ID].IsSystem {
		return fmt.Errorf("cannot modify system role %q", role.ID)
	}
	s.roles[role.ID] = role
	return nil
}

func (s *MemoryStore) DeleteRole(_ context.Context, roleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	role, exists := s.roles[roleID]
	if !exists {
		return fmt.Errorf("role %q not found", roleID)
	}
	if role.IsSystem {
		return fmt.Errorf("cannot delete system role %q", roleID)
	}
	delete(s.roles, roleID)
	return nil
}

func (s *MemoryStore) AssignRole(_ context.Context, userID, tenantID, roleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := tenantID + ":" + userID
	if s.userAssignments[key] == nil {
		s.userAssignments[key] = make(map[string]struct{})
	}
	s.userAssignments[key][roleID] = struct{}{}
	return nil
}

func (s *MemoryStore) RevokeRole(_ context.Context, userID, tenantID, roleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := tenantID + ":" + userID
	if s.userAssignments[key] != nil {
		delete(s.userAssignments[key], roleID)
	}
	return nil
}

func (s *MemoryStore) GetUserRoles(_ context.Context, userID, tenantID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := tenantID + ":" + userID
	roles := make([]string, 0)
	for roleID := range s.userAssignments[key] {
		roles = append(roles, roleID)
	}
	return roles, nil
}

func (s *MemoryStore) ListUsersWithRole(_ context.Context, tenantID, roleID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := make([]string, 0)
	for key, roleSet := range s.userAssignments {
		if _, ok := roleSet[roleID]; ok {
			// key is "tenantID:userID", extract userID
			if len(key) > len(tenantID)+1 && key[:len(tenantID)+1] == tenantID+":" {
				users = append(users, key[len(tenantID)+1:])
			}
		}
	}
	return users, nil
}

// ── Enhanced RBAC Engine ────────────────────────────────────────────────────

// Engine is the production RBAC engine with dynamic role support.
type Engine struct {
	store         PermissionStore
	builtinRoles  map[string]*Role
	defaultRoleID string
}

// NewEngine creates a new RBAC engine with the given store.
func NewEngine(store PermissionStore, defaultRoleID string) *Engine {
	e := &Engine{
		store:         store,
		builtinRoles:  make(map[string]*Role),
		defaultRoleID: defaultRoleID,
	}
	e.registerBuiltinRoles()
	return e
}

// registerBuiltinRoles creates the default system roles if they don't exist.
func (e *Engine) registerBuiltinRoles() {
	builtin := []*Role{
		{
			ID: "admin", Name: "Administrator", IsSystem: true,
			Description: "Full access to all resources",
			Permissions: []Permission{
				{ID: "admin:*", Resource: "*", Action: "*", Scope: "global"},
			},
		},
		{
			ID: "editor", Name: "Editor", IsSystem: true,
			Description: "Can create, edit, and delete agents and configurations",
			Permissions: []Permission{
				{ID: "agents:read", Resource: "agents", Action: "read", Scope: "tenant"},
				{ID: "agents:write", Resource: "agents", Action: "write", Scope: "tenant"},
				{ID: "agents:delete", Resource: "agents", Action: "delete", Scope: "tenant"},
				{ID: "agents:execute", Resource: "agents", Action: "execute", Scope: "tenant"},
				{ID: "models:read", Resource: "models", Action: "read", Scope: "tenant"},
				{ID: "models:write", Resource: "models", Action: "write", Scope: "tenant"},
				{ID: "skills:read", Resource: "skills", Action: "read", Scope: "tenant"},
				{ID: "skills:write", Resource: "skills", Action: "write", Scope: "tenant"},
				{ID: "config:read", Resource: "config", Action: "read", Scope: "tenant"},
				{ID: "config:write", Resource: "config", Action: "write", Scope: "tenant"},
				{ID: "monitor:read", Resource: "monitor", Action: "read", Scope: "tenant"},
			},
		},
		{
			ID: "viewer", Name: "Viewer", IsSystem: true,
			Description: "Read-only access to agents and chat",
			Permissions: []Permission{
				{ID: "agents:read", Resource: "agents", Action: "read", Scope: "tenant"},
				{ID: "agents:execute", Resource: "agents", Action: "execute", Scope: "tenant"},
				{ID: "models:read", Resource: "models", Action: "read", Scope: "tenant"},
				{ID: "skills:read", Resource: "skills", Action: "read", Scope: "tenant"},
			},
		},
		{
			ID: "operator", Name: "Operator", IsSystem: true,
			Description: "Can reload, monitor, but not edit agents",
			Permissions: []Permission{
				{ID: "agents:read", Resource: "agents", Action: "read", Scope: "tenant"},
				{ID: "config:read", Resource: "config", Action: "read", Scope: "tenant"},
				{ID: "monitor:read", Resource: "monitor", Action: "read", Scope: "tenant"},
				{ID: "monitor:write", Resource: "monitor", Action: "write", Scope: "tenant"},
			},
		},
	}

	for _, role := range builtin {
		e.builtinRoles[role.ID] = role
		// Also persist to store (ignore "already exists" errors)
		_ = e.store.CreateRole(context.Background(), role)
	}
}

// CheckPermission checks if a user has permission to perform an action on a resource.
// This is the main entry point for authorization checks.
func (e *Engine) CheckPermission(ctx context.Context, user *User, req ResourcePermission) bool {
	if user == nil || user.Disabled {
		return false
	}

	// Get user's roles
	roleIDs, err := e.store.GetUserRoles(ctx, user.ID, "")
	if err != nil || len(roleIDs) == 0 {
		// Fall back to the user's direct role if no assignments
		roleIDs = []string{string(user.Role)}
	}

	for _, roleID := range roleIDs {
		role := e.resolveRole(roleID)
		if role == nil {
			continue
		}
		if e.roleMatchesPermission(role, req) {
			return true
		}
	}
	return false
}

// resolveRole looks up a role by ID, checking builtins first, then the store.
func (e *Engine) resolveRole(roleID string) *Role {
	if role, ok := e.builtinRoles[roleID]; ok {
		return role
	}
	role, _ := e.store.GetRole(context.Background(), roleID)
	return role
}

// roleMatchesPermission checks if a role grants the requested permission.
func (e *Engine) roleMatchesPermission(role *Role, req ResourcePermission) bool {
	for _, perm := range role.Permissions {
		if perm.Resource == "*" || perm.Resource == req.ResourceType {
			if perm.Action == "*" || perm.Action == req.Action {
				return true
			}
		}
	}
	return false
}

// AssignRoleToUser assigns a role to a user within a tenant.
func (e *Engine) AssignRoleToUser(ctx context.Context, userID, tenantID, roleID string) error {
	// Validate role exists
	if _, err := e.store.GetRole(ctx, roleID); err != nil {
		if _, ok := e.builtinRoles[roleID]; !ok {
			return fmt.Errorf("role %q does not exist", roleID)
		}
	}
	return e.store.AssignRole(ctx, userID, tenantID, roleID)
}

// GetUserPermissions returns all effective permissions for a user.
func (e *Engine) GetUserPermissions(ctx context.Context, user *User) ([]Permission, error) {
	if user == nil {
		return nil, fmt.Errorf("user is nil")
	}

	roleIDs, err := e.store.GetUserRoles(ctx, user.ID, "")
	if err != nil {
		return nil, err
	}
	if len(roleIDs) == 0 {
		roleIDs = []string{string(user.Role)}
	}

	seen := make(map[string]struct{})
	var perms []Permission
	for _, roleID := range roleIDs {
		role := e.resolveRole(roleID)
		if role == nil {
			continue
		}
		for _, p := range role.Permissions {
			if _, ok := seen[p.ID]; !ok {
				seen[p.ID] = struct{}{}
				perms = append(perms, p)
			}
		}
	}
	return perms, nil
}

// GetDefaultRoleID returns the default role ID for new users.
func (e *Engine) GetDefaultRoleID() string {
	return e.defaultRoleID
}
