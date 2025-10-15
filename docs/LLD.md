# Manglekit SDK — Low-Level Design (LLD)

> Module path: `github.com/duynguyendang/manglekit`  
> Go version: 1.24.1  
> Last Updated: 2025-10-15  
> Status: Sandwich and Declarative orchestrators are production-ready. The fluent builder and provider ecosystem are stable. Current focus areas: consistent context propagation, LLM MaxTokens handling, and hybrid fusion configuration.

---

## 1. Design Tenets
- Public entry points: `sdk.New` and the fluent builder (`builder.go`). They enforce defaults (`TopK=8`, `MaxTokens=512`), assert concrete types for registry‑constructed components, and accumulate LIFO `ResourceClosers` for clean shutdown.
- Provider discovery flows through the global registry (`registry.go`) and the option‑type maps in `typemap.go`, keeping programmatic and YAML/env assembly in sync.
- Sandwich orchestrator (`pipeline/sandwich.go`) executes a fixed, rule‑wrapped flow; records timings in `Answer.Meta` (`retrieve_ms`, `rerank_ms`, `llm_ms`); and stores pre‑rule evidence as `Meta["original_docs"]`.
- Declarative orchestrator (`pipeline/declarative/orchestrator.go`) queries a `core.FlowController` to determine stage order, dispatches tools via a shared execution context, and supports `core.PostRuleEvaluator` for pre‑LLM gating.
- Observability is pluggable. If `core.Observability.Logger` is nil, a lightweight structured `StdLogger` is installed; tracing and metrics hooks are optional and skipped when unset. No direct `fmt.Printf` is used in production paths.

---

## 2. Package Layout (authoritative)
```
github.com/duynguyendang/manglekit
├── builder.go                 # Fluent builder + dependency resolution/runtime clients
├── builder_test.go           
├── config.go                  # YAML/env loaders feeding the builder
├── registry.go                # Global registries for provider constructors
├── sdk.go                     # Convenience wrapper around pipeline.NewSandwich
├── typemap.go                 # Option-type ↔ provider-name lookup tables
├── core/
│   ├── rules.go               # Stage enums, RuleSet/FlowController/PostRule contracts
│   ├── schema.go              # SchemaParser contract
│   └── types.go               # Doc/Query/Answer/Options, observability, errors
├── retrieve/
│   ├── options.go             # Typed options structs for public APIs
│   └── retrieve.go            # Request/Result + Retriever interfaces
├── rerank/
│   ├── options.go             # Reranker option structs
│   └── rerank.go              # Request struct + Reranker interface
├── embed/
│   └── options.go             # Google/OpenAI embedder options
├── llm/
│   ├── llm.go                 # Request/Response + Client interface
│   ├── options.go             # Google/OpenAI options
│   └── prompt.go              # PromptBuilder + default RAG template
├── pipeline/
│   ├── sandwich.go            # Default orchestrator implementation
│   ├── sandwich_test.go       # Stage-level unit tests
│   └── declarative/
│       ├── orchestrator.go    # Flow-driven orchestrator
│       └── orchestrator_test.go
├── internal/
│   ├── embedders/{google,openai}/
│   ├── providers/
│   │   ├── bm25/              # Sparse retriever
│   │   ├── dense/             # Dense retriever
│   │   ├── hybrid/            # Reciprocal rank fusion retriever
│   │   ├── llm/{google,openai}/
│   │   ├── mangle/            # Rules engine + converters
│   │   ├── rerank/cosine/
│   │   ├── retrievers/inmemory/
│   │   ├── schemaparsers/{jsonschema,rdf}/
│   │   └── state/{inmemory,redis}/
│   ├── vectorstores/localvec/
│   └── logger/
│       ├── std_logger.go
│       └── zap_adapter.go
├── providers/all/all.go       # Blank import of bundled providers
├── examples/                  # 01-basic-rag … 10_chatbot runnable demos
├── docs/                      # HLD, LLD, CONTEXT, CSD, LOGGING, code-review
└── state/                     # State provider option types
```

---

