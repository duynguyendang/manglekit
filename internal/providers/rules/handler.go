package rules

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

// Handler is the component handler for RuleSets.
type Handler struct{}

// Kind returns the component kind.
func (h *Handler) Kind() core.Kind {
	return core.KindRules
}

// BuildComponent builds the RuleSet component and assigns it to the resolved map.
func (h *Handler) BuildComponent(
	ctx context.Context,
	builderDI any,
	factory any,
	resolved *core.Resolved,
	cfg core.ProviderOptions,
	name string,
) (core.ResourceCloser, error) {
	deps := diapi.RuleSetDeps{}
	f, ok := factory.(core.Factory)
	if !ok {
		return nil, fmt.Errorf("invalid factory type for RuleSet handler")
	}
	built, err := f.Build(ctx, deps, cfg)
	if err != nil {
		return nil, fmt.Errorf("factory for %s '%s' failed: %w", core.KindRules, name, err)
	}

	ruleSet, ok := built.(core.RuleSet)
	if !ok {
		return nil, fmt.Errorf("component %s is not a valid RuleSet", name)
	}
	resolved.Rules[name] = ruleSet

	return core.NopCloser, nil
}
