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
	"encoding/json"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	toolv1 "github.com/superagent-ai/superagent-base/backend/api/grpc/gen/tool/v1"
	"github.com/superagent-ai/superagent-base/backend/pkg/tool"
)

// ToolHandler implements toolv1.ToolServiceServer by delegating to tool.Manager.
type ToolHandler struct {
	toolv1.UnimplementedToolServiceServer
	mgr *tool.Manager
}

// NewToolHandler creates a ToolHandler backed by mgr.
// When mgr is nil the handler returns Unavailable for all calls.
func NewToolHandler(mgr *tool.Manager) *ToolHandler {
	return &ToolHandler{mgr: mgr}
}

// ListTools returns all tools registered in the manager.
func (h *ToolHandler) ListTools(ctx context.Context, _ *toolv1.ListToolsRequest) (*toolv1.ListToolsResponse, error) {
	if h.mgr == nil {
		return nil, status.Error(codes.Unavailable, "tool manager not initialised")
	}

	einoTools := h.mgr.List()
	protoTools := make([]*toolv1.Tool, 0, len(einoTools))
	for _, t := range einoTools {
		info, err := t.Info(ctx)
		if err != nil {
			continue
		}
		protoTools = append(protoTools, &toolv1.Tool{
			Id:      info.Name,
			Name:    info.Name,
			Description: info.Desc,
			Enabled: true,
		})
	}

	return &toolv1.ListToolsResponse{
		Tools: protoTools,
		Total: int32(len(protoTools)),
	}, nil
}

// GetTool retrieves a single tool by name (ToolId maps to the registered name).
func (h *ToolHandler) GetTool(ctx context.Context, req *toolv1.GetToolRequest) (*toolv1.GetToolResponse, error) {
	if req.ToolId == "" {
		return nil, status.Error(codes.InvalidArgument, "tool_id is required")
	}
	if h.mgr == nil {
		return nil, status.Error(codes.Unavailable, "tool manager not initialised")
	}

	t, ok := h.mgr.Get(req.ToolId)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "tool %q not found", req.ToolId)
	}

	info, err := t.Info(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get tool info: %v", err)
	}

	return &toolv1.GetToolResponse{
		Tool: &toolv1.Tool{
			Id:          info.Name,
			Name:        info.Name,
			Description: info.Desc,
			Enabled:     true,
		},
	}, nil
}

// InvokeTool executes a tool identified by ToolId with the provided Parameters.
// Parameters (structpb.Struct) are serialised to JSON and passed to InvokableRun.
// The raw string output is returned as {"result": "<output>"} in Output.
func (h *ToolHandler) InvokeTool(ctx context.Context, req *toolv1.InvokeToolRequest) (*toolv1.InvokeToolResponse, error) {
	if req.ToolId == "" {
		return nil, status.Error(codes.InvalidArgument, "tool_id is required")
	}
	if h.mgr == nil {
		return nil, status.Error(codes.Unavailable, "tool manager not initialised")
	}

	t, ok := h.mgr.Get(req.ToolId)
	if !ok {
		return nil, status.Errorf(codes.NotFound, "tool %q not found", req.ToolId)
	}

	// Serialise the parameters struct to a JSON string for InvokableRun.
	argsJSON := "{}"
	if req.Parameters != nil {
		b, err := json.Marshal(req.Parameters.AsMap())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "marshal parameters: %v", err)
		}
		argsJSON = string(b)
	}

	start := time.Now()
	out, err := t.InvokableRun(ctx, argsJSON)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		return &toolv1.InvokeToolResponse{
			Success:     false,
			Error:       err.Error(),
			ExecutionMs: elapsed,
		}, nil
	}

	// Wrap the raw string output in a structpb.Struct{"result": out}.
	outputStruct, structErr := structpb.NewStruct(map[string]any{"result": out})
	if structErr != nil {
		return nil, status.Errorf(codes.Internal, "encode output: %v", structErr)
	}

	return &toolv1.InvokeToolResponse{
		Success:     true,
		Output:      outputStruct,
		ExecutionMs: elapsed,
	}, nil
}
