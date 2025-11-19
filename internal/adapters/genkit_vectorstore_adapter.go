package adapters

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
)

// GenkitVectorStoreAdapter wraps a Genkit-backed Retriever and provides a vector store-like interface.
// This adapter is provider-agnostic and works with any Genkit vector store plugin
// (Pinecone, LocalVec, Weaviate, Qdrant, Milvus, etc.).
//
// Note: This adapter delegates Genkit retriever operations to core.VectorStore semantics.
// Since Genkit retrievers are primarily read-oriented, write operations (AddDocuments)
// may not be fully supported by all backends.
type GenkitVectorStoreAdapter struct {
	retriever core.Retriever
	logger    core.Logger
	provider  string // For logging and debugging
}

// NewGenkitVectorStoreAdapter creates a new adapter wrapping a Genkit retriever as a vector store.
// provider is used for logging/debugging to identify which Genkit plugin is being used.
func NewGenkitVectorStoreAdapter(retriever core.Retriever, provider string, logger core.Logger) *GenkitVectorStoreAdapter {
	return &GenkitVectorStoreAdapter{
		retriever: retriever,
		logger:    logger,
		provider:  provider,
	}
}

// Search delegates a vector store search to the underlying Genkit retriever.
// It converts the search request into a Manglekit retrieval request and returns results as core.Docs.
//
// queryText is the text query to search for.
// queryVector is the query vector (may be ignored if retriever uses text-based search).
// topK is the number of results to return.
// filter is the optional metadata filters.
func (a *GenkitVectorStoreAdapter) Search(
	ctx context.Context,
	queryText string,
	queryVector []float32,
	topK int,
	filter map[string]any,
) ([]core.Doc, error) {
	if a.retriever == nil {
		return nil, fmt.Errorf("genkit vector store adapter (%s): underlying retriever is nil", a.provider)
	}

	if a.logger != nil {
		a.logger.Debugf(
			"delegating vector store search to Genkit retriever",
			"provider", a.provider,
			"query_text", queryText,
			"topk", topK,
		)
	}

	// Create a Manglekit retrieval request
	req := core.RetrieveRequest{
		Query: queryText,
		TopK:  topK,
		Meta:  filter,
	}

	// Call the underlying retriever
	result, err := a.retriever.Retrieve(ctx, req)
	if err != nil {
		if a.logger != nil {
			a.logger.Debugf(
				"genkit vector store retriever delegation failed",
				"provider", a.provider,
				"error", err.Error(),
			)
		}
		return nil, fmt.Errorf("genkit vector store (%s) search failed: %w", a.provider, err)
	}

	if a.logger != nil {
		a.logger.Debugf(
			"vector store search completed successfully",
			"provider", a.provider,
			"doc_count", len(result.Docs),
		)
	}

	return result.Docs, nil
}

// AddDocuments adds documents to the vector store.
// Not all Genkit backends support document ingestion. This returns ErrNotSupported
// for read-only backends.
func (a *GenkitVectorStoreAdapter) AddDocuments(
	ctx context.Context,
	docs []core.Doc,
) error {
	if a.retriever == nil {
		return fmt.Errorf("genkit vector store adapter (%s): underlying retriever is nil", a.provider)
	}

	if a.logger != nil {
		a.logger.Debugf(
			"AddDocuments called on Genkit vector store (read-only)",
			"provider", a.provider,
			"operation", "add_documents",
			"result", "not_supported",
		)
	}

	// Most Genkit vector store backends are read-only
	return fmt.Errorf("genkit vector store (%s): %w", a.provider, core.ErrNotSupported)
}

// Name returns a human-readable name for the adapter.
func (a *GenkitVectorStoreAdapter) Name() string {
	return fmt.Sprintf("genkit-vectorstore(%s)", a.provider)
}
