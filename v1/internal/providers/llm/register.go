package llm

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/providers/llm/genkit"
)

// Register registers all LLM providers with the MangleKit registry.
// LLMs are registered as thin factories that delegate to Genkit plugins:
// - "google": Google Generative AI models via genkit googlegenai plugin
// - "openai": OpenAI models via genkit compat_oai plugin
// - "genkit-llm": Generic Genkit model lookup for dynamic or custom models
//
// To add support for a new LLM provider (e.g., Anthropic, Cohere):
// 1. Create a new file: anthropic.go
// 2. Define AnthropicOptions struct
// 3. Implement RegisterAnthropic function as a thin factory
// 4. Call RegisterAnthropic(r) here
// 5. Keep the pattern: only configure + delegate to Genkit
func Register(r *manglekit.Registry) error {
	// Register native LLM thin factories
	RegisterOpenAI(r)
	RegisterGoogle(r)

	// Register generic Genkit LLM provider for dynamic model lookup
	if err := genkit.Register(r); err != nil {
		return err
	}

	return nil
}
