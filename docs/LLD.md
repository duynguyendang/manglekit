# Manglekit SDK — Low-Level Design (LLD)

> Module path: `github.com/duynguyendang/manglekit`  
> Go version: 1.24.1  
> Last Updated: 2025-10-17
> Status: **Architectural refactor (type-safe registry + builder).** The registry is generic and option-driven; the builder assembles a typed `core.Resolved` for orchestrators. Sandwich is production-ready; Declarative registration is stubbed.

---

## 1. Design Tenets
- Public entry points: `sdk.New` and the fluent builder (`builder.go`). They enforce defaults (`TopK=8`, `MaxTokens=512`), assert concrete types for registry‑constructed components, and accumulate LIFO `ResourceClosers` for clean shutdown.
- **Dependency Injection first, with a convenience singleton.** Prefer passing an instance `Registry` into the Builder. A convenience `sdk.GlobalRegistry()` exists for simple apps, but isolated registries are recommended for tests and modularity.
- Sandwich orchestrator (`pipeline/sandwich.go`) executes a fixed, rule‑wrapped flow; records timings in `Answer.Meta` (`retrieve_ms`, `rerank_ms`, `llm_ms`); and stores pre‑rule evidence as `Meta["original_docs"]`.
- Declarative orchestrator (`pipeline/declarative/orchestrator.go`) queries a `core.FlowController` to determine stage order, dispatches tools via a shared execution context, and supports `core.PostRuleEvaluator` for pre‑LLM gating.
- Observability is pluggable. If `core.Observability.Logger` is nil, a lightweight structured `StdLogger` is installed; tracing and metrics hooks are optional and skipped when unset. No direct `fmt.Printf` is used in production paths.

---

## 2. Package Layout (authoritative)
```
github.com/duynguyendang/manglekit
├── builder.go                 # Fluent builder + spec-driven dependency resolution
├── from_config.go             # Translate validated config -> fluent builder
├── factories.go               # (doc stub) factory type defs moved to registry.go
├── registry.go                # Instance registry; typed factories + options/type maps + orchestrators
├── sdk/                       # Convenience accessors (e.g., global Registry)
│   └── sdk.go                 # sdk.GlobalRegistry()
├── typemap.go                 # (doc stub) mapping logic owned by Registry
├── config/                    # YAML/env loaders and validation
│   ├── loader.go
│   ├── types.go
│   └── validate.go
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
│   └── declarative/
│       ├── orchestrator.go    # Flow-driven orchestrator
│       └── (tests TBD)
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
├── providers/                 # Provider registration helpers and sets
│   └── all/all.go             # RegisterAll: applies standard provider set
├── examples/                  # 01-basic-rag … 10_chatbot runnable demos
├── docs/                      # HLD, LLD, CONTEXT, CSD, LOGGING, code-review
└── state/                     # State provider option types
```

---

## 3. Core Contracts & Observability
- `core.Doc` contains chunk identifiers, provenance, payload text, and free-form metadata. Providers populate common keys for rules to inspect.
- `core.Query.Meta` carries filters, expansion terms, `history`, and user context. Pre rules may mutate these.
- `core.Answer.Meta` aggregates timings (`manglekit.retrieve_ms`, `manglekit.rerank_ms`, `manglekit.llm_ms`, `manglekit.rules_post_ms`), best rerank score, `original_docs`, token usage, and rule emissions.
- `core.Resolved` is the typed container of built components passed to orchestrators.
- `core.OptionsLike` holds global settings (TopK, MaxTokens, FallbackThreshold, Observability, ResourceClosers) during build.
- Observability (`Logger`, `Tracer`, `Meter`) is optional. A default structured `StdLogger` is installed when nil.

---

## 4. Builder API
- `NewBuilder(reg *Registry)`: initializes defaults and binds a registry.
- `With(opts any)`: add a component using its typed options; the registry maps option type → provider kind/name.
- `WithKind(kind core.Kind, name string, opts any)`: add a component explicitly (used by config loaders).
- Orchestrator: `WithOrchestrator("sandwich"|...)` selects from registered orchestrators; defaults to `sandwich`.
- Global config: `WithTopK`, `WithMaxTokens`, `WithFallbackThreshold`, `WithObservability`, `WithGenkit`.
- Build is spec-driven: embedder → vector store → retriever → reranker → rules → llm → state provider; each receives typed deps (`diapi`).
- Sub-builds: `BuildRetriever` exists to support patterns like hybrid retrievers creating sub-retrievers.

---

## 5. Registry & Provider Wiring
- Generic registration: `manglekit.Register[T,D,O](reg, optsSample O, fn)` where `O` implements `core.ProviderOptions` with `ProviderName()` and `ProviderKind()`.
- Internally stored as `GenericFactory` with `Build(ctx, deps any, cfg any) (any, error)`; type safety is preserved via the typed wrapper.
- Option type mapping: the registry records option type → provider name and kind; the builder uses this for `With(opts)`.
- Kinds enumerate component families: `retriever`, `vector_store`, `reranker`, `rules`, `llm`, `embedder`, `state_provider`, `orchestrator`, `schema_parser`.
- Orchestrators without options use `core.NilOptions{Name, Kind: core.KindOrchestrator}` when registering.

---

