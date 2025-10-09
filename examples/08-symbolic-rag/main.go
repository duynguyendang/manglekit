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
	builder, err := manglekit.NewBuilderFromYAML("./config.yaml")
	if err != nil {
		log.Fatalf("Failed to create builder from YAML: %v", err)
	}

	orch, err := builder.Build()
	if err != nil {
		log.Fatalf("Failed to build orchestrator: %v", err)
	}

	// The developer asks a natural language question.
	query := core.Query{
		Text: "Which document describes the Builder function?",
	}

	// Run the pipeline.
	answer, err := orch.Run(context.Background(), query)
	if err != nil {
		log.Fatalf("Pipeline run failed: %v", err)
	}

	fmt.Println("AI Answer:", answer.Text)
	// Expected Output: An LLM-generated answer like "The Builder function is described in jules_LLD.md. It provides a fluent API for constructing the orchestrator."
}
