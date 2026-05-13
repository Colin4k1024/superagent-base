package coze

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/sse"

	"github.com/superagent-ai/superagent-base/backend/pkg/agentdef"
	"github.com/superagent-ai/superagent-base/backend/pkg/logs"
)

// AdminHandler provides endpoints for system monitoring and administration.
type AdminHandler struct {
	runtime *agentdef.AgentRuntime
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(rt *agentdef.AgentRuntime) *AdminHandler {
	return &AdminHandler{runtime: rt}
}

// HandleStatus returns aggregated runtime status.
// GET /api/v1/admin/status
func (h *AdminHandler) HandleStatus(_ context.Context, c *app.RequestContext) {
	if h.runtime == nil {
		c.JSON(503, map[string]string{"error": "runtime not available"})
		return
	}

	uptime := time.Since(h.runtime.StartTime()).Seconds()
	agents := h.runtime.AgentInfoList()
	lastReload := h.runtime.LastReloadAt()

	var lastReloadStr string
	if !lastReload.IsZero() {
		lastReloadStr = lastReload.Format(time.RFC3339)
	}

	c.JSON(200, map[string]any{
		"uptime_seconds": int(uptime),
		"agent_count":    len(agents),
		"agents":         agents,
		"health":         "ok",
		"ready": map[string]string{
			"agent_runtime": "ok",
			"http":          "ok",
		},
		"last_reload_at": lastReloadStr,
		"start_time":     h.runtime.StartTime().Format(time.RFC3339),
	})
}

// HandleReload triggers a hot-reload of all agent definitions.
// POST /api/v1/admin/reload
func (h *AdminHandler) HandleReload(ctx context.Context, c *app.RequestContext) {
	if h.runtime == nil {
		c.JSON(503, map[string]string{"error": "runtime not available"})
		return
	}

	if err := h.runtime.Reload(ctx); err != nil {
		c.JSON(500, map[string]string{"error": fmt.Sprintf("reload failed: %v", err)})
		return
	}

	agents := h.runtime.ListAgents()
	c.JSON(200, map[string]any{
		"message":     "reload successful",
		"agent_count": len(agents),
		"agents":      agents,
	})
}

// HandleLogStream streams log entries as Server-Sent Events.
// GET /api/v1/admin/logs
func (h *AdminHandler) HandleLogStream(_ context.Context, c *app.RequestContext) {
	broadcaster := logs.GetBroadcaster()
	logCh, unsubscribe := broadcaster.Subscribe()

	// Set SSE headers.
	c.Response.Header.Set("Cache-Control", "no-cache")
	c.Response.Header.Set("Connection", "keep-alive")
	c.Response.Header.Set("X-Accel-Buffering", "no")
	c.Response.Header.Set("Access-Control-Allow-Origin", "*")

	w := sse.NewWriter(c)

	// Stream log entries until client disconnects.
	for entry := range logCh {
		data := logs.MarshalEntry(entry)
		if err := w.WriteEvent("", "log", data); err != nil {
			// Client disconnected.
			break
		}
	}

	unsubscribe()
	_ = w.Close()
}
