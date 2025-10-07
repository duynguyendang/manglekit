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

// This example demonstrates how to use Mangle to enforce row- and column-level
// security policies when chatting with structured data.
//
// We define a set of documents and user attributes as Datalog facts. Then, we
// use Mangle rules to decide:
// 1. Which documents a user can retrieve (row-level security).
// 2. Which columns are visible or masked (column-level security).
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

	// 2. Build the pipeline.
	pipeline, err := builder.Build()
	if err != nil {
		log.Fatalf("failed to build pipeline: %v", err)
	}

	// 3. Add documents programmatically to the in-memory retriever.
	// In a real application, this data would come from a database or data warehouse.
	docs := []core.Doc{
		{
			ID:   "A123",
			Text: "customer_name: Acme Corp, email: contact@acme.com, revenue: 100000, notes: Initial deal",
		},
		{
			ID:   "B456",
			Text: "lead_name: Globex Inc, email: sales@globex.inc, score: 95",
		},
		{
			ID:   "S777",
			Text: "account: Initech, deal_size: 250000, owner: bsmith, notes: Q3 expansion plan",
		},
	}
	updatableRetriever, ok := pipeline.Retriever().(retrieve.Updatable)
	if !ok {
		log.Fatalf("retriever does not support updates")
	}
	if err := updatableRetriever.Upsert(docs); err != nil {
		log.Fatalf("failed to upsert documents: %v", err)
	}

	// 4. Define the user context and query.
	// The user attributes are used by the Mangle rules to make access control decisions.
	query := core.Query{
		Text: "Summarize the documents about sales and marketing",
		Meta: map[string]any{
			"user_attribute": []map[string]string{
				{"key": "user_id", "value": "alice"},
				{"key": "role", "value": "analyst"},
				{"key": "department", "value": "sales"},
				{"key": "doc_id", "value": "A123"},
				{"key": "purpose", "value": "analytics"},
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
