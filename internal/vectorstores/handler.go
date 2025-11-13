package vectorstores

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/firebase/genkit/go/ai"
)

// VectorStoreOptions is an interface that all vector store options must satisfy.
type VectorStoreOptions interface {
	// GetEmbedderName returns the name of the embedder defined in config.
	GetEmbedderName() string
}

// ProviderNameGetter is an interface for options that have a ProviderName method.
type ProviderNameGetter interface {
	ProviderName() string
}

// Handler is the component handler for VectorStores.
type Handler struct{}

// NewHandler returns a new ComponentHandler for VectorStores.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindVectorStore
}

// genkitRetrieverAdapter wraps a Manglekit Retriever (which may delegate to Genkit)
// and adapts it to the core.VectorStore interface. It delegates Search to the wrapped
// retriever and returns ErrNotSupported for AddDocuments, as Genkit-delegated retrievers
// are read-only.
type genkitRetrieverAdapter struct {
	retriever core.Retriever
	logger    core.Logger
}

// newGenkitRetrieverAdapter creates a new adapter for a retriever.
func newGenkitRetrieverAdapter(retriever core.Retriever, logger core.Logger) *genkitRetrieverAdapter {
	return &genkitRetrieverAdapter{
		retriever: retriever,
		logger:    logger,
	}
}

// Search adapts a Manglekit Search request into a Manglekit Retrieve request and
// returns the results as core.Doc slices.
func (g *genkitRetrieverAdapter) Search(
	ctx context.Context,
	queryText string,
	queryVector []float32,
	topK int,
	filter map[string]any,
) ([]core.Doc, error) {
	// Create a Manglekit retrieval request
	req := core.RetrieveRequest{
		Query: queryText,
		TopK:  topK,
		Meta:  filter,
	}

	// Call the underlying retriever
	result, err := g.retriever.Retrieve(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("retriever delegation failed: %w", err)
	}

	return result.Docs, nil
}

// AddDocuments returns ErrNotSupported for Genkit-delegated retrievers, as they
// are read-only and do not support document ingestion.
func (g *genkitRetrieverAdapter) AddDocuments(
	ctx context.Context,
	docs []core.Doc,
) error {
	if g.logger != nil {
		g.logger.Debugf(
			"AddDocuments called on read-only vector store (Genkit-delegated retriever)",
			"operation", "add_documents",
			"result", "not_supported",
		)
	}
	return core.ErrNotSupported
}

// BuildComponent builds the VectorStore component. It first attempts to find
// a native Manglekit VectorStore factory. If found, it executes the native path.
// If not found (e.g., provider: "pinecone"), it delegates to a Genkit-based retriever.
func (h *Handler) BuildComponent(
	ctx context.Context,
	builderDI any,
	factory any,
	resolved *core.Resolved,
	cfg core.ProviderOptions,
	name string,
) (core.ResourceCloser, error) {
	b, ok := builderDI.(diapi.Builder)
	if !ok {
		return nil, fmt.Errorf("invalid builder DI type for VectorStore handler: got %T", builderDI)
	}

	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for VectorStore handler: got %T", factory)
	}

	var deps any
	var err error

	switch vcfg := cfg.(type) {
	case VectorStoreOptions:
		var emb ai.Embedder
		embedderName := vcfg.GetEmbedderName()
		if embedderName != "" {
			emb, err = b.GetEmbedder(embedderName)
			if err != nil {
				return nil, fmt.Errorf("failed to get embedder '%s' for vector store '%s': %w", embedderName, name, err)
			}
		}
		deps = diapi.VectorStoreDeps{
			CoreDeps: b.GetCoreDeps(),
			Embedder: emb,
		}
	default:
		// This case handles vector stores that might not have or need an embedder.
		// For example, a purely lexical vector store.
		deps = diapi.NoopDeps{
			CoreDeps: b.GetCoreDeps(),
		}
	}

	if err != nil {
		return nil, err
	}

	// STEP 1: Attempt to build using the native Manglekit factory
	built, err := f.Build(ctx, deps, cfg)
	if err == nil {
		// Native build succeeded
		store, ok := built.(core.VectorStore)
		if !ok {
			return nil, fmt.Errorf("component %s is not a valid VectorStore", name)
		}
		resolved.VectorStores[name] = store
		return core.NopCloser, nil
	}

	// STEP 2: If native factory failed, try to delegate to a retriever (Genkit-based)
	// Extract provider name from config
	providerName := extractProviderName(cfg)
	if providerName == "" {
		return nil, fmt.Errorf(
			"native VectorStore factory for '%s' failed (%v) and provider name could not be extracted for Genkit delegation",
			name, err,
		)
	}

	// Log the delegation attempt
	coreDeps := b.GetCoreDeps()
	if coreDeps.Obs.Logger != nil {
		coreDeps.Obs.Logger.Debugf(
			"Native VectorStore factory not found. Attempting Genkit delegation to retriever.",
			"vector_store_name", name,
			"provider_name", providerName,
			"native_error", err.Error(),
		)
	}

	// Delegate to Genkit by retrieving a retriever with the provider name
	// The expectation is that the retriever (e.g., pinecone, chroma) is configured
	// elsewhere in the config and can be resolved by name.
	retriever, err := b.GetRetriever(providerName)
	if err != nil {
		return nil, fmt.Errorf(
			"native VectorStore factory failed and Genkit delegation to retriever '%s' also failed: %w",
			providerName, err,
		)
	}

	// Wrap the retriever in the adapter
	adapter := newGenkitRetrieverAdapter(retriever, coreDeps.Obs.Logger)
	resolved.VectorStores[name] = adapter

	if coreDeps.Obs.Logger != nil {
		coreDeps.Obs.Logger.Debugf(
			"Successfully delegated VectorStore to Genkit retriever",
			"vector_store_name", name,
			"genkit_retriever", providerName,
		)
	}

	return core.NopCloser, nil
}

// extractProviderName extracts the provider name from the config options.
func extractProviderName(cfg core.ProviderOptions) string {
	// Try ProviderName() method if available
	if pn, ok := cfg.(ProviderNameGetter); ok {
		return pn.ProviderName()
	}
	return ""
}
