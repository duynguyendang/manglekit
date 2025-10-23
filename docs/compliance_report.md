# Manglekit Provider Compliance Report

## 1. Compliance Matrix

### Retriever

| Method | `bm25` | `dense` | `hybrid` | `in-memory` | Evidence |
|---|---|---|---|---|---|
| `Retrieve(context.Context, core.RetrieveRequest) (core.RetrieveResult, error)` | ✔ | ✔ | ✔ | ✔ | `core/interfaces.go:123-129` |
| `Upsert(context.Context, []core.Doc) error` | ✖ | ✖ | ✖ | ✔ | `core/interfaces.go:139-143` |
| `Replace(context.Context, []core.Doc) error` | ✖ | ✖ | ✖ | ✔ | `core/interfaces.go:145-149` |

**Registration:**
- `bm25`: `internal/providers/bm25/bm25.go:41-46`
- `dense`: `internal/providers/dense/dense.go:29-38`
- `hybrid`: `internal/providers/hybrid/hybrid.go:34-49`
- `in-memory`: `internal/providers/retrievers/inmemory/inmemory.go:48-53`

### Reranker

| Method | `cosine` | Evidence |
|---|---|---|
| `Rerank(context.Context, core.RerankRequest) ([]core.ScoredDoc, error)` | ✔ | `core/interfaces.go:84-91` |

**Registration:** `internal/providers/rerank/cosine/cosine.go:34-41`

### LLM

| Method | `google` | `openai` | Evidence |
|---|---|---|---|
| `Complete(context.Context, core.LLMRequest) (core.LLMResponse, error)` | ✔ | ✔ | `core/interfaces.go:44-51` |

**Registration:**
- `google`: `internal/providers/llm/google.go:37-48`
- `openai`: `internal/providers/llm/openai.go:61-71`

### Embedder

| Method | `google` | `openai` | Evidence |
|---|---|---|---|
| `Embed(context.Context, *ai.EmbedRequest) (*ai.EmbedResponse, error)` | ✔ | ✔ | `vendor/github.com/firebase/genkit/go/ai/embedder.go` |
| `Register(api.Registry)` | ✔ | ✔ | `vendor/github.com/firebase/genkit/go/ai/embedder.go` |

**Registration:**
- `google`: `internal/embedders/register.go:18-24`
- `openai`: `internal/embedders/register.go:26-35`
- `groq`: `internal/embedders/register.go:37-46`

### Rules Engine (RuleSet)

| Method | `mangle` | Evidence |
|---|---|---|
| `Evaluate(core.Stage, core.Query, *core.Answer) (core.RuleResult, error)` | ✔ | `core/rules.go:73-80` |

**Registration:** `internal/providers/rules/register.go:11-16`

### Schema Parser

| Method | `jsonschema` | `rdf` | Evidence |
|---|---|---|---|
| `Parse(io.Reader) ([]ast.Atom, error)` | ✔ | ✔ | `core/schema.go:18-24` |
| `Predicates() []ast.PredicateSym` | ✔ | ✔ | `core/schema.go:26-29` |

**Registration:**
- `jsonschema`: `internal/providers/schemaparsers/jsonschema/parser.go:24-29`
- `rdf`: `internal/providers/schemaparsers/rdf/parser.go:25-30`

### Vector Store

| Method | `localvec` | Evidence |
|---|---|---|
| `AddDocuments(context.Context, []core.Doc) error` | ✔ | `core/types.go:50-55` |
| `Search(context.Context, string, []float32, int, map[string]any) ([]core.Doc, error)` | ✔ | `core/types.go:57-64` |

**Registration:** `internal/vectorstores/register.go:15-20`

## 2. Per-Provider Findings

### Retriever: bm25
- **Summary Verdict:** Fail
- **Evidence:** `internal/providers/bm25/bm25.go`
- **Notes:**
  - Registration Key: `bm25`
  - Constructor: `New(BM25Options)`
  - Dependencies: None
  - Config/Lifecycle/Observability/Tests: Configuration is sound. No `Close` method needed. Logging is implemented. Fails to implement `Updatable` interface. No tests.

### Retriever: dense
- **Summary Verdict:** Fail
- **Evidence:** `internal/providers/dense/dense.go`
- **Notes:**
  - Registration Key: `dense`
  - Constructor: `New(ai.Embedder, core.VectorStore)`
  - Dependencies: `Embedder`, `VectorStore` (correctly wired)
  - Config/Lifecycle/Observability/Tests: Configuration correctly declares dependencies. No `Close` method needed. Fails to implement `Updatable` interface. No tests.

