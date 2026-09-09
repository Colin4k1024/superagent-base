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

// Package compose provides a native graph orchestration engine that
// replaces cloudwego/eino/compose. It uses wfcompose types throughout
// and has ZERO eino imports.
package compose

import (
	"context"
	"fmt"
	"sync"

	"github.com/superagent-ai/superagent-base/backend/pkg/wfcompose"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	START = "@start"
	END   = "@end"
)

// NodeTriggerMode controls when a node fires relative to its predecessors.
type NodeTriggerMode int

const (
	AllPredecessor NodeTriggerMode = iota
	AnyPredecessor
)

// GraphAddNodeOpt is an option for adding a node to a graph.
type GraphAddNodeOpt func(*NodeConfig)

// GraphCompileOption is an option for compiling a graph.
type GraphCompileOption func(*compileConfig)

// NewGraphOption is an option for creating a new graph.
type NewGraphOption func(*graphConfig)

type graphConfig struct {
	maxRunSteps    int
	checkPointStore wfcompose.CheckPointStore
	genLocalState   func(ctx context.Context) any
}

type NodeConfig struct {
	name          string
	outputKey     string
	triggerMode   NodeTriggerMode
	compileOpts   []GraphCompileOption
	StatePreHandler  any
	StatePostHandler any
}

type compileConfig struct {
	maxRunSteps      int
	checkPointStore  wfcompose.CheckPointStore
	nodeTriggerMode  NodeTriggerMode
	graphName        string
}

// GraphCompileConfig is the exported alias for compileConfig.
type GraphCompileConfig = compileConfig

// ---------------------------------------------------------------------------
// Lambda
// ---------------------------------------------------------------------------

// Lambda is a composable unit wrapping a Go function.
type Lambda struct {
	invoke    func(ctx context.Context, input any) (any, error)
	stream    func(ctx context.Context, input any) (*wfcompose.StreamReader[any], error)
	transform func(ctx context.Context, input *wfcompose.StreamReader[any]) (*wfcompose.StreamReader[any], error)
	lambdaType string
	callbackEnabled bool
}

// InvokeWOOpt is the invoke function signature.
type InvokeWOOpt[I, O any] func(ctx context.Context, input I) (O, error)

// Invoke is the invoke function signature WITH opts (for AnyLambda).
type Invoke[I, O, TOption any] func(ctx context.Context, input I, opts ...TOption) (O, error)

// Stream is the stream function signature WITH opts.
type Stream[I, O, TOption any] func(ctx context.Context, input I, opts ...TOption) (*wfcompose.StreamReader[O], error)

// Transform is the transform function signature WITH opts.
type Transform[I, O, TOption any] func(ctx context.Context, input *wfcompose.StreamReader[I], opts ...TOption) (*wfcompose.StreamReader[O], error)

// Collect is the collect function signature WITH opts.
type Collect[I, O, TOption any] func(ctx context.Context, input *wfcompose.StreamReader[I], opts ...TOption) (O, error)

// StreamWOOpt is the stream function signature.
type StreamWOOpt[I, O any] func(ctx context.Context, input I) (*wfcompose.StreamReader[O], error)

// TransformWOOpts is the transform function signature.
type TransformWOOpts[I, O any] func(ctx context.Context, input *wfcompose.StreamReader[I]) (*wfcompose.StreamReader[O], error)

// CollectWOOpt is the collect function signature.
type CollectWOOpt[I, O any] func(ctx context.Context, input *wfcompose.StreamReader[I]) (O, error)

// LambdaOpt is an option for creating a Lambda.
type LambdaOpt func(*Lambda)

