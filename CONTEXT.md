---
context_type: codebase_overview
project: manglekit
language: go
version: 2025.10
last_updated: 2025-10-13
---

# Manglekit Project Context

## Overview
Manglekit is a Go 1.24+ toolkit that combines Google’s Mangle Datalog engine with retrieval, reranking, and LLM calls. The default “Sandwich” pipeline runs **Mangle-Pre → Retrieval → optional Rerank → LLM → Mangle-Post**; an alternate declarative orchestrator drives stage ordering from Mangle facts. Providers are registered at init-time and wired at runtime through the global registry in `registry.go`.

---

## Core Building Blocks
- ⚠️ **Registry (`registry.go`)**: Global maps hold constructors for retrievers, rerankers, LLMs, embedders, schema parsers, and generic “components”. `RegisterOptions` maps provider names to option pointer types, but production providers do not currently call it.
- ✅ **Core types (`core/types.go`)**: `Query`, `Answer`, `Doc`, `Citation`, and `Options` structure pipeline data. `Options` embeds `Observability` hooks plus `ResourceClosers`. Errors surface as sentinel values (`ErrInvalidOptions`, `ErrNoEvidence`, `ErrDenied`).
- ✅ **Observability contracts**: `Logger` (Info/Error), `Tracer` (`StartSpan`), and `Meter` (`Record`) let pipelines instrument work without binding to specific libraries.
- ✅ **SDK entry point (`sdk.go`)**: `New` validates that retriever and LLM are present, fills defaults (`TopK=8`, `MaxTokens=512`), type-asserts concrete interfaces, and delegates to `pipeline.NewSandwich`.

---

## Orchestrator Construction
- ⚠️ **Fluent builder (`builder.go`)**
  - `NewBuilder` seeds state, including `providerNames`, `clients`, and `ResourceClosers`.
  - `With*` methods accept typed option pointers; `WithEmbedder` also accepts a pre-built `ai.Embedder`. Methods stash config and accumulate errors if the options type was not registered via `RegisterOptions`.
  - `resolveProviderConfig` lazily creates shared clients for `"google"` (Genkit + Generative AI), `"openai"`, and `"groq"`, wiring shutdown callbacks that close transports or cancel contexts.
  - `resolveDependencies` infers missing embedder names from retriever/reranker requests before resolving provider configs.
  - `buildComponents` executes in dependency order (Embedder → VectorStore → Retriever → Reranker → Rules → LLM) and appends closers when instances expose `Close(context.Context) error`.
  - `Build` chooses the orchestrator (`sandwich` default, `declarative` optional). Declarative builds a `core.FlowController`, materialises all declared tools, and hands them to `declarative.New`.
- ⚠️ **Configuration (`config.go`)**
  - `Config` captures orchestrator selection plus component slots (`embedder`, `retriever`, `vectorStore`, `reranker`, `rules`, `llm`) and provider family defaults (`google`, `openai`, `groq`, `mangle`).
  - `NewBuilderFromYAML` expands env vars, unmarshals into `Config`, resolves `path:"resolve"` struct tags relative to the file, and invokes the builder’s `With*` methods using typed options (requires prior `RegisterOptions` calls).
  - `NewBuilderFromEnv` mirrors the YAML flow using `MKT_*` environment variables, again depending on registered option types.

---

## Execution Pipelines
- ⚠️ **Sandwich orchestrator (`pipeline/sandwich.go`)**
  - Bootstraps logging/tracing/metrics via `core.Observability`, defaulting to `fmt.Printf` when no logger is provided.
  - **Pre rules**: Executes `Rules.Evaluate(core.Pre, q, nil)`, recording latency and applying policy-driven mutations through `RuleResult.Mutate`. Denials short-circuit with `ErrDenied`.
  - **Retrieval**: Calls `retrieve.Retriever.Retrieve`, stores latency in `Answer.Meta["retrieve_ms"]`, and snapshots the raw docs in `Answer.Meta["original_docs"]`.
  - **Rerank & fallback**: When a reranker is present, rescoring populates `Answer.Citations`, `Answer.Meta["best_score"]`, and metrics. `FallbackThreshold` is enforced only inside this reranker branch.
  - **LLM**: Builds context passages, invokes `llm.Client.Complete`, records `"llm_ms"` and `"token_usage"`, and stores the returned text.
  - **Post rules**: Re-runs `Rules.Evaluate(core.Post, q, &answer)` for policy enforcement, allowing additional mutations or denials.
  - Returns `ErrNoEvidence` when no documents survive retrieval (or rerank fallback) and propagates wrapped errors for component failures.
