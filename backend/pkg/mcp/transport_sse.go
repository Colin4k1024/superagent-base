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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// SSETransport implements Transport using Server-Sent Events.
//
// Client → server: HTTP POST to postURL with the JSON-RPC request body.
// Server → client: SSE stream at sseURL; each event carries a JSON-RPC response.
//
// The SSE connection is established lazily on the first Send call and kept alive
// until Close is called.
type SSETransport struct {
	sseURL  string
	postURL string
	headers map[string]string
	client  *http.Client

	mu      sync.Mutex
	pending map[int64]chan *JSONRPCResponse

	doneCh chan struct{}
	once   sync.Once
	connWg sync.WaitGroup
}

// NewSSETransport creates an SSETransport. sseURL is the endpoint to subscribe
// to; postURL is the endpoint to POST requests to. If postURL is empty it
// defaults to sseURL with "/message" appended.
func NewSSETransport(sseURL, postURL string, headers map[string]string) *SSETransport {
	if postURL == "" {
		postURL = strings.TrimSuffix(sseURL, "/") + "/message"
	}
	return &SSETransport{
		sseURL:  sseURL,
		postURL: postURL,
		headers: headers,
		client:  &http.Client{},
		pending: make(map[int64]chan *JSONRPCResponse),
		doneCh:  make(chan struct{}),
	}
}

// Send POSTs the request and waits for the matching response on the SSE stream.
func (t *SSETransport) Send(ctx context.Context, req *JSONRPCRequest) (*JSONRPCResponse, error) {
	// Start the SSE listener once.
	t.once.Do(func() {
		t.connWg.Add(1)
		go t.listenSSE()
	})

	ch := make(chan *JSONRPCResponse, 1)
	t.mu.Lock()
	t.pending[req.ID] = ch
	t.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		t.mu.Lock()
		delete(t.pending, req.ID)
		t.mu.Unlock()
		return nil, fmt.Errorf("mcp sse: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, t.postURL, bytes.NewReader(data))
	if err != nil {
		t.mu.Lock()
		delete(t.pending, req.ID)
		t.mu.Unlock()
		return nil, fmt.Errorf("mcp sse: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range t.headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := t.client.Do(httpReq)
	if err != nil {
		t.mu.Lock()
		delete(t.pending, req.ID)
		t.mu.Unlock()
		return nil, fmt.Errorf("mcp sse: post request: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		t.mu.Lock()
		delete(t.pending, req.ID)
		t.mu.Unlock()
		return nil, fmt.Errorf("mcp sse: unexpected status %d", resp.StatusCode)
	}

	select {
	case rpcResp := <-ch:
		return rpcResp, nil
	case <-ctx.Done():
		t.mu.Lock()
		delete(t.pending, req.ID)
		t.mu.Unlock()
		return nil, ctx.Err()
	case <-t.doneCh:
		return nil, fmt.Errorf("mcp sse: transport closed")
	}
}

// Close terminates the SSE connection and unblocks all pending callers.
func (t *SSETransport) Close() error {
	t.once.Do(func() {}) // ensure once has fired so listenSSE won't start
	close(t.doneCh)
	t.connWg.Wait()
	return nil
}

// listenSSE connects to the SSE endpoint and dispatches incoming events.
func (t *SSETransport) listenSSE() {
	defer t.connWg.Done()

	req, err := http.NewRequest(http.MethodGet, t.sseURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "text/event-stream")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	var dataLine string
	for scanner.Scan() {
		select {
		case <-t.doneCh:
			return
		default:
		}

		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			dataLine = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			continue
		}
		// Blank line signals end of event.
		if line == "" && dataLine != "" {
			t.dispatch(dataLine)
			dataLine = ""
		}
	}
}

func (t *SSETransport) dispatch(data string) {
	var resp JSONRPCResponse
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return
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
