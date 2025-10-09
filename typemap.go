package manglekit

import (
	"reflect"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/embed"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
)

// optionsTypeToName is the internal, central mapping from a provider's Go Options
// struct `reflect.Type` to its registered string name. This is a key part of the
// builder's "magic," allowing it to infer the provider name automatically when a
// user calls a `With...` method, which enables a simpler and more type-safe API.
// For example, when `builder.WithRetriever(&retrieve.BM25Options{})` is called,
// the builder looks up the type of `*retrieve.BM25Options` in this map to
// discover the provider name "bm25".
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
	reflect.TypeOf(&retrieve.InMemoryOptions{}): "in-memory",
	reflect.TypeOf(&retrieve.DenseOptions{}):    "dense",
	reflect.TypeOf(&retrieve.HybridOptions{}):   "hybrid",

	// Rules
	reflect.TypeOf(&core.MangleOptions{}): "mangle",

	// Vector Stores
	reflect.TypeOf(&core.LocalvecOptions{}): "localvec",
}

// nameToOptionsType is the inverse of optionsTypeToName, mapping a provider's
// string name back to the `reflect.Type` of its corresponding Options struct.
// This map is essential for the YAML-based configuration, as it allows the
// `NewBuilderFromYAML` function to dynamically create the correct options struct
// instance based on the `name` field in the YAML configuration.
var nameToOptionsType = make(map[string]reflect.Type)

// init populates the nameToOptionsType map by inverting the optionsTypeToName map
// at program startup.
func init() {
	for t, name := range optionsTypeToName {
		// Note: For provider names shared by multiple components (e.g., "google"
		// is used for both LLM and Embedder), this loop will cause the last entry
		// in the map to overwrite previous ones. This is acceptable because the
		// primary use of `nameToOptionsType` is to create a struct of the correct
		// *kind* of options, which is then populated from a generic `map[string]any`.
		// The builder's `configureComponent` function handles this dynamic assignment.
		nameToOptionsType[name] = t
	}

	// Manually register aliases to disambiguate provider families that share the
	// same root name between different component types (e.g., embedder vs LLM).
	nameToOptionsType["google"] = reflect.TypeOf(&llm.GoogleOptions{})
	nameToOptionsType["openai"] = reflect.TypeOf(&llm.OpenAIOptions{})
	nameToOptionsType["groq"] = reflect.TypeOf(&llm.OpenAIOptions{})
	nameToOptionsType["google-embedder"] = reflect.TypeOf(&embed.GoogleEmbedderOptions{})
	nameToOptionsType["openai-embedder"] = reflect.TypeOf(&embed.OpenAIEmbedderOptions{})
}
