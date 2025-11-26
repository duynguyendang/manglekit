package vector

import (
	"context"
	"fmt"

	"github.com/firebase/genkit/go/ai"
)

// GenkitRetriever wraps a Genkit ai.Retriever and implements the DocumentRetriever interface.
// This adapter translates between Manglekit's simple DocumentRetriever interface and Genkit's
// more feature-rich ai.Retriever API, enabling any Genkit-registered retriever to be used
// with Manglekit's vector search actions.
//
// This adapter works with any Genkit retriever plugin (Pinecone, LocalVec, Weaviate, etc.)
// and eliminates the need for provider-specific implementations.
type GenkitRetriever struct {
	retriever ai.Retriever
	embedder  ai.Embedder
}

// NewGenkitRetriever creates a new GenkitRetriever wrapping the provided Genkit retriever.
//
// retriever is the Genkit ai.Retriever to wrap (e.g., from Pinecone, LocalVec plugins).
// embedder is an optional Genkit ai.Embedder used for query embedding if needed.
// If the retriever handles embedding internally, embedder can be nil.
func NewGenkitRetriever(retriever ai.Retriever, embedder ai.Embedder) *GenkitRetriever {
	return &GenkitRetriever{
		retriever: retriever,
		embedder:  embedder,
	}
}

// Retrieve implements the DocumentRetriever interface.
// It takes a query string and returns semantically similar documents from the Genkit retriever.
//
// ctx is the request context.
// query is the search query (typically a user question or prompt).
//
// It returns a slice of Document structs, or an error if retrieval fails.
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
