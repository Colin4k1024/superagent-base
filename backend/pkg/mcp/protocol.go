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

// Package mcp implements a minimal MCP (Model Context Protocol) client that
// supports tool listing and calling over stdio and SSE transports.
package mcp

import "encoding/json"

// MCP JSON-RPC method names.
const (
	MethodInitialize = "initialize"
	MethodToolsList  = "tools/list"
	MethodToolsCall  = "tools/call"
)

// MCP protocol version advertised during initialization.
const ProtocolVersion = "2024-11-05"

// JSONRPCRequest is a JSON-RPC 2.0 request envelope.
type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// JSONRPCResponse is a JSON-RPC 2.0 response envelope.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is the JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string { return e.Message }

// JSONSchema is a minimal JSON Schema representation used in ToolDefinition.
// Only the fields relevant to tool input descriptions are included.
type JSONSchema struct {
	Type       string                 `json:"type,omitempty"`
	Properties map[string]*JSONSchema `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
	Items      *JSONSchema            `json:"items,omitempty"`
	Enum       []any                  `json:"enum,omitempty"`
	// Description is not standard JSON Schema but MCP servers emit it.
	Description string `json:"description,omitempty"`
}

// ToolDefinition is an MCP tool descriptor returned by tools/list.
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema *JSONSchema `json:"inputSchema"`
}

// ToolCallResult is the response payload for a tools/call request.
type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock is a single piece of content in a ToolCallResult.
type ContentBlock struct {
	// Type is one of "text", "image", or "resource".
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ServerInfo is returned by the server during initialization.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// initializeParams is the params object for the initialize request.
type initializeParams struct {
	ProtocolVersion string     `json:"protocolVersion"`
	ClientInfo      ServerInfo `json:"clientInfo"`
	Capabilities    struct{}   `json:"capabilities"`
}

// initializeResult is the result object for the initialize response.
type initializeResult struct {
	ProtocolVersion string     `json:"protocolVersion"`
	ServerInfo      ServerInfo `json:"serverInfo"`
}

// toolsListResult is the result object for the tools/list response.
type toolsListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

// toolsCallParams is the params object for the tools/call request.
type toolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}
