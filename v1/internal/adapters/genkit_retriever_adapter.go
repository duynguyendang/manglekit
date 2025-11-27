package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// GenkitRetrieverAdapter wraps a Genkit ai.Retriever and adapts it to the core.Retriever interface.
// This adapter is provider-agnostic and works with any Genkit retriever plugin
// (Pinecone, LocalVec, Weaviate, Qdrant, Milvus, etc.).
//
// This eliminates the need for the old "dense" orchestrator, as Genkit retrievers
// already perform semantic search (embedding + vector store lookup) internally.
// By wrapping them directly, we get a simpler, cleaner architecture.
// GenkitIndexer is an interface for Genkit providers that support indexing.
// Since Genkit doesn't have a unified Indexer interface yet, we define one here
// to abstract over different provider implementations (like localvec).
type GenkitIndexer interface {
	Index(ctx context.Context, docs []*ai.Document) error
}

type GenkitRetrieverAdapter struct {
	retriever      *genkit.Genkit // The Genkit instance needed to call Retrieve
	genktRetriever ai.Retriever   // The actual ai.Retriever to delegate to
	indexer        GenkitIndexer  // The optional indexer for Upsert operations
	logger         core.Logger
	provider       string // For logging and debugging
}

// NewGenkitRetrieverAdapter creates a new adapter wrapping a Genkit retriever.
//
// genkitInstance is the Genkit generator instance needed for retrieval operations.
// retriever is the Genkit ai.Retriever to wrap (e.g., from Pinecone, LocalVec plugin).
// indexer is the GenkitIndexer to wrap (optional, can be nil if read-only).
// provider is used for logging/debugging to identify which Genkit plugin is being used.
// logger is optional; if nil, no debug logging will occur.
func NewGenkitRetrieverAdapter(genkitInstance *genkit.Genkit, retriever ai.Retriever, indexer GenkitIndexer, provider string, logger core.Logger) *GenkitRetrieverAdapter {
	return &GenkitRetrieverAdapter{
		retriever:      genkitInstance,
		genktRetriever: retriever,
		indexer:        indexer,
		logger:         logger,
		provider:       provider,
	}
}

// Retrieve delegates the retrieval request directly to the underlying Genkit retriever.
// It is a simple pass-through that converts Manglekit RetrieveRequest/RetrieveResult
// to Genkit's retrieval semantics.
//
// ctx is the context for the API call.
// req is the Manglekit retrieval request (query, TopK, metadata filters).
//
// It returns a RetrieveResult containing the retrieved documents, or an error if retrieval fails.
func (a *GenkitRetrieverAdapter) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	if a.retriever == nil {
		return core.RetrieveResult{}, fmt.Errorf("genkit retriever adapter (%s): genkit instance is nil", a.provider)
	}
	if a.genktRetriever == nil {
		return core.RetrieveResult{}, fmt.Errorf("genkit retriever adapter (%s): underlying retriever is nil", a.provider)
	}

	if a.logger != nil {
		a.logger.Debugf(
			"delegating retrieval request to Genkit provider",
			"provider", a.provider,
			"query", req.Query,
			"topk", req.TopK,
		)
	}

	// Create a Genkit document for the query
	queryDoc := ai.DocumentFromText(req.Query, nil)

	// Call Genkit's Retrieve function with the wrapper retriever
	genkitResult, err := genkit.Retrieve(
		ctx,
		a.retriever,
		ai.WithRetriever(a.genktRetriever),
		ai.WithDocs(queryDoc),
		ai.WithConfig(map[string]any{"k": req.TopK}),
	)
	if err != nil {
		if a.logger != nil {
			a.logger.Debugf(
				"genkit retriever delegation failed",
				"provider", a.provider,
				"error", err.Error(),
			)
		}
		return core.RetrieveResult{}, fmt.Errorf("genkit retriever (%s) retrieve failed: %w", a.provider, err)
	}

	if a.logger != nil {
		docCount := 0
		if genkitResult != nil {
			docCount = len(genkitResult.Documents)
		}
		a.logger.Debugf(
			"retrieval completed successfully",
			"provider", a.provider,
			"doc_count", docCount,
		)
	}

	// Convert Genkit documents to Manglekit core.Doc format
	docs := make([]core.Doc, 0)
	if genkitResult != nil {
		for _, genkitDoc := range genkitResult.Documents {
			// Extract text content from the Genkit document
			docText := ""
			if len(genkitDoc.Content) > 0 && genkitDoc.Content[0].Text != "" {
				docText = genkitDoc.Content[0].Text
			}

			// Extract ID from metadata or use a placeholder
			docID := ""
			if genkitDoc.Metadata != nil {
				if id, ok := genkitDoc.Metadata["doc_id"].(string); ok {
					docID = id
				}
			}

			// Filter by metadata if filters were provided
			if len(req.Meta) > 0 {
				if !matchesFilters(genkitDoc.Metadata, req.Meta) {
					continue
				}
			}

			doc := core.Doc{
				ID:     docID,
				Text:   docText,
				Source: extractSource(genkitDoc.Metadata),
				Meta:   genkitDoc.Metadata,
			}
			docs = append(docs, doc)
		}
	}

	result := core.RetrieveResult{
		Docs: docs,
		Meta: map[string]any{
			"provider": a.provider,
			"source":   "genkit_retriever",
		},
	}

	return result, nil
}

