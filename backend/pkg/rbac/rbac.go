package rbac

import "context"

type contextKey struct{}

// WithUser attaches a User to the context.
func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, contextKey{}, user)
}

// GetUser retrieves the User from context. Returns nil if not set.
func GetUser(ctx context.Context) *User {
	u, _ := ctx.Value(contextKey{}).(*User)
	return u
}

// HasPermission checks if the given role is allowed to perform action on resource.
func HasPermission(role Role, resource, action string) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if (p.Resource == "*" || p.Resource == resource) &&
			(p.Action == "*" || p.Action == action) {
			return true
		}
	}
	return false
}

// CheckPermission checks the user stored in ctx for the given resource and action.
// Returns false if no user is present or the user is disabled.
func CheckPermission(ctx context.Context, resource, action string) bool {
	user := GetUser(ctx)
	if user == nil {
		return false
	}
	if user.Disabled {
		return false
	}
	return HasPermission(user.Role, resource, action)
}