## 3. Core Contracts & Observability
- `core.Doc` contains chunk identifiers, provenance, payload text, and free-form metadata. Provider implementations populate common keys (`doc_id`, `source`, sparse scores) for rules to inspect.
- `core.Query.Meta` acts as the coordination channel for filters, expansion terms, `user_context`, and conversation `history`. Mangle Pre rules mutate these keys so retrievers receive policy-compliant constraints.
- `core.Answer.Meta` aggregates timings (`retrieve_ms`, `rerank_ms`, `llm_ms`), best rerank score, `original_docs`, LLM token usage, and rule emissions (`rule_results`, `redactions`, `denial_reason`, etc.).
- `core.Options` stores components as `any` to avoid import cycles; the builder fills them with concrete instances, applies defaults, and appends provider-specific `ResourceClosers`.
- Observability hooks are optional interfaces (`Logger`, `Tracer`, `Meter`). When `Logger` is nil, the builder installs a default structured `StdLogger`; tracing/metrics are no-ops when unset.

---

## 4. Builder & Configuration
- `builder.NewBuilder()` issues an empty builder that captures typed configuration via `With*` methods. Each call records the provider name determined from `typemap.go` and stashes the typed config under `typedConfig`.
- `config.NewBuilderFromYAML` expands environment variables, resolves any `path:"resolve"` struct tags relative to the YAML file, hydrates typed option structs, and chains the corresponding `With*` calls.
- `config.NewBuilderFromEnv` mirrors the YAML loader but sources component names/params from `MKT_*` environment variables and resolves paths relative to `$PWD`.
- `builder.resolveProviderConfig` creates external clients up front (Google Genkit + generative AI, OpenAI/Groq) and pushes cleanup callbacks onto `core.Options.ResourceClosers`.
- `builder.Build()` inspects `config.Orchestrator.Type`:  
  * Sandwich mode builds components in a fixed order (embedder → vector store → retriever → reranker → rules → LLM) before creating `pipeline.NewSandwich`.  
  * Declarative mode requires a `FlowController`, builds every `config.tools` entry while resolving dependencies, and instantiates `declarative.New`.

---

## 5. Provider Wiring
- Every provider package registers itself via the appropriate `manglekit.Register*` call from its `init()`, exposing a constructor with a fully typed signature.
- Option structs exposed to users live in public packages (`retrieve/options.go`, `rerank/options.go`, `embed/options.go`, `llm/options.go`) so programmatic flows remain type-safe.
- The builder infers dependencies: dense retrievers and cosine rerankers automatically request the configured embedder; localvec requires both an embedder and corpus path; declarative tools declare dependencies by referencing other tool names in their params.
- Rules providers (`internal/providers/mangle`) support both code-first (converters define EDB) and file-first (rule files declare EDB) modes, selectable via `core.MangleOptions.FileFirst`.

---

## 6. Retrieval & Vector Storage
- BM25 (`internal/providers/bm25`) indexes Markdown directories, parses YAML front matter into metadata, and attaches Okapi scores to each document’s metadata.
- Dense retrieval (`internal/providers/dense`) embeds the query through the configured Genkit/OpenAI embedder, passes metadata filters to the injected `core.VectorStore`, and searches for semantic matches.
- Hybrid retrieval (`internal/providers/hybrid`) executes sparse and dense lookups concurrently via `errgroup`, fuses rankings with Reciprocal Rank Fusion, and truncates to `TopK`.
- Local vector storage (`internal/vectorstores/localvec`) relies on Genkit’s localvec plugin: it initializes a collection, indexes corpus documents (front matter included), and filters matches using metadata. Resource cleanup is a known gap.
- The in-memory retriever (`internal/providers/retrievers/inmemory`) implements `retrieve.Updatable` for demos and small corpora.

---

## 7. Sandwich Pipeline Mechanics
- Pre rules: `pipeline.Sandwich.Run` calls `RuleSet.Evaluate(core.Pre)` to normalize the query, seed filters/expansions, and optionally deny the request early.
- Retrieval: the configured retriever receives `retrieve.Request{Query, TopK, Meta}`, with filters/expansions forwarded via `Meta`.
- Rerank: if configured, `rerank.Reranker` reorders documents, captures the best score, and materializes citations.
- Fallback threshold: `FallbackThreshold > 0` short-circuits the pipeline with `core.ErrNoEvidence` when the best score is insufficient.
- LLM call: `llm.Client.Complete` receives the query prompt, grounded context, max tokens, and metadata; responses log token usage and latency.
- Post rules: `RuleSet.Evaluate(core.Post)` filters citations, applies redactions, or denies the answer. The orchestrator records rule timings and finalizes `Answer.Meta`.
- Cleanup: `Sandwich.Close` drains `ResourceClosers` in LIFO order so clients like Genkit shut down cleanly.

