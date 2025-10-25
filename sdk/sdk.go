package sdk

import (
	"context"
	"fmt"
	"reflect"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/providers/all"
	"github.com/mitchellh/mapstructure"
)

// Load is a high-level function that loads a Manglekit orchestrator from a YAML
// configuration byte slice. It handles registry creation and component handler
// registration automatically.
func Load(ctx context.Context, data []byte) (core.Orchestrator, error) {
	// 1. Create a new registry.
	registry := manglekit.NewRegistry()

	// 2. Create a new builder and register all component handlers.
	builder := manglekit.NewBuilder(registry).WithHandlers(all.ComponentHandlers()...)

	// 3. Build the orchestrator from the configuration.
	orch, _, err := builder.FromConfig(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("failed to build orchestrator from config: %w", err)
	}
	return orch, nil
}

// FromConfig loads a Manglekit orchestrator from a YAML configuration byte slice.
// The caller is responsible for creating and populating the registry.
func FromConfig(ctx context.Context, registry *manglekit.Registry, data []byte) (core.Orchestrator, error) {
	if registry == nil {
		return nil, fmt.Errorf("registry cannot be nil")
	}

	cfg, err := config.ParseConfig(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	builder := manglekit.NewBuilder(registry).
		WithTopK(cfg.TopK).
		WithMaxTokens(cfg.MaxTokens).
		WithOrchestrator(cfg.Orchestrator).
		WithUpdatable(cfg.Updatable).
		WithFallbackThreshold(cfg.FallbackThreshold)

	for _, comp := range cfg.Components {
		if comp.Type == "" {
			return nil, fmt.Errorf("component %q is missing required field 'type'", comp.Name)
		}
		var foundType reflect.Type
		for t, name := range registry.OptionsTypeToName {
			if name == comp.Type && registry.OptionsTypeToKind[t] == comp.Kind {
				foundType = t
				break
			}
		}

		if foundType == nil {
			return nil, fmt.Errorf("could not find options type for kind=%s, type=%s", comp.Kind, comp.Type)
		}

		// Create a new instance of the options struct.
		optsPtr := reflect.New(foundType)
		opts := optsPtr.Interface()

		// Unmarshal the YAML params into the new options struct.
		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Result:           opts,
			WeaklyTypedInput: true,
			TagName:          "yaml",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create mapstructure decoder: %w", err)
		}
		if err := decoder.Decode(comp.Params); err != nil {
			return nil, fmt.Errorf("failed to decode params for %s '%s': %w", comp.Kind, comp.Name, err)
		}

		builder.With(comp.Name, opts)
	}

	orch, _, err := builder.Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build orchestrator: %w", err)
	}
	return orch, nil
}
