# Manglekit SDK — Low-Level Design (LLD)

> Module path: `github.com/duynguyendang/manglekit`  
> Go version: 1.24.1  
> Last Updated: 2025-10-17
> Status: **Major architectural refactoring complete.** The builder and registry architecture is now significantly more robust, type-safe, and extensible. Sandwich and Declarative orchestrators are production-ready.

---

## 1. Design Tenets
- Public entry points: `sdk.New` and the fluent builder (`builder.go`). They enforce defaults (`TopK=8`, `MaxTokens=512`), assert concrete types for registry‑constructed components, and accumulate LIFO `ResourceClosers` for clean shutdown.
- **Dependency Injection over global state.** The component `Registry` is now an instance created in the application entry point (`main.go`). Providers are explicitly registered with this instance, which is then injected into the `Builder`. This pattern eliminates global state and ensures clear, testable dependency resolution.
- Sandwich orchestrator (`pipeline/sandwich.go`) executes a fixed, rule‑wrapped flow; records timings in `Answer.Meta` (`retrieve_ms`, `rerank_ms`, `llm_ms`); and stores pre‑rule evidence as `Meta["original_docs"]`.
- Declarative orchestrator (`pipeline/declarative/orchestrator.go`) queries a `core.FlowController` to determine stage order, dispatches tools via a shared execution context, and supports `core.PostRuleEvaluator` for pre‑LLM gating.
- Observability is pluggable. If `core.Observability.Logger` is nil, a lightweight structured `StdLogger` is installed; tracing and metrics hooks are optional and skipped when unset. No direct `fmt.Printf` is used in production paths.

---

