package main

import (
	"context"
	"fmt"
	"log"

	"github.com/superagent-ai/superagent-base/sdk"
	"github.com/superagent-ai/superagent-base/sdk/tool"
)

func main() {
	registry := tool.NewRegistry()
	registry.Register(tool.New("web_search", "Search the web", func(ctx context.Context, args map[string]any) (map[string]any, error) {
		query := args["query"].(string)
		return map[string]any{"results": []string{"Result for: " + query}}, nil
	}))

	registry.Register(tool.New("calculate", "Calculate math", func(ctx context.Context, args map[string]any) (map[string]any, error) {
		_ = args["expression"].(string)
		return map[string]any{"result": "42"}, nil
	}))

	fmt.Println("Registered tools:")
	for _, t := range registry.List() {
		fmt.Printf("  - %s: %s\n", t.Name(), t.Description())
	}

	result, err := registry.Invoke(context.Background(), "web_search", map[string]any{"query": "Go SDK"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Result: %v\n", result)

	rt, _ := sdk.NewRuntime()
	defer rt.Shutdown()
}
