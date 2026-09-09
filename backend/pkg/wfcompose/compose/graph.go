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
	"reflect"
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

// AddBranch adds a conditional branch to an existing node, preserving its lambda.
func (g *Graph[I, O]) AddBranch(key string, branch *GraphBranch, opts ...GraphAddNodeOpt) error {
	// Apply options to the existing node's config
	if existing, ok := g.nodeMap[key]; ok {
		existing.branch = branch
		for _, opt := range opts {
			opt(&existing.config)
		}
		return nil
	}
	// Node doesn't exist yet: create a branch-only entry
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

	// Evaluate graph options (NewGraphOption) to get genLocalState
	gc := &graphConfig{}
	for _, opt := range g.opts {
		opt(gc)
	}

	cg := &compiledGraph{
		nodes:         g.nodeMap,
		outEdges:      outEdges,
		genLocalState: gc.genLocalState,
		compileConfig: cc,
	}

	return cg, nil
}

// ---------------------------------------------------------------------------
// compiledGraph
// ---------------------------------------------------------------------------

type staticValueEntry struct {
	path FieldPath
	val  any
}

type compiledGraph struct {
	nodes         map[string]*nodeEntry
	outEdges      map[string][]string
	fieldMappings map[string][]depEntry
	staticValues  map[string][]staticValueEntry
	genLocalState func(ctx context.Context) any
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
	result, err := cg.executeGraph(ctx, input)
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

	return output, nil
}

func (cg *compiledGraph) executeLambda(ctx context.Context, node *nodeEntry, input any) (any, error) {
	if node.lambda == nil {
		return nil, fmt.Errorf("compose: lambda node %q has no lambda", node.key)
	}

	// State is already set up by executeChain (shared across all nodes)

	// Call state pre-handler if configured (using reflection for generic state type)
	if node.config.StatePreHandler != nil {
		state := ctx.Value(stateKey{})
		if state != nil {
			preVal := reflect.ValueOf(node.config.StatePreHandler)
			if preVal.Kind() == reflect.Func {
				args := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(input), reflect.ValueOf(state)}
				results := preVal.Call(args)
				if len(results) >= 2 && !results[1].IsNil() {
					errVal := results[1].Interface()
					if err, ok := errVal.(error); ok {
						return nil, fmt.Errorf("compose: state pre-handler for node %q failed: %w", node.key, err)
					}
				}
				if len(results) >= 1 && !results[0].IsNil() {
					if newInput, ok := results[0].Interface().(map[string]any); ok {
						input = newInput
					}
				}
			}
		}
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
	selected, err := cg.evalBranchCondition(ctx, node, input)
	if err != nil {
		return nil, err
	}
	// For backward compatibility: execute the first selected non-end target
	b := node.branch
	for target, shouldExec := range selected {
		if !shouldExec {
			continue
		}
		if b.endNodes[target] {
			return input, nil
		}
		return cg.executeNode(ctx, target, input)
	}
	return input, nil
}

// executeValue executes the graph and returns a single value (non-streaming).
func (cg *compiledGraph) executeValue(ctx context.Context, input any) (any, error) {
	entryKeys := cg.outEdges[START]
	if len(entryKeys) == 0 {
		return nil, fmt.Errorf("compose: no entry node")
	}
	return cg.executeGraph(ctx, input)
}

