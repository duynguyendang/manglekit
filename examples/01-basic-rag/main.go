// Package main for 01-basic-rag demonstrates the simplest way to create and run
// a MangleKit pipeline. It uses the `NewBuilderFromYAML` function to load a
// complete "sandwich" pipeline configuration from a local `config.yaml` file.
// This example showcases how to initialize the system, run a query, and print
// the response and its citations.
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

// The simplest way to get started.
//
// This example demonstrates the basic RAG pipeline.
func main() {
	_ = godotenv.Load()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatalf("failed to get current file path")
	}
	dir := filepath.Dir(currentFile)
	configPath := filepath.Join(dir, "config.yaml")

	// Build the pipeline from the local config file.
	builder, err := manglekit.NewBuilderFromYAML(configPath)
	if err != nil {
		log.Fatalf("failed to create builder from yaml: %v", err)
	}

	ctx := context.Background()
	pipeline, err := builder.Build(ctx)
	if err != nil {
		log.Fatalf("failed to build pipeline: %v", err)
	}
	defer pipeline.Close(ctx)

	// Run the pipeline.
	resp, err := pipeline.Run(ctx, core.Query{Text: "what is the features of mangle?"})
	if err != nil {
		log.Fatalf("failed to run pipeline: %v", err)
	}
	fmt.Println(resp.Text)
	fmt.Println("\nCitations:")
	for _, citation := range resp.Citations {
		fmt.Printf("- %s (%s)\n", citation.Snippet, citation.Source)
	}
}
