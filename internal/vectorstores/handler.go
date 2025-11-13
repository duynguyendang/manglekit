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

// Handler is the component handler for VectorStores.
// This handler processes native Manglekit VectorStore factories only.
// For Genkit vector store providers (e.g., Pinecone, Chroma), use the
// genkit-vectorstore factory in internal/providers/vectorstores/genkitvectorstore/,
// which wraps Genkit retrievers via the GenkitVectorStoreAdapter.
type Handler struct{}

// NewHandler returns a new ComponentHandler for VectorStores.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindVectorStore
}

// BuildComponent builds the VectorStore component using native Manglekit factories only.
// This handler does NOT attempt to fallback to Genkit delegation.
// For Genkit-backed vector stores, use the genkit-vectorstore factory instead.
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

	// Build using the native Manglekit factory
	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindVectorStore, name, err)
	}

	// Verify the result is a valid VectorStore
	store, ok := built.(core.VectorStore)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid VectorStore", name)
	}

	resolved.VectorStores[name] = store
	return core.NopCloser, nil
}
