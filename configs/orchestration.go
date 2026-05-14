package configs

import (
	"context"

	"github.com/cloudwego/eino/compose"
)

func Buildtest(ctx context.Context) (r compose.Runnable[any, any], err error) {
	const Graph1 = "Graph1"
	g := compose.NewGraph[any, any]()
	graph1KeyOftest12, err := buildtest12(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddGraphNode(Graph1, graph1KeyOftest12,
		compose.WithGraphCompileOptions(
			compose.WithGraphName("test12")))
	_ = g.AddEdge(compose.START, Graph1)
	_ = g.AddEdge(Graph1, compose.END)
	r, err = g.Compile(ctx, compose.WithGraphName("test"))
	if err != nil {
		return nil, err
	}
	return r, err
}

func buildtest12(ctx context.Context) (ag compose.AnyGraph, err error) {
	g := compose.NewGraph[any, any]()
	return g, err
}
