/*
 * Copyright 2025 coze-dev Authors
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

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// ToolHandler is a function that executes a tool given parsed arguments and
// returns a ToolCallResult or an error.
type ToolHandler func(ctx context.Context, args map[string]any) (*ToolCallResult, error)

// registeredTool pairs a ToolDefinition with its handler.
type registeredTool struct {
	def     ToolDefinition
	handler ToolHandler
}

// Server is an MCP server that exposes registered tools to MCP clients.
// It handles incoming JSON-RPC requests and dispatches them to the appropriate
// method handler.
type Server struct {
	info  ServerInfo
	tools map[string]registeredTool
	mu    sync.RWMutex
}

// NewServer creates a Server advertising the given name and version.
func NewServer(name, version string) *Server {
	return &Server{
		info:  ServerInfo{Name: name, Version: version},
		tools: make(map[string]registeredTool),
	}
}

// RegisterTool adds or replaces a tool on the server.
func (s *Server) RegisterTool(def ToolDefinition, handler ToolHandler) {
	s.mu.Lock()
	s.tools[def.Name] = registeredTool{def: def, handler: handler}
	s.mu.Unlock()
}

// UnregisterTool removes a tool by name. It is a no-op if the tool does not
// exist.
func (s *Server) UnregisterTool(name string) {
	s.mu.Lock()
	delete(s.tools, name)
	s.mu.Unlock()
}

// HandleRequest dispatches a JSON-RPC request to the appropriate method and
// always returns a non-nil response.
func (s *Server) HandleRequest(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	var (
		result any
		rpcErr *RPCError
	)

	switch req.Method {
	case MethodInitialize:
		result, rpcErr = s.handleInitialize()
	case MethodToolsList:
		result, rpcErr = s.handleToolsList()
	case MethodToolsCall:
		result, rpcErr = s.handleToolsCall(ctx, req.Params)
	default:
		rpcErr = &RPCError{
			Code:    -32601,
			Message: fmt.Sprintf("method not found: %s", req.Method),
		}
	}

	resp := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Error:   rpcErr,
	}

	if rpcErr == nil && result != nil {
		raw, err := json.Marshal(result)
		if err != nil {
			resp.Error = &RPCError{Code: -32603, Message: "internal error: marshal result"}
		} else {
			resp.Result = raw
		}
	}

	return resp
}

// handleInitialize returns the server capabilities and info.
func (s *Server) handleInitialize() (any, *RPCError) {
	return initializeResult{
		ProtocolVersion: ProtocolVersion,
		ServerInfo:      s.info,
	}, nil
}

// handleToolsList returns the list of registered tool definitions.
func (s *Server) handleToolsList() (any, *RPCError) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	defs := make([]ToolDefinition, 0, len(s.tools))
	for _, t := range s.tools {
		defs = append(defs, t.def)
	}
	return toolsListResult{Tools: defs}, nil
}

// handleToolsCall invokes a registered tool handler.
func (s *Server) handleToolsCall(ctx context.Context, rawParams any) (any, *RPCError) {
	// rawParams arrives as map[string]any when decoded from JSON.
	params, err := decodeToolsCallParams(rawParams)
	if err != nil {
		return nil, &RPCError{Code: -32602, Message: "invalid params: " + err.Error()}
	}

	s.mu.RLock()
	rt, ok := s.tools[params.Name]
	s.mu.RUnlock()
	if !ok {
		return nil, &RPCError{
			Code:    -32602,
			Message: fmt.Sprintf("tool not found: %s", params.Name),
		}
	}

	result, err := rt.handler(ctx, params.Arguments)
	if err != nil {
		// Return a tool-level error result rather than a protocol error so the
		// client can surface the message without treating it as a transport failure.
		return &ToolCallResult{
			Content: []ContentBlock{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	return result, nil
}

// decodeToolsCallParams normalises rawParams (which may be a map or already a
// toolsCallParams after re-marshalling) into a toolsCallParams.
func decodeToolsCallParams(rawParams any) (toolsCallParams, error) {
	if rawParams == nil {
		return toolsCallParams{}, fmt.Errorf("params required")
	}
	data, err := json.Marshal(rawParams)
	if err != nil {
		return toolsCallParams{}, err
	}
	var p toolsCallParams
	if err := json.Unmarshal(data, &p); err != nil {
		return toolsCallParams{}, err
	}
	if p.Name == "" {
		return toolsCallParams{}, fmt.Errorf("name is required")
	}
	return p, nil
}
