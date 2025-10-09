// Package dense provides an implementation of a dense retriever, which performs
// semantic search using vector embeddings.
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
	// Register the type-safe constructor with the MangleKit framework.
	manglekit.RegisterRetriever("dense", New)
}

// Dense implements the `retrieve.Retriever` interface for dense, vector-based
// search. It acts as an orchestrator, combining an `ai.Embedder` and a
// `core.VectorStore` to perform semantic retrieval. It does not store any
// documents itself but delegates the storage and search operations to the
// vector store.
type Dense struct {
	embedder    ai.Embedder
	vectorStore core.VectorStore
}

// New is the constructor for the Dense retriever. It is registered with the
// MangleKit registry for the "dense" provider name. The builder is responsible
// for constructing and injecting the required embedder and vector store dependencies.
//
// embedder is the component used to convert the query text into a vector embedding.
// vectorStore is the vector database used to store and search for document vectors.
// It returns an initialized `retrieve.Retriever` or an error if dependencies are missing.
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

// Retrieve performs a search by first converting the query text into a vector
// embedding, and then using that vector to search the configured vector store.
// This method satisfies the `retrieve.Retriever` interface.
//
// req contains the query string, the number of results to return (`TopK`), and
// any metadata to be used for filtering within the vector store.
// It returns a `retrieve.Result` containing the documents found by the vector
// store, or an error if the query embedding or vector search fails.
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