## 6. Retrieval & Vector Storage
- BM25 (`internal/providers/bm25`): indexes Markdown directories, parses YAML front matter into metadata, attaches Okapi scores in metadata.
- Dense (`internal/providers/dense`): embeds queries via configured embedder, queries injected `core.VectorStore`, supports metadata filters.
- Hybrid (`internal/providers/hybrid`): runs sparse and dense in parallel, fuses with RRF, truncates to `TopK`.
- Localvec (`internal/vectorstores/localvec`): integrates Genkit localvec; indexes corpus with metadata; known gap on lifecycle cleanup.
- In-memory retriever (`internal/providers/retrievers/inmemory`): implements `retrieve.Updatable` for demos/small corpora.

---

## 7. Sandwich Orchestrator Mechanics
- Factory: `pipeline.NewSandwich(ctx, core.Resolved)` receives fully typed deps; no runtime type assertions.
- Context: builds `pipeline.PipelineContext` with `Query`, session `History`, timers, and `Answer.Meta`.
- Stages: `PreRulesStage` → `RetrieveStage` → `RerankStage` → `LLMStage` → `PostRulesStage`, executed by `pipeline.Runner`.
- Retrieval: `Retriever.Retrieve` populates `OriginalDocs`; metrics (`manglekit.retrieve_ms`) recorded; evidence saved to `Answer.Meta["original_docs"]`.
- Rerank: optional `Reranker` emits `Citations` with scores; `best_score` tracked for fallback; metrics `manglekit.rerank_ms` recorded.
- LLM: `llm.Client.Complete` consumes query and passages; `token_usage` captured; `MaxTokens` forwarded; metrics `manglekit.llm_ms` recorded.
- Post rules: `RuleSet.Evaluate(core.Post)` may redact citations or deny; records `manglekit.rules_post_ms`.
- Cleanup: `Close()` drains `ResourceClosers` LIFO; conversation state persisted via `StateProvider` when configured.

---

## 8. Declarative Orchestrator Mechanics
- Registration: a stub is registered under `internal/providers/orchestrators` using `core.NilOptions{Name: "declarative", Kind: core.KindOrchestrator}`.
- Stage discovery: queries `flow_stage/3` and `stage_tool/2` from `core.FlowController` to order stages and bind tools.
- Pre evaluation: `Evaluate(core.Pre)` can deny, mutate `Query`, or flag stages to skip.
- Execution context: shared map carries `query`, `docs`, `answer`, `meta`, and denial flags across tools.
- Tool dispatch:
  - Retrievers add `docs` and stash `original_docs` in `answer.meta`.
  - Rerankers produce ordered docs and citation scores; update `meta["best_score"]`.
  - LLM builds context passages and captures `token_usage`.
  - Post rules: if tool implements `core.PostRuleEvaluator`, applies filters/denials; sets `rules_post_ms`, propagates `denied` and `denial_reason`.
- Skips and denials propagate so later tools respect prior decisions; state managed via `StateProvider`.

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
- Repository currently contains few or no tests under `pipeline/*`.
- Recommended: Sandwich happy path, retriever failures, pre-rule denials, fallback thresholds.
- Declarative: flow resolution, tool dispatch, and denial handling when non-stubbed.
- Providers: BM25/dense/hybrid retrieval, cosine reranking, embedders, rules evaluation.
- Additional: localvec lifecycle, tool dependency wiring order, and LLM error handling.

---

## 11. Open Items / Known Issues
- **Resolved:** Global registry state hinders testing. (The registry is now an injected instance).
- **Resolved:** Lack of type safety due to `any` and reflection in factories. (Factories are now strongly-typed).
- **Resolved:** Builder is not OCP compliant. (Provider-specific logic has been moved into their respective factories).
- **Duplicated orchestration logic:** Conversational state handling exists in both orchestrators and could be centralized.
- **LLM `MaxTokens` ignored:** Current OpenAI/Google clients do not propagate `req.MaxTokens`, so orchestrator defaults cannot constrain completion length.
- **Hybrid RRF constant:** `internal/providers/hybrid/hybrid.go` uses a hard‑coded `k=60` with no configuration hook.
- **Context propagation:** Some providers may not consistently thread `context.Context` through external calls.

---

## 12. Extension Hooks
- Register providers using the generic API:
  - `manglekit.Register[T,D,O](reg, optsSample, func(ctx, deps, cfg) (T, error))` where `optsSample` implements `core.ProviderOptions`.
  - For orchestrators without options, pass `core.NilOptions{Name: "sandwich", Kind: core.KindOrchestrator}`.
- Select orchestrator with `WithOrchestrator`. Configure components using `With(opts)` or `WithKind(...)`.
- `sdk.GlobalRegistry()` provides a convenience singleton; create isolated registries for tests.

---

## 13. Alignment with HLD
- Principles (SDK-first, dual orchestrators, registry-driven extensibility, fail-fast construction, stateless engines with external state) match HLD §1.2.
- Components map to packages: builder/config/registry manage dependency graphs; providers cover BM25/dense/hybrid retrievers, cosine reranker, Google/OpenAI embedders and LLMs, localvec vector store, Mangle rules and schema parsing; orchestrators ship as Sandwich and Declarative implementations.
- Observability, lifecycle, and usage patterns align with HLD §4 via examples 01–10.
- Non-functionals—concurrency, scaling, enforcement via rules, and observability hooks—align with HLD §5; open items tracked in §11.
