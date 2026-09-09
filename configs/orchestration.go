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

package configs

import (
	"context"

	"github.com/google/adk-go/compose"
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
