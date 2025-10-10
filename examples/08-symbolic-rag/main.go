// Package main for 08-symbolic-rag demonstrates a symbolic RAG pipeline, where
// the retrieval process is guided by symbolic reasoning and knowledge representation.
//
// In this example, the user asks a natural language question. The Mangle rules
// in `rules/policy.dlog` first parse this query to extract key entities (like
// "Builder function"). They then use a knowledge base of facts (`rules/kb.facts`)
// to find the specific document that is known to describe this entity. Finally,
// the pipeline retrieves only that specific document to pass to the LLM. This
// shows how symbolic reasoning can dramatically improve retrieval precision
// over purely semantic or keyword-based methods.
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
	_ = godotenv.Load() // load .env file if exists
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

	ctx := context.Background()
	orch, err := builder.Build()
	if err != nil {
		log.Fatalf("Failed to build orchestrator: %v", err)
	}
	defer orch.Close(ctx)

	// The developer asks a natural language question.
	query := core.Query{
		Text: "Which document describes the Builder function?",
	}

	// Run the pipeline.
	answer, err := orch.Run(ctx, query)
	if err != nil {
		log.Fatalf("Pipeline run failed: %v", err)
	}

	fmt.Println("AI Answer:", answer.Text)
	// Expected Output: An LLM-generated answer like "The Builder function is described in jules_LLD.md. It provides a fluent API for constructing the orchestrator."
}
