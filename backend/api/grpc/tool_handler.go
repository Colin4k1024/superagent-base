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

package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	toolv1 "github.com/superagent-ai/superagent-base/backend/api/grpc/gen/tool/v1"
)

// ToolHandler implements toolv1.ToolServiceServer by delegating to the
// plugin domain service.
type ToolHandler struct {
	toolv1.UnimplementedToolServiceServer
}

// NewToolHandler creates a ToolHandler.
// The plugin domain service is accessed via the crossdomain package (set up
// in application.Init) rather than injected directly here.
func NewToolHandler() *ToolHandler {
	return &ToolHandler{}
}

// ListTools lists tools available for an agent or space.
func (h *ToolHandler) ListTools(_ context.Context, _ *toolv1.ListToolsRequest) (*toolv1.ListToolsResponse, error) {
	// TODO: delegate to crossdomain/plugin.GetDefaultSVC().
	return &toolv1.ListToolsResponse{Tools: []*toolv1.Tool{}}, nil
}

// GetTool retrieves tool details by ID.
func (h *ToolHandler) GetTool(_ context.Context, req *toolv1.GetToolRequest) (*toolv1.GetToolResponse, error) {
	if req.ToolId == "" {
		return nil, status.Error(codes.InvalidArgument, "tool_id is required")
	}
	// TODO: delegate to crossdomain/plugin.GetDefaultSVC().
	return &toolv1.GetToolResponse{
		Tool: &toolv1.Tool{Id: req.ToolId},
	}, nil
}

// InvokeTool executes a tool with given parameters.
func (h *ToolHandler) InvokeTool(_ context.Context, req *toolv1.InvokeToolRequest) (*toolv1.InvokeToolResponse, error) {
	if req.ToolId == "" {
		return nil, status.Error(codes.InvalidArgument, "tool_id is required")
	}
	// TODO: delegate to crossdomain/plugin.GetDefaultSVC().ExecTool.
	return &toolv1.InvokeToolResponse{
		Success: true,
		Output:  nil,
	}, nil
}
