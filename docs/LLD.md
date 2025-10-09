# Manglekit SDK — Low-Level Design (LLD)

> Module path: `github.com/duynguyendang/manglekit`  
> Go version: 1.24.1  
> Last Updated: 2025-10-09  
> Status: Sandwich and declarative orchestrators ship with the YAML-driven builder + tool graph, including post-rule evaluators for redact/drop/deny metadata. Example 04 exercises stage skipping and LLM opt-out paths. Embedder `Register` hooks and HTTP hosting layers remain TODO.

---

## 1. Design Tenets
- Single orchestrator entry point via `sdk.New` or the fluent `Builder`; the builder reads orchestrator type from configuration and wires a tool graph for declarative flows when requested.
- Providers self-register in `init()` through `registry.go`, exposing typed constructors; the builder asserts signatures and injects shared clients (Genkit, OpenAI) before invocation.
- Sandwich pipeline enforces `Mangle → Retrieval → Rerank → LLM → Mangle` ordering, persisting `original_docs`, `best_score`, and latency metrics into `Answer.Meta`.
- Declarative orchestration separates workflow definition (facts) from concrete tool wiring, using a shared execution context plus `RuleResult.SkippedStages` to gate runtime behavior.
- All components are constructed through typed options (see `typemap.go`) to avoid stringly-typed APIs while retaining YAML-driven `tools` configuration and programmatic overrides.

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
├── examples/
│   ├── 04-declarative-flow/   # Flow demo using Datalog-defined stages + tool graph.
│   ├── 05-chat-with-data/     # Sandwich example wired via YAML config.
│   └── …                      # Additional runnable guides (01-basic, 08-symbolic-rag).
└── docs/…                     # Design docs (this file, etc.).
```

---

## 3. Core Contracts & Observability
- `core.Doc` encapsulates chunk metadata. `Meta` is free-form and reused downstream (e.g., Mangle converters). `core.LocalvecOptions` remains for backwards compatibility; the builder prefers explicit vector store wiring.
- `core.Query` carries user text plus `Meta` (filters, expansion terms, user context, dynamic facts). Mangle `Mutate` callbacks write new filters back into this struct so retrievers see policy outputs.
- `core.Answer` stores final text, citations, and diagnostics. `Meta` is the primary coordination surface for later stages and now carries `original_docs`, `token_usage`, `best_score`, `denial_reason`, `rule_results`, and `redactions` when emitted by rules.
- `core.Options` holds `any` for Retriever/Reranker/LLM to avoid import cycles; `sdk.New` and the builder enforce concrete types and fill defaults (`TopK=8`, `MaxTokens=512`).
- `core.Observability` provides optional Logger/Tracer/Meter hooks. When absent, Sandwich falls back to `fmt.Printf` logging; when present, components record `manglekit.*_ms` metrics.
- `core.RuleSet` and `core.FlowController` define the rules contract; `RuleResult` carries `Mutate` and `SkippedStages` for declarative flows, while denying requests returns `core.ErrDenied`.
- `core.PostRuleEvaluator` lets rule engines run after retrieval (before LLM). Declarative flows detect this interface to enforce drop/redact policies mid-pipeline.
- `core.SchemaParser` enables custom schema-to-fact ingestion referenced by rules via `core.SchemaSource`.

---

## 4. Pipelines
### Sandwich Orchestrator (`pipeline/sandwich.go`)
1. Starts observability: emits `pipeline run started` via `Logger` or `fmt.Printf`, and opens `manglekit.Run` span when a tracer is provided.
2. Pre-rules: calls `RuleSet.Evaluate(core.Pre, q, nil)`, records `manglekit.rules_pre_ms`, applies `Mutate` to persist filters/expansions, and immediately returns `core.ErrDenied` on policy failures.
3. Retrieval: invokes `retriever.Retrieve` with `TopK` (default 8) and enriched metadata. Captures latency under `answer.Meta["retrieve_ms"]`, retains documents in `answer.Meta["original_docs"]`, and copies retriever metadata when present.
4. Rerank (optional): converts `rerank.ScoredDoc` into `core.Citation`s, stores `best_score`, records `manglekit.rerank_ms`, and logs counts. Without a reranker, `best_score` remains `0`.
5. Fallback: compares `best_score` against `Options.FallbackThreshold`, returning `core.ErrNoEvidence` if the threshold is not met.
6. LLM: materializes passages, calls `llm.Client.Complete` with `MaxTokens`, tracks `answer.Meta["llm_ms"]` and `answer.Meta["token_usage"]`, and records metrics.
7. Post-rules: replays `RuleSet.Evaluate(core.Post, q, &answer)`, leveraging `original_docs` to filter citations. Policy denials return `core.ErrDenied`; `Mutate` callbacks can redact or rewrite the answer.
8. Finishes with success logs; `answer.Meta` now contains timings, `best_score`, and any rule outputs.

### Declarative Orchestrator (`pipeline/declarative/orchestrator.go`)
- Reads the execution plan from Mangle facts via `flow_stage/3` + `stage_tool/2`, sorting by numeric order.
- Runs pre-rules once to populate filters, detect `RuleResult.SkippedStages`, and cache `Mutate` changes into a shared execution context (`query`, `answer`, `docs`, `meta`).
- Dispatches tools by type:
  * `retrieve.Retriever`: logs filter metadata, persists docs, annotates `retrieved_count`, and stores `original_docs` on the answer.
  * `rerank.Reranker`: rebuilds docs and citations, propagating `best_score` into context meta.
  * `core.PostRuleEvaluator`: executes Mangle post-rules before any LLM call, merging `result.Meta` (`rule_results`, `redactions`, `dropped_docs`) into both the answer and execution meta, with optional logging/metrics via `Observability`.
  * `llm.Client`: skipped when post-rules denied; otherwise generates text and token usage.
- Missing tools or unsupported types terminate the run with contextual errors. The final answer (including denial metadata) is pulled from the execution context, ensuring stage outputs remain observable.

---

## 5. Builder & Configuration
- `config.Config` models orchestrator selection (sandwich vs declarative), pipeline defaults, provider credentials, and either component wiring (`componentCfg{Name, Params}`) or a declarative `tools` map.
- `NewBuilderFromYAML` expands env vars, materializes typed option structs via JSON round-trip, records the config directory for `path:"resolve"` handling, and seeds the fluent builder.
- `typemap.go` maps option pointer types to provider names (`*retrieve.BM25Options` → `"bm25"`). The inverse map powers YAML-driven tool construction and aliasing (`google-embedder`, `groq`).
- Builder flow:
  1. `With*` methods store provider names plus typed configs; `WithFlow` overrides the declarative flow name.
  2. `resolveDependencies` infers missing embedders/vector stores from other components and tracks provider families for credential resolution.
  3. `resolveProviderConfig` uses `ProviderConfigs` or env (`GOOGLE_API_KEY`, `OPENAI_API_KEY`, `GROQ_API_KEY`) to instantiate shared clients (`genkit.Genkit`, `genai.Client`, `openai.Client`) cached on the builder, and records shutdown callbacks for anything that needs explicit tear down.
  4. `buildComponents` executes in dependency order (embedder → vector store → retriever → reranker → rules → LLM), injecting typed options into registry constructors.
  5. Declarative flows call `buildTools`, which repeatedly resolves dependencies using a string heuristic, then dispatches `buildSingleTool` for embedder/vector store/retriever/LLM wiring (e.g., passing constructed tools into `dense`, `hybrid`, `localvec`).
  6. `Build` switches on orchestrator: sandwich returns `pipeline.NewSandwich`; declarative asserts the rules engine implements `core.FlowController`, injects it under `mangle_rules`, and calls `declarative.New`.
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
- **Mangle RuleSet (`internal/providers/mangle/rules.go`)**: Loads rules/facts via globbing, optionally merges schema facts (`SchemaSource`), and initializes converters. Supports `DefaultConverters` (query/user/doc). Honors `FileFirst` to defer EDB declarations to `.dlog`. Evaluates once to seed a base store; each request clones that store, injects runtime facts, and re-evaluates strata.
  * **Pre stage (`Evaluate(core.Pre)`)**: emits `expansion_terms`, `filters`, `skip_stage`, `deny`. `Mutate` writes filters/expansions into query meta and pre-populates answer denial metadata.
  * **Declarative post hook (`Post`)**: ingests current docs + meta (`best_score`, `retrieved_count`), collects `deny`, `drop_doc`, `redact`, and returns `core.PostRuleResult`. Outputs include filtered docs, `rule_results`, `dropped_docs`, `redactions`, and `denied_reason`; redactions support built-in patterns (`phone`, `email`) or `regex:` labels.
  * **Evaluate(core.Post)**: for sandwich flows, replays citations from `answer.Meta["original_docs"]`, filters denied documents, and records `mangle_denied_reasons`.
  * **FlowController.Query**: Streams solutions for declarative planning by parsing single-atom queries (`flow_stage/3`, `stage_tool/2`, etc.).
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
- `examples/04-declarative-flow/main.go`: Builds the declarative orchestrator from YAML, loads a Datalog `flow.dlog`, and demonstrates policy-driven stage skipping (guest vs employee, `(no_llm)` flag) with redaction outputs surfaced via answer meta.
- `examples/05-chat-with-data/main.go`: Loads `.env`, builds the sandwich orchestrator from `config.yaml`, imports `providers/all`, and runs against a markdown corpus with policy rules.
- No production HTTP binary yet; `cmd` directory is absent in this snapshot.

---

## 9. Testing Coverage
- `pipeline/sandwich_test.go`: Validates happy path, retrieval failure, pre-rule denial, and fallback threshold behavior.
- Provider tests: `internal/providers/{bm25,dense,hybrid,rerank/cosine}` cover indexing, fusion, converter pipelines, and similarity ordering.
- `internal/providers/mangle/rules_test.go`: Exercises pre/post evaluation, declarative `Post` hook (drop/deny/redact), and ensures metadata (`dropped_docs`, `redactions`, `rule_results`) surfaces correctly.
- Additional integration coverage can be added by exercising `localvec` and LLM providers with mocks; declarative orchestrator currently lacks direct unit tests.

---

## 10. Known Gaps & Risks
- Embedder `Register` methods (`internal/embedders/*`) panic when invoked; Genkit plugin registration is incomplete.
- Provider clients use `context.Background()` with no timeout/cancellation; long-running calls cannot be cancelled by callers.
- Sandwich fallback relies on reranker-provided `bestScore`; without a reranker the fallback threshold path always returns `ErrNoEvidence`.
- `localvec.New` re-initializes Genkit and LocalVec each build; resource reuse and shutdown hooks are absent. `Search` hard-requires `ctx` to carry `"query_text"`.
- Declarative tool dependency detection treats any string param as a dependency, risking false positives/negatives.
- `resolveProviderConfig` now registers cleanup callbacks so that the orchestrator's `Close(ctx)` shuts down the shared `genai.Client` and cancels the Genkit runtime context.
- Logging defaults to stdout prints when `Observability.Logger` nil; consider wiring `internal/logger` helper.
- No HTTP server or ingestion pipeline yet; examples rely on CLI context only.
- Declarative orchestrator lacks dedicated unit tests; behavior is currently exercised via runnable examples.

---

## 11. Extension Hooks
- New providers should register constructors via `manglekit.Register*` and supply typed options mapped in `typemap.go`.
- Additional fact converters or schema parsers can be registered as `Registry.Component` or `Registry.SchemaParser` entries for discovery by the Mangle provider.
- Declarative flows can add tools by extending `config.tools` and ensuring builder support via `buildSingleTool`.
