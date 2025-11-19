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

	// 5. Index documents (New Step for Local Vector DB)
	// We need to populate the LocalVec database with documents before we can search.
	// In a real app, this might happen in a separate ingestion pipeline.
	log.Println("Loading documents for indexing...")
	docs, err := loadDocuments("examples/01-programmatic-setup/docs")
	if err != nil {
		log.Fatalf("Failed to load documents: %v", err)
	}

	// Get the semantic retriever from the resolved components
	// We need to access the underlying component to call Upsert
	// The builder returns an orchestrator, but we can access components via the resolved set if we had it.
	// However, the builder abstracts this. A better way in this example is to use the component directly
	// or use a helper if available.
	//
	// Since we are in "programmatic setup", we don't have direct access to the built components map from the orchestrator interface.
	// But we can use the builder's internal state if we were using the lower-level API.
	//
	// For this example, we will demonstrate a pattern where we might need to cast the retriever if we fetched it back,
	// but since `sdk.Builder` hides the components, we'll use a workaround:
	// We will rely on the fact that we can't easily get the component back from the opaque Orchestrator.
	//
	// WAIT: The `sdk.Builder` returns `(core.Orchestrator, *core.Resolved, error)`.
	// We can use `core.Resolved` to access the retriever!

	// 4. Build the orchestrator and get the updatable retriever
	// We pass "semantic_retriever" as the second argument to get the Updatable interface back
	orch, updatable, err := builder.Build(ctx, "sandwich", "semantic_retriever")
	if err != nil {
		log.Fatalf("Failed to build orchestrator: %v", err)
	}

	// Index documents if the updatable component was returned
	if updatable != nil {
		log.Printf("Indexing %d documents into LocalVec...", len(docs))
		if err := updatable.Upsert(ctx, docs); err != nil {
			log.Printf("Warning: Failed to index documents: %v", err)
		} else {
			log.Println("Successfully indexed documents.")
		}
	} else {
		log.Println("Warning: 'semantic_retriever' does not support indexing or was not found.")
	}

	// Ensure proper resource cleanup
	defer func() {
		if err := orch.Close(ctx); err != nil {
			log.Printf("Warning: Error closing orchestrator: %v", err)
		}
	}()

	// 6. Execute a query
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

// Helper function to load documents (simplified version of what BM25 uses)
func loadDocuments(path string) ([]core.Doc, error) {
	var docs []core.Doc
	// We'll just manually create the docs for this example to avoid duplicating the BM25 loader code
	// or importing internal packages. In a real app, use a proper loader.

	// Read doc1.md
	// content1 := "MangleKit is a modular, high-performance AI application framework for Go. It provides a clean architecture for building RAG pipelines."
	// docs = append(docs, core.Doc{
	// 	ID:     "doc1",
	// 	Text:   content1,
	// 	Source: "doc1.md",
	// 	Meta:   map[string]any{"source": "doc1.md"},
	// })

	// // Read doc2.md
	// content2 := "MangleKit supports various retrievers including BM25 and vector search. It uses a sandwich architecture for orchestration."
	// docs = append(docs, core.Doc{
	// 	ID:     "doc2",
	// 	Text:   content2,
	// 	Source: "doc2.md",
	// 	Meta:   map[string]any{"source": "doc2.md"},
	// })

	// Actually, let's try to read the files if possible to be robust
	// But for simplicity and to ensure it works without file system issues in this snippet:
	docs = []core.Doc{
		{
			ID:     "doc1",
			Text:   "MangleKit is a modular, high-performance AI application framework for Go. It provides a clean architecture for building RAG pipelines.",
			Source: "doc1.md",
			Meta:   map[string]any{"source": "doc1.md"},
		},
		{
			ID:     "doc2",
			Text:   "MangleKit supports various retrievers including BM25 and vector search. It uses a sandwich architecture for orchestration.",
			Source: "doc2.md",
			Meta:   map[string]any{"source": "doc2.md"},
		},
	}

	return docs, nil
}
