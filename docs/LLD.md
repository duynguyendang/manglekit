# Manglekit SDK — Low-Level Design (LLD)

> Module path: `github.com/duynguyendang/manglekit`  
> Go version: 1.21  
> Last Updated: 2025-05-29  
> Status: Sandwich orchestrator, provider registry, Genkit-based vector store/LLM adapters, and Mangle rule integration are implemented and unit-tested. Declarative flow execution is available when a FlowController-compatible rules engine is configured. Embedder registry hooks (`Register`) and HTTP hosting layers remain TODO.

---

## 1. Design Tenets
- Single orchestrator entry point via `sdk.New` or the fluent `Builder`; callers only import `manglekit`.
- Providers self-register in `init()` through `registry.go`, exposing typed constructors; builder performs type assertions before invocation.
- Sandwich pipeline enforces `Mangle → Retrieval → Rerank → LLM → Mangle` ordering with observability hooks and meta breadcrumbs.
- Declarative orchestration separates workflow definition (facts) from concrete tool wiring, allowing rules to gate/skip stages at runtime.
- All components are constructed through typed options (see `typemap.go`) to avoid stringly-typed APIs while retaining configuration flexibility.

---

## 2. Package Layout (authoritative)
```
github.com/duynguyendang/manglekit
├── sdk.go                     # Thin wrapper around pipeline.NewSandwich; default TopK/MaxTokens.
├── builder.go                 # Fluent Builder implementation and dependency resolution.
├── config.go                  # YAML loader, env expansion, path resolution, Builder bootstrap.
├── registry.go                # Component registries + helpers (no Must*/Try*).
├── typemap.go                 # Options-type ↔ provider-name lookups used by the Builder.
│
├── core/
│   ├── types.go               # Doc/Query/Answer/Citation, Options, Observability, errors.
│   ├── rules.go               # RuleSet, FlowController, FactConverter, SchemaSource.
│   └── schema.go              # SchemaParser interface for rule-time schema ingestion.
│
├── retrieve/                  # Public retrieval contracts and typed options.
├── rerank/                    # Reranker interfaces and options.
├── embed/                     # Embedder option structs (Google/OpenAI).
├── llm/                       # LLM interfaces, options, and PromptBuilder helper.
├── pipeline/
│   ├── sandwich.go            # Default orchestrator implementation.
│   ├── sandwich_test.go       # Unit tests for pipeline orchestration.
│   └── declarative/           # Mangle-driven declarative orchestrator.
│
├── internal/                  # Provider implementations hidden from consumers.
│   ├── embedders/{google,openai}/
│   ├── providers/
│   │   ├── bm25/              # Sparse retriever over markdown corpora.
│   │   ├── dense/             # Embedder + vector store retriever.
│   │   ├── hybrid/            # Reciprocal Rank Fusion retriever.
│   │   ├── llm/{google,openai}/
│   │   ├── mangle/            # FlowController + default converters.
│   │   ├── rerank/cosine/     # Cosine similarity reranker.
│   │   ├── retrievers/inmemory/
│   │   └── schemaparsers/{jsonschema,rdf}/
│   ├── vectorstores/localvec/ # Genkit LocalVec-backed VectorStore.
│   └── logger/logger.go       # zap logger helper (not wired yet).
│
├── providers/all/all.go       # One-stop import to register all internal providers.
├── examples/05-chat-with-data/ # Sandwich example wired via YAML config.
└── docs/…                     # Design docs (this file, etc.).
```

---

