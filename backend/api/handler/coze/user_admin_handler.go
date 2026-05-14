package coze

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/superagent-ai/superagent-base/backend/pkg/rbac"
)

// UserAdminHandler provides CRUD operations for the in-memory UserStore.
type UserAdminHandler struct {
	store *rbac.UserStore
}

// NewUserAdminHandler creates a UserAdminHandler backed by the given store.
func NewUserAdminHandler(store *rbac.UserStore) *UserAdminHandler {
	return &UserAdminHandler{store: store}
}

// userView is the JSON representation of a user (API key excluded).
type userView struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email,omitempty"`
	Role      rbac.Role `json:"role"`
	Disabled  bool      `json:"disabled,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func toView(u *rbac.User) userView {
	return userView{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      u.Role,
		Disabled:  u.Disabled,
		CreatedAt: u.CreatedAt,
	}
}

// HandleList returns all registered users without API keys.
// GET /api/v1/admin/users
func (h *UserAdminHandler) HandleList(_ context.Context, c *app.RequestContext) {
	users := h.store.List()
	views := make([]userView, 0, len(users))
	for _, u := range users {
		views = append(views, toView(u))
	}
	c.JSON(200, map[string]any{"users": views})
}

// createUserRequest is the request body for user creation.
type createUserRequest struct {
	Name   string    `json:"name"`
	Email  string    `json:"email"`
	Role   rbac.Role `json:"role"`
	APIKey string    `json:"api_key"`
}

// HandleCreate adds a new user to the store.
// POST /api/v1/admin/users
// Body: {name, email, role, api_key}
func (h *UserAdminHandler) HandleCreate(_ context.Context, c *app.RequestContext) {
	var req createUserRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]any{"code": 400, "msg": fmt.Sprintf("invalid request body: %v", err)})
		return
	}

	if req.Name == "" {
		c.JSON(400, map[string]any{"code": 400, "msg": "name is required"})
		return
	}
	if req.APIKey == "" {
		c.JSON(400, map[string]any{"code": 400, "msg": "api_key is required"})
		return
	}
	if req.Role == "" {
		req.Role = rbac.RoleViewer
	}

	user := &rbac.User{
		ID:        fmt.Sprintf("u-%d", time.Now().UnixNano()),
		Name:      req.Name,
		Email:     req.Email,
		Role:      req.Role,
		APIKey:    req.APIKey,
		CreatedAt: time.Now(),
	}
	h.store.Register(user)

	c.JSON(201, map[string]any{"user": toView(user)})
}

// HandleDelete removes a user by ID.
// DELETE /api/v1/admin/users/:id
func (h *UserAdminHandler) HandleDelete(_ context.Context, c *app.RequestContext) {
	id := c.Param("id")
	if !h.store.Remove(id) {
		c.JSON(404, map[string]any{"code": 404, "msg": fmt.Sprintf("user %q not found", id)})
		return
	}
	c.JSON(200, map[string]any{"id": id, "message": "deleted"})
}

// updateUserRequest is the request body for role/disabled updates.
type updateUserRequest struct {
	Role     rbac.Role `json:"role"`
	Disabled *bool     `json:"disabled"`
}

// HandleUpdate modifies the role or disabled flag of an existing user.
// PUT /api/v1/admin/users/:id
// Body: {role?, disabled?}
func (h *UserAdminHandler) HandleUpdate(_ context.Context, c *app.RequestContext) {
	id := c.Param("id")

	var req updateUserRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]any{"code": 400, "msg": fmt.Sprintf("invalid request body: %v", err)})
		return
	}

	if !h.store.Update(id, req.Role, req.Disabled) {
		c.JSON(404, map[string]any{"code": 404, "msg": fmt.Sprintf("user %q not found", id)})
		return
	}

	u, _ := h.store.LookupByID(id)
	c.JSON(200, map[string]any{"user": toView(u)})
}
