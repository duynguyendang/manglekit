package embedders

import (
	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/internal/embedders/google"
	"github.com/duynguyendang/manglekit/internal/embedders/openai"
)

// Register registers all embedder providers with the MangleKit registry.
func Register(r *manglekit.Registry) {
	openai.Register(r)
	google.Register(r)
}