## 2. Package Layout (authoritative)
```
github.com/duynguyendang/manglekit
├── builder.go                 # Fluent builder + dependency resolution/runtime clients
├── from_config.go             # Translate validated config -> fluent builder
├── factories.go               # (doc stub) factory type defs moved to registry.go
├── registry.go                # Instance registry; typed factories + options/type maps + orchestrators
├── sdk.go                     # Convenience wrapper around pipeline.NewSandwich
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
- `core.Doc` contains chunk identifiers, provenance, payload text, and free-form metadata. Provider implementations populate common keys (`doc_id`, `source`, sparse scores) for rules to inspect.
- `core.Query.Meta` acts as the coordination channel for filters, expansion terms, `user_context`, and conversation `history`. Mangle Pre rules mutate these keys so retrievers receive policy-compliant constraints.
- `core.Answer.Meta` aggregates timings (`retrieve_ms`, `rerank_ms`, `llm_ms`), best rerank score, `original_docs`, LLM token usage, and rule emissions (`rule_results`, `redactions`, `denial_reason`, etc.).
- `core.Options` stores components as `any` to avoid import cycles; the builder fills them with concrete instances, applies defaults, and appends provider-specific `ResourceClosers`.
- Observability hooks are optional interfaces (`Logger`, `Tracer`, `Meter`). When `Logger` is nil, the builder installs a default structured `StdLogger`; tracing/metrics are no-ops when unset.

---

## 4. Builder API
- `NewBuilder(reg *Registry)`: initializes defaults and binds an instance `Registry` (no globals).
- `With*` methods set components via typed options: `WithRetriever/WithReranker/WithLLM/WithEmbedder/WithVectorStore/WithStateProvider` resolve the provider name from the options type using `Registry.RegisterOptions` mappings.
- Orchestrator: `WithOrchestrator("sandwich"|...)` selects a factory from `Registry.OrchestratorFactories`; defaults to `sandwich`. `WithFlow` names the declarative flow.
- Other config: `WithTopK`, `WithMaxTokens`, `WithFallbackThreshold`, `WithObservability`, `WithGenkit`.
- Build order: embedder → vector store → retriever → reranker → rules → llm → state provider. Each factory receives typed `diapi.Deps` and may add `ResourceClosers`.
- Sub-builds: complex components (e.g., `hybrid` retriever) call `Builder.BuildRetriever` via injected `diapi.BuildSubRetriever`.

---

## 5. Registry & Provider Wiring
- Instance-based `Registry` holds typed factory maps: `Retrievers`, `Rerankers`, `LLMs`, `Embedders`, `VectorStores`, `RuleSets`, `StateProviders`, `SchemaParsers`, `FactConverters`, and `OrchestratorFactories`.
- Options mapping: `RegisterOptions(providerName, (*T)(nil))` records option type ↔ name within the registry; lookup used by the Builder to resolve provider names from typed options.
- Provider registration is explicit and centralized (e.g., via `providers/all.RegisterAll(reg)`). The side-effect `init()` pattern is not used.
- Option structs for programmatic flows live under `retrieve/`, `rerank/`, `embed/`, and `llm/` packages.

---

## 6. Retrieval & Vector Storage
- BM25 (`internal/providers/bm25`): indexes Markdown directories, parses YAML front matter into metadata, attaches Okapi scores in metadata.
- Dense (`internal/providers/dense`): embeds queries via configured embedder, queries injected `core.VectorStore`, supports metadata filters.
- Hybrid (`internal/providers/hybrid`): runs sparse and dense in parallel, fuses with RRF, truncates to `TopK`.
- Localvec (`internal/vectorstores/localvec`): integrates Genkit localvec; indexes corpus with metadata; known gap on lifecycle cleanup.
- In-memory retriever (`internal/providers/retrievers/inmemory`): implements `retrieve.Updatable` for demos/small corpora.

---

## 7. Sandwich Orchestrator Mechanics
- Context: builds `pipeline.PipelineContext` with `Query`, session `History`, timers, and `Answer.Meta`.
- Stages: `PreRulesStage` → `RetrieveStage` → `RerankStage` → `LLMStage` → `PostRulesStage`, executed by `pipeline.Runner`.
- Retrieval: `Retriever.Retrieve` populates `OriginalDocs`; metrics (`manglekit.retrieve_ms`) recorded; evidence saved to `Answer.Meta["original_docs"]`.
- Rerank: optional `Reranker` emits `Citations` with scores; `best_score` tracked for fallback; metrics `manglekit.rerank_ms` recorded.
- LLM: `llm.Client.Complete` consumes query and passages; `token_usage` captured; `MaxTokens` forwarded; metrics `manglekit.llm_ms` recorded.
- Post rules: `RuleSet.Evaluate(core.Post)` may redact citations or deny; records `manglekit.rules_post_ms`.
- Cleanup: `Close()` drains `ResourceClosers` LIFO; conversation state persisted via `StateProvider` when configured.

---

## 8. Declarative Orchestrator Mechanics
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
- Repository currently contains no Go test files under `pipeline/*` in tree.
- Recommended targets: Sandwich happy path, retriever failures, pre-rule denials, fallback thresholds.
- Declarative: flow resolution, tool dispatch, and denial handling.
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
- Register via `reg.Register*` on a `*Registry`, and register typed options with `reg.RegisterOptions("name", (*OptionsType)(nil))`.
- Converters and schema parsers: `RegisterFactConverter` / `RegisterSchemaParser`; referenced by rules/flows.
- Select orchestrator with `WithOrchestrator`; set the declarative flow with `WithFlow`.

---

## 13. Alignment with HLD
- Principles (SDK-first, dual orchestrators, registry-driven extensibility, fail-fast construction, stateless engines with external state) match HLD §1.2.
- Components map to packages: builder/config/registry manage dependency graphs; providers cover BM25/dense/hybrid retrievers, cosine reranker, Google/OpenAI embedders and LLMs, localvec vector store, Mangle rules and schema parsing; orchestrators ship as Sandwich and Declarative implementations.
- Observability, lifecycle, and usage patterns align with HLD §4 via examples 01–10.
- Non-functionals—concurrency, scaling, enforcement via rules, and observability hooks—align with HLD §5; open items tracked in §11.
