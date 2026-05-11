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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// sseClient represents a single connected SSE subscriber.
type sseClient struct {
	ch chan string
}

// sseHub manages the set of live SSE connections and broadcasts messages to
// all of them.
type sseHub struct {
	mu      sync.RWMutex
	clients map[*sseClient]struct{}
}

func newSSEHub() *sseHub {
	return &sseHub{clients: make(map[*sseClient]struct{})}
}

func (h *sseHub) add(c *sseClient) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *sseHub) remove(c *sseClient) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

func (h *sseHub) broadcast(msg string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.ch <- msg:
		default:
			// Drop if the client's buffer is full rather than blocking.
		}
	}
}

// sseHandler is an http.Handler that streams an SSE connection for one client.
type sseHandler struct {
	hub    *sseHub
	client *sseClient
}

func (h *sseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	h.hub.add(h.client)
	defer h.hub.remove(h.client)

	for {
		select {
		case msg := <-h.client.ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// ServeSSE returns an http.Handler that implements the MCP SSE transport:
//
//   - GET  /sse     — opens an SSE stream; server pushes JSON-RPC responses here.
//   - POST /message — receives JSON-RPC requests; response is sent via SSE.
//
// Mount it at a path prefix, e.g.:
//
//	mux.Handle("/mcp/", http.StripPrefix("/mcp", server.ServeSSE()))
func (s *Server) ServeSSE() http.Handler {
	hub := newSSEHub()

	mux := http.NewServeMux()

	// SSE stream endpoint — each GET opens a persistent push channel.
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		c := &sseClient{ch: make(chan string, 16)}
		h := &sseHandler{hub: hub, client: c}
		h.ServeHTTP(w, r)
	})

	// POST /message — receive a JSON-RPC request, dispatch, push result via SSE.
	mux.HandleFunc("/message", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB max
		if err != nil {
			http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
			return
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(body, &req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		resp := s.HandleRequest(r.Context(), &req)

		data, err := json.Marshal(resp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Push to all connected SSE clients.
		hub.broadcast(string(data))

		// Also return 202 Accepted so the HTTP caller knows the POST landed.
		w.WriteHeader(http.StatusAccepted)
	})

	return mux
}
