package vector

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/duynguyendang/manglekit/core"
)

// Document represents a retrieved document from a vector store.
type Document struct {
	Content string `json:"content"`
	Source  string `json:"source"`
}

// DocumentRetriever defines the interface for a vector search backend.
// This represents the retrieval "Muscle" layer that performs similarity search.
type DocumentRetriever interface {
	Retrieve(ctx context.Context, query string) ([]Document, error)
}

// RetrieverAction wraps a DocumentRetriever and implements core.Action.
// It treats vector search as a universal action (Query-in, Documents-out).
type RetrieverAction struct {
	name      string
	retriever DocumentRetriever
}

// NewRetrieverAction creates a new RetrieverAction wrapping the given DocumentRetriever.
func NewRetrieverAction(name string, retriever DocumentRetriever) *RetrieverAction {
	return &RetrieverAction{
		name:      name,
		retriever: retriever,
	}
}

// Execute expects a string payload (the search query), calls the retriever,
// and returns the retrieved documents as a JSON string wrapped in an Envelope.
// The output is formatted for downstream consumption (e.g., by an LLM action).
func (r *RetrieverAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	query, ok := input.Payload.(string)
	if !ok {
		return core.Envelope{}, fmt.Errorf("%w: invalid input type, expected string but got %T", core.ErrSystemError, input.Payload)
	}

	docs, err := r.retriever.Retrieve(ctx, query)
	if err != nil {
		return core.Envelope{}, fmt.Errorf("retrieval failed: %w", err)
	}

	// Convert documents to JSON string for downstream LLM consumption
	docsJSON, err := json.Marshal(docs)
	if err != nil {
		return core.Envelope{}, fmt.Errorf("failed to marshal documents: %w", err)
	}

	output := core.NewEnvelope(string(docsJSON))
	output.SetMeta("doc_count", strconv.Itoa(len(docs)))
	output.SetMeta("action_name", r.name)

	return output, nil
}

// Metadata returns the metadata for this retriever action.
func (r *RetrieverAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: r.name,
		Type: "retriever",
	}
}

// FormatDocsAsContext is a helper function that formats retrieved documents
// as a context string suitable for LLM prompts.
func FormatDocsAsContext(docsJSON string) (string, error) {
	var docs []Document
	if err := json.Unmarshal([]byte(docsJSON), &docs); err != nil {
		return "", fmt.Errorf("failed to unmarshal documents: %w", err)
	}

	if len(docs) == 0 {
		return "No relevant documents found.", nil
	}

	var context string
	for i, doc := range docs {
		context += fmt.Sprintf("Document %d (Source: %s):\n%s\n\n", i+1, doc.Source, doc.Content)
	}

	return context, nil
}
