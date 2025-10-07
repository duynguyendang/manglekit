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

// This example demonstrates how to use the hot-reload feature.
//
// The hot-reload feature allows you to update the documents in a retriever
// at runtime.
//
// In this example, we'll create an in-memory retriever, run the pipeline,
// update the documents, and then run the pipeline again to show that the
// documents have been updated.
func main() {
	_ = godotenv.Load()

	// 1. Load the base configuration from YAML.
	builder, err := manglekit.NewBuilderFromYAML("./config.yaml")
	if err != nil {
		log.Fatalf("failed to create builder from yaml: %v", err)
	}

	// 2. Programmatically configure the in-memory retriever with initial documents.
	builder.WithRetriever(&retrieve.InMemoryOptions{
		Documents: []core.Doc{
			{
				ID:   "doc1",
				Text: "Mangle is a tool for building and running RAG pipelines.",
			},
		},
	})

	// 3. Build the pipeline.
	pipeline, err := builder.Build()
	if err != nil {
		log.Fatalf("failed to build pipeline: %v", err)
	}

	// 4. Run the pipeline for the first time.
	fmt.Println("--- First run ---")
	resp, err := pipeline.Run(context.Background(), core.Query{Text: "what is mangle?"})
	if err != nil {
		log.Fatalf("failed to run pipeline: %v", err)
	}
	fmt.Println(resp.Text)

	// 5. Update the documents in the retriever.
	// We can get the retriever from the pipeline and then cast it to the
	// `retrieve.Updatable` interface.
	updatable, ok := pipeline.Retriever().(retrieve.Updatable)
	if !ok {
		log.Fatalf("retriever is not updatable")
	}

	// The Upsert method expects a slice of core.Doc values.
	err = updatable.Upsert([]core.Doc{
		{
			ID:   "doc1", // Note: This will replace the original doc1
			Text: "Mangle is a super-cool tool for building and running RAG pipelines.",
		},
	})
	if err != nil {
		log.Fatalf("failed to upsert documents: %v", err)
	}

	// 6. Run the pipeline again to see the updated answer.
	fmt.Println("\n--- Second run ---")
	resp, err = pipeline.Run(context.Background(), core.Query{Text: "what is mangle?"})
	if err != nil {
		log.Fatalf("failed to run pipeline: %v", err)
	}
	fmt.Println(resp.Text)
}