## 3. Core Contracts & Observability
- `core.Doc` encapsulates chunk metadata. `Meta` is free-form and reused downstream (e.g., Mangle converters). `core.LocalvecOptions` remains for backwards compatibility; builder prefers explicit vector store wiring.
- `core.Query` carries user text plus `Meta` (filters, expansion terms, user context, dynamic facts). Mutations from rules are written back into this struct.
- `core.Answer` stores final text, citations, and diagnostics. `Citation.Score` uses struct tag `json_:"score,omitempty"` (typo) which prevents JSON emission—tracked as a known bug.
- `core.Options` holds `any` for Retriever/Reranker/LLM to avoid import cycles; `sdk.New` and builder enforce concrete types. Defaults: `TopK=8`, `MaxTokens=512`.
- `core.Observability` provides optional Logger/Tracer/Meter hooks. When absent, Sandwich falls back to `fmt.Printf` logging.
- `core.RuleSet` and `core.FlowController` define the rules contract; `RuleResult` supports `Allowed`, `Reason`, `Mutate`, and `SkippedStages` for declarative flows.
- `core.SchemaParser` enables custom schema-to-fact ingestion referenced by rules via `core.SchemaSource`.

---

## 4. Pipelines
### Sandwich Orchestrator (`pipeline/sandwich.go`)
1. Logs span start (`Logger`, stdout, and optional tracer span `manglekit.Run`).
2. Pre-stage: invokes `RuleSet.Evaluate(core.Pre, Query, nil)`. `Mutate` callbacks can enrich filters/expansion terms or deny requests. Denials return `core.ErrDenied` with reason. Metrics recorded via `Meter.Record("manglekit.rules_pre_ms", …)`.
3. Retrieval: issues `retrieve.Request{Query, TopK, Meta}`. Stores latency under `answer.Meta["retrieve_ms"]` and persist `Meta["original_docs"]` for post-rules. Errors are wrapped (`"retrieve failed: %w"`).
4. Rerank (optional): adapts `rerank.ScoredDoc` into citations, tracking `bestScore` and emitting `manglekit.rerank_ms`. Without a reranker, `bestScore` stays `0`.
5. Fallback threshold: if `FallbackThreshold` > 0 and `bestScore` below, returns `core.ErrNoEvidence`. With no reranker, this check always sees `bestScore=0`.
6. LLM: builds passage slice and calls `llm.Client.Complete`. Tracks `answer.Meta["llm_ms"]` and `answer.Meta["token_usage"]`.
7. Post-stage: runs `RuleSet.Evaluate(core.Post, Query, &Answer)`; can mutate answer or deny with `core.ErrDenied`. Uses pre-captured `original_docs` to inspect dropped citations.
8. Success logging and return.

### Declarative Orchestrator (`pipeline/declarative/orchestrator.go`)
- Loads flow plan from Mangle facts: `flow_stage/3` (order) and `stage_tool/2`. Stages are sorted numerically.
- Pre-rules evaluate once; `Mutate` results are re-applied to `Query`/`Answer` inside the shared execution context map. `RuleResult.SkippedStages` allows selective stage skipping.
- Tools map holds concrete instances keyed by name (retrievers, rerankers, LLMs, vector stores, etc.). Dispatch uses type assertions:
  * `retrieve.Retriever`: sets `docs` and records `original_docs`.
  * `rerank.Reranker`: rebuilds docs + citations.
  * `llm.Client`: produces answer text and token usage.
- Any missing tool or unsupported type aborts execution with contextual error. Final answer is read from the context map.

---

## 5. Builder & Configuration
- `config.Config` models orchestrator selection, component wiring (`componentCfg{Name, Params}`), provider-level settings, and pipeline defaults (`TopK`, `MaxTokens`, `FallbackThreshold`).
- `NewBuilderFromYAML` expands env vars, marshals `map[string]any` into typed option structs via temporary JSON, and resolves relative paths with `path:"resolve"` tags (uses `resolvePathsInStruct`). The resultant builder chains `With*` calls.
- `typemap.go` maps option pointer types to provider names (`*retrieve.BM25Options` → `"bm25"`). Inverse map fuels dynamic config loading.
- Builder flow:
  1. Chain `With*` calls store provider name + typed config in `*_Params`.
  2. `resolveDependencies` infers missing embedder choice from other components and populates `providerNames` for API key lookup.
  3. `resolveProviderConfig` instantiates shared clients (`genai.Client`, `genkit.Genkit`, `openai.Client`) using config or environment (`GOOGLE_API_KEY`, `OPENAI_API_KEY`, `GROQ_API_KEY`).
  4. `buildComponents` executes in dependency order (embedder → vector store → retriever → reranker → rules → LLM).
  5. Declarative flows additionally call `buildTools`, iterating until all dependencies are satisfied. Tool parameters can reference other tools by name (string heuristic).
  6. `Build` selects orchestrator: `"sandwich"` calls `New(b.opts)`; `"declarative"` requires `core.FlowController`, builds tools, and invokes `declarative.New`.