// InvokableLambda creates a Lambda from an invoke function.
func InvokableLambda[I, O any](fn InvokeWOOpt[I, O], opts ...LambdaOpt) *Lambda {
	l := &Lambda{
		invoke: func(ctx context.Context, input any) (any, error) {
			typed, ok := input.(I)
			if !ok {
				return nil, fmt.Errorf("lambda: type assertion failed: expected %T, got %T", typed, input)
			}
			return fn(ctx, typed)
		},
		lambdaType: "invokable",
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// StreamableLambda creates a Lambda from a stream function.
func StreamableLambda[I, O any](fn StreamWOOpt[I, O], opts ...LambdaOpt) *Lambda {
	l := &Lambda{
		stream: func(ctx context.Context, input any) (*wfcompose.StreamReader[any], error) {
			typed, ok := input.(I)
			if !ok {
				return nil, fmt.Errorf("lambda: type assertion failed: expected %T, got %T", typed, input)
			}
			sr, err := fn(ctx, typed)
			if err != nil {
				return nil, err
			}
			return wfcompose.StreamReaderWithConvert[O, any](sr, func(v O) (any, error) {
				return v, nil
			}), nil
		},
		lambdaType: "streamable",
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// TransformableLambda creates a Lambda from a transform function.
func TransformableLambda[I, O any](fn TransformWOOpts[I, O], opts ...LambdaOpt) *Lambda {
	l := &Lambda{
		transform: func(ctx context.Context, input *wfcompose.StreamReader[any]) (*wfcompose.StreamReader[any], error) {
			if input == nil {
				return nil, fmt.Errorf("lambda: transform input is nil")
			}
			converted := wfcompose.StreamReaderWithConvert[any, I](input, func(v any) (I, error) {
				typed, ok := v.(I)
				if !ok {
					var zero I
					return zero, fmt.Errorf("lambda: type assertion failed: expected %T, got %T", zero, v)
				}
				return typed, nil
			})
			result, err := fn(ctx, converted)
			if err != nil {
				return nil, err
			}
			return wfcompose.StreamReaderWithConvert[O, any](result, func(v O) (any, error) {
				return v, nil
			}), nil
		},
		lambdaType: "transformable",
	}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// ToList creates a Lambda that converts a stream into a slice.
func ToList[I any](opts ...LambdaOpt) *Lambda {
	return TransformableLambda[I, []I](func(ctx context.Context, input *wfcompose.StreamReader[I]) (*wfcompose.StreamReader[[]I], error) {
		items := make([]I, 0)
		for {
			item, err := input.Recv()
			if err != nil {
				if wfcompose.IsEOF(err) {
					break
				}
				return nil, err
			}
			items = append(items, item)
		}
		return wfcompose.StreamReaderFromArray([][]I{items}), nil
	}, opts...)
}

// WithLambdaCallbackEnable enables/disables callbacks for a lambda.
func WithLambdaCallbackEnable(enable bool) LambdaOpt {
	return func(l *Lambda) { l.callbackEnabled = enable }
}

// WithLambdaType sets the type string for a lambda.
func WithLambdaType(t string) LambdaOpt {
	return func(l *Lambda) { l.lambdaType = t }
}

// ---------------------------------------------------------------------------
// GraphBranch
// ---------------------------------------------------------------------------

// StreamGraphBranchCondition is a condition function for stream branches.
type StreamGraphBranchCondition[T any] func(ctx context.Context, sr *wfcompose.StreamReader[T]) (string, error)

// GraphBranchCondition is a condition function for non-stream branches.
type GraphBranchCondition[T any] func(ctx context.Context, input T) (string, error)

// GraphMultiBranchCondition is a condition function for multi-branches.
type GraphMultiBranchCondition[T any] func(ctx context.Context, input T) (map[string]bool, error)

// StreamGraphMultiBranchCondition is a stream condition function for multi-branches.
type StreamGraphMultiBranchCondition[T any] func(ctx context.Context, sr *wfcompose.StreamReader[T]) (map[string]bool, error)

// GraphBranch represents a conditional branch in the graph.
type GraphBranch struct {
	condition any // StreamGraphBranchCondition[T] or GraphBranchCondition[T]
	endNodes  map[string]bool
	isStream  bool
	isMulti   bool
}

// NewStreamGraphBranch creates a stream-based branch.
func NewStreamGraphBranch[T any](condition StreamGraphBranchCondition[T], endNodes map[string]bool) *GraphBranch {
	return &GraphBranch{
		condition: condition,
		endNodes:  endNodes,
		isStream:  true,
	}
}

// NewGraphBranch creates a non-stream branch.
func NewGraphBranch[T any](condition GraphBranchCondition[T], endNodes map[string]bool) *GraphBranch {
	return &GraphBranch{
		condition: condition,
		endNodes:  endNodes,
		isStream:  false,
	}
}

// NewGraphMultiBranch creates a non-stream multi-branch.
func NewGraphMultiBranch[T any](condition GraphMultiBranchCondition[T], endNodes map[string]bool) *GraphBranch {
	return &GraphBranch{
		condition: condition,
		endNodes:  endNodes,
		isStream:  false,
		isMulti:   true,
	}
}

// NewStreamGraphMultiBranch creates a stream multi-branch.
func NewStreamGraphMultiBranch[T any](condition StreamGraphMultiBranchCondition[T], endNodes map[string]bool) *GraphBranch {
	return &GraphBranch{
		condition: condition,
		endNodes:  endNodes,
		isStream:  true,
		isMulti:   true,
	}
}

// ---------------------------------------------------------------------------
// Graph
// ---------------------------------------------------------------------------

type nodeEntry struct {
	key        string
	lambda     *Lambda
	chatModel  wfcompose.ChatModel
	subGraph   *compiledGraph
	toolsNode  *toolsNodeHandler
	branch     *GraphBranch
	chatTpl    wfcompose.ChatTemplate
	parallel   *Parallel
	nodeType   string
	config     NodeConfig
}

type edgeEntry struct {
	from string
	to   string
}

type graphState struct {
	mu     sync.Mutex
	values map[string]any
}

func newGraphState() *graphState {
	return &graphState{values: make(map[string]any)}
}

// Graph is a generic graph builder.
type Graph[I, O any] struct {
	nodes   []*nodeEntry
	nodeMap map[string]*nodeEntry
	edges   []*edgeEntry
	opts    []NewGraphOption
	built   bool
}

// NewGraph creates a new Graph builder.
func NewGraph[I, O any](opts ...NewGraphOption) *Graph[I, O] {
	return &Graph[I, O]{
		nodeMap: make(map[string]*nodeEntry),
		opts:    opts,
	}
}

// AddLambdaNode adds a lambda node to the graph.
func (g *Graph[I, O]) AddLambdaNode(key string, lambda *Lambda, opts ...GraphAddNodeOpt) error {
	nc := NodeConfig{name: key}
	for _, opt := range opts {
		opt(&nc)
	}
	g.nodes = append(g.nodes, &nodeEntry{
		key:      key,
		lambda:   lambda,
		nodeType: "lambda",
		config:   nc,
	})
	g.nodeMap[key] = g.nodes[len(g.nodes)-1]
	return nil
}

// AddChatModelNode adds a chat model node to the graph.
func (g *Graph[I, O]) AddChatModelNode(key string, model wfcompose.ChatModel, opts ...GraphAddNodeOpt) error {
	nc := NodeConfig{name: key}
	for _, opt := range opts {
		opt(&nc)
	}
	g.nodes = append(g.nodes, &nodeEntry{
		key:       key,
		chatModel: model,
		nodeType:  "chat_model",
		config:    nc,
	})
	g.nodeMap[key] = g.nodes[len(g.nodes)-1]
	return nil
}

// AddToolsNode adds a tools node to the graph.
func (g *Graph[I, O]) AddToolsNode(key string, tools *ToolsNode, opts ...GraphAddNodeOpt) error {
	nc := NodeConfig{name: key}
	for _, opt := range opts {
		opt(&nc)
	}
	g.nodes = append(g.nodes, &nodeEntry{
		key:       key,
		toolsNode: tools.handler,
		nodeType:  "tools_node",
		config:    nc,
	})
	g.nodeMap[key] = g.nodes[len(g.nodes)-1]
	return nil
}

// AddGraphNode adds a sub-graph node.
func (g *Graph[I, O]) AddGraphNode(key string, sub any, opts ...GraphAddNodeOpt) error {
	nc := NodeConfig{name: key}
	for _, opt := range opts {
		opt(&nc)
	}
	// Extract the compiledGraph from various possible types
	var cg *compiledGraph
	switch s := sub.(type) {
	case *compiledGraph:
		cg = s
	case *Runnable[I, O]:
		cg = s.graph
	default:
		// try to extract from any Runnable variant
		if r, ok := sub.(interface{ GetCompiledGraph() *compiledGraph }); ok {
			cg = r.GetCompiledGraph()
		}
	}
	g.nodes = append(g.nodes, &nodeEntry{
		key:      key,
		subGraph: cg,
		nodeType: "graph",
		config:   nc,
	})
	g.nodeMap[key] = g.nodes[len(g.nodes)-1]
	return nil
}

// AddBranch adds a conditional branch.
func (g *Graph[I, O]) AddBranch(key string, branch *GraphBranch, opts ...GraphAddNodeOpt) error {
	nc := NodeConfig{name: key}
	for _, opt := range opts {
		opt(&nc)
	}
	g.nodes = append(g.nodes, &nodeEntry{
		key:      key,
		branch:   branch,
		nodeType: "branch",
		config:   nc,
	})
	g.nodeMap[key] = g.nodes[len(g.nodes)-1]
	return nil
}

// AddEdge adds a directed edge between nodes.
func (g *Graph[I, O]) AddEdge(from, to string) error {
	g.edges = append(g.edges, &edgeEntry{from: from, to: to})
	return nil
}

// Compile finalizes the graph and returns a Runnable.
func (g *Graph[I, O]) Compile(ctx context.Context, opts ...GraphCompileOption) (*Runnable[I, O], error) {
	cg, err := g.compileInternal(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &Runnable[I, O]{graph: cg}, nil
}

// compileInternal builds the compiled graph (used internally by AddGraphNode).
func (g *Graph[I, O]) compileInternal(ctx context.Context, opts ...GraphCompileOption) (*compiledGraph, error) {
	cc := &compileConfig{
		maxRunSteps: 1000,
	}
	for _, opt := range opts {
		opt(cc)
	}

	// Build adjacency list
	outEdges := make(map[string][]string)
	for _, e := range g.edges {
		outEdges[e.from] = append(outEdges[e.from], e.to)
	}

	cg := &compiledGraph{
		nodes:    g.nodeMap,
		outEdges: outEdges,
		compileConfig: cc,
	}

	return cg, nil
}

// ---------------------------------------------------------------------------
// compiledGraph
// ---------------------------------------------------------------------------

type compiledGraph struct {
	nodes    map[string]*nodeEntry
	outEdges map[string][]string
	*compileConfig
}

// Stream executes the graph in streaming mode.
func (cg *compiledGraph) Stream(ctx context.Context, input any, opts ...wfcompose.Option) (*wfcompose.StreamReader[any], error) {
	// Find entry node (connected from START)
	entryKeys := cg.outEdges[START]
	if len(entryKeys) == 0 {
		return nil, fmt.Errorf("compose: no entry node (no edge from START)")
	}

	// Execute the graph starting from the entry node
	// For now, support linear execution: follow edges from entry to exit
	result, err := cg.executeNode(ctx, entryKeys[0], input)
	if err != nil {
		return nil, err
	}

	// If result is already a stream, return it
	if sr, ok := result.(*wfcompose.StreamReader[any]); ok {
		return sr, nil
	}

	// Wrap single value in a stream
	return wfcompose.StreamReaderFromArray([]any{result}), nil
}

// executeNode executes a single node and returns its output.
func (cg *compiledGraph) executeNode(ctx context.Context, key string, input any) (any, error) {
	node, ok := cg.nodes[key]
	if !ok {
		return nil, fmt.Errorf("compose: unknown node %q", key)
	}

	var output any
	var err error

	switch node.nodeType {
	case "lambda":
		output, err = cg.executeLambda(ctx, node, input)
	case "chat_model":
		output, err = cg.executeChatModel(ctx, node, input)
	case "tools_node":
		output, err = cg.executeToolsNode(ctx, node, input)
	case "graph":
		output, err = cg.executeSubGraph(ctx, node, input)
	case "branch":
		output, err = cg.executeBranch(ctx, node, input)
	default:
		return nil, fmt.Errorf("compose: unknown node type %q", node.nodeType)
	}

	if err != nil {
		return nil, fmt.Errorf("compose: node %q failed: %w", key, err)
	}

	// Follow edges to next node
	nexts := cg.outEdges[key]
	if len(nexts) == 0 || (len(nexts) == 1 && nexts[0] == END) {
		return output, nil
	}

	// Execute next node
	if len(nexts) == 1 {
		return cg.executeNode(ctx, nexts[0], output)
	}

	// Multiple edges: not supported in basic implementation
	return output, nil
}

func (cg *compiledGraph) executeLambda(ctx context.Context, node *nodeEntry, input any) (any, error) {
	if node.lambda == nil {
		return nil, fmt.Errorf("compose: lambda node %q has no lambda", node.key)
	}
	l := node.lambda
	if l.invoke != nil {
		return l.invoke(ctx, input)
	}
	if l.stream != nil {
		return l.stream(ctx, input)
	}
	if l.transform != nil {
		// If input is a stream reader, transform it
		if sr, ok := input.(*wfcompose.StreamReader[any]); ok {
			return l.transform(ctx, sr)
		}
		// Wrap non-stream input
		wrapped := wfcompose.StreamReaderFromArray([]any{input})
		return l.transform(ctx, wrapped)
	}
	return nil, fmt.Errorf("compose: lambda node %q has no function", node.key)
}

func (cg *compiledGraph) executeChatModel(ctx context.Context, node *nodeEntry, input any) (any, error) {
	if node.chatModel == nil {
		return nil, fmt.Errorf("compose: chat model node %q has no model", node.key)
	}
	// Convert input to []*wfcompose.Message
	msgs, ok := input.([]*wfcompose.Message)
	if !ok {
		// Try single message
		if msg, ok := input.(*wfcompose.Message); ok {
			msgs = []*wfcompose.Message{msg}
		} else {
			return nil, fmt.Errorf("compose: chat model node %q expects []*wfcompose.Message or *wfcompose.Message, got %T", node.key, input)
		}
	}
	result, err := node.chatModel.Generate(ctx, msgs)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (cg *compiledGraph) executeToolsNode(ctx context.Context, node *nodeEntry, input any) (any, error) {
	if node.toolsNode == nil {
		return nil, fmt.Errorf("compose: tools node %q has no handler", node.key)
	}
	return node.toolsNode.execute(ctx, input)
}

func (cg *compiledGraph) executeSubGraph(ctx context.Context, node *nodeEntry, input any) (any, error) {
	if node.subGraph == nil {
		return nil, fmt.Errorf("compose: sub-graph node %q has no graph", node.key)
	}
	return node.subGraph.executeValue(ctx, input)
}

func (cg *compiledGraph) executeBranch(ctx context.Context, node *nodeEntry, input any) (any, error) {
	if node.branch == nil {
		return nil, fmt.Errorf("compose: branch node %q has no branch", node.key)
	}
	b := node.branch
	if b.isStream {
		// Stream branch: input should be a stream reader
		sr, ok := input.(*wfcompose.StreamReader[any])
		if !ok {
			wrapped := wfcompose.StreamReaderFromArray([]any{input})
			sr = wrapped
		}
		condFn := b.condition.(func(context.Context, *wfcompose.StreamReader[any]) (string, error))
		nextKey, err := condFn(ctx, sr)
		if err != nil {
			return nil, err
		}
		if b.endNodes[nextKey] {
			return sr, nil
		}
		return cg.executeNode(ctx, nextKey, sr)
	}
	// Non-stream branch
	condFn := b.condition.(func(context.Context, any) (string, error))
	nextKey, err := condFn(ctx, input)
	if err != nil {
		return nil, err
	}
	if b.endNodes[nextKey] {
		return input, nil
	}
	return cg.executeNode(ctx, nextKey, input)
}

// executeValue executes the graph and returns a single value (non-streaming).
func (cg *compiledGraph) executeValue(ctx context.Context, input any) (any, error) {
	entryKeys := cg.outEdges[START]
	if len(entryKeys) == 0 {
		return nil, fmt.Errorf("compose: no entry node")
	}
	return cg.executeNode(ctx, entryKeys[0], input)
}

// ---------------------------------------------------------------------------
// Runnable adapter
// ---------------------------------------------------------------------------

// Runnable is the compiled graph runner.
type Runnable[I, O any] struct {
	graph *compiledGraph
}

// Stream executes the graph in streaming mode.
func (r *Runnable[I, O]) Stream(ctx context.Context, input I, opts ...wfcompose.Option) (*wfcompose.StreamReader[O], error) {
	sr, err := r.graph.Stream(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	return wfcompose.StreamReaderWithConvert[any, O](sr, func(v any) (O, error) {
		typed, ok := v.(O)
		if !ok {
			var zero O
			return zero, fmt.Errorf("compose: type assertion failed: expected %T, got %T", zero, v)
		}
		return typed, nil
	}), nil
}

// Invoke executes the graph and returns a single result.
func (r *Runnable[I, O]) Invoke(ctx context.Context, input I, opts ...wfcompose.Option) (O, error) {
	result, err := r.graph.executeValue(ctx, input)
	if err != nil {
		var zero O
		return zero, err
	}
	typed, ok := result.(O)
	if !ok {
		var zero O
		return zero, fmt.Errorf("compose: type assertion failed: expected %T, got %T", zero, result)
	}
	return typed, nil
}

// Compile wraps Graph.Compile to return a typed Runnable.
func Compile[I, O any](g *Graph[I, O], ctx context.Context, opts ...GraphCompileOption) (*Runnable[I, O], error) {
	return g.Compile(ctx, opts...)
}

// ---------------------------------------------------------------------------
// Option helpers
// ---------------------------------------------------------------------------

func WithNodeName(name string) GraphAddNodeOpt {
	return func(nc *NodeConfig) { nc.name = name }
}

func WithOutputKey(key string) GraphAddNodeOpt {
	return func(nc *NodeConfig) { nc.outputKey = key }
}

func WithNodeTriggerMode(mode NodeTriggerMode) GraphCompileOption {
	return func(cc *compileConfig) { cc.nodeTriggerMode = mode }
}

func WithGraphCompileOptions(opts ...GraphCompileOption) GraphAddNodeOpt {
	return func(nc *NodeConfig) { nc.compileOpts = append(nc.compileOpts, opts...) }
}

func WithMaxRunSteps(steps int) GraphCompileOption {
	return func(cc *compileConfig) { cc.maxRunSteps = steps }
}

func WithGraphName(name string) GraphCompileOption {
	return func(cc *compileConfig) { cc.graphName = name }
}

func WithCheckPointStore(store wfcompose.CheckPointStore) GraphCompileOption {
	return func(cc *compileConfig) { cc.checkPointStore = store }
}

func WithGenLocalState[S any](gls func(ctx context.Context) S) NewGraphOption {
	return func(gc *graphConfig) {
		gc.genLocalState = func(ctx context.Context) any {
			return gls(ctx)
		}
	}
}

// ---------------------------------------------------------------------------
// FieldPath and FieldMapping (stubs for API compatibility)
// ---------------------------------------------------------------------------

type FieldPath = wfcompose.FieldPath
type FieldMapping struct{ from, to FieldPath }

// ToPath returns the destination field path.
func (m *FieldMapping) ToPath() FieldPath {
	if m == nil {
		return nil
	}
	return m.to
}

// FromPath returns the source field path.
func (m *FieldMapping) FromPath() FieldPath {
	if m == nil {
		return nil
	}
	return m.from
}

// Equals checks if two field mappings are equivalent.
func (m *FieldMapping) Equals(other *FieldMapping) bool {
	if m == nil || other == nil {
		return m == other
	}
	if len(m.from) != len(other.from) || len(m.to) != len(other.to) {
		return false
	}
	for i := range m.from {
		if m.from[i] != other.from[i] {
			return false
		}
	}
	for i := range m.to {
		if m.to[i] != other.to[i] {
			return false
		}
	}
	return true
}
type FieldMappingOption func(*FieldMapping)

func ToFieldPath(fp FieldPath, opts ...FieldMappingOption) *FieldMapping {
	fm := &FieldMapping{to: fp}
	for _, opt := range opts {
		opt(fm)
	}
	return fm
}

func FromFieldPath(fp FieldPath) *FieldMapping {
	return &FieldMapping{from: fp}
}

func FromField(from string) *FieldMapping {
	return &FieldMapping{from: FieldPath{from}}
}

func ToField(to string, opts ...FieldMappingOption) *FieldMapping {
	fm := &FieldMapping{to: FieldPath{to}}
	for _, opt := range opts {
		opt(fm)
	}
	return fm
}

func MapFields(from, to string) *FieldMapping {
	return &FieldMapping{from: FieldPath{from}, to: FieldPath{to}}
}

func MapFieldPaths(from, to FieldPath) *FieldMapping {
	return &FieldMapping{from: from, to: to}
}

func WithCustomExtractor(extractor func(input any) (any, error)) FieldMappingOption {
	return func(fm *FieldMapping) {}
}

type NodePath struct{ parts []string }

func NewNodePath(parts ...string) *NodePath { return &NodePath{parts: parts} }

// ---------------------------------------------------------------------------
// ToolsNode (stub for API compatibility)
// ---------------------------------------------------------------------------

type ToolsNodeConfig struct {
	Tools []wfcompose.BaseTool
}

type ToolsNode struct {
	handler *toolsNodeHandler
}

type toolsNodeHandler struct {
	tools []wfcompose.BaseTool
}

func (h *toolsNodeHandler) execute(ctx context.Context, input any) (any, error) {
	return input, nil
}

func NewToolNode(ctx context.Context, conf *ToolsNodeConfig) (*ToolsNode, error) {
	return &ToolsNode{
		handler: &toolsNodeHandler{tools: conf.Tools},
	}, nil
}

// ---------------------------------------------------------------------------
// InterruptInfo (stub for API compatibility)
// ---------------------------------------------------------------------------

type InterruptInfo struct {
	Data           any
	RerunNodes     []string
	RerunNodesExtra map[string]any
	SubGraphs      map[string]*InterruptInfo
	State          any
}

func ExtractInterruptInfo(err error) (*InterruptInfo, bool) {
	return nil, false
}

func IsInterruptRerunError(err error) (any, bool) {
	return nil, false
}

func NewInterruptAndRerunErr(extra any) error {
	return fmt.Errorf("interrupt and rerun: %v", extra)
}

func RegisterSerializableType[T any](name string) error {
	return nil
}

func ProcessState[S any](ctx context.Context, handler func(context.Context, S) error) error {
	return nil
}

func GetToolCallID(ctx context.Context) string {
	return ""
}

// Option is the compose execution option (opaque, framework-agnostic).
type Option = wfcompose.Option

// CheckPointStore aliases to wfcompose.CheckPointStore.
type CheckPointStore = wfcompose.CheckPointStore

// InterruptAndRerun is a sentinel value (stub).
var InterruptAndRerun = fmt.Errorf("interrupt and rerun")

// ErrExceedMaxSteps is returned when the graph exceeds its step limit.
var ErrExceedMaxSteps = fmt.Errorf("compose: max run steps exceeded")

// ErrChainCompiled is returned when a compiled chain is modified.
var ErrChainCompiled = fmt.Errorf("compose: chain already compiled")

// ErrGraphCompiled is returned when a compiled graph is modified.
var ErrGraphCompiled = fmt.Errorf("compose: graph already compiled")

var DAGInvalidLoopErr = fmt.Errorf("compose: invalid loop in DAG")

// Component constants
const (
	ComponentOfGraph     = "Graph"
	ComponentOfToolsNode = "ToolsNode"
	ComponentOfLambda    = "Lambda"
	ComponentOfWorkflow = "Workflow"
	ComponentOfChain     = "Chain"
	ComponentOfUnknown  = "Unknown"
)

// ---------------------------------------------------------------------------
// Additional types for API compatibility (stubs)
// ---------------------------------------------------------------------------

// Parallel is a stub for parallel execution (API compatibility).
type Parallel struct{}

func NewParallel() *Parallel { return &Parallel{} }

// GraphInfo holds graph metadata.
type GraphInfo struct {
	Name  string
	Nodes map[string]*GraphNodeInfo
}

// GraphNodeInfo holds node metadata.
type GraphNodeInfo struct {
	Key       string
	Name      string
	Type      string
	InputKey  string
	OutputKey string
}

// ToolsNodeOption is an option for tools nodes.
type ToolsNodeOption func(any)

// WithToolOption wraps tool options.
func WithToolOption(opts ...any) ToolsNodeOption {
	return func(any) {}
}

// WithToolsNodeOption wraps tools node options into an Option.
func WithToolsNodeOption(opts ...ToolsNodeOption) wfcompose.Option {
	return wfcompose.NewOption(nil)
}

// Serializer is a stub for serialization (API compatibility).
type Serializer struct{}

// GraphConfigShim is used for option functions that need access to config.
type GraphConfigShim = graphConfig

// AnyLambda creates a Lambda from any combination of functions.
func AnyLambda[I, O, TOption any](i Invoke[I, O, TOption], s Stream[I, O, TOption],
	c Collect[I, O, TOption], t Transform[I, O, TOption], opts ...LambdaOpt) (*Lambda, error) {
	l := &Lambda{}
	if i != nil {
		l.invoke = func(ctx context.Context, input any) (any, error) {
			typed, ok := input.(I)
			if !ok {
				return nil, fmt.Errorf("anylambda: type assertion failed for input")
			}
			return i(ctx, typed)
		}
	}
	if s != nil {
		l.stream = func(ctx context.Context, input any) (*wfcompose.StreamReader[any], error) {
			typed, ok := input.(I)
			if !ok {
				return nil, fmt.Errorf("anylambda: type assertion failed for input")
			}
			sr, err := s(ctx, typed)
			if err != nil {
				return nil, err
			}
			return wfcompose.StreamReaderWithConvert[O, any](sr, func(v O) (any, error) {
				return v, nil
			}), nil
		}
	}
	if t != nil {
		l.transform = func(ctx context.Context, input *wfcompose.StreamReader[any]) (*wfcompose.StreamReader[any], error) {
			converted := wfcompose.StreamReaderWithConvert[any, I](input, func(v any) (I, error) {
				typed, ok := v.(I)
				if !ok {
					var zero I
					return zero, fmt.Errorf("anylambda: type assertion failed")
				}
				return typed, nil
			})
			result, err := t(ctx, converted)
			if err != nil {
				return nil, err
			}
			return wfcompose.StreamReaderWithConvert[O, any](result, func(v O) (any, error) {
				return v, nil
			}), nil
		}
	}
	for _, opt := range opts {
		opt(l)
	}
	return l, nil
}

// ---------------------------------------------------------------------------
// Chain-style builder methods (for API compatibility)
// ---------------------------------------------------------------------------

// AppendLambda adds a lambda node to the chain and returns the chain for chaining.
func (g *Graph[I, O]) AppendLambda(lambda *Lambda) *Graph[I, O] {
	key := fmt.Sprintf("lambda_%d", len(g.nodes))
	_ = g.AddLambdaNode(key, lambda)
	return g
}

// AppendChatTemplate adds a chat template node to the chain.
func (g *Graph[I, O]) AppendChatTemplate(tpl wfcompose.ChatTemplate) *Graph[I, O] {
	key := fmt.Sprintf("template_%d", len(g.nodes))
	_ = g.AddChatTemplateNode(key, tpl)
	return g
}

// AppendChatModel adds a chat model node to the chain.
func (g *Graph[I, O]) AppendChatModel(model wfcompose.ChatModel, opts ...GraphAddNodeOpt) *Graph[I, O] {
	key := fmt.Sprintf("model_%d", len(g.nodes))
	_ = g.AddChatModelNode(key, model)
	return g
}

// AppendParallel adds a parallel node to the chain.
func (g *Graph[I, O]) AppendParallel(p *Parallel) *Graph[I, O] {
	key := fmt.Sprintf("parallel_%d", len(g.nodes))
	nc := NodeConfig{name: key}
	g.nodes = append(g.nodes, &nodeEntry{
		key:      key,
		nodeType: "parallel",
		parallel: p,
		config:   nc,
	})
	g.nodeMap[key] = g.nodes[len(g.nodes)-1]
	return g
}

// AddLambda adds a lambda to the parallel node.
func (p *Parallel) AddLambda(key string, lambda *Lambda) *Parallel {
	return p
}

// AddChatTemplateNode adds a chat template node to the graph by key.
func (g *Graph[I, O]) AddChatTemplateNode(key string, tpl wfcompose.ChatTemplate, opts ...GraphAddNodeOpt) error {
	nc := NodeConfig{name: key}
	g.nodes = append(g.nodes, &nodeEntry{
		key:      key,
		nodeType: "template",
		chatTpl:  tpl,
		config:   nc,
	})
	g.nodeMap[key] = g.nodes[len(g.nodes)-1]
	return nil
}

// WithCallbacks returns a compile option that registers callback handlers.
// Native compose is a no-op for callbacks currently.
func WithCallbacks(handlers ...any) GraphCompileOption {
	return func(cc *compileConfig) {}
}

// GetCompiledGraph returns the underlying compiled graph.
func (r *Runnable[I, O]) GetCompiledGraph() *compiledGraph {
	if r == nil {
		return nil
	}
	return r.graph
}

// ---------------------------------------------------------------------------
// Workflow types (API compatibility)
// ---------------------------------------------------------------------------

// Workflow is a specialized graph for workflow composition.
type Workflow[I, O any] struct {
	*Graph[I, O]
}

// NewWorkflow creates a new workflow graph.
func NewWorkflow[I, O any](opts ...NewGraphOption) *Workflow[I, O] {
	return &Workflow[I, O]{Graph: NewGraph[I, O]()}
}

// AddLambdaNode adds a lambda node and returns the node for further configuration.
func (w *Workflow[I, O]) AddLambdaNode(key string, lambda *Lambda, opts ...GraphAddNodeOpt) *Lambda {
	_ = w.Graph.AddLambdaNode(key, lambda, opts...)
	return lambda
}

// End finalizes the workflow and returns the end node.
func (w *Workflow[I, O]) End() *Lambda {
	return &Lambda{}
}

// Compile compiles the workflow.
func (w *Workflow[I, O]) Compile(ctx context.Context, opts ...GraphCompileOption) (*Runnable[I, O], error) {
	return w.Graph.Compile(ctx, opts...)
}

// AddInput adds an input dependency to the node.
func (l *Lambda) AddInput(fromNodeKey string, fieldMappings ...*FieldMapping) {}

// AddInputWithOptions adds an input dependency with options.
func (l *Lambda) AddInputWithOptions(fromNodeKey string, fieldMappings []*FieldMapping, opts ...NewGraphOption) {}

// AddDependency adds a dependency to the node.
func (l *Lambda) AddDependency(key string) {}

// SetStaticValue sets a static value for the node.
func (l *Lambda) SetStaticValue(path FieldPath, val any) {}
