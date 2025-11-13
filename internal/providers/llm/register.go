package llm

import (
	"fmt"

	"github.com/duynguyendang/manglekit"
)

// Register registers all LLM providers with the MangleKit registry.
func Register(r *manglekit.Registry) error {
	// Register native providers
	RegisterOpenAI(r)
	RegisterGoogle(r)

	// Register generic Genkit LLM factory (supports any provider via config)
	if err := RegisterGenkit(r); err != nil {
		return fmt.Errorf("failed to register genkit LLM factory: %w", err)
	}

	return nil
}