- ⚠️ **Declarative orchestrator (`pipeline/declarative/orchestrator.go`)**
  - `getFlowStages` queries Mangle facts (`flow_stage/3`, `stage_tool/2`) to assemble an ordered stage list.
  - Pre-rules (`RuleResult`) may deny or provide `SkippedStages` and metadata mutations. Execution context maps (`contextKeyQuery`, `contextKeyDocs`, etc.) carry state across stages.
  - Stage dispatch handles retrievers, rerankers, `llm.Client`, and `core.PostRuleEvaluator` instances. Retrieves and reranks emit debug logs with `fmt.Printf`; post rules capture metrics via `Meter` when available.
  - Post-rule stages filter/redact documents, enforce denials (`ErrDenied`), and prune citations to match surviving evidence. Final answers come from the shared context map.

---

## Rules Engine (`internal/providers/mangle`) ⚠️
- **Initialization (`New`)**: Parses optional schemas through registered schema parsers (e.g., `jsonschema`, `rdf`), instantiates default and custom converters, decides between code-first vs file-first predicate declarations, loads `.dlog` / `.facts` files, stratifies the program, seeds a base fact store, and performs an initial evaluation.
- **Converters**: Built-ins include `QueryConverter` (tokens, versions, normalized text), `UserContextConverter` (metadata to `user_attribute/2` facts with optional IRI handling), and `DocumentConverter` (document IDs, text, metadata fan-out).
- **Evaluate (pre/post)**: Clones the base store, injects converter-derived facts, re-evaluates the program, and returns `RuleResult` with `Allowed`, `Reason`, `Mutate` (injecting `filters` and `expansion_terms`), and optional `SkippedStages`.
- **Post hook (`Post`)**: Accepts the query, evidence, and execution metadata; adds `retrieved_doc` facts; evaluates drop/redact/deny rules; applies regex-based redactions; filters documents; records applied rule metadata; and returns `core.PostRuleResult`.
- **Query**: Implements read-only fact queries by running the engine and manually unifying results into Go maps.

---

## Providers & Components
- ✅ **Retrieval**
  - ✅ `internal/providers/bm25`: Walks Markdown files (parsing YAML front matter), builds TF-IDF vocabulary, applies Okapi BM25 scoring (default `TopK=10`), and annotates scores in document metadata.
  - ⚠️ `internal/providers/dense`: Couples an `ai.Embedder` with a `core.VectorStore`; embeds the query, passes vectors plus `Meta["filters"]` to the store, and relies on `context.WithValue(ctx, "query_text", ...)` for LocalVec compatibility.
  - ⚠️ `internal/providers/hybrid`: Executes sparse and dense retrievers in parallel (`errgroup`), fuses results with Reciprocal Rank Fusion (`k=60`), and trims to `TopK`.
  - ⚠️ `internal/providers/retrievers/inmemory`: Thread-safe map-backed retriever implementing `retrieve.Updatable`; `Upsert`/`Replace` emit `fmt.Printf` diagnostics.
- ⚠️ **Vector store**
  - ⚠️ `internal/vectorstores/localvec`: Spins up Genkit’s LocalVec retriever with a managed context, indexes Markdown documents at startup, filters matches via metadata equality checks, and exposes `Close` to cancel the Genkit background context. `Search` requires the original query text via `ctx.Value("query_text")` (string key).
- ✅ **Rerank**
  - ⚠️ `internal/providers/rerank/cosine`: Uses a shared embedder to embed query and docs concurrently (`errgroup`), computes cosine similarity with a custom float32 `sqrt`, sorts descending, and truncates to `TopK`.