// executeGraph runs the full DAG using BFS, handling branches and convergence.
func (cg *compiledGraph) executeGraph(ctx context.Context, input any) (any, error) {
	// Create state once for the entire graph execution
	if cg.genLocalState != nil {
		state := cg.genLocalState(ctx)
		if state != nil {
			ctx = context.WithValue(ctx, stateKey{}, state)
		}
	}

	// Build inEdges from outEdges
	inEdges := make(map[string][]string)
	for from, tos := range cg.outEdges {
		for _, to := range tos {
			if to == END {
				continue
			}
			inEdges[to] = append(inEdges[to], from)
		}
	}

	nodeOutputs := make(map[string]any)
	executed := make(map[string]bool)

	// Store raw input under START so field mappings with fromKey=START can resolve
	nodeOutputs[START] = input
	executed[START] = true

	// BFS queue: all nodes directly connected from START
	queue := make([]string, 0)
	for _, next := range cg.outEdges[START] {
		if next != END {
			queue = append(queue, next)
		}
	}

	var lastOutput any = input
	var output any
	var err error
	maxIterations := 10000

	for len(queue) > 0 && maxIterations > 0 {
		maxIterations--
		progress := false
		var deferred []string

		for len(queue) > 0 {
			nextKey := queue[0]
			queue = queue[1:]

			if nextKey == END || executed[nextKey] {
				continue
			}

			// Check if all predecessors are executed
			allPredsDone := true
			for _, pred := range inEdges[nextKey] {
				if !executed[pred] {
					allPredsDone = false
					break
				}
			}

			if !allPredsDone {
				deferred = append(deferred, nextKey)
				continue
			}

			progress = true
			resolvedInput := cg.resolveAllInputs(nextKey, nodeOutputs, inEdges)

			// Handle branch node specially
			node := cg.nodes[nextKey]
			if node != nil && node.branch != nil {
				// If the node also has a lambda, execute it first
				var branchInput any = resolvedInput
				if node.lambda != nil {
					lambdaOutput, err := cg.executeLambda(ctx, node, resolvedInput)
					if err != nil {
						return nil, fmt.Errorf("compose: node %q failed: %w", nextKey, err)
					}
					nodeOutputs[nextKey] = lambdaOutput
					lastOutput = lambdaOutput
					branchInput = lambdaOutput
				} else {
					nodeOutputs[nextKey] = resolvedInput
					lastOutput = resolvedInput
				}

				selected, err := cg.evalBranchCondition(ctx, node, branchInput)
				if err != nil {
					return nil, fmt.Errorf("compose: node %q failed: %w", nextKey, err)
				}
				executed[nextKey] = true
				cg.markExecuted(ctx, nextKey)

				for target, shouldExec := range selected {
					if shouldExec && !executed[target] && target != END {
						queue = append(queue, target)
					}
				}
				for _, next := range cg.outEdges[nextKey] {
					if !executed[next] && next != END {
						queue = append(queue, next)
					}
				}
				continue
			}

			// Execute regular node
			output, err = cg.executeNode(ctx, nextKey, resolvedInput)
			if err != nil {
				return nil, err
			}
			nodeOutputs[nextKey] = output
			executed[nextKey] = true
			cg.markExecuted(ctx, nextKey)
			lastOutput = output

			for _, next := range cg.outEdges[nextKey] {
				if !executed[next] && next != END {
					queue = append(queue, next)
				}
			}
		}

		if !progress && len(deferred) > 0 {
			// Force-execute deferred nodes (predecessors that will never run)
			forceKey := deferred[0]
			deferred = deferred[1:]
			progress = true

			resolvedInput := cg.resolveAllInputs(forceKey, nodeOutputs, inEdges)

			node := cg.nodes[forceKey]
			if node != nil && node.branch != nil {
				var branchInput any = resolvedInput
				if node.lambda != nil {
					lambdaOutput, err := cg.executeLambda(ctx, node, resolvedInput)
					if err != nil {
						return nil, fmt.Errorf("compose: node %q failed: %w", forceKey, err)
					}
					nodeOutputs[forceKey] = lambdaOutput
					lastOutput = lambdaOutput
					branchInput = lambdaOutput
				} else {
					nodeOutputs[forceKey] = resolvedInput
					lastOutput = resolvedInput
				}

				selected, err := cg.evalBranchCondition(ctx, node, branchInput)
				if err != nil {
					return nil, fmt.Errorf("compose: node %q failed: %w", forceKey, err)
				}
				executed[forceKey] = true
				cg.markExecuted(ctx, forceKey)

				for target, shouldExec := range selected {
					if shouldExec && !executed[target] && target != END {
						queue = append(queue, target)
					}
				}
				for _, next := range cg.outEdges[forceKey] {
					if !executed[next] && next != END {
						queue = append(queue, next)
					}
				}
			} else {
				output, err = cg.executeNode(ctx, forceKey, resolvedInput)
				if err != nil {
					return nil, err
				}
				nodeOutputs[forceKey] = output
				executed[forceKey] = true
				cg.markExecuted(ctx, forceKey)
				lastOutput = output

				for _, next := range cg.outEdges[forceKey] {
					if !executed[next] && next != END {
						queue = append(queue, next)
					}
				}
			}
		}

		queue = deferred
	}

	// If END has field mappings, resolve and return collected outputs
	if cg.fieldMappings != nil {
		if _, ok := cg.fieldMappings[END]; ok {
			return cg.resolveAllInputs(END, nodeOutputs, inEdges), nil
		}
	}

	// Return output of the node that connects to END
	for key, out := range nodeOutputs {
		for _, next := range cg.outEdges[key] {
			if next == END {
				return out, nil
			}
		}
	}

	return lastOutput, nil
}

