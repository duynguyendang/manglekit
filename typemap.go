package manglekit

import (
	"reflect"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/embed"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
)

// optionsTypeToName is the internal, central mapping of an Options struct's reflect.Type
// to its registered provider name. The builder uses this map to infer the
// provider name, enabling a simpler and more type-safe API.
var optionsTypeToName = map[reflect.Type]string{
	// Embedders
	reflect.TypeOf(&embed.GoogleEmbedderOptions{}): "google",
	reflect.TypeOf(&embed.OpenAIEmbedderOptions{}): "openai",

	// LLMs
	reflect.TypeOf(&llm.GoogleOptions{}): "google",
	reflect.TypeOf(&llm.OpenAIOptions{}): "openai",

	// Rerankers
	reflect.TypeOf(&rerank.CosineOptions{}):  "cosine",
	reflect.TypeOf(&rerank.ColbertOptions{}): "colbert",

	// Retrievers
	reflect.TypeOf(&retrieve.BM25Options{}):     "bm25",
	reflect.TypeOf(&retrieve.InMemoryOptions{}): "inmemory",
	reflect.TypeOf(&retrieve.DenseOptions{}):    "dense",
	reflect.TypeOf(&retrieve.HybridOptions{}):   "hybrid",

	// Rules
	reflect.TypeOf(&core.MangleOptions{}): "mangle",

	// Vector Stores
	reflect.TypeOf(&core.LocalvecOptions{}): "localvec",
}

// nameToOptionsType is the inverse of optionsTypeToName, mapping a string name
// to the reflect.Type of the corresponding Options struct. It's used for
// dynamically constructing components from a string-based configuration (e.g., YAML).
var nameToOptionsType = make(map[string]reflect.Type)

// init populates the nameToOptionsType map by inverting the optionsTypeToName map.
func init() {
	for t, name := range optionsTypeToName {
		// For names shared by multiple components (e.g., "google" for LLM and Embedder),
		// this will overwrite. This is acceptable because the builder's With... methods
		// provide the necessary context. When looking up by name, we primarily need
		// a representative type to instantiate, and the specific fields will be
		// populated from the config map.
		nameToOptionsType[name] = t
	}
}