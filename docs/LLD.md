# Manglekit SDK — Low-Level Design (LLD)

> Module path: `github.com/duynguyendang/manglekit`  
> Go version: 1.24.1  
> Last Updated: 2025-10-10  
> Status: Sandwich orchestrator + fluent builder are production-ready; declarative orchestrator executes Mangle-defined flows (with post-rule gating) but is only exercised via examples; embedder `Register` hooks remain TODO.

---

## 1. Design Tenets
- `sdk.New` and the fluent `Builder` provide the only supported entry points; they ensure defaults (`TopK=8`, `MaxTokens=512`), enforce type assertions against registry constructors, and accumulate `ResourceClosers` for shared clients.
- Provider registries (`registry.go`, `typemap.go`) keep a single source of truth for provider names ↔ option structs so programmatic and YAML-driven flows share wiring.
- The Sandwich orchestrator enforces the fixed stage order (rules → retrieve → rerank → fallback → LLM → rules), records timings in `Answer.Meta`, and keeps the unreduced evidence in `original_docs` for downstream rule inspection.
- The declarative orchestrator queries a `core.FlowController` to derive stage order at runtime, shares execution state via a context map, and invokes any `core.PostRuleEvaluator` before calling an LLM to propagate denies, drops, and redactions.
- Observability is optional: when `core.Observability` fields are nil the pipeline falls back to `fmt.Printf` logging and skips tracing/metrics so examples can run without extra setup.

---

## 2. Package Layout (authoritative)
```
github.com/duynguyendang/manglekit
├── builder.go                 # Fluent builder + dependency resolution
├── config.go                  # YAML/env loader feeding the builder
├── registry.go                # Global registries for provider constructors
├── sdk.go                     # Convenience wrapper around pipeline.NewSandwich
├── typemap.go                 # Option-type ↔ provider-name lookup tables
├── core/
│   ├── rules.go               # Stage enums, RuleSet/FlowController contracts
│   ├── schema.go              # SchemaParser contract
│   └── types.go               # Doc/Query/Answer/Options, observability, errors
├── retrieve/
│   ├── options.go             # Typed options structs
│   ├── retrieve.go            # Request/Result + Retriever interfaces
│   ├── bm25.go                # Public wiring helpers (implementation is internal)
│   ├── hybrid.go              # Request helper for hybrid retrievers
│   └── inmemory.go            # Request helper for in-memory retrievers
├── rerank/
│   ├── options.go             # Cosine + ColBERT option structs
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
│       └── orchestrator.go    # Flow-driven orchestrator
├── internal/
│   ├── embedders/{google,openai}/
│   ├── providers/
│   │   ├── bm25/
│   │   ├── dense/
│   │   ├── hybrid/
│   │   ├── llm/{google,openai}/
│   │   ├── mangle/
│   │   ├── rerank/cosine/
│   │   ├── retrievers/inmemory/
│   │   └── schemaparsers/{jsonschema,rdf}/
│   ├── vectorstores/localvec/
│   └── logger/logger.go       # zap helper (not yet wired)
├── providers/all/all.go       # Blank import of every bundled provider
├── examples/                  # 01-basic-rag … 09-genkit-tool runnable demos
├── apps/                      # Experimental integrations (e.g., rdf-knowledge-base)
├── docs/                      # HLD, LLD, CONTEXT, CSD
└── rules/                     # Placeholder for sample .dlog programs
```

---

## 3. Core Contracts & Observability
- `core.Doc` carries chunk ID/source/text plus free-form `Meta`; BM25 and LocalVec populate fields like `doc_id`, `source`, and sparse scores for downstream rules.
- `core.Query.Meta` is the coordination channel for filters, expansion terms, and `user_context`; Mangle pre-rules mutate these keys so retrievers receive policy-aware constraints.
- `core.Answer.Meta` holds timings (`retrieve_ms`, `llm_ms`), `original_docs`, `best_score`, and LLM `token_usage`; declarative post-rules merge additional keys (`rule_results`, `redactions`, `denied_reason`, `rules_post_ms`) while rule denials attach `mangle_denied_reasons`.
- `core.Options` stores components as `any` to avoid import cycles; the builder fills them with the correct concrete types, sets defaults, and appends any provider `ResourceClosers`.
- `core.RuleSet`, `core.FlowController`, and `core.PostRuleEvaluator` define the policy surface: Sandwich uses `Evaluate(pre|post)` while the declarative orchestrator also expects `Query` and optional `Post`.
- `core.Observability` (`Logger`, `Tracer`, `Meter`) is pluggable; when unset the orchestrators downgrade to basic stdout logs and skip spans/metrics.

---