// resolveAllInputs resolves field mappings from ALL executed predecessors.
func (cg *compiledGraph) resolveAllInputs(nodeKey string, nodeOutputs map[string]any, inEdges map[string][]string) map[string]any {
	result := make(map[string]any)

	if cg.fieldMappings != nil {
		if entries, ok := cg.fieldMappings[nodeKey]; ok {
			for _, entry := range entries {
				predOutput, exists := nodeOutputs[entry.fromKey]
				if !exists {
					continue
				}
				predMap, ok := predOutput.(map[string]any)
				if !ok {
					result[entry.fromKey] = predOutput
					continue
				}
				if len(entry.fieldMappings) == 0 {
					for k, v := range predMap {
						result[k] = v
					}
				} else {
					for _, fm := range entry.fieldMappings {
						val, found := getNestedValue(predMap, fm.from)
						if found {
							setNestedValue(result, fm.to, val)
						}
					}
				}
			}
			// Apply static values
			if cg.staticValues != nil {
				if svs, ok := cg.staticValues[nodeKey]; ok {
					for _, sv := range svs {
						setNestedValue(result, sv.path, sv.val)
					}
				}
			}
			return result
		}
	}

	// No field mappings: merge all predecessor outputs using inEdges
	for _, predKey := range inEdges[nodeKey] {
		predOutput, exists := nodeOutputs[predKey]
		if !exists {
			continue
		}
		predMap, ok := predOutput.(map[string]any)
		if !ok {
			result[predKey] = predOutput
			continue
		}
		for k, v := range predMap {
			result[k] = v
		}
	}
	// Apply static values
	if cg.staticValues != nil {
		if svs, ok := cg.staticValues[nodeKey]; ok {
			for _, sv := range svs {
				setNestedValue(result, sv.path, sv.val)
			}
		}
	}
	return result
}

// evalBranchCondition evaluates a branch condition using reflection.
func (cg *compiledGraph) evalBranchCondition(ctx context.Context, node *nodeEntry, input any) (map[string]bool, error) {
	b := node.branch
	if b == nil {
		return nil, fmt.Errorf("compose: branch node %q has no branch", node.key)
	}

	if b.isMulti {
		if b.isStream {
			sr, ok := input.(*wfcompose.StreamReader[any])
			if !ok {
				sr = wfcompose.StreamReaderFromArray([]any{input})
			}
			condVal := reflect.ValueOf(b.condition)
			results := condVal.Call([]reflect.Value{
				reflect.ValueOf(ctx),
				reflect.ValueOf(sr),
			})
			selected, ok := results[0].Interface().(map[string]bool)
			if !ok {
				return nil, fmt.Errorf("compose: multi-branch condition returned non-map result")
			}
			var err error
			if !results[1].IsNil() {
				err = results[1].Interface().(error)
			}
			return selected, err
		}
		condVal := reflect.ValueOf(b.condition)
		results := condVal.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(input),
		})
		selected, ok := results[0].Interface().(map[string]bool)
		if !ok {
			return nil, fmt.Errorf("compose: multi-branch condition returned non-map result")
		}
		var err error
		if !results[1].IsNil() {
			err = results[1].Interface().(error)
		}
		return selected, err
	}

	// Single branch
	var nextKey string
	var err error
	var ok bool
	if b.isStream {
		var sr *wfcompose.StreamReader[any]
		sr, ok = input.(*wfcompose.StreamReader[any])
		if !ok {
			sr = wfcompose.StreamReaderFromArray([]any{input})
		}
		condVal := reflect.ValueOf(b.condition)
		results := condVal.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(sr),
		})
		nextKey, ok = results[0].Interface().(string)
		if !ok {
			return nil, fmt.Errorf("compose: branch condition returned non-string result")
		}
		if !results[1].IsNil() {
			err = results[1].Interface().(error)
		}
	} else {
		condVal := reflect.ValueOf(b.condition)
		results := condVal.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(input),
		})
		nextKey, ok = results[0].Interface().(string)
		if !ok {
			return nil, fmt.Errorf("compose: branch condition returned non-string result")
		}
		if !results[1].IsNil() {
			err = results[1].Interface().(error)
		}
	}
	return map[string]bool{nextKey: true}, err
}