---

## 8. Declarative Orchestrator Mechanics
- Stage discovery: `getFlowStages` queries `flow_stage/3` and `stage_tool/2` facts from the `FlowController`, builds an ordered stage list, and validates tool assignments.
- Pre evaluation: `flowController.Evaluate(core.Pre)` can deny requests, mutate the query, or flag stages to skip.
- Execution context: a shared map holds the evolving `query`, `docs`, `answer`, `meta`, and denial flags. Each tool updates this map.
- Tool dispatch:  
  * Retrievers populate `docs`, `meta["retrieved_count"]`, and stash `original_docs` in the answer.  
  * Rerankers embed in parallel, emit citations, and set `meta["best_score"]`.  
  * LLM clients build prompts via `llm.PromptBuilder` and capture token usage.  
  * `core.PostRuleEvaluator` instances (e.g., the Mangle engine) can drop/redact evidence, deny the request, and emit audit metadata.
- Skips & denials propagate through the context so later tools respect prior decisions.

---

## 9. Examples & Integration
- `examples/01-basic-rag`: Sandwich pipeline loaded from YAML with local Markdown evidence.
- `examples/02-logic-layer-mode`: Focuses on pre-rule normalization and filter emission without custom retrievers.
- `examples/03-custom-prompt`: Overrides the default prompt template via configuration.
- `examples/04-declarative-flow`: Runs a Datalog-defined flow featuring stage skipping and LLM opt-out.
- `examples/05-chat-with-data`: Full Sandwich pipeline using `.env`, `providers/all`, and vector retrieval.
- `examples/06-schema-validation`: Demonstrates schema parser integration and post-rule enforcement.
- `examples/07-rdf-knowledge-base`: Consumes RDF triples and surfaces canonicalized metadata through rules.
- `examples/08-symbolic-rag`: Emphasizes deterministic post-rule gating.
- `examples/09-genkit-tool`: Shows Genkit tool registration inside orchestrated flows.
- `examples/10_chatbot`: Chatbot sample wiring stateful sessions.

---

## 10. Testing Coverage
- `pipeline/sandwich_test.go` covers the happy path, retriever failures, pre-rule denials, and fallback thresholds.
- `pipeline/declarative/orchestrator_test.go` validates flow resolution, tool dispatch, and denial handling.
- Provider suites exercise BM25 indexing, dense/hybrid retrieval, cosine reranking, embedders, and Mangle rule evaluation.
- Additional targets: localvec lifecycle, tool dependency resolution ordering, and LLM error handling.

---

## 11. Known Gaps & Risks
- LLM `MaxTokens` ignored: current OpenAI/Google clients do not propagate `req.MaxTokens`, so orchestrator defaults cannot constrain completion length.
- Hybrid RRF constant: `internal/providers/hybrid/hybrid.go` uses a hard‑coded `k=60` with no configuration hook.
- Heuristic tool deps: declarative tool dependency detection treats any string param as a dependency, risking false positives for literal strings.
- Duplicated orchestration logic: conversational state handling exists in both orchestrators and could be centralized.
- Context propagation: some providers may not consistently thread `context.Context` through external calls.

---

## 12. Extension Hooks
- Register providers via `manglekit.Register*` and a typed options struct mirrored in `typemap.go`.
- Converters and schema parsers register under `Registry.Component`/`Registry.SchemaParser` and can be referenced in `core.MangleOptions`.
- Declarative flows extend `config.tools`; prefer explicit dependency keys (e.g., `retriever: "hybrid"`, `embedder: "my_embedder"`) over ambiguous strings.

---

## 13. Alignment with HLD
- Principles (SDK-first, dual orchestrators, registry-driven extensibility, fail-fast construction, stateless engines with external state) match HLD §1.2.
- Components map to packages: builder/config/registry manage dependency graphs; providers cover BM25/dense/hybrid retrievers, cosine reranker, Google/OpenAI embedders and LLMs, localvec vector store, Mangle rules and schema parsing; orchestrators ship as Sandwich and Declarative implementations.
- Observability, lifecycle, and usage patterns align with HLD §4 via examples 01–10.
- Non-functionals—concurrency, scaling, enforcement via rules, and observability hooks—align with HLD §5; open items tracked in §11.