### Retriever: hybrid
- **Summary Verdict:** Fail
- **Evidence:** `internal/providers/hybrid/hybrid.go`
- **Notes:**
  - Registration Key: `hybrid`
  - Constructor: `New([]core.Retriever, float64)`
  - Dependencies: `Retriever` (list, correctly wired)
  - Config/Lifecycle/Observability/Tests: Configuration correctly declares dependencies. No `Close` method needed. Uses `errgroup` for safe concurrency. Fails to implement `Updatable` interface. No tests.

### Retriever: in-memory
- **Summary Verdict:** Pass with caveats
- **Evidence:** `internal/providers/retrievers/inmemory/inmemory.go`
- **Notes:**
  - Registration Key: `in-memory`
  - Constructor: `New(InMemoryOptions)`
  - Dependencies: None
  - Config/Lifecycle/Observability/Tests: Configuration is sound. Implements `Updatable`. Thread-safe. No tests.

### Reranker: cosine
- **Summary Verdict:** Pass with caveats
- **Evidence:** `internal/providers/rerank/cosine/cosine.go`
- **Notes:**
  - Registration Key: `cosine`
  - Constructor: `New(CosineOptions, ai.Embedder)`
  - Dependencies: `Embedder` (correctly wired)
  - Config/Lifecycle/Observability/Tests: Configuration is sound. No `Close` method needed. Uses `errgroup` for safe concurrency. No tests.

### LLM: google
- **Summary Verdict:** Pass with caveats
- **Evidence:** `internal/providers/llm/google.go`
- **Notes:**
  - Registration Key: `google`
  - Constructor: `NewGoogle(GoogleOptions, ai.Model, *genkit.Genkit)`
  - Dependencies: `genkit.Genkit` (correctly wired)
  - Config/Lifecycle/Observability/Tests: Configuration is sound. No `Close` method needed. Integration test present, but no unit tests.

### LLM: openai
- **Summary Verdict:** Pass with caveats
- **Evidence:** `internal/providers/llm/openai.go`
- **Notes:**
  - Registration Key: `openai`
  - Constructor: `NewOpenAI(OpenAIOptions, *genkit.Genkit)`
  - Dependencies: `genkit.Genkit` (correctly wired)
  - Config/Lifecycle/Observability/Tests: Configuration is sound, supports custom `base_url`. No `Close` method needed. Integration test present, but no unit tests.

### Embedder: google-embedder
- **Summary Verdict:** Pass with caveats
- **Evidence:** `internal/embedders/google/google.go`
- **Notes:**
  - Registration Key: `google`
  - Constructor: `New(embed.GoogleEmbedderOptions, *genkit.Genkit)`
  - Dependencies: `genkit.Genkit` (correctly wired)
  - Config/Lifecycle/Observability/Tests: Configuration is sound. No `Close` method needed. No tests.

### Embedder: openai-embedder
- **Summary Verdict:** Pass with caveats
- **Evidence:** `internal/embedders/openai/openai.go`
- **Notes:**
  - Registration Key: `openai`, `groq`
  - Constructor: Factory function in `register.go`
  - Dependencies: `genkit.Genkit` (correctly wired)
  - Config/Lifecycle/Observability/Tests: Configuration is sound. No `Close` method needed. No tests.

### Rules Engine: mangle
- **Summary Verdict:** Pass with caveats
- **Evidence:** `internal/providers/mangle/rules.go`
- **Notes:**
  - Registration Key: `mangle`
  - Constructor: `New(context.Context, core.MangleOptions, *manglekit.Registry)`
  - Dependencies: `manglekit.Registry` (for schema parsers)
  - Config/Lifecycle/Observability/Tests: Complex configuration, but appears correct. No `Close` method, but resource-intensive init. No tests.

### Schema Parser: jsonschema
- **Summary Verdict:** Pass with caveats
- **Evidence:** `internal/providers/schemaparsers/jsonschema/parser.go`
- **Notes:**
  - Registration Key: `jsonschema`
  - Constructor: `New(map[string]any)`
  - Dependencies: None
  - Config/Lifecycle/Observability/Tests: Configuration is sound. No `Close` method needed. No tests.

### Schema Parser: rdf
- **Summary Verdict:** Pass with caveats
- **Evidence:** `internal/providers/schemaparsers/rdf/parser.go`
- **Notes:**
  - Registration Key: `rdf`
  - Constructor: `New(map[string]any)`
  - Dependencies: None
  - Config/Lifecycle/Observability/Tests: Configuration is sound. No `Close` method needed. No tests.

