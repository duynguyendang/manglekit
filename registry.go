package manglekit

import (
	"fmt"
)

// Registry is the global, central store for all registered component constructors.
// It is the primary mechanism for extending MangleKit with new providers.
//
// Provider implementations are expected to register their constructor functions
// in an `init()` block within their package. This makes them discoverable by the
// framework's Builder, which can then instantiate them based on configuration.
//
// For example, a new retriever provider would register itself like this:
//
//	func init() {
//	    manglekit.RegisterRetriever("my-retriever", NewMyRetriever)
//	}
//
// Constructors are stored as `any` and are expected to be functions with specific
// signatures (e.g., `func(opts MyOptions) (my.Interface, error)`). The Builder
// is responsible for retrieving a constructor and performing a type assertion to
// the expected function signature before calling it.
var Registry = struct {
	// Retriever holds registered `retrieve.Retriever` constructors.
	Retriever map[string]any
	// Reranker holds registered `rerank.Reranker` constructors.
	Reranker map[string]any
	// Rules holds registered `core.RuleSet` constructors.
	Rules map[string]any
	// LLM holds registered `llm.Client` constructors.
	LLM map[string]any
	// Embedder holds registered `ai.Embedder` constructors.
	Embedder map[string]any
	// SchemaParser holds registered `core.SchemaParser` constructors.
	SchemaParser map[string]any
	// Component holds registered generic component constructors, such as `core.VectorStore`.
	Component map[string]any
}{
	Retriever:    make(map[string]any),
	Reranker:     make(map[string]any),
	Rules:        make(map[string]any),
	LLM:          make(map[string]any),
	Embedder:     make(map[string]any),
	SchemaParser: make(map[string]any),
	Component:    make(map[string]any),
}

// Get retrieves a constructor function from the specified registry map. It is a
// helper used by the Builder to find and instantiate components.
//
// registry is the specific component map to search in (e.g., `Registry.Retriever`).
// name is the string name under which the component was registered.
// It returns the constructor as `any` or an error if the name is not found in the map.
func Get(registry map[string]any, name string) (any, error) {
	c, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown component: %s", name)
	}
	return c, nil
}

// RegisterRetriever adds a retriever constructor to the global registry. This
// is typically called from an `init()` function in a provider package.
func RegisterRetriever(name string, c any) { Registry.Retriever[name] = c }

// RegisterReranker adds a reranker constructor to the global registry. This
// is typically called from an `init()` function in a provider package.
func RegisterReranker(name string, c any) { Registry.Reranker[name] = c }

// RegisterRules adds a rules engine constructor to the global registry. This
// is typically called from an `init()` function in a provider package.
func RegisterRules(name string, c any) { Registry.Rules[name] = c }

// RegisterLLM adds a language model constructor to the global registry. This
// is typically called from an `init()` function in a provider package.
func RegisterLLM(name string, c any) { Registry.LLM[name] = c }

// RegisterEmbedder adds a text embedder constructor to the global registry. This
// is typically called from an `init()` function in a provider package.
func RegisterEmbedder(name string, c any) { Registry.Embedder[name] = c }

// RegisterSchemaParser adds a schema parser constructor to the global registry.
// This is typically called from an `init()` function in a provider package.
func RegisterSchemaParser(name string, c any) { Registry.SchemaParser[name] = c }

// Register adds a generic component constructor (e.g., a vector store) to the
// global registry. This is typically called from an `init()` function in a
// provider package.
func Register(name string, c any) { Registry.Component[name] = c }

// Note: All Try... and Must... functions have been removed.
// The builder is now responsible for type-safe construction.
