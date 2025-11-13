package genkitvectorstore

import (
"context"
"fmt"

"github.com/duynguyendang/manglekit"
"github.com/duynguyendang/manglekit/core"
"github.com/duynguyendang/manglekit/core/diapi"
)

// Register registers the genkit-vectorstore factory with the Manglekit registry.
// This factory creates vector store instances by wrapping Genkit retrievers
// and adapting them via GenkitVectorStoreAdapter.
func Register(r *manglekit.Registry) {
manglekit.Register(r, &GenkitVectorStoreOptions{},
func(ctx context.Context, deps diapi.VectorStoreDeps, cfg *GenkitVectorStoreOptions) (core.VectorStore, error) {
// Validate required fields
if cfg.Provider == "" {
return nil, fmt.Errorf("genkit-vectorstore: provider is required")
}

if cfg.Index == "" {
return nil, fmt.Errorf("genkit-vectorstore: index is required")
}

// TODO: In a future implementation, this factory would:
// 1. Initialize the Genkit vector store plugin (e.g., Pinecone, Chroma)
// 2. Create a Genkit retriever from the vector store
// 3. Wrap the retriever in the GenkitVectorStoreAdapter
//
// For now, this factory is a placeholder. The correct pattern is:
// - Genkit provides vector store backends (Pinecone, Chroma, etc.)
// - Manglekit retrievers (e.g., dense, hybrid) depend on core.VectorStore
// - GenkitVectorStoreAdapter wraps Genkit retrievers as core.VectorStore

// This is a placeholder; full implementation requires Genkit SDK setup
return nil, fmt.Errorf("genkit-vectorstore: factory implementation in progress for provider %q", cfg.Provider)
},
)
}

// NOTE: The GenkitVectorStoreAdapter in internal/adapters/genkit_vectorstore_adapter.go
// is designed to wrap a core.Retriever (which may be backed by Genkit)
// and adapt it to the core.VectorStore interface.
// This allows dense and hybrid retrievers to accept any Genkit vector store backend
// without hard-coding provider logic.
