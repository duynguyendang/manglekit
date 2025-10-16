package llm

import "github.com/duynguyendang/manglekit"

// Register registers all LLM providers.
func Register(r *manglekit.Registry) {
	RegisterGoogle(r)
	RegisterOpenAI(r)
}