- `providers/all/all.go` blank-imports every internal provider to populate the registries.

---

## 6. Providers & Components
### Retrieval
- **BM25 (`internal/providers/bm25/bm25.go`)**: Walks markdown directory, parses YAML front matter, and indexes content using `go-nlp/tfidf` + `go-nlp/bm25`. Adds `doc_id`, `source`, and retrieval `score` into `core.Doc.Meta`. `TopK` defaults to 10.
- **Dense (`internal/providers/dense/dense.go`)**: Embeds query via `ai.Embedder`, searches a `core.VectorStore`, and forwards filters from `req.Meta["filters"]`. Requires embedder + vector store; errors bubble up with context.
- **Hybrid (`internal/providers/hybrid/hybrid.go`)**: Runs BM25 and dense retrievers concurrently (`errgroup`), fuses results via Reciprocal Rank Fusion (`k=60`), and truncates to request `TopK`. Dense leg optional.
- **In-memory (`internal/providers/retrievers/inmemory/inmemory.go`)**: Thread-safe map store implementing `retrieve.Updatable`. `Retrieve` returns `TopK` slice; `Upsert`/`Replace` mutate internal map with ID checks.

### Vector Store
- **LocalVec (`internal/vectorstores/localvec/localvec.go`)**: Wraps Genkit `localvec` retriever. Requires embedder and document path; indexes markdown on startup. `Search` demands `ctx.Value("query_text")` and supports metadata filtering. `AddDocuments` re-indexes new docs.

### Reranker
- **Cosine (`internal/providers/rerank/cosine/cosine.go`)**: Embeds query + docs (parallel goroutines) using shared embedder, computes cosine similarity, sorts desc, respects request `TopK`. Custom `sqrt` avoids math dependency. Uses `context.Background()` for embedding calls.

### LLM Clients
- **OpenAI/Groq (`internal/providers/llm/openai.go`)**: Takes `llm.OpenAIOptions`, builds prompts via `llm.PromptBuilder`, and invokes chat completions with `openai-go` on `context.Background()`. Captures token usage (`prompt`, `completion`) and returns first choice. Supports Groq via alternate client/base URL.
- **Google (`internal/providers/llm/google.go`)**: Wraps Genkit `googlegenai` model. Builds prompts with `PromptBuilder`, calls `model.Generate`, concatenates message text, and records usage when available.

### Embedders
- **Google (`internal/embedders/google/google.go`)**: Uses `genai.Client.EmbeddingModel`. Determines vector dimension from model (default `embedding-001`). `Embed` batches documents; `Register` method currently panics (`TODO`).
- **OpenAI/Groq (`internal/embedders/openai/openai.go`)**: Calls OpenAI Embeddings API, converts float64 → float32, optional dimension override. Also registers under `"groq"`. `Register` is unimplemented (panics).

### Rules & Converters
- **Mangle RuleSet (`internal/providers/mangle/rules.go`)**: Loads rules/facts via globbing, optionally merges schema facts (`SchemaSource`), and initializes converters. Supports `DefaultConverters` (query/user/doc). Honors `FileFirst` flag to defer EDB declarations to `.dlog`. Evaluates stratified program once to seed base store; each `Evaluate` copies base store, injects runtime facts, and re-evaluates strata.
  * **Pre stage**: emits `expansion_terms`, `filters`, `skip_stage`, `deny`. `Mutate` writes filters/expansions into query meta and attaches deny reasons to answer meta.
  * **Post stage**: replays original docs kept in `answer.Meta["original_docs"]`, filters citations via deny facts, and records deny reasons in answer meta.
  * **FlowController.Query**: Streams facts matching a query atom for declarative orchestrator planning.
