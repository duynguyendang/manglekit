package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/duynguyendang/manglekit/adapters/knowledge"
	"github.com/duynguyendang/manglekit/sdk"
)

func main() {
	ctx := context.Background()

	// 1. Initialize Manglekit Client with Blueprint
	// We load the blueprint which defines the logic rules.
	client, err := sdk.NewClient(ctx,
		sdk.WithBlueprintPath("examples/context_aware_rag/blueprint.dl"),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// 2. Load Knowledge Base (N-Quads)
	// This loads the static knowledge graph.
	kbPath := "examples/context_aware_rag/knowledge_base.nq"
	f, err := os.Open(kbPath)
	if err != nil {
		log.Fatalf("Failed to open knowledge base: %v", err)
	}
	defer f.Close()

	loader := knowledge.NewNQuadsLoader()
	facts, err := loader.Parse(f)
	if err != nil {
		log.Fatalf("Failed to parse N-Quads: %v", err)
	}

	// Load facts into the client (Persistent Knowledge)
	if err := client.LoadFacts(facts); err != nil {
		log.Fatalf("Failed to load facts: %v", err)
	}

	// 3. Scenario A (Employee View)
	// We inject the "employee" role as a transient fact for this query.
	fmt.Println("--- Scenario A: Employee View ---")
	employeeFacts := []string{fmt.Sprintf("request_role(%q)", "employee")}

	// Execute Query: result(Status)
	// We need to cast Evaluator to Queryable interface locally
	type Queryable interface {
		Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error)
	}

	eng, ok := client.Engine().(Queryable)
	if !ok {
		log.Fatalf("Engine does not support Query")
	}

	results, err := eng.Query(ctx, employeeFacts, "result(Status)")
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	for _, res := range results {
		fmt.Printf("Status: %s\n", res["Status"])
	}

	// 4. Scenario B (Executive View)
	// We inject the "executive" role as a transient fact for this query.
	fmt.Println("\n--- Scenario B: Executive View ---")
	executiveFacts := []string{fmt.Sprintf("request_role(%q)", "executive")}

	// Execute Query: result(Status)
	resultsExec, err := eng.Query(ctx, executiveFacts, "result(Status)")
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}
	for _, res := range resultsExec {
		fmt.Printf("Status: %s\n", res["Status"])
	}
}
