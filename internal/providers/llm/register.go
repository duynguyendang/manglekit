package llm

import (
	"github.com/duynguyendang/manglekit"
)

// Register registers all LLM providers and the LLM kind handler.
func Register(r *manglekit.Registry) {
	RegisterGoogle(r)
	RegisterOpenAI(r)
	r.RegisterHandler(&Handler{})
}
