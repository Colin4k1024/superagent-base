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

package dag

import (
	"fmt"
	"sort"
)

// Graph is the DAG representation. It is immutable after Build.
type Graph struct {
	nodes map[NodeKey]*Node
	edges []*Edge

	// adjacency: node → outgoing edges
	outEdges map[NodeKey][]*Edge
	// reverse adjacency: node → incoming edges
	inEdges map[NodeKey][]*Edge

	entry NodeKey
	exit  NodeKey
}

// GraphBuilder constructs a Graph incrementally.
type GraphBuilder struct {
	nodes map[NodeKey]*Node
	edges []*Edge
	entry NodeKey
	exit  NodeKey
}

// NewGraphBuilder creates a new GraphBuilder.
func NewGraphBuilder() *GraphBuilder {
	return &GraphBuilder{
		nodes: make(map[NodeKey]*Node),
	}
}

// AddNode registers a node. Panics on duplicate keys.
func (b *GraphBuilder) AddNode(node *Node) *GraphBuilder {
	if _, exists := b.nodes[node.Meta.Key]; exists {
		panic(fmt.Sprintf("dag: duplicate node key %q", node.Meta.Key))
	}
	b.nodes[node.Meta.Key] = node
	return b
}

// AddEdge adds a directed edge. Both endpoints must already be registered.
func (b *GraphBuilder) AddEdge(edge Edge) *GraphBuilder {
	if _, ok := b.nodes[edge.From]; !ok {
		panic(fmt.Sprintf("dag: edge from unknown node %q", edge.From))
	}
	if _, ok := b.nodes[edge.To]; !ok {
		panic(fmt.Sprintf("dag: edge to unknown node %q", edge.To))
	}
	b.edges = append(b.edges, &edge)
	return b
}

// SetEntry designates the entry node (source of execution).
func (b *GraphBuilder) SetEntry(key NodeKey) *GraphBuilder {
	b.entry = key
	return b
}

// SetExit designates the exit node (sink, its output becomes the graph output).
func (b *GraphBuilder) SetExit(key NodeKey) *GraphBuilder {
	b.exit = key
	return b
}

// Build finalizes the graph and returns an immutable Graph.
func (b *GraphBuilder) Build() (*Graph, error) {
	if len(b.nodes) == 0 {
		return nil, fmt.Errorf("dag: graph has no nodes")
	}
	if b.entry == "" {
		return nil, fmt.Errorf("dag: entry node not set")
	}
	if _, ok := b.nodes[b.entry]; !ok {
		return nil, fmt.Errorf("dag: entry node %q not found", b.entry)
	}

	g := &Graph{
		nodes:    b.nodes,
		edges:    b.edges,
		outEdges: make(map[NodeKey][]*Edge),
		inEdges:  make(map[NodeKey][]*Edge),
		entry:    b.entry,
		exit:     b.exit,
	}

	for _, e := range b.edges {
		g.outEdges[e.From] = append(g.outEdges[e.From], e)
		g.inEdges[e.To] = append(g.inEdges[e.To], e)
	}

	if cycle := g.detectCycle(); cycle != nil {
		return nil, fmt.Errorf("dag: cycle detected: %v", cycle)
	}

	return g, nil
}

// Nodes returns all node keys in sorted order.
func (g *Graph) Nodes() []NodeKey {
	keys := make([]NodeKey, 0, len(g.nodes))
	for k := range g.nodes {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

// Node returns the node with the given key, or nil.
func (g *Graph) Node(key NodeKey) *Node {
	return g.nodes[key]
}

// Entry returns the entry node key.
func (g *Graph) Entry() NodeKey { return g.entry }

// Exit returns the exit node key (may be empty).
func (g *Graph) Exit() NodeKey { return g.exit }

// OutEdges returns all outgoing edges from a node.
func (g *Graph) OutEdges(key NodeKey) []*Edge {
	return g.outEdges[key]
}

// InEdges returns all incoming edges to a node.
func (g *Graph) InEdges(key NodeKey) []*Edge {
	return g.inEdges[key]
}

// detectCycle returns a cycle path if one exists, or nil.
func (g *Graph) detectCycle() []NodeKey {
	const (
		white = 0 // unvisited
		gray  = 1 // in progress
		black = 2 // done
	)
	color := make(map[NodeKey]int)
	var path []NodeKey

	var dfs func(key NodeKey) []NodeKey
	dfs = func(key NodeKey) []NodeKey {
		color[key] = gray
		path = append(path, key)
		for _, e := range g.outEdges[key] {
			switch color[e.To] {
			case gray:
				// found cycle
				cycleStart := 0
				for i, n := range path {
					if n == e.To {
						cycleStart = i
						break
					}
				}
				cycle := append([]NodeKey{}, path[cycleStart:]...)
				cycle = append(cycle, e.To)
				return cycle
			case white:
				if c := dfs(e.To); c != nil {
					return c
				}
			}
		}
		path = path[:len(path)-1]
		color[key] = black
		return nil
	}

	for key := range g.nodes {
		if color[key] == white {
			if cycle := dfs(key); cycle != nil {
				return cycle
			}
		}
	}
	return nil
}

// TopologicalSort returns nodes in execution order (entry first).
// Returns an error if the graph has a cycle (should have been caught in Build).
func (g *Graph) TopologicalSort() ([]NodeKey, error) {
	inDegree := make(map[NodeKey]int)
	for key := range g.nodes {
		inDegree[key] = len(g.inEdges[key])
	}

	var queue []NodeKey
	for key, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, key)
		}
	}

	var order []NodeKey
	for len(queue) > 0 {
		// sort queue for deterministic order
		sort.Slice(queue, func(i, j int) bool { return queue[i] < queue[j] })
		curr := queue[0]
		queue = queue[1:]
		order = append(order, curr)

		for _, e := range g.outEdges[curr] {
			inDegree[e.To]--
			if inDegree[e.To] == 0 {
				queue = append(queue, e.To)
			}
		}
	}

	if len(order) != len(g.nodes) {
		return nil, fmt.Errorf("dag: topological sort failed (cycle exists)")
	}
	return order, nil
}
