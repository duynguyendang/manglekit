package dense

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/ai"
)

func init() {
	// Register the type-safe constructor directly.
	manglekit.RegisterRetriever("dense", New)
}

// Dense implements the retrieve.Retriever interface for dense vector search.
// It orchestrates the process of converting a text query into a vector embedding
// and then using that vector to search a pluggable core.VectorStore.
type Dense struct {
	embedder    ai.Embedder
	vectorStore core.VectorStore
}

// New creates a new Dense retriever. It is the constructor function registered
// with the MangleKit registry for the "dense" retriever. It requires an embedder
// and a vector store, which are injected by the builder.
//
// embedder is the component used to convert the query text into a vector.
// vectorStore is the database used to store and search for document vectors.
// It returns an initialized Dense retriever or an error if dependencies are missing.
func New(embedder ai.Embedder, vectorStore core.VectorStore) (retrieve.Retriever, error) {
	if embedder == nil {
		return nil, fmt.Errorf("dense: an embedder is required")
	}
	if vectorStore == nil {
		return nil, fmt.Errorf("dense: a vectorStore is required")
	}

	return &Dense{
		embedder:    embedder,
		vectorStore: vectorStore,
	}, nil
}

// Retrieve performs a search by first embedding the query and then searching the
// vector store with the resulting vector.
// This method satisfies the core.Retriever interface.
//
// req contains the query string, TopK, and any metadata filters.
// It returns a retrieve.Result containing the documents found in the vector
// store, or an error if embedding or searching fails.
func (d *Dense) Retrieve(req retrieve.Request) (retrieve.Result, error) {
	// 1. Embed the query text.
	embedReq := &ai.EmbedRequest{
		Input: []*ai.Document{ai.DocumentFromText(req.Query, nil)},
	}
	embedResp, err := d.embedder.Embed(context.Background(), embedReq)
	if err != nil {
		return retrieve.Result{}, fmt.Errorf("dense: failed to embed query: %w", err)
	}
	if len(embedResp.Embeddings) == 0 {
		return retrieve.Result{}, fmt.Errorf("dense: embedder returned no embeddings for query")
	}
	queryVector := embedResp.Embeddings[0].Embedding

	// 2. Search the vector store.
	// WORKAROUND: Pass query text in context for localvec compatibility.
	ctx := context.WithValue(context.Background(), "query_text", req.Query)

	var filter map[string]any
	if f, ok := req.Meta["filters"].(map[string]any); ok {
		filter = f
	}

	docs, err := d.vectorStore.Search(ctx, queryVector, req.TopK, filter)
	if err != nil {
		return retrieve.Result{}, fmt.Errorf("dense: vector store search failed: %w", err)
	}

	return retrieve.Result{Docs: docs}, nil
}