// markExecuted marks a node as executed in the state (using reflection).
func (cg *compiledGraph) markExecuted(ctx context.Context, key string) {
	state := ctx.Value(stateKey{})
	if state == nil {
		return
	}
	// Use reflection to set ExecutedNodes[key] = true
	v := reflect.ValueOf(state)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	executedField := v.FieldByName("ExecutedNodes")
	if !executedField.IsValid() {
		return
	}
	if executedField.Kind() != reflect.Map {
		return
	}
	// Set key=true in the map
	keyVal := reflect.ValueOf(key)
	boolVal := reflect.ValueOf(true)
	if !keyVal.Type().AssignableTo(executedField.Type().Key()) {
		// Try converting key to the map's key type
		keyVal = keyVal.Convert(executedField.Type().Key())
	}
	executedField.SetMapIndex(keyVal, boolVal)
}

// resolveInput resolves field mappings from predecessor output to node input.
func (cg *compiledGraph) resolveInput(nodeKey, predKey string, predOutput any) map[string]any {
	result := make(map[string]any)
	if cg.fieldMappings == nil {
		// No field mappings: pass entire predecessor output
		result[predKey] = predOutput
		return result
	}

	entries, ok := cg.fieldMappings[nodeKey]
	if !ok {
		result[predKey] = predOutput
		return result
	}

	predMap, ok := predOutput.(map[string]any)
	if !ok {
		result[predKey] = predOutput
		return result
	}

	for _, entry := range entries {
		if entry.fromKey != predKey {
			continue
		}
		if len(entry.fieldMappings) == 0 {
			// No specific mappings: merge entire predecessor output
			for k, v := range predMap {
				result[k] = v
			}
		} else {
			// Apply specific field mappings
			for _, fm := range entry.fieldMappings {
				fromFields := fm.from
				toFields := fm.to
				val, found := getNestedValue(predMap, fromFields)
				if found {
					setNestedValue(result, toFields, val)
				}
			}
		}
	}
	return result
}

// getNestedValue retrieves a value from a nested map using a field path.
func getNestedValue(m map[string]any, path FieldPath) (any, bool) {
	if len(path) == 0 {
		return nil, false
	}
	v, ok := m[path[0]]
	if !ok {
		return nil, false
	}
	for i := 1; i < len(path); i++ {
		m2, ok := v.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok = m2[path[i]]
		if !ok {
			return nil, false
		}
	}
	return v, true
}

