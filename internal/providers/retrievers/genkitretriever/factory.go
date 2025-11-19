package genkitretriever

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/adapters"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/localvec"
)

// Register registers the generic Genkit retriever factory with the Manglekit registry.
// This factory supports ANY Genkit retriever provider by dispatching based on configuration.
func Register(r *manglekit.Registry) error {
	factory := func(ctx context.Context, deps diapi.GenkitRetrieverDeps, cfg *GenkitRetrieverOptions) (core.Retriever, error) {
		if cfg.Provider == "" {
			return nil, fmt.Errorf("provider is required in GenkitRetrieverOptions (e.g., 'pinecone', 'localvec', 'weaviate')")
		}

		if deps.Embedder == nil {
			return nil, fmt.Errorf("embedder dependency is required but nil")
		}

		// Create a Genkit instance to work with retrievers
		genkitInst := genkit.Init(ctx)

		// Dispatch to the appropriate Genkit provider plugin to get an ai.Retriever
		genkitRetriever, indexer, err := createGenkitRetriever(ctx, genkitInst, cfg, deps.Embedder)
		if err != nil {
			return nil, fmt.Errorf("failed to create genkit retriever for provider %q: %w", cfg.Provider, err)
		}

		if genkitRetriever == nil {
			return nil, fmt.Errorf("genkit provider %q returned nil retriever", cfg.Provider)
		}

		// Wrap the Genkit retriever in our adapter
		adapter := adapters.NewGenkitRetrieverAdapter(
			genkitInst,
			genkitRetriever,
			indexer,
			cfg.Provider,
			deps.CoreDeps.Obs.Logger,
		)

		// Log successful creation
		if deps.CoreDeps.Obs.Logger != nil {
			deps.CoreDeps.Obs.Logger.Debugf(
				"created genkit retriever via dynamic factory",
				"provider", cfg.Provider,
				"embedder", cfg.Embedder,
			)
		}

		return adapter, nil
	}

	return manglekit.Register(r, &GenkitRetrieverOptions{}, factory)
}

// createGenkitRetriever dispatches to the appropriate Genkit provider plugin based on configuration.
// This is the extensibility point: new providers are added here via switch cases.
//
// Supported providers:
//   - "pinecone": Pinecone vector database
//   - "localvec": Firebase Genkit LocalVec (file-based, for development)
//   - "weaviate": Weaviate vector database
//   - "qdrant": Qdrant vector database
//   - "milvus": Milvus vector store
//   - Others: Any Genkit-compatible retriever plugin
//
// To add support for a new provider:
//  1. Ensure the Genkit plugin package is available
//  2. Add a case to this switch statement
//  3. Implement the provider creation logic
//  4. Update documentation
//  5. NO Manglekit recompilation needed if using ProviderConfig for custom params
func createGenkitRetriever(ctx context.Context, g *genkit.Genkit, opts *GenkitRetrieverOptions, embedder ai.Embedder) (ai.Retriever, adapters.GenkitIndexer, error) {
	switch opts.Provider {
	case "openai":
		// Note: OpenAI doesn't natively provide retrievers, but we include this for completeness.
		// Users should use "pinecone" or "localvec" with OpenAI embeddings instead.
		return nil, nil, fmt.Errorf(
			"openai is not a retriever provider (it's only an embedder provider)\n" +
				"Use 'pinecone', 'localvec', 'weaviate', 'qdrant', or 'milvus' instead with openai as the embedding model",
		)

	case "localvec":
		return createLocalVecRetriever(g, opts, embedder)

	case "pinecone":
		return createPineconeRetriever(g, opts, embedder)

	case "weaviate":
		return createWeaviateRetriever(g, opts, embedder)

	case "qdrant":
		return createQdrantRetriever(g, opts, embedder)

	case "milvus":
		return createMilvusRetriever(g, opts, embedder)

	case "google", "googlegenai", "vertex":
		// Google/Vertex is primarily an embedder, not a retriever
		return nil, nil, fmt.Errorf(
			"google/vertex is not a retriever provider (it's only an embedder provider)\n" +
				"Use 'pinecone', 'localvec', 'weaviate', 'qdrant', or 'milvus' instead with google as the embedding model",
		)

	default:
		return nil, nil, fmt.Errorf(
			"unsupported retriever provider: %q\n"+
				"Supported providers: pinecone, localvec, weaviate, qdrant, milvus, etc.\n"+
				"Tip: Ensure the provider's Genkit plugin is initialized in your genkit.Genkit instance",
			opts.Provider,
		)
	}
}

