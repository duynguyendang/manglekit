package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	_ "github.com/duynguyendang/manglekit/providers/all"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatalf("failed to get current file path")
	}
	dir := filepath.Dir(currentFile)
	configPath := filepath.Join(dir, "config.yaml")
	builder, err := manglekit.NewBuilderFromYAML(configPath)

	if err != nil {
		log.Fatalf("Failed to create builder from YAML: %v", err)
	}

	// Build the orchestrator from the configured builder.
	orch, err := builder.Build()
	if err != nil {
		log.Fatalf("Failed to build orchestrator: %v", err)
	}

	// Simulate a query from user "alice".
	// The Mangle rules will use the `current_user` fact to find the document
	// assigned to her.
	query := core.Query{
		Text: "test simple rule",
		Meta: map[string]any{
			"dynamic_facts": []map[string]any{
				{"predicate": "current_user", "args": []any{"alice"}},
			},
		},
	}

	// Run the pipeline.
	answer, err := orch.Run(context.Background(), query)
	if err != nil {
		log.Fatalf("Pipeline run failed: %v", err)
	}

	fmt.Println("AI Answer:", answer.Text)
	// Expected Output: An LLM-generated answer like "The customer name is 'Innovate Inc.'."
}
