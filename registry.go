package manglekit

import (
	"fmt"
)

// Registry is the global, central store for all registered component constructors.
// Provider implementations use `init()` functions to register their constructors
// here, making them available to the Builder.
//
// Constructors are stored as `any` and are expected to be functions with specific
// signatures (e.g., `func(opts MyOptions) (my.Interface, error)`). The Builder
// is responsible for retrieving a constructor and performing a type assertion to
// the expected function signature before calling it. This approach enables a
// type-safe dependency injection model.
var Registry = struct {
	// Retriever holds registered retriever constructors.
	Retriever map[string]any
	// Reranker holds registered reranker constructors.
	Reranker map[string]any
	// Rules holds registered rules engine constructors.
	Rules map[string]any
	// LLM holds registered language model constructors.
	LLM map[string]any
	// Embedder holds registered text embedder constructors.
	Embedder map[string]any
	// SchemaParser holds registered schema parser constructors.
	SchemaParser map[string]any
	// Component holds registered generic component constructors, such as vector stores.
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

// Get retrieves a constructor function from the specified registry map. It provides
// a consistent error message if the component is not found.
//
// registry is the specific component map to search in (e.g., Registry.Retriever).
// name is the name of the component to retrieve.
// It returns the constructor as `any` or an error if not found.
func Get(registry map[string]any, name string) (any, error) {
	c, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown component: %s", name)
	}
	return c, nil
}

// RegisterRetriever adds a retriever constructor to the registry.
func RegisterRetriever(name string, c any) { Registry.Retriever[name] = c }

// RegisterReranker adds a reranker constructor to the registry.
func RegisterReranker(name string, c any) { Registry.Reranker[name] = c }

// RegisterRules adds a rules engine constructor to the registry.
func RegisterRules(name string, c any) { Registry.Rules[name] = c }

// RegisterLLM adds a language model constructor to the registry.
func RegisterLLM(name string, c any) { Registry.LLM[name] = c }

// RegisterEmbedder adds a text embedder constructor to the registry.
func RegisterEmbedder(name string, c any) { Registry.Embedder[name] = c }

// RegisterSchemaParser adds a schema parser constructor to the registry.
func RegisterSchemaParser(name string, c any) { Registry.SchemaParser[name] = c }

// Register adds a generic component constructor (e.g., a vector store) to the registry.
func Register(name string, c any) { Registry.Component[name] = c }

// Note: All Try... and Must... functions have been removed.
// The builder is now responsible for type-safe construction.