// matchesFilters checks if a document's metadata matches the provided filter criteria.
func matchesFilters(metadata map[string]any, filters map[string]any) bool {
	for key, expectedVal := range filters {
		actualVal, ok := metadata[key]
		if !ok {
			return false
		}
		if actualVal != expectedVal {
			// Try string comparison as fallback
			if s1, ok1 := actualVal.(string); ok1 {
				if s2, ok2 := expectedVal.(string); ok2 {
					if !strings.EqualFold(s1, s2) {
						return false
					}
					continue
				}
			}
			return false
		}
	}
	return true
}

// extractSource extracts the source/URI from document metadata.
func extractSource(metadata map[string]any) string {
	if metadata == nil {
		return ""
	}
	if source, ok := metadata["source"].(string); ok {
		return source
	}
	if uri, ok := metadata["uri"].(string); ok {
		return uri
	}
	return ""
}

// Upsert adds or updates documents in the underlying Genkit index.
// It implements the core.Updatable interface.
func (a *GenkitRetrieverAdapter) Upsert(ctx context.Context, docs []core.Doc) error {
	if a.indexer == nil {
		return fmt.Errorf("genkit retriever adapter (%s): indexer is nil (provider does not support indexing or failed to initialize)", a.provider)
	}

	if a.logger != nil {
		a.logger.Debugf(
			"upserting documents to Genkit provider",
			"provider", a.provider,
			"count", len(docs),
		)
	}

	// Convert core.Doc to Genkit ai.Document
	var genkitDocs []*ai.Document
	for _, d := range docs {
		// Ensure metadata is not nil
		meta := d.Meta
		if meta == nil {
			meta = make(map[string]any)
		}
		// Preserve ID and Source in metadata if not already present
		if _, ok := meta["doc_id"]; !ok && d.ID != "" {
			meta["doc_id"] = d.ID
		}
		if _, ok := meta["source"]; !ok && d.Source != "" {
			meta["source"] = d.Source
		}

		gDoc := ai.DocumentFromText(d.Text, meta)
		genkitDocs = append(genkitDocs, gDoc)
	}

	// Index the documents
	err := a.indexer.Index(ctx, genkitDocs)
	if err != nil {
		return fmt.Errorf("genkit retriever (%s) upsert failed: %w", a.provider, err)
	}

	return nil
}

// Replace clears and replaces the index.
// For now, it just calls Upsert as many vector stores handle upsert as replace-if-exists.
// True "replace collection" semantics depend on the specific Genkit plugin capabilities.
func (a *GenkitRetrieverAdapter) Replace(ctx context.Context, docs []core.Doc) error {
	// TODO: Check if Genkit supports a "clear" or "reset" operation for the indexer.
	// For now, we assume Upsert is sufficient for adding/updating.
	return a.Upsert(ctx, docs)
}
