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
)

// Register registers the generic Genkit retriever factory with the Manglekit registry.
// This factory supports ANY Genkit retriever provider by dispatching based on configuration.
func Register(r *manglekit.Registry) error {
	factory := func(ctx context.Context, deps diapi.NoopDeps, cfg *GenkitRetrieverOptions) (core.Retriever, error) {
		if cfg.Provider == "" {
			return nil, fmt.Errorf("provider is required in GenkitRetrieverOptions (e.g., 'pinecone', 'chroma', 'weaviate')")
		}

		if cfg.Model == "" {
			return nil, fmt.Errorf("model is required in GenkitRetrieverOptions for provider %q", cfg.Provider)
		}

		// Create a Genkit instance to work with retrievers
		genkitInst := genkit.Init(ctx)

		// Dispatch to the appropriate Genkit provider plugin to get an ai.Retriever
		genkitRetriever, err := createGenkitRetriever(ctx, genkitInst, cfg)
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
			cfg.Provider,
			deps.Obs.Logger,
		)

		// Log successful creation
		if deps.Obs.Logger != nil {
			deps.Obs.Logger.Debugf(
				"created genkit retriever via dynamic factory",
				"provider", cfg.Provider,
				"model", cfg.Model,
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
//   - "chroma": Chroma vector store
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
func createGenkitRetriever(ctx context.Context, g *genkit.Genkit, opts *GenkitRetrieverOptions) (ai.Retriever, error) {
	switch opts.Provider {
	case "openai":
		// Note: OpenAI doesn't natively provide retrievers, but we include this for completeness.
		// Users should use "pinecone" or "chroma" with OpenAI embeddings instead.
		return nil, fmt.Errorf(
			"openai is not a retriever provider (it's only an embedder provider)\n" +
				"Use 'pinecone', 'chroma', 'weaviate', 'qdrant', or 'milvus' instead with openai as the embedding model",
		)

	case "pinecone":
		return createPineconeRetriever(g, opts)

	case "chroma":
		return createChromaRetriever(g, opts)

	case "weaviate":
		return createWeaviateRetriever(g, opts)

	case "qdrant":
		return createQdrantRetriever(g, opts)

	case "milvus":
		return createMilvusRetriever(g, opts)

	case "google", "googlegenai", "vertex":
		// Google/Vertex is primarily an embedder, not a retriever
		return nil, fmt.Errorf(
			"google/vertex is not a retriever provider (it's only an embedder provider)\n" +
				"Use 'pinecone', 'chroma', 'weaviate', 'qdrant', or 'milvus' instead with google as the embedding model",
		)

	default:
		return nil, fmt.Errorf(
			"unsupported retriever provider: %q\n"+
				"Supported providers: pinecone, chroma, weaviate, qdrant, milvus, etc.\n"+
				"Tip: Ensure the provider's Genkit plugin is initialized in your genkit.Genkit instance",
			opts.Provider,
		)
	}
}

// createPineconeRetriever creates a Pinecone retriever.
// Placeholder for now; implement when Genkit Pinecone plugin is available.
func createPineconeRetriever(g *genkit.Genkit, opts *GenkitRetrieverOptions) (ai.Retriever, error) {
	// TODO: Implement when Genkit exposes Pinecone plugin
	return nil, fmt.Errorf("pinecone retriever support not yet implemented (waiting for Genkit plugin)")
}

// createChromaRetriever creates a Chroma retriever.
// Placeholder for now; implement when Genkit Chroma plugin is available.
func createChromaRetriever(g *genkit.Genkit, opts *GenkitRetrieverOptions) (ai.Retriever, error) {
	// TODO: Implement when Genkit exposes Chroma plugin
	return nil, fmt.Errorf("chroma retriever support not yet implemented (waiting for Genkit plugin)")
}

// createWeaviateRetriever creates a Weaviate retriever.
// Placeholder for now; implement when Genkit Weaviate plugin is available.
func createWeaviateRetriever(g *genkit.Genkit, opts *GenkitRetrieverOptions) (ai.Retriever, error) {
	// TODO: Implement when Genkit exposes Weaviate plugin
	return nil, fmt.Errorf("weaviate retriever support not yet implemented (waiting for Genkit plugin)")
}

// createQdrantRetriever creates a Qdrant retriever.
// Placeholder for now; implement when Genkit Qdrant plugin is available.
func createQdrantRetriever(g *genkit.Genkit, opts *GenkitRetrieverOptions) (ai.Retriever, error) {
	// TODO: Implement when Genkit exposes Qdrant plugin
	return nil, fmt.Errorf("qdrant retriever support not yet implemented (waiting for Genkit plugin)")
}

// createMilvusRetriever creates a Milvus retriever.
// Placeholder for now; implement when Genkit Milvus plugin is available.
func createMilvusRetriever(g *genkit.Genkit, opts *GenkitRetrieverOptions) (ai.Retriever, error) {
	// TODO: Implement when Genkit exposes Milvus plugin
	return nil, fmt.Errorf("milvus retriever support not yet implemented (waiting for Genkit plugin)")
}
