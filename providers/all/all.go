// Package all provides a convenient way to register all standard Manglekit providers.
// By importing this package, all providers will be automatically available to the Builder.
package all

import (
	// LLM Providers
	_ "github.com/duynguyendang/manglekit/internal/providers/llm"

	// Embedder Providers
	_ "github.com/duynguyendang/manglekit/internal/embedders/google"
	_ "github.com/duynguyendang/manglekit/internal/embedders/openai"

	// Retriever Providers
	_ "github.com/duynguyendang/manglekit/internal/providers/bm25"
	_ "github.com/duynguyendang/manglekit/internal/providers/dense"
	_ "github.com/duynguyendang/manglekit/internal/providers/hybrid"
	_ "github.com/duynguyendang/manglekit/internal/providers/retrievers/inmemory"

	// Reranker Providers
	_ "github.com/duynguyendang/manglekit/internal/providers/rerank/cosine"

	// Rules Providers
	_ "github.com/duynguyendang/manglekit/internal/providers/mangle"

	// Schema Parser Providers
	_ "github.com/duynguyendang/manglekit/internal/providers/schemaparsers/jsonschema"
)