- ⚠️ **Embedders**
  - ⚠️ `internal/embedders/openai`: Registers as `"openai"` and `"groq"`, builds embeddings through the OpenAI API (optional `Dimensions`), and converts float64 arrays to float32.
  - ⚠️ `internal/embedders/google`: Registers as `"google-embedder"`, wraps Genkit’s Google AI embedder (default model `embedding-001`), and currently mismatches the builder’s `"google"` alias.
- ⚠️ **LLM clients**
  - ⚠️ `internal/providers/llm/google`: Wraps a Genkit model, builds prompts with `llm.PromptBuilder`, streams text parts into the final answer, and returns `Usage` maps keyed `"prompt"` / `"completion"`.
  - ⚠️ `internal/providers/llm/openai`: Supports OpenAI-compatible APIs (OpenAI/Groq), builds prompts via `PromptBuilder`, and calls Chat Completions, returning the first choice plus usage counters.
- ✅ **Schema parsers**
  - ✅ `internal/providers/schemaparsers/jsonschema`: Converts JSON Schema documents into `schema/1`, `field/3`, and constraint facts.
  - ✅ `internal/providers/schemaparsers/rdf`: Decodes Turtle RDF triples into `triple/3` facts with smart IRI handling.
- ⚠️ **Mocks (`internal/providers/mock`)**: Lightweight retriever, reranker, LLM, and tool implementations for tests, alongside helpers to convert between Datalog constants and Go objects.
- ⚠️ **Catch-all import (`providers/all/all.go`)**: Blank-importing registers every bundled provider family.

---

## Prompting (`llm/prompt.go`) ✅
- `PromptBuilder` caches compiled `text/template` instances, guarded by a RW mutex, and injects helper functions (`toJSON`, `join`, `truncate`).
- The default RAG template instructs the model to ground answers strictly in the provided context and to refuse when evidence is missing.

---

## Applications & Samples
- ✅ **`apps/rdf-knowledge-base`**: CLI demo that loads RDF data via the Mangle rules engine, normalises user input into query metadata, runs `RuleSet.Evaluate(core.Pre, ...)`, and prints sheet associations plus canonical keyword expansions. Uses `godotenv` to source environment variables.

---

## Observability & Logging ⚠️
- Pipelines emit structured logs when a `Logger` is provided and fall back to `fmt.Printf` otherwise. Latencies are recorded via `Meter.Record` under metric names like `manglekit.retrieve_ms`, `manglekit.rerank_ms`, `manglekit.llm_ms`, and `manglekit.rules_post_ms`.
- Builders collect `ResourceClosers` from providers (Genkit clients, HTTP transports, vector stores) and unwind them LIFO on shutdown or build failure.
- Trace integration is opt-in through `Tracer.StartSpan`; the Sandwich pipeline wraps the entire run in a span when available.

---

## Testing & Tooling ⚠️
- Unit tests exist for the sandwich orchestrator (`pipeline/sandwich_test.go`), declarative orchestrator (`pipeline/declarative/orchestrator_test.go`), BM25, dense, hybrid retrievers, cosine reranker, and various rule-engine behaviours.
- `builder_test.go` registers mock providers and option types to exercise YAML/env configuration paths.
- `Makefile` defines `fmt`, `lint`, `test`, `build`, and `run` targets, though the build/run targets point to a non-existent `./cmd/agent`.
- `go.mod` pins Firebase Genkit, Google Mangle, OpenAI-go, and supporting libraries; `go.sum` tracks module hashes.

---

