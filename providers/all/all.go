// Package all imports all standard Manglekit providers, causing their `init()`
// functions to be executed. This allows users to register all standard components
// with a single, simple import.
package all

import (
	// Import each provider package to trigger its init() function, which
	// in turn calls the registration function on the global registry.
	_ "github.com/duynguyendang/manglekit/internal/embedders/google"
	_ "github.com/duynguyendang/manglekit/internal/embedders/openai"
	_ "github.com/duynguyendang/manglekit/internal/providers/bm25"
	_ "github.com/duynguyendang/manglekit/internal/providers/dense"
	_ "github.com/duynguyendang/manglekit/internal/providers/hybrid"
	_ "github.com/duynguyendang/manglekit/internal/providers/llm"
	_ "github.com/duynguyendang/manglekit/internal/providers/mangle"
	_ "github.com/duynguyendang/manglekit/internal/providers/orchestrators"
	_ "github.com/duynguyendang/manglekit/internal/providers/rerank"
	_ "github.com/duynguyendang/manglekit/internal/providers/retrievers"
	_ "github.com/duynguyendang/manglekit/internal/providers/schemaparsers/jsonschema"
	_ "github.com/duynguyendang/manglekit/internal/providers/schemaparsers/rdf"
	_ "github.com/duynguyendang/manglekit/internal/providers/state"
	_ "github.com/duynguyendang/manglekit/internal/vectorstores"
)