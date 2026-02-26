package vector

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/duynguyendang/manglekit-wip/core"
	"github.com/firebase/genkit/go/ai"
)

// Document represents a single unit of retrieved knowledge.
type Document struct {
	// Content is the textual body of the document.
	Content string `json:"content"`
	// Source indicates the origin of the document (e.g., filename, URL).
	Source string `json:"source"`
}

// DocumentRetriever defines the interface for a vector search backend.
// It abstracts the underlying storage mechanism (Pinecone, Weaviate, Local).
type DocumentRetriever interface {
	// Retrieve finds semantically similar documents for a given query.
	//
	// Parameters:
	//   - ctx: The execution context.
	//   - query: The search string.
	//
	// Returns:
	//   - A slice of Document matches, or an error.
	Retrieve(ctx context.Context, query string) ([]Document, error)
}

// RetrieverAction wraps a DocumentRetriever into a core.Action.
// This allows retrieval operations to be governed by policies and traced.
type RetrieverAction struct {
	name      string
	retriever DocumentRetriever
}

// NewRetrieverAction creates a new RetrieverAction.
//
// Parameters:
//   - name: The unique name for this action.
//   - retriever: The retrieval backend implementation.
//
// Returns:
//   - A pointer to the initialized RetrieverAction.
func NewRetrieverAction(name string, retriever DocumentRetriever) *RetrieverAction {
	return &RetrieverAction{
		name:      name,
		retriever: retriever,
	}
}

// Execute performs the vector search.
// It expects a string payload (query) and returns a JSON string payload (list of documents).
//
// Parameters:
//   - ctx: The execution context.
//   - input: The input Envelope (Payload must be string).
//
// Returns:
//   - A result Envelope containing a JSON string of retrieved documents.
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

// Metadata returns the action's metadata.
func (r *RetrieverAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: r.name,
		Type: "retriever",
	}
}

// FormatDocsAsContext is a utility to format a JSON list of documents into a readable context string.
// This is typically used to inject retrieved knowledge into an LLM prompt.
//
// Parameters:
//   - docsJSON: The JSON string returned by RetrieverAction.Execute.
//
// Returns:
//   - A formatted string (e.g., "Document 1...\nDocument 2...").
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

// NewGenkitRetrieverAction creates a protected core.Action from a Genkit ai.Retriever.
//
// Parameters:
//   - name: The unique name for this action.
//   - retriever: The Genkit retriever instance.
//   - embedder: Optional embedder (if required by the retriever).
//
// Returns:
//   - A core.Action that can be used with `client.Protect()`.
func NewGenkitRetrieverAction(name string, retriever ai.Retriever, embedder ai.Embedder) core.Action {
	genkitRetriever := NewGenkitRetriever(retriever, embedder)
	return NewRetrieverAction(name, genkitRetriever)
}
