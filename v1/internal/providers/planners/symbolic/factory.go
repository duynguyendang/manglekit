package symbolic

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// Options configures the symbolic planner.
type Options struct {
	// ReasonerName specifies which reasoner to use for plan generation.
	// This must reference a registered reasoner component.
	ReasonerName string `yaml:"reasoner"`
}

// ProviderName returns the unique name for this planner provider.
func (o *Options) ProviderName() string {
	return "symbolic"
}

// ProviderKind returns the component kind (planner).
func (o *Options) ProviderKind() core.Kind {
	return core.KindPlanner
}

// GetProviderOptions returns the options for type-safe access.
func (o *Options) GetProviderOptions() any {
	return o
}

// Register registers the symbolic planner factory with the registry.
func Register(r *manglekit.Registry) {
	manglekit.Register(r, &Options{},
		func(ctx context.Context, deps diapi.PlannerDeps, cfg *Options) (core.Planner, error) {
			return NewFactory(deps, cfg)
		},
	)
}

// NewFactory creates a new symbolic planner instance.
// It validates the configuration and resolves the required reasoner
// from the dependency map.
func NewFactory(deps diapi.PlannerDeps, cfg *Options) (core.Planner, error) {
	// Validate configuration
	if cfg.ReasonerName == "" {
		return nil, fmt.Errorf("symbolic planner requires a reasoner name")
	}

	// Look up the reasoner
	reasoner, ok := deps.Reasoners[cfg.ReasonerName]
	if !ok {
		availableReasoners := make([]string, 0, len(deps.Reasoners))
		for name := range deps.Reasoners {
			availableReasoners = append(availableReasoners, name)
		}
		return nil, fmt.Errorf("reasoner '%s' not found; available reasoners: %v",
			cfg.ReasonerName, availableReasoners)
	}

	// Get logger from core dependencies
	logger := deps.CoreDeps.Obs.Logger
	if logger == nil {
		return nil, fmt.Errorf("logger not provided in core dependencies")
	}

	// Create and return the planner
	return &SymbolicPlanner{
		log:      logger,
		reasoner: reasoner,
	}, nil
}
