package vectorstores

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// VectorStoreOptions is an interface that all vector store options must satisfy.
type VectorStoreOptions interface {
	// GetEmbedderName returns the name of the embedder defined in config.
	GetEmbedderName() string
}

// Handler is the component handler for VectorStores.
type Handler struct{}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindVectorStore
}

// BuildComponent builds the VectorStore component.
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

	// Assert the options to your common interface
	vcfg, ok := cfg.(VectorStoreOptions)
	if !ok {
		// This is a safety check
		return nil, fmt.Errorf("vectorstore options do not implement GetEmbedderName(): %T", cfg)
	}

	// Get the *single dependency* all vector stores need
	emb, err := b.GetEmbedder(vcfg.GetEmbedderName())
	if err != nil {
		return nil, err
	}

	// Create the *single dependency struct* all factories expect
	deps := diapi.VectorStoreDeps{
		Embedder: emb,
	}

	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for VectorStore handler")
	}

	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindVectorStore, name, err)
	}

	store, ok := built.(core.VectorStore)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid VectorStore", name)
	}
	resolved.VectorStores[name] = store
	return core.NopCloser, nil
}