// createPineconeRetriever creates a Pinecone retriever.
// Placeholder for now; implement when Genkit Pinecone plugin is available.
func createPineconeRetriever(g *genkit.Genkit, opts *GenkitRetrieverOptions, embedder ai.Embedder) (ai.Retriever, adapters.GenkitIndexer, error) {
	// TODO: Implement when Genkit exposes Pinecone plugin
	return nil, nil, fmt.Errorf("pinecone retriever support not yet implemented (waiting for Genkit plugin)")
}

// localVecIndexerWrapper wraps localvec.DocStore to implement adapters.GenkitIndexer
type localVecIndexerWrapper struct {
	docStore *localvec.DocStore
}

func (w *localVecIndexerWrapper) Index(ctx context.Context, docs []*ai.Document) error {
	return localvec.Index(ctx, docs, w.docStore)
}

// createLocalVecRetriever creates a local vector retriever using Firebase Genkit's LocalVec plugin.
// LocalVec is a lightweight, file-based vector database perfect for development and testing.
func createLocalVecRetriever(g *genkit.Genkit, opts *GenkitRetrieverOptions, embedder ai.Embedder) (ai.Retriever, adapters.GenkitIndexer, error) {
	// Initialize the LocalVec plugin if not already done
	if err := localvec.Init(); err != nil {
		return nil, nil, fmt.Errorf("failed to initialize localvec plugin: %w", err)
	}

	if embedder == nil {
		return nil, nil, fmt.Errorf("embedder is required for LocalVec retriever (got nil)")
	}

	// Configure LocalVec with the specified directory and embedder
	config := localvec.Config{
		Dir:      opts.Endpoint, // Use endpoint as the storage directory
		Embedder: embedder,
	}

	// Define the retriever with LocalVec
	docStore, retriever, err := localvec.DefineRetriever(
		g,
		opts.IndexName,
		config,
		&ai.RetrieverOptions{
			Label: fmt.Sprintf("LocalVec Collection: %s", opts.IndexName),
		},
	)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to define localvec retriever: %w", err)
	}

	// Wrap docStore in our indexer interface
	indexer := &localVecIndexerWrapper{docStore: docStore}

	return retriever, indexer, nil
}

// createWeaviateRetriever creates a Weaviate retriever.
// Placeholder for now; implement when Genkit Weaviate plugin is available.
func createWeaviateRetriever(g *genkit.Genkit, opts *GenkitRetrieverOptions, embedder ai.Embedder) (ai.Retriever, adapters.GenkitIndexer, error) {
	// TODO: Implement when Genkit exposes Weaviate plugin
	return nil, nil, fmt.Errorf("weaviate retriever support not yet implemented (waiting for Genkit plugin)")
}

// createQdrantRetriever creates a Qdrant retriever.
// Placeholder for now; implement when Genkit Qdrant plugin is available.
func createQdrantRetriever(g *genkit.Genkit, opts *GenkitRetrieverOptions, embedder ai.Embedder) (ai.Retriever, adapters.GenkitIndexer, error) {
	// TODO: Implement when Genkit exposes Qdrant plugin
	return nil, nil, fmt.Errorf("qdrant retriever support not yet implemented (waiting for Genkit plugin)")
}

// createMilvusRetriever creates a Milvus retriever.
// Placeholder for now; implement when Genkit Milvus plugin is available.
func createMilvusRetriever(g *genkit.Genkit, opts *GenkitRetrieverOptions, embedder ai.Embedder) (ai.Retriever, adapters.GenkitIndexer, error) {
	// TODO: Implement when Genkit exposes Milvus plugin
	return nil, nil, fmt.Errorf("milvus retriever support not yet implemented (waiting for Genkit plugin)")
}
