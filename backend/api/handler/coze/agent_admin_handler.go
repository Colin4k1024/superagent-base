package coze

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/superagent-ai/superagent-base/backend/pkg/agentdef"
)

// AgentAdminHandler manages agent YAML CRUD operations.
type AgentAdminHandler struct {
	runtime   *agentdef.AgentRuntime
	configDir string // path to configs/agents/
}

// NewAgentAdminHandler creates an AgentAdminHandler.
func NewAgentAdminHandler(rt *agentdef.AgentRuntime, configDir string) *AgentAdminHandler {
	return &AgentAdminHandler{runtime: rt, configDir: configDir}
}

// agentFileItem is the list-view representation of an agent.
type agentFileItem struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Status      string `json:"status"` // "loaded" | "error" | "unknown"
	File        string `json:"file"`
}

// yamlBody is the request body for create/update/validate endpoints.
type yamlBody struct {
	YAML string `json:"yaml"`
}

// HandleList returns all agent YAML files with their runtime status.
// GET /api/v1/admin/agents
func (h *AgentAdminHandler) HandleList(_ context.Context, c *app.RequestContext) {
	entries, err := os.ReadDir(h.configDir)
	if err != nil {
		c.JSON(500, map[string]any{"code": 500, "msg": fmt.Sprintf("read config dir: %v", err)})
		return
	}

	items := make([]agentFileItem, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		data, readErr := os.ReadFile(filepath.Join(h.configDir, name))
		if readErr != nil {
			items = append(items, agentFileItem{File: name, Status: "error"})
			continue
		}

		def, parseErr := agentdef.Parse(data)
		if parseErr != nil {
			items = append(items, agentFileItem{File: name, Status: "error"})
			continue
		}

		status := "unknown"
		if h.runtime != nil {
			if _, ok := h.runtime.GetAgent(def.Metadata.Name); ok {
				status = "loaded"
			} else {
				status = "error"
			}
		}

		items = append(items, agentFileItem{
			Name:        def.Metadata.Name,
			Type:        def.Spec.Type,
			Description: descriptionFromDef(def),
			Status:      status,
			File:        name,
		})
	}

	c.JSON(200, map[string]any{"agents": items})
}

// HandleGet returns a single agent's definition and raw YAML.
// GET /api/v1/admin/agents/:name
func (h *AgentAdminHandler) HandleGet(_ context.Context, c *app.RequestContext) {
	agentName := c.Param("name")
	data, file, err := h.findAgentFile(agentName)
	if err != nil {
		c.JSON(404, map[string]any{"code": 404, "msg": fmt.Sprintf("agent %q not found", agentName)})
		return
	}

	def, parseErr := agentdef.Parse(data)
	if parseErr != nil {
		c.JSON(500, map[string]any{"code": 500, "msg": fmt.Sprintf("parse %s: %v", file, parseErr)})
		return
	}

	c.JSON(200, map[string]any{
		"agent": def,
		"yaml":  string(data),
	})
}

// HandleCreate creates a new agent YAML file.
// POST /api/v1/admin/agents
func (h *AgentAdminHandler) HandleCreate(ctx context.Context, c *app.RequestContext) {
	var body yamlBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(400, map[string]any{"code": 400, "msg": fmt.Sprintf("invalid request body: %v", err)})
		return
	}
	if body.YAML == "" {
		c.JSON(400, map[string]any{"code": 400, "msg": "yaml field is required"})
		return
	}

	def, err := agentdef.Parse([]byte(body.YAML))
	if err != nil {
		c.JSON(400, map[string]any{"code": 400, "msg": fmt.Sprintf("invalid agent YAML: %v", err)})
		return
	}

	filename := def.Metadata.Name + ".yaml"
	destPath := filepath.Join(h.configDir, filename)

	if _, statErr := os.Stat(destPath); statErr == nil {
		c.JSON(409, map[string]any{"code": 409, "msg": fmt.Sprintf("agent %q already exists", def.Metadata.Name)})
		return
	}

	if err := os.WriteFile(destPath, []byte(body.YAML), 0o644); err != nil {
		c.JSON(500, map[string]any{"code": 500, "msg": fmt.Sprintf("write file: %v", err)})
		return
	}

	if h.runtime != nil {
		if reloadErr := h.runtime.Reload(ctx); reloadErr != nil {
			// Log but do not fail the request — the file was written successfully.
			c.JSON(201, map[string]any{
				"name":    def.Metadata.Name,
				"message": fmt.Sprintf("created (reload warning: %v)", reloadErr),
			})
			return
		}
	}

	c.JSON(201, map[string]any{"name": def.Metadata.Name, "message": "created"})
}

