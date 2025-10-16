package manglekit

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/duynguyendang/manglekit/config"
)

// NewBuilderFromConfig creates a new Builder instance from a validated Config object.
// This function is the primary entrypoint for creating a builder from a static configuration.
// It translates the config struct into a series of type-safe builder calls.
func NewBuilderFromConfig(ctx context.Context, cfg *config.Config, reg *Registry) (*Builder, error) {
	// First, normalize and validate the configuration.
	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	b := NewBuilder(reg)

	// Helper function to resolve provider options from a map[string]any to a typed struct.
	resolveOptions := func(providerName string, optionsMap map[string]any) (any, error) {
		if optionsMap == nil {
			// If no options are provided, we may still need an empty struct.
			optionsMap = make(map[string]any)
		}

		optsType, ok := reg.NameToOptionsType(providerName)
		if !ok {
			return nil, fmt.Errorf("no options type registered for provider %q", providerName)
		}

		optsPtr := reflect.New(optsType.Elem()).Interface()

		jsonParams, err := json.Marshal(optionsMap)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params for %q: %w", providerName, err)
		}
		if err := json.Unmarshal(jsonParams, optsPtr); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params into options struct for %q: %w", providerName, err)
		}
		return optsPtr, nil
	}

	// Configure LLM
	if cfg.LLM != nil {
		opts, err := resolveOptions(cfg.LLM.Provider, cfg.LLM.Options)
		if err != nil {
			return nil, fmt.Errorf("failed to configure llm: %w", err)
		}
		b.WithLLM(opts)
	}

	// Configure Embedder
	if cfg.Embedder != nil {
		opts, err := resolveOptions(cfg.Embedder.Provider, cfg.Embedder.Options)
		if err != nil {
			return nil, fmt.Errorf("failed to configure embedder: %w", err)
		}
		b.WithEmbedder(opts)
	}

	// Configure Retriever
	if cfg.Retrieve != nil {
		opts, err := resolveOptions(cfg.Retrieve.Provider, cfg.Retrieve.Options)
		if err != nil {
			return nil, fmt.Errorf("failed to configure retriever: %w", err)
		}
		b.WithRetriever(opts)
	}

	// Configure Reranker
	if cfg.Rerank != nil {
		opts, err := resolveOptions(cfg.Rerank.Provider, cfg.Rerank.Options)
		if err != nil {
			return nil, fmt.Errorf("failed to configure reranker: %w", err)
		}
		b.WithReranker(opts)
	}

	// Configure Vector Store
	if cfg.Vector != nil {
		opts, err := resolveOptions(cfg.Vector.Provider, cfg.Vector.Options)
		if err != nil {
			return nil, fmt.Errorf("failed to configure vector store: %w", err)
		}
		b.WithVectorStore(opts)
	}

	// Configure State Provider
	if cfg.State != nil {
		opts, err := resolveOptions(cfg.State.Provider, cfg.State.Options)
		if err != nil {
			return nil, fmt.Errorf("failed to configure state provider: %w", err)
		}
		b.WithStateProvider(opts)
	}

	// Configure TopK and MaxTokens
	b.WithTopK(cfg.TopK)
	b.WithMaxTokens(cfg.MaxTokens)

	return b, nil
}