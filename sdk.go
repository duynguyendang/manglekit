package manglekit

import (
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/pipeline"
	"github.com/duynguyendang/manglekit/retrieve"
)

// New creates a new MangleKit orchestrator using the default "Sandwich" pipeline.
// This is the simplest way to get started if you are not using the Builder.
// It requires, at a minimum, a configured Retriever and LLM client in the options.
//
// This function is a convenience wrapper around pipeline.NewSandwich. For more
// complex setups, such as configuring providers from a YAML file or using
// custom components, it is recommended to use the Builder.
//
// opts is a struct containing the configuration and components for the orchestrator.
// A valid `Retriever` and `LLM` must be provided.
// It returns a core.Orchestrator ready to process queries, or an error if
// the configuration is invalid.
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