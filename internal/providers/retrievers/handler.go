package retrievers

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// Handler is the component handler for Retrievers.
type Handler struct{}

// NewHandler returns a new ComponentHandler for Retrievers.
func NewHandler() core.ComponentHandler {
	return &Handler{}
}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindRetriever
}

// BuildComponent builds the Retriever component by multiplexing based on the options type.
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
		return nil, fmt.Errorf("invalid builder DI type for Retriever handler: got %T", builderDI)
	}

	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for Retriever handler: got %T", factory)
	}

	providerWithOptions, ok := cfg.(diapi.ProviderWithOptions)
	if !ok {
		return nil, fmt.Errorf("provider options for %s do not implement diapi.ProviderWithOptions", name)
	}
	opts := providerWithOptions.GetProviderOptions()

	coreDeps := b.GetCoreDeps()
	var deps any
	switch typedOpts := opts.(type) {
	case diapi.SubRetrieversDep:
		hybridDeps := diapi.RetrieverDeps{
			CoreDeps:      coreDeps,
			SubRetrievers: make(map[string]core.Retriever),
		}
		for _, subName := range typedOpts.GetSubRetrievers() {
			r, err := b.GetRetriever(subName)
			if err != nil {
				return nil, fmt.Errorf("failed to get sub-retriever '%s' for hybrid retriever '%s': %w", subName, name, err)
			}
			hybridDeps.SubRetrievers[subName] = r
		}
		deps = hybridDeps

	case diapi.EmbedderDep:
		embedder, err := b.GetEmbedder(typedOpts.GetEmbedder())
		if err != nil {
			return nil, fmt.Errorf("failed to get embedder '%s' for dense retriever '%s': %w", typedOpts.GetEmbedder(), name, err)
		}

		if vsDep, ok := opts.(diapi.VectorStoreDep); ok {
			vs, err := b.GetVectorStore(vsDep.GetVectorStore())
			if err != nil {
				return nil, fmt.Errorf("failed to get vector store '%s' for dense retriever '%s': %w", vsDep.GetVectorStore(), name, err)
			}
			deps = diapi.DenseRetrieverDeps{
				CoreDeps:    coreDeps,
				Embedder:    embedder,
				VectorStore: vs,
			}
		} else {
			return nil, fmt.Errorf("dense retriever '%s' is missing a vector store dependency", name)
		}

	default:
		deps = diapi.NoopDeps{CoreDeps: coreDeps}
	}

	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindRetriever, name, err)
	}

	retriever, ok := built.(core.Retriever)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid Retriever", name)
	}
	if err := b.SetRetriever(name, retriever); err != nil {
		return nil, fmt.Errorf("failed to set retriever '%s': %w", name, err)
	}
	return core.NopCloser, nil
}
