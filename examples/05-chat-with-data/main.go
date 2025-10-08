package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	_ "github.com/duynguyendang/manglekit/providers/all"
)

func main() {
	// The NewBuilderFromYAML function will read the config, resolve paths,
	// and set up the builder with the tools we defined.
	// The path is relative to the execution directory. Since we run `go run .`
	// from this directory, the path is simply "config.yaml".
	builder, err := manglekit.NewBuilderFromYAML("config.yaml")
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
		Text: "Based on the document assigned to me, what is the customer name?",
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