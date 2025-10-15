package manglekit

import (
	"context"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/ai"
)

// FactoryDeps is a map of dependencies that can be injected into a factory.
type FactoryDeps map[string]any

// RetrieverFactory defines the contract for a retriever constructor.
type RetrieverFactory func(ctx context.Context, opts any, deps FactoryDeps) (retrieve.Retriever, error)

// LLMFactory defines the contract for an LLM client constructor.
type LLMFactory func(ctx context.Context, opts any, deps FactoryDeps) (llm.Client, error)

// EmbedderFactory defines the contract for an embedder constructor.
type EmbedderFactory func(ctx context.Context, opts any, deps FactoryDeps) (ai.Embedder, error)

// RerankerFactory defines the contract for a reranker constructor.
type RerankerFactory func(ctx context.Context, opts any, deps FactoryDeps) (rerank.Reranker, error)

// StateProviderFactory defines the contract for a state provider constructor.
type StateProviderFactory func(ctx context.Context, opts any, deps FactoryDeps) (core.StateProvider, error)

// VectorStoreFactory defines the contract for a vector store constructor.
type VectorStoreFactory func(ctx context.Context, opts any, deps FactoryDeps) (core.VectorStore, error)

// RuleSetFactory defines the contract for a ruleset constructor.
type RuleSetFactory func(ctx context.Context, opts any, deps FactoryDeps) (core.RuleSet, error)

// FactConverterFactory defines the contract for a fact converter constructor.
type FactConverterFactory func(ctx context.Context, opts any, deps FactoryDeps) (core.FactConverter, error)

// SchemaParserFactory defines the contract for a schema parser constructor.
type SchemaParserFactory func(ctx context.Context, opts any, deps FactoryDeps) (core.SchemaParser, error)

// ComponentFactory is a generic function signature for component constructors.
// It accepts a provider-specific options struct (`options`) and a map of
// resolved dependencies (`deps`), returning the initialized component as `any`.
type ComponentFactory func(ctx context.Context, options any, deps FactoryDeps) (any, error)
