package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/embed"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/bm25"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/genkitretriever"
	"github.com/duynguyendang/manglekit/internal/providers/retrievers/hybrid"
	"github.com/duynguyendang/manglekit/internal/providers/state/inmemory"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Note: .env file not found, will fall back to environment variables")
	}

	ctx := context.Background()

	// 1. Create a new programmatic builder
	// This automatically registers all standard providers via providers/all.Register()
	builder, err := sdk.NewBuilder(ctx)
	if err != nil {
		log.Fatalf("Failed to create builder: %v", err)
	}

	// 2. Add components programmatically
	// Google Embedder - provides embeddings for semantic search
	_ = embed.GoogleEmbedderOptions{} // Use default model (text-embedding-004)
	googleEmbedderOpts := &embed.GoogleEmbedderOptions{
		// Model defaults to "text-embedding-004", which is fine
		// API key is read from GOOGLE_API_KEY environment variable
	}

	// Genkit-based Semantic Retriever - semantic/dense search using LocalVec
	// LocalVec is a lightweight, file-based vector database (Genkit plugin)
	_ = genkitretriever.GenkitRetrieverOptions{} // Ensure package is imported
	genkitRetrieverOpts := &genkitretriever.GenkitRetrieverOptions{
		Provider:  "localvec",                // Uses LocalVec Genkit plugin
		Embedder:  "google_embedder",         // Reference to the Manglekit-registered embedder
		Endpoint:  "/tmp/manglekit-localvec", // LocalVec storage directory
		IndexName: "documents",               // LocalVec collection/index name
	}

	// BM25 Retriever (now sub-retriever) - keyword-based search
	_ = bm25.BM25Options{} // Ensure package is imported
	bm25SubRetrieverOpts := &bm25.BM25Options{
		Path: "examples/01-programmatic-setup/docs",
	}

	// Hybrid Retriever - combines BM25 and LocalVec semantic search
	// Uses Reciprocal Rank Fusion (RRF) to merge results from both retrievers
	_ = hybrid.HybridOptions{} // Ensure package is imported
	hybridRetrieverOpts := &hybrid.HybridOptions{
		Retrievers: []string{"keyword_retriever", "semantic_retriever"}, // BM25 + LocalVec
		// RRF_K can be customized if needed; defaults to 60.0
	}

	// Google LLM - language model for generation
	googleOpts := &llm.GoogleOptions{
		Model:          "gemini-2.5-flash",
		PromptTemplate: "Explain in details ",
		// API key is read from GOOGLE_API_KEY environment variable
	}

	// Mangle RuleSet - policy and rule engine
	// Note: Set DefaultConverters to true to enable default fact converters
	mangleOpts := &core.MangleOptions{
		Path:              []string{"examples/rules/acme-rules.dlog"},
		DefaultConverters: true,
	}

	// In-memory StateProvider - session state management
	stateOpts := &inmemory.Options{}

	// Sandwich Orchestrator - RAG pipeline orchestrator
	sandwichOpts := &sandwich.Options{
		LLM:           "google",
		Retriever:     "hybrid_retriever", // Changed to hybrid retriever
		RuleSet:       "mangle",
		StateProvider: "inmemory",
	}

	// 3. Configure the builder with all components
	builder.
		WithOptions("google_embedder", googleEmbedderOpts).
		WithOptions("semantic_retriever", genkitRetrieverOpts).
		WithOptions("keyword_retriever", bm25SubRetrieverOpts).
		WithOptions("hybrid_retriever", hybridRetrieverOpts).
		WithOptions("google", googleOpts).
		WithOptions("mangle", mangleOpts).
		WithOptions("inmemory", stateOpts).
		WithOptions("sandwich", sandwichOpts)

	// 4. Build the orchestrator
	orch, _, err := builder.Build(ctx, "sandwich", "")
	if err != nil {
		log.Fatalf("Failed to build orchestrator: %v", err)
	}

	// Ensure proper resource cleanup
	defer func() {
		if err := orch.Close(ctx); err != nil {
			log.Printf("Warning: Error closing orchestrator: %v", err)
		}
	}()

	// 5. Execute a query
	// Use a query that matches content in the test documents
	// The test documents contain: "MangleKit", "Go", "framework", "Retrieval-Augmented-Generation"
	query := core.Query{
		Text: "What is manglekit?",
	}
	log.Printf("Executing query: %s\n", query.Text)

	answer, err := orch.Execute(ctx, "session-123", query)
	if err != nil {
		log.Fatalf("Failed to execute query: %v", err)
	}

	fmt.Printf("\nAnswer: %s\n", answer.Text)
	if len(answer.Citations) > 0 {
		fmt.Println("\nCitations:")
		for _, citation := range answer.Citations {
			fmt.Printf("  - %s (Source: %s)\n", citation.ID, citation.Source)
		}
	}
}
