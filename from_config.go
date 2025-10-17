package manglekit

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
)

import "github.com/firebase/genkit/go/genkit"

// NewBuilderFromConfig creates a new Builder instance from a validated Config object.
func NewBuilderFromConfig(ctx context.Context, cfg *config.Config, reg *Registry, g *genkit.Genkit) (*Builder, error) {
	cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	if g == nil {
		g = genkit.Init(ctx)
	}
	b := NewBuilder(reg)
	b.WithGenkit(g)

	// Helper to resolve provider options from a map[string]any to a typed struct.
	resolveOptions := func(providerName string, kind core.Kind, optionsMap map[string]any) (any, error) {
		if optionsMap == nil {
			optionsMap = make(map[string]any)
		}

		// Find the options type associated with the provider name and kind.
		var optsType reflect.Type
		for t, name := range reg.optionsTypeToName {
			if name == providerName && reg.optionsTypeToKind[t] == kind {
				optsType = t
				break
			}
		}
		if optsType == nil {
			return nil, fmt.Errorf("no options type registered for provider %q with kind %q", providerName, kind)
		}

		optsPtr := reflect.New(optsType).Interface()
		jsonParams, err := json.Marshal(optionsMap)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params for %q: %w", providerName, err)
		}
		if err := json.Unmarshal(jsonParams, optsPtr); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params for %q: %w", providerName, err)
		}
		return reflect.ValueOf(optsPtr).Elem().Interface(), nil
	}

	// Configure components using the new WithKind method.
	if cfg.LLM != nil {
		opts, err := resolveOptions(cfg.LLM.Provider, core.KindLLM, cfg.LLM.Options)
		if err != nil {
			return nil, err
		}
		b.WithKind(core.KindLLM, cfg.LLM.Provider, opts)
	}
	if cfg.Embedder != nil {
		opts, err := resolveOptions(cfg.Embedder.Provider, core.KindEmbedder, cfg.Embedder.Options)
		if err != nil {
			return nil, err
		}
		b.WithKind(core.KindEmbedder, cfg.Embedder.Provider, opts)
	}
	if cfg.Retrieve != nil {
		opts, err := resolveOptions(cfg.Retrieve.Provider, core.KindRetriever, cfg.Retrieve.Options)
		if err != nil {
			return nil, err
		}
		b.WithKind(core.KindRetriever, cfg.Retrieve.Provider, opts)
	}
	if cfg.Rerank != nil {
		opts, err := resolveOptions(cfg.Rerank.Provider, core.KindReranker, cfg.Rerank.Options)
		if err != nil {
			return nil, err
		}
		b.WithKind(core.KindReranker, cfg.Rerank.Provider, opts)
	}
	if cfg.Vector != nil {
		opts, err := resolveOptions(cfg.Vector.Provider, core.KindVectorStore, cfg.Vector.Options)
		if err != nil {
			return nil, err
		}
		b.WithKind(core.KindVectorStore, cfg.Vector.Provider, opts)
	}
	if cfg.State != nil {
		opts, err := resolveOptions(cfg.State.Provider, core.KindStateProvider, cfg.State.Options)
		if err != nil {
			return nil, err
		}
		b.WithKind(core.KindStateProvider, cfg.State.Provider, opts)
	}

	b.WithTopK(cfg.TopK)
	b.WithMaxTokens(cfg.MaxTokens)

	return b, nil
}
