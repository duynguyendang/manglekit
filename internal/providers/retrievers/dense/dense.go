// Package dense provides an implementation of a dense retriever, which performs
// semantic search using vector embeddings.
package dense

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/firebase/genkit/go/ai"
)

// DenseOptions provides a type-safe way to configure a dense (vector-based)
// retriever. It explicitly declares its dependencies by name, allowing the
// builder to inject the correct components.
type DenseOptions struct {
	Embedder    string `yaml:"embedder"`
	VectorStore string `yaml:"vectorStore"`
}

func (o *DenseOptions) ProviderName() string { return "dense" }
func (o *DenseOptions) ProviderKind() core.Kind   { return core.KindRetriever }
func (o *DenseOptions) GetEmbedderName() string { return o.Embedder }
func (o *DenseOptions) GetEmbedder() string    { return o.Embedder }
func (o *DenseOptions) GetVectorStore() string { return o.VectorStore }
func (o *DenseOptions) GetProviderOptions() any { return o }

func Register(r *manglekit.Registry) {
	manglekit.Register(r, &DenseOptions{},
		func(ctx context.Context, deps diapi.DenseRetrieverDeps, cfg *DenseOptions) (core.Retriever, error) {
			if deps.Embedder == nil {
				return nil, fmt.Errorf("dense retriever factory requires an 'embedder' dependency, but it was not provided")
			}
			if deps.VectorStore == nil {
				return nil, fmt.Errorf("dense retriever factory requires a 'vectorStore' dependency, but it was not provided")
			}
			return New(deps)
		},
	)
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
// It returns an initialized `core.Retriever` or an error if dependencies are missing.
func New(deps diapi.DenseRetrieverDeps) (core.Retriever, error) {
	if deps.Embedder == nil {
		return nil, fmt.Errorf("dense: an embedder is required")
	}
	if deps.VectorStore == nil {
		return nil, fmt.Errorf("dense: a vectorStore is required")
	}

	return &Dense{
		embedder:    deps.Embedder,
		vectorStore: deps.VectorStore,
	}, nil
}

// Retrieve performs a search by first converting the query text into a vector
// embedding, and then using that vector to search the configured vector store.
// This method satisfies the `retrieve.Retriever` interface.
//
// ctx is the context for the API call.
// req contains the query string, the number of results to return (`TopK`), and
// any metadata to be used for filtering within the vector store.
// It returns a `core.RetrieveResult` containing the documents found by the vector
// store, or an error if the query embedding or vector search fails.
func (d *Dense) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	// 1. Embed the query text.
	embedReq := &ai.EmbedRequest{
		Input: []*ai.Document{ai.DocumentFromText(req.Query, nil)},
	}
	embedResp, err := d.embedder.Embed(ctx, embedReq)
	if err != nil {
		return core.RetrieveResult{}, fmt.Errorf("dense: failed to embed query: %w", err)
	}
	if len(embedResp.Embeddings) == 0 {
		return core.RetrieveResult{}, fmt.Errorf("dense: embedder returned no embeddings for query")
	}
	queryVector := embedResp.Embeddings[0].Embedding

	// 2. Search the vector store.
	var filter map[string]any
	if f, ok := req.Meta["filters"].(map[string]any); ok {
		filter = f
	}

	docs, err := d.vectorStore.Search(ctx, req.Query, queryVector, req.TopK, filter)
	if err != nil {
		return core.RetrieveResult{}, fmt.Errorf("dense: vector store search failed: %w", err)
	}

	return core.RetrieveResult{Docs: docs}, nil
}
