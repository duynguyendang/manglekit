package vector

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
)

// GenkitRetriever adapts the Genkit ai.Retriever interface to the Manglekit DocumentRetriever interface.
// It acts as a bridge, allowing Manglekit to use any retriever plugin compatible with Genkit (e.g., Pinecone, Chroma).
type GenkitRetriever struct {
	retriever ai.Retriever
	embedder  ai.Embedder
}

// NewGenkitRetriever initializes a new adapter for a Genkit retriever.
//
// Parameters:
//   - retriever: The Genkit retriever instance.
//   - embedder: An optional embedder. If the retriever handles embeddings internally, this can be nil.
//
// Returns:
//   - A pointer to the initialized GenkitRetriever.
func NewGenkitRetriever(retriever ai.Retriever, embedder ai.Embedder) *GenkitRetriever {
	return &GenkitRetriever{
		retriever: retriever,
		embedder:  embedder,
	}
}

// Retrieve executes a search query against the underlying Genkit retriever.
// It converts the results into Manglekit's standard Document format.
//
// Parameters:
//   - ctx: The execution context.
//   - query: The search query string.
//
// Returns:
//   - A slice of Document structs found by the retriever.
func (gr *GenkitRetriever) Retrieve(ctx context.Context, query string) ([]Document, error) {
	if gr.retriever == nil {
		return nil, fmt.Errorf("genkit retriever: underlying retriever is nil")
	}

	// Create a Genkit document for the query
	// Genkit's Retriever.Retrieve API expects a RetrieverRequest with a query Document
	req := &ai.RetrieverRequest{
		Query: ai.DocumentFromText(query, nil),
	}

	result, err := gr.retriever.Retrieve(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("genkit retriever failed: %w", err)
	}

	if result == nil {
		return []Document{}, nil
	}

	// Convert Genkit documents to Manglekit Document format
	docs := make([]Document, 0, len(result.Documents))
	for _, genkitDoc := range result.Documents {
		// Extract text content from the Genkit document
		content := ""
		if len(genkitDoc.Content) > 0 && genkitDoc.Content[0].Text != "" {
			content = genkitDoc.Content[0].Text
		}

		// Extract source from metadata or use a placeholder
		source := ""
		if genkitDoc.Metadata != nil {
			if s, ok := genkitDoc.Metadata["source"].(string); ok {
				source = s
			} else if u, ok := genkitDoc.Metadata["uri"].(string); ok {
				source = u
			}
		}

		doc := Document{
			Content: content,
			Source:  source,
		}
		docs = append(docs, doc)
	}

	return docs, nil
}
