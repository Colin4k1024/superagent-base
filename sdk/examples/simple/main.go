package main

import (
	"context"
	"fmt"
	"log"

	"github.com/superagent-ai/superagent-base/sdk"
)

func main() {
	rt, err := sdk.NewRuntime(
		sdk.WithAgentsDir("configs/agents"),
		sdk.WithModel(sdk.ModelRuntimeConfig{
			BaseURL: "http://localhost:8000/v1",
			APIKey:  "sk-xxx",
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer rt.Shutdown()

	agent, ok := rt.GetAgent("research-agent")
	if !ok {
		log.Fatal("agent not found")
	}

	ch, err := agent.Chat(context.Background(), "session-1", "What is quantum computing?")
	if err != nil {
		log.Fatal(err)
	}

	for chunk := range ch {
		fmt.Print(chunk)
	}
	fmt.Println()
}
