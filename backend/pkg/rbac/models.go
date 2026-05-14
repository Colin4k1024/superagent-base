// Package rbac provides a foundational Role-Based Access Control model for
// superagent-base. This is a placeholder for future multi-tenant expansion —
// the current implementation is self-contained with no external dependencies
// (no DB, no Redis, no JWT). Wire it into middleware only when multi-tenant
// support is fully designed.
package rbac

import "time"

// Role defines a user's permission level.
type Role string

const (
	RoleAdmin    Role = "admin"    // Full access to all resources
	RoleEditor   Role = "editor"   // Can create/edit/delete agents and configs
	RoleViewer   Role = "viewer"   // Read-only access to agents and chat
	RoleOperator Role = "operator" // Can reload, monitor, but not edit agents
)

// User represents an authenticated user in the system.
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email,omitempty"`
	Role      Role      `json:"role"`
	APIKey    string    `json:"-"` // never exposed in JSON
	CreatedAt time.Time `json:"created_at"`
	Disabled  bool      `json:"disabled,omitempty"`
}

// Permission defines an action on a resource.
type Permission struct {
	Resource string // "agents", "models", "skills", "config", "monitor"
	Action   string // "read", "write", "delete", "execute"
}

// RolePermissions maps each role to its allowed permissions.
var RolePermissions = map[Role][]Permission{
	RoleAdmin: {
		{Resource: "*", Action: "*"},
	},
	RoleEditor: {
		{Resource: "agents", Action: "read"},
		{Resource: "agents", Action: "write"},
		{Resource: "agents", Action: "delete"},
		{Resource: "agents", Action: "execute"},
		{Resource: "models", Action: "read"},
		{Resource: "models", Action: "write"},
		{Resource: "models", Action: "delete"},
		{Resource: "skills", Action: "read"},
		{Resource: "skills", Action: "write"},
		{Resource: "skills", Action: "delete"},
		{Resource: "config", Action: "read"},
		{Resource: "config", Action: "write"},
		{Resource: "monitor", Action: "read"},
	},
	RoleViewer: {
		{Resource: "agents", Action: "read"},
		{Resource: "agents", Action: "execute"},
		{Resource: "models", Action: "read"},
		{Resource: "skills", Action: "read"},
		{Resource: "monitor", Action: "read"},
	},
	RoleOperator: {
		{Resource: "agents", Action: "read"},
		{Resource: "agents", Action: "execute"},
		{Resource: "models", Action: "read"},
		{Resource: "skills", Action: "read"},
		{Resource: "config", Action: "read"},
		{Resource: "monitor", Action: "read"},
		{Resource: "monitor", Action: "write"},
	},
}
