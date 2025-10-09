package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
)

// This example demonstrates how to configure and run a Manglekit pipeline
// entirely from environment variables. It's particularly useful for deployment
// in containerized or CI/CD environments.
//
// Before running, set the required environment variables.
//
// # Minimal setup for an in-memory retriever and Google Gemini LLM:
//
// export GOOGLE_API_KEY="your-google-api-key"
//
// export MKT_RETRIEVER_NAME="in-memory"
// export MKT_RETRIEVER_PARAMS='{"documents": [
//   {"doc_id": "doc1", "content": "Manglekit is a framework for building RAG applications."},
//   {"doc_id": "doc2", "content": "It uses a rules engine to control the pipeline."}
// ]}'
//
// export MKT_LLM_NAME="google"
// export MKT_LLM_PARAMS='{"model": "gemini-1.5-flash-latest"}'
//
// # To run the example:
//
// go run ./examples/09-env-config/main.go
func main() {
	// NewBuilderFromEnv reads the MKT_* environment variables to configure the pipeline.
	builder, err := manglekit.NewBuilderFromEnv()
	if err != nil {
		log.Fatalf("Error creating builder from environment: %v", err)
	}

	// Build the orchestrator. The builder handles dependency injection, such as
	// creating the Google AI client and passing it to the LLM provider.
	orchestrator, err := builder.Build()
	if err != nil {
		log.Fatalf("Error building orchestrator: %v", err)
	}

	log.Println("Orchestrator built successfully from environment variables.")

	// Define a simple query to test the pipeline.
	query := core.Query{
		Text: "What is Manglekit?",
		Meta: map[string]any{"user": "test-user"},
	}

	// Execute the query.
	answer, err := orchestrator.Run(context.Background(), query)
	if err != nil {
		log.Fatalf("Error executing query: %v", err)
	}

	// Print the result.
	fmt.Println("\nFinal Answer:")
	fmt.Printf("Text: %s\n", answer.Text)
	fmt.Println("Citations:")
	for _, citation := range answer.Citations {
		fmt.Printf("  - DocID: %s, Source: %s\n", citation.ID, citation.Source)
	}

	// Example of required environment variables:
	if os.Getenv("GOOGLE_API_KEY") == "" {
		fmt.Println("\nWarning: GOOGLE_API_KEY is not set. The LLM call will fail.")
	}
	if os.Getenv("MKT_RETRIEVER_NAME") == "" {
		fmt.Println("\nWarning: MKT_RETRIEVER_NAME is not set. Retrieval will be empty.")
	}
}