## Known Gaps (machine-readable)
| Severity | Component | File | Description |
| --- | --- | --- | --- |
| Critical | Provider options registry | builder.go, registry.go | Production providers never call `RegisterOptions`, so `optionsTypeToName` remains empty and builder lookups fail for real option structs. |
| Critical | OpenAI client wiring | builder.go | `buildSingleTool` asserts `openai.Client` by value, but `resolveProviderConfig` stores `*openai.Client`, causing panic when instantiating OpenAI/Groq tools. |
| High | Google embedder aliasing | builder.go, internal/embedders/google/google.go | Builder expects provider `"google"` while the embedder registers `"google-embedder"`, preventing automatic Google embedder construction. |
| High | CLI build targets | Makefile | `make build` and `make run` reference missing `./cmd/agent`, so default workflows fail. |
| High | Empty provider stubs | llm/google.go, llm/openai.go | Root-level files are empty, indicating incomplete or dead code paths that confuse navigation and coverage. |
| Medium | Logging consistency | pipeline/sandwich.go, pipeline/declarative/orchestrator.go, internal/providers/mangle/rules.go | Direct `fmt.Printf` calls bypass the optional logger, leading to noisy stdout in production. |
| Medium | Context propagation | internal/vectorstores/localvec/localvec.go | `Search` requires `ctx.Value("query_text")` with a plain string key, creating a fragile dependency between retrievers and vector stores. |
| Medium | Fallback behaviour | pipeline/sandwich.go | `FallbackThreshold` is enforced only when a reranker is configured, so non-reranked pipelines always proceed to the LLM. |
| Medium | Provider family coverage | builder.go | `resolveProviderConfig` handles only `"google"`, `"openai"`, and `"groq"`, limiting extension to other provider families or aliases. |
| Low | Heuristic constants | internal/providers/hybrid/hybrid.go, pipeline/sandwich.go | Values like the RRF constant (`k=60`) are hard-coded without configuration hooks. |
| Low | In-memory retriever logging | internal/providers/retrievers/inmemory/inmemory.go | `Upsert` and `Replace` log counts directly with `fmt.Printf`, with no way to silence in quiet environments. |

---

## Known Gaps (detailed)
- **Critical**: No production provider calls `RegisterOptions`, leaving `optionsTypeToName` empty. Any `With*` call with real option structs (or YAML/env load) yields “unregistered options type” errors (`builder.go`, `registry.go`).
- **Critical**: `buildSingleTool` asserts `openai.Client` (value) while `resolveProviderConfig` stores `*openai.Client`, causing a panic when wiring OpenAI/Groq tools (`builder.go`).
- **High**: Embedder names are inconsistent—`internal/embedders/google` registers `"google-embedder"`, but the builder expects `"google"` and resolves provider config only for `"google"`, so Google embedders cannot be instantiated without manual overrides (`builder.go`, `internal/embedders/google/google.go`).
- **High**: Makefile targets `./cmd/agent` for build/run even though no such package exists, so `make build`/`make run` fail out of the box (`Makefile`).
- **High**: Root-level `llm/google.go` and `llm/openai.go` files are empty placeholders, signalling incomplete or dead code paths that can confuse readers and coverage tools (`llm/google.go`, `llm/openai.go`).
- **Medium**: Several components print directly to stdout (`pipeline/sandwich.go`, `pipeline/declarative/orchestrator.go`, `internal/providers/mangle/rules.go`, `internal/providers/retrievers/inmemory/inmemory.go`), bypassing the optional logger and cluttering logs in production.
- **Medium**: `localvec.Search` depends on `ctx.Value("query_text")` with a raw string key, creating a fragile contract between retrievers and vector stores (`internal/vectorstores/localvec/localvec.go`).
- **Medium**: `FallbackThreshold` is only enforced when a reranker is configured; Sandwich pipelines without rerankers always call the LLM even when confidence should trigger a fallback (`pipeline/sandwich.go`).
- **Medium**: `resolveProviderConfig` handles only `"google"`, `"openai"`, and `"groq"`; additional provider families (or hyphenated names) cannot pick up shared configuration without extending the switch (`builder.go`).
- **Low**: Reciprocal Rank Fusion constant `k=60` and other heuristic values are hard-coded with no configuration hooks (`internal/providers/hybrid/hybrid.go`, `pipeline/sandwich.go`).
- **Low**: In-memory retriever `Upsert`/`Replace` log counts via `fmt.Printf` with no way to disable them (`internal/providers/retrievers/inmemory/inmemory.go`).

---

This file is designed to help coding agents understand Manglekit’s architecture, dependencies, and current implementation state.