// setNestedValue sets a value in a nested map using a field path.
func setNestedValue(m map[string]any, path FieldPath, val any) {
	if len(path) == 0 {
		return
	}
	if len(path) == 1 {
		m[path[0]] = val
		return
	}
	sub, ok := m[path[0]].(map[string]any)
	if !ok {
		sub = make(map[string]any)
		m[path[0]] = sub
	}
	setNestedValue(sub, path[1:], val)
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

type stateKey struct{}

func ProcessState[S any](ctx context.Context, handler func(context.Context, S) error) error {
	v := ctx.Value(stateKey{})
	if v == nil {
		return nil
	}
	s, ok := v.(S)
	if !ok {
		return nil
	}
	return handler(ctx, s)
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
	workflowNodes map[string]*WorkflowNode
}

// NewWorkflow creates a new workflow graph.
func NewWorkflow[I, O any](opts ...NewGraphOption) *Workflow[I, O] {
	return &Workflow[I, O]{
		Graph:         NewGraph[I, O](opts...),
		workflowNodes: make(map[string]*WorkflowNode),
	}
}

// initNode creates or retrieves a WorkflowNode for the given key.
func (w *Workflow[I, O]) initNode(key string) *WorkflowNode {
	if n, ok := w.workflowNodes[key]; ok {
		return n
	}
	n := &WorkflowNode{key: key}
	w.workflowNodes[key] = n
	return n
}

// AddLambdaNode adds a lambda node and returns the node for further configuration.
func (w *Workflow[I, O]) AddLambdaNode(key string, lambda *Lambda, opts ...GraphAddNodeOpt) *WorkflowNode {
	_ = w.Graph.AddLambdaNode(key, lambda, opts...)
	return w.initNode(key)
}

// End finalizes the workflow and returns the end node.
func (w *Workflow[I, O]) End() *WorkflowNode {
	return w.initNode(END)
}

// Compile compiles the workflow.
func (w *Workflow[I, O]) Compile(ctx context.Context, opts ...GraphCompileOption) (*Runnable[I, O], error) {
	// Resolve dependencies into edges
	for key, node := range w.workflowNodes {
		for _, dep := range node.depEntries {
			_ = w.Graph.AddEdge(dep.fromKey, key)
		}
	}
	r, err := w.Graph.Compile(ctx, opts...)
	if err != nil {
		return nil, err
	}
	// Store field mappings in the compiled graph
	for key, node := range w.workflowNodes {
		if len(node.depEntries) > 0 {
			if r.graph.fieldMappings == nil {
				r.graph.fieldMappings = make(map[string][]depEntry)
			}
			r.graph.fieldMappings[key] = node.depEntries
		}
		if len(node.staticValues) > 0 {
			if r.graph.staticValues == nil {
				r.graph.staticValues = make(map[string][]staticValueEntry)
			}
			r.graph.staticValues[key] = node.staticValues
		}
	}
	return r, nil
}

// AddChatTemplateNode adds a chat template node.
func (w *Workflow[I, O]) AddChatTemplateNode(key string, tpl wfcompose.ChatTemplate, opts ...GraphAddNodeOpt) *WorkflowNode {
	_ = w.Graph.AddChatTemplateNode(key, tpl, opts...)
	return w.initNode(key)
}

// WorkflowNode represents a node in a workflow, tracking dependencies.
type WorkflowNode struct {
	key          string
	depEntries   []depEntry
	staticValues []staticValueEntry
}

type depEntry struct {
	fromKey      string
	fieldMappings []*FieldMapping
}

// AddInput adds an input dependency to the node.
func (n *WorkflowNode) AddInput(fromNodeKey string, fieldMappings ...*FieldMapping) *WorkflowNode {
	n.depEntries = append(n.depEntries, depEntry{fromKey: fromNodeKey, fieldMappings: fieldMappings})
	return n
}

// AddInputWithOptions adds an input dependency with options.
func (n *WorkflowNode) AddInputWithOptions(fromNodeKey string, fieldMappings []*FieldMapping, opts ...NewGraphOption) *WorkflowNode {
	n.depEntries = append(n.depEntries, depEntry{fromKey: fromNodeKey, fieldMappings: fieldMappings})
	return n
}

// AddDependency adds a dependency to the node.
func (n *WorkflowNode) AddDependency(key string) *WorkflowNode {
	n.depEntries = append(n.depEntries, depEntry{fromKey: key})
	return n
}

// SetStaticValue sets a static value for the node (called without chaining).
func (n *WorkflowNode) SetStaticValue(path FieldPath, val any) {
	n.staticValues = append(n.staticValues, staticValueEntry{path: path, val: val})
}
