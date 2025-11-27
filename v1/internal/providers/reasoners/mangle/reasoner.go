// Package mangle provides a Reasoner implementation using the Mangle Datalog engine.
package mangle

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/v1"
	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
	"github.com/duynguyendang/manglekit/v1/internal/providers/rules/mangle"
	"github.com/google/mangle/ast"
)

// Reasoner is a Mangle Datalog engine that implements the core.Reasoner interface.
type Reasoner struct {
	RuleSet *mangle.RuleSet
}

// Execute runs the Mangle Datalog engine to perform reasoning.
func (r *Reasoner) Execute(ctx context.Context, req core.ReasonerRequest) (core.ReasonerResponse, error) {
	// 1. Convert input map to Mangle facts.
	// This is a simplified conversion. A more robust implementation would
	// use a configurable converter.
	var inputFacts []ast.Atom
	for key, value := range req.Input {
		inputFacts = append(inputFacts, ast.NewAtom(
			key,
			ast.String(fmt.Sprintf("%v", value)),
		))
	}

	// 2. Call the underlying RuleSet's Reason method.
	outputFacts, err := r.RuleSet.Reason(ctx, inputFacts)
	if err != nil {
		return core.ReasonerResponse{}, err
	}

	// 3. Convert output facts back to a map.
	// This is also a simplified conversion.
	output := make(map[string]any)
	for _, fact := range outputFacts {
		if fact.Predicate.Arity == 2 {
			key, err := constantToString(fact.Args[0].(ast.Constant))
			if err != nil {
				continue
			}
			value, err := constantToString(fact.Args[1].(ast.Constant))
			if err != nil {
				continue
			}
			output[key] = value
		}
	}

	return core.ReasonerResponse{Output: output}, nil
}

// Register registers the Mangle Reasoner provider.
func Register(r *manglekit.Registry) {
	// Re-using MangleOptions for the Reasoner.
	// A wrapper might be needed if the options diverge.
	reasonerOpts := &core.MangleOptions{}
	manglekit.Register(r, reasonerOpts,
		func(ctx context.Context, deps diapi.RuleSetDeps, cfg *core.MangleOptions) (core.Reasoner, error) {
			rs, err := mangle.New(ctx, *cfg, deps.Registry.(*manglekit.Registry))
			if err != nil {
				return nil, fmt.Errorf("failed to create Mangle RuleSet for Reasoner: %w", err)
			}
			return &Reasoner{RuleSet: rs}, nil
		},
	)
}

func constantToString(c ast.Constant) (string, error) {
	if v, err := c.StringValue(); err == nil {
		return v, nil
	}
	if v, err := c.NameValue(); err == nil {
		return v, nil
	}
	return "", fmt.Errorf("unsupported constant type: %v", c.Type)
}
