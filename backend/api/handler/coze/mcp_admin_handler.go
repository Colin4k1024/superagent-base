package coze

import (
	"context"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/superagent-ai/superagent-base/backend/pkg/mcp"
)

// MCPAdminHandler manages MCP server connections via HTTP.
// Authentication is enforced by middleware.APIKeyAdminAuthMW at the route group level.
type MCPAdminHandler struct {
	registry *mcp.Registry
}

// NewMCPAdminHandler creates an MCPAdminHandler.
func NewMCPAdminHandler(registry *mcp.Registry) *MCPAdminHandler {
	return &MCPAdminHandler{registry: registry}
}

// mcpServerItem is the list-view representation of a connected MCP server.
type mcpServerItem struct {
	Name       string `json:"name"`
	Transport  string `json:"transport"`
	Status     string `json:"status"`
	ToolsCount int    `json:"tools_count"`
}

// connectRequest is the request body for POST /api/v1/admin/mcp/servers.
type connectRequest struct {
	Name      string            `json:"name"`
	Transport string            `json:"transport"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	URL       string            `json:"url,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
}

// HandleList returns all connected MCP servers with metadata.
// GET /api/v1/admin/mcp/servers
func (h *MCPAdminHandler) HandleList(ctx context.Context, c *app.RequestContext) {
	if h.registry == nil {
		c.JSON(200, map[string]any{"servers": []mcpServerItem{}})
		return
	}

	names := h.registry.ListServers()
	items := make([]mcpServerItem, 0, len(names))
	for _, name := range names {
		item := mcpServerItem{
			Name:   name,
			Status: "connected",
		}
		// Attempt to get tool count; treat errors as 0 (non-fatal for listing).
		if client, ok := h.registry.GetClient(name); ok {
			if tools, err := client.ListTools(ctx); err == nil {
				item.ToolsCount = len(tools)
			}
		}
		items = append(items, item)
	}

	c.JSON(200, map[string]any{"servers": items})
}

// HandleConnect connects a new MCP server.
// POST /api/v1/admin/mcp/servers
func (h *MCPAdminHandler) HandleConnect(ctx context.Context, c *app.RequestContext) {
	if h.registry == nil {
		c.JSON(503, map[string]any{"code": 503, "msg": "MCP registry not available"})
		return
	}

	var req connectRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, map[string]any{"code": 400, "msg": fmt.Sprintf("invalid request body: %v", err)})
		return
	}
	if req.Name == "" {
		c.JSON(400, map[string]any{"code": 400, "msg": "name is required"})
		return
	}
	if req.Transport == "" {
		c.JSON(400, map[string]any{"code": 400, "msg": "transport is required (stdio or sse)"})
		return
	}

	// Check for existing connection — return 409 without disconnecting.
	if _, exists := h.registry.GetClient(req.Name); exists {
		c.JSON(409, map[string]any{"code": 409, "msg": fmt.Sprintf("server %q is already connected", req.Name)})
		return
	}

	cfg := mcp.ServerConfig{
		Name:      req.Name,
		Transport: req.Transport,
		Command:   req.Command,
		Args:      req.Args,
		URL:       req.URL,
		Env:       req.Env,
	}

	if err := h.registry.Connect(ctx, cfg); err != nil {
		c.JSON(400, map[string]any{"code": 400, "msg": fmt.Sprintf("connect failed: %v", err)})
		return
	}

	// Fetch tool list for the response; non-fatal if it fails.
	var tools []mcp.ToolDefinition
	if client, ok := h.registry.GetClient(req.Name); ok {
		tools, _ = client.ListTools(ctx)
	}

	c.JSON(201, map[string]any{
		"name":    req.Name,
		"message": "connected",
		"tools":   tools,
	})
}

// HandleDisconnect disconnects a named MCP server.
// DELETE /api/v1/admin/mcp/servers/:name
func (h *MCPAdminHandler) HandleDisconnect(_ context.Context, c *app.RequestContext) {
	if h.registry == nil {
		c.JSON(503, map[string]any{"code": 503, "msg": "MCP registry not available"})
		return
	}

	name := c.Param("name")

	// Verify the server exists before attempting disconnect.
	if _, exists := h.registry.GetClient(name); !exists {
		c.JSON(404, map[string]any{"code": 404, "msg": fmt.Sprintf("server %q not found", name)})
		return
	}

	if err := h.registry.Disconnect(name); err != nil {
		c.JSON(500, map[string]any{"code": 500, "msg": fmt.Sprintf("disconnect failed: %v", err)})
		return
	}

	c.JSON(200, map[string]any{"name": name, "message": "disconnected"})
}

// HandleListTools returns the tool list from a named MCP server.
// GET /api/v1/admin/mcp/servers/:name/tools
func (h *MCPAdminHandler) HandleListTools(ctx context.Context, c *app.RequestContext) {
	if h.registry == nil {
		c.JSON(503, map[string]any{"code": 503, "msg": "MCP registry not available"})
		return
	}

	name := c.Param("name")

	client, ok := h.registry.GetClient(name)
	if !ok {
		c.JSON(404, map[string]any{"code": 404, "msg": fmt.Sprintf("server %q not found", name)})
		return
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		c.JSON(500, map[string]any{"code": 500, "msg": fmt.Sprintf("list tools failed: %v", err)})
		return
	}

	c.JSON(200, map[string]any{"tools": tools})
}
