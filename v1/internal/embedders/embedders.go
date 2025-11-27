package embedders

import (
	"errors"
	"fmt"

	"github.com/duynguyendang/manglekit/v1"
	"github.com/duynguyendang/manglekit/v1/internal/embedders/google"
	"github.com/duynguyendang/manglekit/v1/internal/embedders/openai"
)

// Register registers all embedder providers with the MangleKit registry.
// Embedders are registered as thin factories that delegate to Genkit plugins:
// - "google": Google Generative AI embeddings via genkit googlegenai plugin
// - "openai": OpenAI embeddings via genkit compat_oai plugin
func Register(r *manglekit.Registry) error {
	var errs []error

	// Register native embedder thin factories
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
