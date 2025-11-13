package embedders

import (
	"errors"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/embedders/google"
	"github.com/duynguyendang/manglekit/internal/embedders/openai"
	"github.com/duynguyendang/manglekit/internal/providers/embedders/genkitembedder"
)

// Register registers all embedder providers with the MangleKit registry.
// This includes both native Manglekit embedders and the generic Genkit embedder factory.
func Register(r *manglekit.Registry) error {
	var errs []error

	// Register native Manglekit embedders (for backward compatibility)
	if err := openai.Register(r); err != nil {
		errs = append(errs, fmt.Errorf("openai embedder registration: %w", err))
	}

	if err := google.Register(r); err != nil {
		errs = append(errs, fmt.Errorf("google embedder registration: %w", err))
	}

	// Register generic Genkit embedder factory (supports ANY Genkit provider)
	if err := genkitembedder.Register(r); err != nil {
		errs = append(errs, fmt.Errorf("genkit embedder registration: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("embedder registration failed: %w", errors.Join(errs...))
	}

	return nil
}
