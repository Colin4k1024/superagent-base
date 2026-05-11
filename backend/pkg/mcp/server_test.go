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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// echoTool is a minimal tool used across tests; it echoes its "text" argument.
var echoToolDef = ToolDefinition{
	Name:        "echo",
	Description: "Echoes the input text.",
	InputSchema: &JSONSchema{
		Type:       "object",
		Properties: map[string]*JSONSchema{"text": {Type: "string"}},
		Required:   []string{"text"},
	},
}

func echoHandler(_ context.Context, args map[string]any) (*ToolCallResult, error) {
	text, _ := args["text"].(string)
	return &ToolCallResult{Content: []ContentBlock{{Type: "text", Text: text}}}, nil
}

func newTestServer() *Server {
	s := NewServer("test-server", "0.1.0")
	s.RegisterTool(echoToolDef, echoHandler)
	return s
}

// ---- initialize ----

func TestHandleRequest_Initialize(t *testing.T) {
	s := newTestServer()
	resp := s.HandleRequest(context.Background(), &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  MethodInitialize,
		Params:  initializeParams{ProtocolVersion: ProtocolVersion},
	})

	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.Equal(t, int64(1), resp.ID)

	var result initializeResult
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	assert.Equal(t, "test-server", result.ServerInfo.Name)
	assert.Equal(t, "0.1.0", result.ServerInfo.Version)
	assert.Equal(t, ProtocolVersion, result.ProtocolVersion)
}

// ---- tools/list ----

func TestHandleRequest_ToolsList(t *testing.T) {
	s := newTestServer()
	resp := s.HandleRequest(context.Background(), &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  MethodToolsList,
	})

	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	var result toolsListResult
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	require.Len(t, result.Tools, 1)
	assert.Equal(t, "echo", result.Tools[0].Name)
}

func TestHandleRequest_ToolsList_Empty(t *testing.T) {
	s := NewServer("empty", "1.0")
	resp := s.HandleRequest(context.Background(), &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  MethodToolsList,
	})

	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	var result toolsListResult
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	assert.Empty(t, result.Tools)
}

// ---- tools/call ----

func TestHandleRequest_ToolsCall_Success(t *testing.T) {
	s := newTestServer()
	resp := s.HandleRequest(context.Background(), &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      4,
		Method:  MethodToolsCall,
		Params:  toolsCallParams{Name: "echo", Arguments: map[string]any{"text": "hello"}},
	})

	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	var result ToolCallResult
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	assert.False(t, result.IsError)
	require.Len(t, result.Content, 1)
	assert.Equal(t, "hello", result.Content[0].Text)
}

func TestHandleRequest_ToolsCall_HandlerError(t *testing.T) {
	s := NewServer("s", "1")
	s.RegisterTool(ToolDefinition{Name: "fail"}, func(_ context.Context, _ map[string]any) (*ToolCallResult, error) {
		return nil, errors.New("something went wrong")
	})

	resp := s.HandleRequest(context.Background(), &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      5,
		Method:  MethodToolsCall,
		Params:  toolsCallParams{Name: "fail"},
	})

	require.NotNil(t, resp)
	// Handler errors surface as IsError result, not a protocol error.
	assert.Nil(t, resp.Error)

	var result ToolCallResult
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "something went wrong")
}

func TestHandleRequest_ToolsCall_UnknownTool(t *testing.T) {
	s := newTestServer()
	resp := s.HandleRequest(context.Background(), &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      6,
		Method:  MethodToolsCall,
		Params:  toolsCallParams{Name: "no-such-tool"},
	})

	require.NotNil(t, resp)
	require.NotNil(t, resp.Error)
	assert.Contains(t, resp.Error.Message, "tool not found")
}

// ---- unknown method ----

func TestHandleRequest_UnknownMethod(t *testing.T) {
	s := newTestServer()
	resp := s.HandleRequest(context.Background(), &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      7,
		Method:  "unknown/method",
	})

	require.NotNil(t, resp)
	require.NotNil(t, resp.Error)
	assert.Equal(t, -32601, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "method not found")
}

// ---- RegisterTool / UnregisterTool ----

func TestUnregisterTool(t *testing.T) {
	s := newTestServer()
	s.UnregisterTool("echo")

	resp := s.HandleRequest(context.Background(), &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      8,
		Method:  MethodToolsList,
	})

	var result toolsListResult
	require.NoError(t, json.Unmarshal(resp.Result, &result))
	assert.Empty(t, result.Tools)
}

// ---- expose helpers ----

func TestExposeAgentAsTool(t *testing.T) {
	def, handler := ExposeAgentAsTool("my-agent", "test agent", func(_ context.Context, msg string) (string, error) {
		return "echo: " + msg, nil
	})

	assert.Equal(t, "my-agent", def.Name)

	result, err := handler(context.Background(), map[string]any{"message": "hi"})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Equal(t, "echo: hi", result.Content[0].Text)
}

func TestExposeWorkflowAsTool(t *testing.T) {
	def, handler := ExposeWorkflowAsTool("my-wf", "test workflow", func(_ context.Context, input map[string]any) (map[string]any, error) {
		return map[string]any{"out": input["in"]}, nil
	})

	assert.Equal(t, "my-wf", def.Name)

	result, err := handler(context.Background(), map[string]any{
		"input": map[string]any{"in": "value"},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "value")
}
