package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/superagent-ai/superagent-base/backend/pkg/agentdef"
)

func main() {
	yamlPath := "configs/agents/document-pipeline.yaml"
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		yamlPath = "../configs/agents/document-pipeline.yaml"
	}
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read YAML: %v\n", err)
		os.Exit(1)
	}

	def, err := agentdef.Parse(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse YAML: %v\n", err)
		os.Exit(1)
	}

	baseURL := envOrDefault("MODEL_BASE_URL", "https://api.minimaxi.com/anthropic")
	apiKey := envOrDefault("MODEL_API_KEY", "")
	modelID := envOrDefault("MODEL_ID", "MiniMax-M2.7-highspeed")
	modelType := envOrDefault("MODEL_TYPE", "claude")

	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "MODEL_API_KEY environment variable is required")
		os.Exit(1)
	}

	builder := agentdef.NewAgentBuilder(agentdef.WithModelConfig(agentdef.ModelRuntimeConfig{
		BaseURL: baseURL,
		APIKey:  apiKey,
		ModelID: modelID,
		Type:    modelType,
	}))
	agent, err := builder.Build(context.Background(), def)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to build agent: %v\n", err)
		os.Exit(1)
	}

	sampleDoc := `Title: Machine Learning Pipeline Optimization Guide
Author: AI Research Team
Date: 2025-03-20

This comprehensive guide covers best practices for optimizing
machine learning pipelines. Key topics include data preprocessing
strategies, feature engineering techniques, model selection
frameworks, hyperparameter tuning with Bayesian optimization,
and deployment considerations for production environments.

The guide recommends using automated feature selection to reduce
dimensionality, implementing cross-validation for robust model
evaluation, and adopting MLOps practices for continuous
integration and delivery of ML models. Performance benchmarks
show 40% improvement in training time and 15% gain in prediction
accuracy when following these optimization strategies.`

	fmt.Println("=== Document Processing Pipeline ===")
	fmt.Printf("Agent: %s\n", agent.Name())
	fmt.Printf("Model: %s (protocol: %s)\n", modelID, modelType)
	fmt.Println()
	fmt.Println("--- Input Document ---")
	fmt.Println(sampleDoc)
	fmt.Println()
	fmt.Println("--- Pipeline Output ---")

	ch, err := agent.Chat(context.Background(), "example-session", sampleDoc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Chat error: %v\n", err)
		os.Exit(1)
	}

	var output strings.Builder
	for tok := range ch {
		output.WriteString(tok)
		fmt.Print(tok)
	}
	fmt.Println()

	if output.Len() == 0 {
		fmt.Println("\nWARNING: Empty response from model. Check API key and connectivity.")
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}