### Vector Store: localvec
- **Summary Verdict:** Pass with caveats
- **Evidence:** `internal/vectorstores/localvec/localvec.go`
- **Notes:**
  - Registration Key: `localvec`
  - Constructor: `New(context.Context, Options, ai.Embedder)`
  - Dependencies: `ai.Embedder` (correctly wired)
  - Config/Lifecycle/Observability/Tests: Configuration is sound. `Close` method correctly manages internal `genkit` instance. No tests.

## 3. Gaps to Append to docs/code-review.md

## Smell: Missing/Incorrect Interface Implementation — bm25
**Location:** `internal/providers/bm25/bm25.go`
**Impact Analysis:** The `bm25` retriever does not implement the `core.Updatable` interface, which means it cannot be updated at runtime. This limits its usefulness in dynamic environments where the knowledge base can change.
**Refactoring Suggestion:** Implement the `Upsert` and `Replace` methods on the `BM25` struct to satisfy the `core.Updatable` interface.
**Status:** Open

## Smell: Missing/Incorrect Interface Implementation — dense
**Location:** `internal/providers/dense/dense.go`
**Impact Analysis:** The `dense` retriever does not implement the `core.Updatable` interface, which means it cannot be updated at runtime. This limits its usefulness in dynamic environments where the knowledge base can change.
**Refactoring Suggestion:** Implement the `Upsert` and `Replace` methods on the `Dense` struct to satisfy the `core.Updatable` interface.
**Status:** Open

## Smell: Missing/Incorrect Interface Implementation — hybrid
**Location:** `internal/providers/hybrid/hybrid.go`
**Impact Analysis:** The `hybrid` retriever does not implement the `core.Updatable` interface, which means it cannot be updated at runtime. This limits its usefulness in dynamic environments where the knowledge base can change.
**Refactoring Suggestion:** Implement the `Upsert` and `Replace` methods on the `Retriever` struct to satisfy the `core.Updatable` interface.
**Status:** Open

## Smell: Insufficient Test Coverage — All Providers
**Location:** `internal/providers/`
**Impact Analysis:** The lack of unit tests for providers makes it difficult to verify their correctness, prevent regressions, and understand their behavior without running the entire application. This is especially critical for providers with complex logic or external dependencies.
**Refactoring Suggestion:** Add unit tests for each provider, covering interface compliance, registry registration, and dependency error paths. Use mocks for external dependencies to isolate the provider's logic.
**Status:** Open

## 4. Registration Integrity Check

- **Duplicate Keys:** None found.
- **Missing Registrations:** All providers listed in the scope are registered.
- **Constructor Signature Deviations:** All constructors and factory functions comply with the expected signatures.
- **Naming Drift:** None found.

## 5. Test Coverage Snapshot

| Provider | Interface Tests | Registry Tests | Dependency Negative Tests | Verdict |
|---|---|---|---|---|
| `bm25` | ✖ | ✖ | ✖ | Fail |
| `dense` | ✖ | ✖ | ✖ | Fail |
| `hybrid` | ✖ | ✖ | ✖ | Fail |
| `in-memory` | ✖ | ✖ | ✖ | Fail |
| `cosine` | ✖ | ✖ | ✖ | Fail |
| `google` (LLM) | ✔ (Integration) | ✖ | ✖ | Pass with caveats |
| `openai` (LLM) | ✔ (Integration) | ✖ | ✖ | Pass with caveats |
| `google-embedder` | ✖ | ✖ | ✖ | Fail |
| `openai-embedder` | ✖ | ✖ | ✖ | Fail |
| `mangle` | ✖ | ✖ | ✖ | Fail |
| `jsonschema` | ✖ | ✖ | ✖ | Fail |
| `rdf` | ✖ | ✖ | ✖ | Fail |
| `localvec` | ✖ | ✖ | ✖ | Fail |

## Final Section

**Registration Readiness Summary:**
- `bm25`: FAIL (Does not implement `Updatable`)
- `dense`: FAIL (Does not implement `Updatable`)
- `hybrid`: FAIL (Does not implement `Updatable`)
- `in-memory`: PASS
- `cosine`: PASS
- `google` (LLM): PASS
- `openai` (LLM): PASS
- `google-embedder`: PASS
- `openai-embedder`: PASS
- `mangle`: PASS
- `jsonschema`: PASS
- `rdf`: PASS
- `localvec`: PASS

**Top 3 Cross-Cutting Fixes:**
1. **Add comprehensive unit tests for all providers.** This is the most critical gap and should be the highest priority.
2. **Implement the `Updatable` interface for all retrievers.** This will greatly improve the flexibility and usefulness of the SDK.
3. **Standardize logging across all providers.** While some providers have logging, it's not consistent. A standard logging interface and practice should be adopted.
