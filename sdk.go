package manglekit

import (
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/pipeline"
	"github.com/duynguyendang/manglekit/retrieve"
)

// New creates a new MangleKit orchestrator using the default "Sandwich" pipeline.
// This function provides a simple, direct way to construct an orchestrator when
// you are building the components programmatically. It requires, at a minimum, a
// configured Retriever and LLM client in the options.
//
// This function is a convenience wrapper around the underlying pipeline constructor.
// For more advanced use cases, such as loading configurations from YAML files or
// using custom components with complex dependencies, it is recommended to use the
// fluent Builder API (see `NewBuilder`).
//
// opts is a struct containing the configuration and components for the orchestrator.
// A valid `Retriever` (of type `retrieve.Retriever`) and `LLM` (of type `llm.Client`)
// must be provided in this struct, otherwise `core.ErrInvalidOptions` will be returned.
// It returns a `core.Orchestrator` ready to process queries, or an error if
// the configuration is invalid or essential components are missing.
func New(opts core.Options) (core.Orchestrator, error) {
	if opts.Retriever == nil || opts.LLM == nil {
		return nil, core.ErrInvalidOptions
	}
	if opts.TopK == 0 {
		opts.TopK = 8
	}
	if opts.MaxTokens == 0 {
		opts.MaxTokens = 512
	}

	// The concrete types for providers are not known at this point.
	// We rely on the builder to have correctly configured them.
	// Here, we just pass the options to the pipeline constructor.
	// Note: a bit of a hack to avoid circular dependencies.
	// The builder will set the concrete types.
	opts.Retriever = opts.Retriever.(retrieve.Retriever)
	opts.LLM = opts.LLM.(llm.Client)
	return pipeline.NewSandwich(opts)
}
