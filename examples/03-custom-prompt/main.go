package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	_ "github.com/duynguyendang/manglekit/providers/all"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/joho/godotenv"
)

// This example demonstrates how to use a custom prompt.
//
// In this example, we'll create an in-memory retriever and then use a custom
// prompt to generate the answer.
func main() {
	_ = godotenv.Load()

	// 1. Load the base configuration from YAML.
	// This will set up the retriever and the LLM with the custom prompt.
	builder, err := manglekit.NewBuilderFromYAML("./config.yaml")
	if err != nil {
		log.Fatalf("failed to create builder from yaml: %v", err)
	}

	// 2. Programmatically configure the in-memory retriever with documents.
	builder.WithRetriever(&retrieve.InMemoryOptions{
		Documents: []core.Doc{
			{
				ID:   "doc1",
				Text: "Mangle is a tool for building and running RAG pipelines.",
			},
			{
				ID:   "doc2",
				Text: "Mangle is easy to use, flexible, and extensible.",
			},
		},
	})

	// 3. Build the pipeline.
	pipeline, err := builder.Build()
	if err != nil {
		log.Fatalf("failed to build pipeline: %v", err)
	}

	// 4. Run the pipeline.
	resp, err := pipeline.Run(context.Background(), core.Query{Text: "what is mangle?"})
	if err != nil {
		log.Fatalf("failed to run pipeline: %v", err)
	}

	fmt.Println(resp.Text)
}
