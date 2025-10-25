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
			Embedder: emb,
		}
	default:
		// This case handles vector stores that might not have or need an embedder.
		// For example, a purely lexical vector store.
		deps = diapi.NoopDeps{}
	}

	if err != nil {
		return nil, err
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
