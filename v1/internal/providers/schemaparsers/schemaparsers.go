package schemaparsers

import (
	"errors"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers/jsonschema"
	"github.com/duynguyendang/manglekit/internal/providers/schemaparsers/rdf"
)

// Register registers all schema parser providers with the MangleKit registry.
func Register(r *manglekit.Registry) error {
	var errs []error

	if err := jsonschema.Register(r); err != nil {
		errs = append(errs, fmt.Errorf("jsonschema parser registration: %w", err))
	}

	if err := rdf.Register(r); err != nil {
		errs = append(errs, fmt.Errorf("rdf parser registration: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("schema parser registration failed: %w", errors.Join(errs...))
	}

	return nil
}