- **Converters (`internal/providers/mangle/converters/`)**:
  * `QueryConverter`: normalizes text, extracts versions (`vX.Y`), tokenizes unique words, emits `raw_query`/`normalized_query`.
  * `UserContextConverter`: converts `query.Meta["user_context"]` to `user_attribute/2` facts, with IRI-aware handling.
  * `DocumentConverter`: maps `core.Doc` to `doc_id`, `doc_content`, and `doc_metadata` facts (strings and string slices).
- **Schema Parsers**:
  * `jsonschema`: Emits `schema/1`, `field/3`, `field_required/2`, `field_format/3`, `field_constraint/4`.
  * `rdf`: Emits `triple/3`, converting IRIs to `ast.Name`.

### Utilities
- `internal/logger/logger.go`: Returns a production `zap.Logger`; not yet used by the builder or pipelines.

---

## 7. Prompt Management
- `llm/prompt.go` defines `DefaultRAGTemplate` instructing citation-bounded answers.
- `llm.PromptBuilder` caches parsed templates behind an RWMutex, injecting helper functions (`toJSON`, `join`, `truncate`). `Build` falls back to default template when user template blank.

---

## 8. Examples & Integration
- `examples/05-chat-with-data/main.go`: Loads `.env`, builds orchestrator from `config.yaml`, imports `providers/all` for registration, and runs Sandwich pipeline against markdown corpus + Mangle rules (`rules/kb.facts`, `rules/retrieval.dlog`).
- Example config wires `mangle` rules (file-first), `bm25` retriever, and Google LLM. API keys supplied via environment (`GOOGLE_API_KEY`).
- No production HTTP binary yet; `cmd` directory is absent in this snapshot.

---

## 9. Testing Coverage
- `pipeline/sandwich_test.go`: Validates happy path, retrieval failure, pre-rule denial, and fallback threshold behavior.
- Provider tests: `internal/providers/{bm25,dense,hybrid,mangle,rerank/cosine}` cover indexing, fusion, converter pipelines, and similarity ordering.
- Additional integration coverage can be added by exercising `localvec` and LLM providers with mocks; currently no automated tests for those packages.

---

## 10. Known Gaps & Risks
- `core.Citation.Score` JSON tag typo prevents serialization; downstream clients cannot inspect scores.
- Embedder `Register` methods (`internal/embedders/*`) panic when invoked; Genkit plugin registration is incomplete.
- Provider clients use `context.Background()` with no timeout/cancellation; long-running calls cannot be cancelled by callers.
- Sandwich fallback relies on reranker-provided `bestScore`; without a reranker the fallback threshold path always returns `ErrNoEvidence`.
- `localvec.New` re-initializes Genkit and LocalVec each build; resource reuse and shutdown hooks are absent. `Search` hard-requires `ctx` to carry `"query_text"`.
- Declarative tool dependency detection treats any string param as a dependency, risking false positives/negatives.
- `resolveProviderConfig` opens `genai.Client` and `genkit.Genkit` instances without explicit Close/Shutdown.
- Logging defaults to stdout prints when `Observability.Logger` nil; consider wiring `internal/logger` helper.
- No HTTP server or ingestion pipeline yet; examples rely on CLI context only.

---

## 11. Extension Hooks
- New providers should register constructors via `manglekit.Register*` and supply typed options mapped in `typemap.go`.
- Additional fact converters or schema parsers can be registered as `Registry.Component` or `Registry.SchemaParser` entries for discovery by the Mangle provider.
- Declarative flows can add tools by extending `config.tools` and ensuring builder support via `buildSingleTool`.
