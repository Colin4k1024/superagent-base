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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// StdioTransport implements Transport by spawning a subprocess and exchanging
// newline-delimited JSON over the process's stdin and stdout.
//
// One goroutine continuously reads stdout and dispatches responses to waiting
// callers via a per-request channel. Send serialises concurrent callers with a
// mutex so that only one request is in flight at a time — acceptable for v1.
type StdioTransport struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader

	mu      sync.Mutex // serialises writes to stdin
	pending map[int64]chan *JSONRPCResponse

	doneCh chan struct{}
	once   sync.Once
}

// NewStdioTransport creates and starts the subprocess identified by command and
// args, and begins reading its stdout in the background.
func NewStdioTransport(command string, args []string, env []string) (*StdioTransport, error) {
	cmd := exec.Command(command, args...)
	if len(env) > 0 {
		cmd.Env = env
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp stdio: stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp stdio: stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp stdio: start process: %w", err)
	}

	t := &StdioTransport{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewReader(stdout),
		pending: make(map[int64]chan *JSONRPCResponse),
		doneCh:  make(chan struct{}),
	}

	go t.readLoop()
	return t, nil
}

// Send writes req as a newline-delimited JSON line to the subprocess stdin and
// waits for the response with matching ID.
func (t *StdioTransport) Send(ctx context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error) {
	ch := make(chan *JSONRPCResponse, 1)

	t.mu.Lock()
	t.pending[req.ID] = ch
	data, err := json.Marshal(req)
	if err != nil {
		delete(t.pending, req.ID)
		t.mu.Unlock()
		return nil, fmt.Errorf("mcp stdio: marshal request: %w", err)
	}
	data = append(data, '\n')
	if _, err = t.stdin.Write(data); err != nil {
		delete(t.pending, req.ID)
		t.mu.Unlock()
		return nil, fmt.Errorf("mcp stdio: write request: %w", err)
	}
	t.mu.Unlock()

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		t.mu.Lock()
		delete(t.pending, req.ID)
		t.mu.Unlock()
		return nil, ctx.Err()
	case <-t.doneCh:
		return nil, fmt.Errorf("mcp stdio: transport closed")
	}
}

// Close terminates the subprocess and unblocks all pending callers.
func (t *StdioTransport) Close() error {
	t.once.Do(func() {
		_ = t.stdin.Close()
		_ = t.cmd.Process.Kill()
		close(t.doneCh)
	})
	return t.cmd.Wait()
}

// readLoop reads newline-delimited JSON from stdout and dispatches responses.
func (t *StdioTransport) readLoop() {
	for {
		line, err := t.stdout.ReadString('\n')
		if err != nil {
			return
		}
		var resp JSONRPCResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			continue
		}
		t.mu.Lock()
		ch, ok := t.pending[resp.ID]
		if ok {
			delete(t.pending, resp.ID)
		}
		t.mu.Unlock()
		if ok {
			ch <- &resp
		}
	}
}
