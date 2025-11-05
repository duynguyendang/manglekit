// main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"

	// CRITICAL: This import registers all handlers, factories,
	// and options samples for the providers we use (sandwich, bm25,
	// openai, etc.). Without this, the config loader will fail.
	//
	_ "github.com/duynguyendang/manglekit/providers/all"
)

func main() {
	// (Optional) Check for API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Println("WARN: OPENAI_API_KEY is not set. LLM calls may fail.")
	}

	ctx := context.Background()

	// 1. Load the pipeline from the config file
	// This is the "Config-First" entrypoint
	log.Println("Reading config.yaml...")
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatalf("unable to get the current filename")
	}
	configPath := filepath.Join(filepath.Dir(filename), "config.yaml")

	configData, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	log.Println("Loading pipeline from config...")
	orchestrator, err := sdk.Load(ctx, configData)
	if err != nil {
		log.Fatalf("Error loading pipeline: %v", err)
	}
	defer orchestrator.Close(ctx) // Ensure resources are cleaned up
	log.Println("Pipeline loaded successfully.")

	// 2. Define a query
	query := core.Query{
		Text: "What is Manglekit?",
	}

	// 3. Execute the pipeline
	log.Printf("Executing query: '%s'\n", query.Text)
	answer, err := orchestrator.Execute(ctx, "session-01", query)
	if err != nil {
		log.Fatalf("Error executing query: %v", err)
	}

	// 4. Print the result
	fmt.Println("---")
	fmt.Printf("Answer: %s\n", answer.Text)
	fmt.Println("---")
	fmt.Println("Citations:")
	for _, citation := range answer.Citations {
		fmt.Printf("- %s (Score: %.4f)\n", citation.Snippet, citation.Score)
	}
}
