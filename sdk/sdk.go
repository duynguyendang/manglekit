package sdk

import (
	"context"
	"fmt"
	"reflect"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/mitchellh/mapstructure"
)

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

	builder := manglekit.NewBuilder(registry)
	for _, comp := range cfg.Components {

		var foundType reflect.Type
		for t, name := range registry.OptionsTypeToName {
			if name == comp.Name && registry.OptionsTypeToKind[t] == comp.Kind {
				foundType = t
				break
			}
		}

		if foundType == nil {
			return nil, fmt.Errorf("could not find options type for kind=%s, name=%s", comp.Kind, comp.Name)
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

		builder.With(opts)
	}

	orch, _, err := builder.Build(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build orchestrator: %w", err)
	}
	return orch, nil
}
