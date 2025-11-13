package core

import "reflect"

// Kind is a typed string that represents the category of a Manglekit component.
// Using a typed string prevents accidental misspellings and provides a clear,
// enumerable set of component types.
type Kind string

const (
	KindRetriever     Kind = "retriever"
	KindReranker      Kind = "reranker"
	KindRules         Kind = "rules"
	KindLLM           Kind = "llm"
	KindEmbedder      Kind = "embedder"
	KindStateProvider Kind = "state_provider"
	KindOrchestrator  Kind = "orchestrator"
	KindSchemaParser  Kind = "schema_parser"
	KindTool          Kind = "tool"
	KindReasoner      Kind = "reasoner"
	KindPlanner       Kind = "planner"
)

// ProviderOptions is the core interface for the new type-safe registration system.
// Any struct that serves as a configuration for a provider must implement this
// interface. It allows the provider to self-identify its unique name and its kind,
// removing the need for string literals during registration.
type ProviderOptions interface {
	// ProviderName returns the unique, machine-readable name of the provider,
	// e.g., "openai-chat", "chroma-db", "bm25".
	ProviderName() string
	// ProviderKind returns the category of the provider, e.g., KindLLM, KindRetriever.
	ProviderKind() Kind
}

// NilOptions is a utility type for components, such as orchestrators, that do not
// have their own specific configuration options but still need to be registered
// with the system. It implements the ProviderOptions interface with fixed values.
type NilOptions struct {
	Name string
	Kind Kind
}

func (o *NilOptions) ProviderName() string { return o.Name }
func (o *NilOptions) ProviderKind() Kind   { return o.Kind }

// NameOf is a helper function that returns the reflect.Type name for a given value.
func NameOf(v any) string {
	if t := reflect.TypeOf(v); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
	}
}
