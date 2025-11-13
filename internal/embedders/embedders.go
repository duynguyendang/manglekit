package embedders

import (
	"errors"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/embedders/google"
	"github.com/duynguyendang/manglekit/internal/embedders/openai"
)

// Register registers all embedder providers with the MangleKit registry.
func Register(r *manglekit.Registry) error {
	var errs []error

	if err := openai.Register(r); err != nil {
		errs = append(errs, fmt.Errorf("openai embedder registration: %w", err))
	}

	if err := google.Register(r); err != nil {
		errs = append(errs, fmt.Errorf("google embedder registration: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("embedder registration failed: %w", errors.Join(errs...))
	}

	return nil
}