## 4. Sandwich Orchestrator (`pipeline/sandwich.go`)
1. Observability: emits `pipeline run started` via `Logger` or stdout and opens a `manglekit.Run` span when a tracer exists.
2. Pre-rules: calls `RuleSet.Evaluate(core.Pre, q, nil)`, records `manglekit.rules_pre_ms`, applies `Mutate` to persist filters/expansion terms, and returns early with `core.ErrDenied` while recording `mangle_denied_reasons` when policies fail.
3. Retrieval: invokes `retriever.Retrieve` with the configured `TopK` and enriched metadata, stores latency in `answer.Meta["retrieve_ms"]`, and snapshots the original evidence list in `answer.Meta["original_docs"]`.
4. Rerank (optional): converts `rerank.ScoredDoc` into citations, stores `best_score`, and records `manglekit.rerank_ms`; without a reranker `best_score` remains `0` and citations stay empty.
5. Fallback: compares `best_score` against `Options.FallbackThreshold`; when the threshold is set and not met it returns `core.ErrNoEvidence` before calling the LLM.
6. LLM: materializes passages, calls `llm.Client.Complete` with `MaxTokens` and `Data=q.Meta`, tracks `answer.Meta["llm_ms"]`, and records the provider `Usage` map under `token_usage`.
7. Post-rules: replays `RuleSet.Evaluate(core.Post, q, &answer)` using `original_docs` to filter citations; denials return `core.ErrDenied`, and `Mutate` can redact or rewrite the answer.
8. Completion: logs success and returns the final `core.Answer`; any configured `ResourceClosers` run later via `Orchestrator.Close`.

---

## 5. Declarative Orchestrator (`pipeline/declarative/orchestrator.go`)
- Derives the execution plan by querying `flow_stage/3` and `stage_tool/2` facts from the `core.FlowController`, sorting stages numerically.
- Runs pre-rules once to populate filters, compute `RuleResult.SkippedStages`, and cache `Mutate` output into the shared execution context (`query`, `answer`, `docs`, `meta`).
- Dispatches tools by interface: `retrieve.Retriever` populates docs and `retrieved_count`, `rerank.Reranker` rebuilds citations and updates `best_score`, `core.PostRuleEvaluator` merges rule metadata (e.g., `rule_results`, `dropped_docs`, `redactions`, `denied_reason`), and `llm.Client` generates text unless the post stage denied the request.
- Maintains an execution `meta` map mirrored into `answer.Meta`, recording durations like `rules_post_ms` and any denial state (`denied`, `denial_reason`).
- Missing tools or unsupported types produce contextual errors; when no stage survives post-rules the orchestrator short-circuits with `core.ErrDenied` while preserving meta explanations.

---

## 6. Builder & Configuration
- `config.Config` models orchestrator choice (sandwich vs declarative), pipeline defaults, provider credentials, and either direct component configs (`componentCfg`) or a declarative `tools` map.
- `NewBuilderFromYAML` expands environment variables, reinstantiates typed option structs via JSON round-trips, and resolves `path:"resolve"` fields relative to the config directory before chaining fluent `With*` calls.
- `typemap.go` maintains the authoritative option-type ↔ provider-name mapping; aliases (e.g., `google-embedder`, `groq`) make YAML configs unambiguous.
- `resolveProviderConfig` instantiates shared clients (Google: `genai.Client` + `genkit.Genkit` with a registered `ResourceCloser`; OpenAI/Groq: `openai.Client` with API key/base URL) based on `ProviderConfigs` or environment variables.
- `buildComponents` honors dependency ordering (embedder → vector store → retriever → reranker → rules → LLM) and populates `core.Options`, while `buildTools` iteratively resolves declarative tool dependencies using a simple string heuristic.
- `Build` branches by orchestrator: sandwich resolves dependencies then calls `sdk.New`; declarative asserts the rules engine implements `core.FlowController`, preloads `mangle_rules`, and invokes `declarative.New`. `closeResources` flushes registered closers on failure paths.

---

