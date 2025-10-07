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
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/joho/godotenv"
)

// This example demonstrates how to use the logic layer mode.
//
// The logic layer mode allows you to provide additional context to the RAG pipeline
// in the form of Datalog facts.
//
// In this example, we'll create an in-memory retriever and then use a Mangle
// rule to specify which documents to retrieve based on runtime context.
func main() {
	_ = godotenv.Load()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatalf("failed to get current file path")
	}
	dir := filepath.Dir(currentFile)
	configPath := filepath.Join(dir, "config.yaml")

	// 1. Load the base configuration from YAML.
	// This will set up the Mangle rules and the LLM.
	builder, err := manglekit.NewBuilderFromYAML(configPath)
	if err != nil {
		log.Fatalf("failed to create builder from yaml: %v", err)
	}

	// 2. Programmatically configure the in-memory retriever with documents.
	// This demonstrates how to mix declarative config with dynamic, runtime data.
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
			{
				ID:   "doc3",
				Text: "The sky is blue.", // This doc is not about Mangle.
			},
		},
	})

	// 3. Build the pipeline.
	pipeline, err := builder.Build()
	if err != nil {
		log.Fatalf("failed to build pipeline: %v", err)
	}

	// 5. Run the pipeline with additional context.
	// We'll pass the 'doc_id' we want to retrieve in the query's Meta field.
	// The userContextConverter will turn this into a `user_attribute("doc_id", "doc2")` fact.
	// Our rule `pre_retrieve(doc) :- user_attribute("doc_id", ID), doc_id(doc, ID).`
	// will then ensure only doc2 is retrieved.
	query := core.Query{
		Text: "what is mangle?",
		Meta: map[string]any{
			"user_context": map[string]any{
				"doc_id": "doc2",
			},
		},
	}

	resp, err := pipeline.Run(context.Background(), query)
	if err != nil {
		log.Fatalf("failed to run pipeline: %v", err)
	}

	fmt.Println(resp.Text)
	fmt.Println("\nCitations:")
	for _, citation := range resp.Citations {
		fmt.Printf("- ID: %s, Snippet: %s\n", citation.ID, citation.Snippet)
	}
}
