package llm

import "github.com/duynguyendang/manglekit"

// Register registers all LLM providers with the MangleKit registry.
func Register(r *manglekit.Registry) {
	RegisterOpenAI(r)
	RegisterGoogle(r)
}