## 7. Providers & Components
- **Retrieval**: `internal/providers/bm25` indexes markdown via `go-nlp/tfidf` + `go-nlp/bm25`, emitting `core.Doc` records with front-matter metadata; `internal/providers/dense` embeds queries with an `ai.Embedder` and searches a `core.VectorStore`, forwarding policy filters; `internal/providers/hybrid` runs BM25 and dense in parallel and fuses results with Reciprocal Rank Fusion; `internal/providers/retrievers/inmemory` offers an updatable in-memory store for tests and demos.
- **Vector store**: `internal/vectorstores/localvec` wraps Genkit LocalVec, loads markdown corpora at build time, requires the caller to pass the raw query via `ctx.Value("query_text")`, and currently re-initializes Genkit on each construction.
- **LLM clients**: `internal/providers/llm/openai` covers OpenAI and Groq using `openai-go`, builds prompts through `llm.PromptBuilder`, and returns usage metrics keyed by `prompt`/`completion`; `internal/providers/llm/google` relies on Genkit's `googlegenai` plugin and aggregates streamed output into a final string, recording token counts when available.
- **Embedders**: `internal/embedders/google` exposes `embedding-001` via `genai.Client` with fixed dimensionality; `internal/embedders/openai` (also registered as `groq`) proxies the embeddings API and casts float64 vectors to float32. Both implement Genkit's `ai.Embedder` interface but leave the `Register` method as a panic.
- **Rules engine**: `internal/providers/mangle` loads `.dlog` programs (file-first or code-first), applies default converters (`Query`, `UserContext`, `Document`), accepts optional schema sources, and evaluates base strata up front. `preProcess` emits filters (`query_filter/3`), expansion facts (`expanded_query/2`), `skip_stage`, and `deny`, mutating query meta; `postProcess` (Sandwich) filters citations and attaches `mangle_denied_reasons`; `Post` (declarative) clones evidence, applies document drops/redactions, collects denial metadata, and returns structured `Meta` (`fired_rules`, `dropped_docs`, `redactions`, `rule_results`, `denied_reason`).
- **Converters**: `QueryConverter` tokenizes/normalizes queries and extracts version strings, `UserContextConverter` converts `user_context` into `user_attribute/2` facts with IRI support, and `DocumentConverter` produces `doc_id`, `doc_content`, `doc_text`, and `doc_metadata` facts, handling string slices in metadata.
- **Schema parsers**: `jsonschema` transforms JSON Schema documents into facts (`schema/1`, `field/3`, `field_required/2`, `field_format/3`, `field_constraint/4`); `rdf` decodes Turtle files into `triple/3` facts with IRI-aware constants.

---

## 8. Prompt Management
- `llm/prompt.go` defines `DefaultRAGTemplate`, instructing models to answer strictly from context and confess when evidence is missing.
- `llm.PromptBuilder` caches compiled Go templates behind an RWMutex, injects helpers (`toJSON`, `join`, `truncate`), and falls back to the default template when the provider options omit a custom prompt.

---

## 9. Examples & Integration
- `examples/01-basic-rag`: Sandwich pipeline wired programmatically with local markdown evidence.
- `examples/02-logic-layer-mode`: Highlights pre-rule normalization and policy filtering without custom retrievers.
- `examples/03-custom-prompt`: Demonstrates supplying a bespoke LLM template through configuration.
- `examples/04-declarative-flow`: Runs the declarative orchestrator with `flow.dlog`, stage skipping, and LLM opt-out paths.
- `examples/05-chat-with-data`: Loads `.env`, imports `providers/all`, and drives a sandwich pipeline from `config.yaml`.
- `examples/06-schema-validation`: Shows schema parser integration where post-rules enforce document structure.
- `examples/07-rdf-knowledge-base`: Consumes RDF triples and reasons over them via Mangle rules.
- `examples/08-symbolic-rag`: Focuses on deterministic responses with heavy post-rule gating.
- `examples/09-genkit-tool`: Illustrates Genkit tool registration and orchestration integration.
- `apps/rdf-knowledge-base`: Experimental app-level wiring for the RDF flow.

---

## 10. Testing Coverage
- `pipeline/sandwich_test.go` exercises happy path, retriever failures, pre-rule denials, and fallback threshold logic.
- Provider suites cover BM25 indexing, dense retriever embeddings, hybrid fusion, cosine reranker scoring, and Mangle rule evaluation (including `Post` redaction/drop/deny flows).
- No unit tests exist for the declarative orchestrator or the LLM/embedder providers; integration expectations are exercised through examples.
- Additional coverage targets include vector store lifecycle (`localvec`), tool dependency resolution, and YAML-driven builder flows.

---

## 11. Known Gaps & Risks
- Embedder `Register` methods panic; Genkit plugin registration for the embedders remains unfinished.
- Provider calls often use `context.Background()` (retrievers, LLM clients, Genkit helpers), so cancellation and deadlines must be managed by callers.
- Sandwich fallback relies on reranker output; without a reranker any positive `FallbackThreshold` causes `ErrNoEvidence`.
- `internal/vectorstores/localvec` re-initializes Genkit/localvec for every build and lacks a `ResourceCloser`, so repeated builds leak background resources.
- Declarative tool dependency detection treats any string parameter as a dependency, which can misorder tool construction.
- Logging defaults to stdout when `Observability.Logger` is nil; `internal/logger` is not wired anywhere.
- OpenAI/Groq clients are never closed explicitly; only Google clients register shutdown callbacks.

---

## 12. Extension Hooks
- New components register constructors via the appropriate `manglekit.Register*` helper and must expose a typed options struct listed in `typemap.go`.
- Additional converters or schema parsers register under `Registry.Component` or `Registry.SchemaParser` for discovery by the Mangle provider.
- Declarative flows extend `config.tools` and rely on `buildSingleTool` to inject dependencies; ensure new tools encode dependencies via params so the string heuristic can discover them.
