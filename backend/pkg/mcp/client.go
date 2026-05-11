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
	"sync/atomic"
)

// Client is a high-level MCP client built on top of a Transport.
// It exposes the tool-related subset of the MCP protocol needed by agents.
type Client struct {
	transport Transport
	info      *ServerInfo
	tools     []ToolDefinition

	nextID atomic.Int64
}

// NewClient wraps a Transport with the MCP client protocol.
func NewClient(transport Transport) *Client {
	return &Client{transport: transport}
}

// Initialize sends the initialize request, exchanges protocol versions, and
// caches the server info. It must be called before any other method.
func (c *Client) Initialize(ctx context.Context) (*ServerInfo, error) {
	params := initializeParams{
		ProtocolVersion: ProtocolVersion,
		ClientInfo: ServerInfo{
			Name:    "superagent-base",
			Version: "0.1.0",
		},
	}

	resp, err := c.call(ctx, MethodInitialize, params)
	if err != nil {
		return nil, fmt.Errorf("mcp initialize: %w", err)
	}

	var result initializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("mcp initialize: unmarshal result: %w", err)
	}

	c.info = &result.ServerInfo
	return c.info, nil
}

// Close shuts down the underlying transport.
func (c *Client) Close() error {
	return c.transport.Close()
}

// ListTools fetches the tool list from the server and caches it.
func (c *Client) ListTools(ctx context.Context) ([]ToolDefinition, error) {
	resp, err := c.call(ctx, MethodToolsList, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp tools/list: %w", err)
	}

	var result toolsListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("mcp tools/list: unmarshal result: %w", err)
	}

	c.tools = result.Tools
	return c.tools, nil
}

// CallTool invokes a named tool with the given arguments.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (*ToolCallResult, error) {
	params := toolsCallParams{Name: name, Arguments: args}

	resp, err := c.call(ctx, MethodToolsCall, params)
	if err != nil {
		return nil, fmt.Errorf("mcp tools/call %q: %w", name, err)
	}

	var result ToolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("mcp tools/call %q: unmarshal result: %w", name, err)
	}

	return &result, nil
}

// call is the internal helper that constructs a JSONRPCRequest, sends it, and
// checks for a protocol-level error in the response.
func (c *Client) call(ctx context.Context, method string, params any) (*JSONRPCResponse, error) {
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextID.Add(1),
		Method:  method,
		Params:  params,
	}

	resp, err := c.transport.Send(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Error != nil {
		return nil, resp.Error
	}

	return resp, nil
}