// HandleUpdate overwrites an existing agent YAML file.
// PUT /api/v1/admin/agents/:name
func (h *AgentAdminHandler) HandleUpdate(ctx context.Context, c *app.RequestContext) {
	agentName := c.Param("name")

	var body yamlBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(400, map[string]any{"code": 400, "msg": fmt.Sprintf("invalid request body: %v", err)})
		return
	}
	if body.YAML == "" {
		c.JSON(400, map[string]any{"code": 400, "msg": "yaml field is required"})
		return
	}

	def, err := agentdef.Parse([]byte(body.YAML))
	if err != nil {
		c.JSON(400, map[string]any{"code": 400, "msg": fmt.Sprintf("invalid agent YAML: %v", err)})
		return
	}

	if def.Metadata.Name != agentName {
		c.JSON(400, map[string]any{
			"code": 400,
			"msg":  fmt.Sprintf("metadata.name %q does not match URL parameter %q", def.Metadata.Name, agentName),
		})
		return
	}

	// Locate the existing file (may be .yaml or .yml).
	_, existingFile, findErr := h.findAgentFile(agentName)
	if findErr != nil {
		c.JSON(404, map[string]any{"code": 404, "msg": fmt.Sprintf("agent %q not found", agentName)})
		return
	}

	destPath := filepath.Join(h.configDir, existingFile)
	if err := os.WriteFile(destPath, []byte(body.YAML), 0o644); err != nil {
		c.JSON(500, map[string]any{"code": 500, "msg": fmt.Sprintf("write file: %v", err)})
		return
	}

	if h.runtime != nil {
		if reloadErr := h.runtime.Reload(ctx); reloadErr != nil {
			c.JSON(200, map[string]any{
				"name":    agentName,
				"message": fmt.Sprintf("updated (reload warning: %v)", reloadErr),
			})
			return
		}
	}

	c.JSON(200, map[string]any{"name": agentName, "message": "updated"})
}

// HandleDelete removes an agent YAML file.
// DELETE /api/v1/admin/agents/:name
func (h *AgentAdminHandler) HandleDelete(ctx context.Context, c *app.RequestContext) {
	agentName := c.Param("name")

	_, existingFile, err := h.findAgentFile(agentName)
	if err != nil {
		c.JSON(404, map[string]any{"code": 404, "msg": fmt.Sprintf("agent %q not found", agentName)})
		return
	}

	filePath := filepath.Join(h.configDir, existingFile)
	if err := os.Remove(filePath); err != nil {
		c.JSON(500, map[string]any{"code": 500, "msg": fmt.Sprintf("remove file: %v", err)})
		return
	}

	if h.runtime != nil {
		if reloadErr := h.runtime.Reload(ctx); reloadErr != nil {
			c.JSON(200, map[string]any{
				"name":    agentName,
				"message": fmt.Sprintf("deleted (reload warning: %v)", reloadErr),
			})
			return
		}
	}

	c.JSON(200, map[string]any{"name": agentName, "message": "deleted"})
}

// HandleValidate parses and validates a YAML payload without writing to disk.
// POST /api/v1/admin/agents/validate
func (h *AgentAdminHandler) HandleValidate(_ context.Context, c *app.RequestContext) {
	var body yamlBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(200, map[string]any{"valid": false, "error": fmt.Sprintf("invalid request body: %v", err)})
		return
	}
	if body.YAML == "" {
		c.JSON(200, map[string]any{"valid": false, "error": "yaml field is required"})
		return
	}

	def, err := agentdef.Parse([]byte(body.YAML))
	if err != nil {
		c.JSON(200, map[string]any{"valid": false, "error": err.Error()})
		return
	}

	c.JSON(200, map[string]any{
		"valid": true,
		"agent": map[string]any{
			"name":        def.Metadata.Name,
			"type":        def.Spec.Type,
			"description": descriptionFromDef(def),
		},
	})
}

// descriptionFromDef extracts a human-readable description from an AgentDefinition.
// It uses the first line of the system prompt as a fallback since AgentSpec has no
// dedicated description field.
func descriptionFromDef(def *agentdef.AgentDefinition) string {
	if def.Spec.SystemPrompt == "" {
		return ""
	}
	line := def.Spec.SystemPrompt
	if idx := strings.IndexByte(line, '\n'); idx >= 0 {
		line = line[:idx]
	}
	const maxLen = 120
	if len(line) > maxLen {
		line = line[:maxLen]
	}
	return strings.TrimSpace(line)
}

// findAgentFile searches configDir for a file whose metadata.name matches agentName.
// It first tries the canonical `<name>.yaml`, then scans all YAML files.
// Returns the raw bytes, filename, and any error.
func (h *AgentAdminHandler) findAgentFile(agentName string) ([]byte, string, error) {
	// Fast path: canonical filename.
	for _, ext := range []string{".yaml", ".yml"} {
		candidate := agentName + ext
		path := filepath.Join(h.configDir, candidate)
		data, err := os.ReadFile(path)
		if err == nil {
			return data, candidate, nil
		}
	}

	// Slow path: scan all YAML files for matching metadata.name.
	entries, err := os.ReadDir(h.configDir)
	if err != nil {
		return nil, "", fmt.Errorf("read dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fname := e.Name()
		if !strings.HasSuffix(fname, ".yaml") && !strings.HasSuffix(fname, ".yml") {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(h.configDir, fname))
		if readErr != nil {
			continue
		}
		def, parseErr := agentdef.Parse(data)
		if parseErr != nil {
			continue
		}
		if def.Metadata.Name == agentName {
			return data, fname, nil
		}
	}

	return nil, "", fmt.Errorf("not found")
}
