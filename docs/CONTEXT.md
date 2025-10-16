---
context_type: codebase_overview
project: manglekit
language: go
version: 2025.10
last_updated: 2025-10-15
---

# Manglekit Project Context

## Overview
Manglekit is a Go 1.24+ toolkit that combines Google’s Mangle Datalog engine with retrieval, reranking, and LLM calls. The default “Sandwich” pipeline runs **Mangle-Pre → Retrieval → optional Rerank → LLM → Mangle-Post**; an alternate declarative orchestrator drives stage ordering from Mangle facts. Provider registration is now handled explicitly in the application's entry point, using a fluent Registration Builder, which creates a `Registry` instance that is injected into the system.

---

## Core Building Blocks
- ⚠️ **Registry (`registry.go`)**: The `Registry` is now an **instance** created in the application entry point. It stores strongly-typed factory functions (e.g., `LLMFactory`, `RetrieverFactory`) for most components. However, the encapsulation is incomplete; the registry modifies global state in `typemap.go` and still uses `any` for client factories.
- ✅ **Providers (`providers/`)**: This new package provides a fluent "Registration Builder" (`providers.NewSet()`) for a streamlined developer experience. Users can chain `With...` methods (e.g., `NewSet().WithOpenAI().WithBM25()`) to explicitly register the desired provider implementations with the `Registry` instance.
- ⚠️ **Type Mapping (`typemap.go`)**: Manages the bidirectional mapping between provider names and their option types using `RegisterOptions`. Critically, this mapping is stored in **global variables**, creating shared state between otherwise isolated `Registry` instances.
- ✅ **Core types & contracts (`core/types.go`, `core/rules.go`)**: `Query`, `Answer`, `Doc`, and `Citation` model pipeline payloads; `Options` now carries the configured `StateProvider`, observability hooks, and a LIFO stack of `ResourceClosers`. Sentinel errors (`ErrInvalidOptions`, `ErrNoEvidence`, `ErrDenied`) and rule interfaces (`RuleSet`, `FlowController`, `PostRuleEvaluator`) gate Sandwich and declarative orchestrators.
- ✅ **Observability contracts**: `Logger` exposes `{Debug,Info,Warn,Error}f` plus `With` for scoped fields, `Tracer.StartSpan` supplies optional spans, and `Meter.Record` captures latency metrics with arbitrary attributes.
- ✅ **SDK entry point (`sdk.go`)**: `New` enforces that a retriever and LLM are present, defaults `TopK` to 8 and `MaxTokens` to 512, asserts concrete interfaces, and hands control to `pipeline.NewSandwich`.
- ✅ **State management (`core/state.go`, `state/types.go`)**: `StateProvider` defines `Get`/`Set`/`Delete`/`Close`; bundled options cover in-memory and Redis-backed implementations with JSON byte payloads.

---

## Orchestrator Construction
- ⚠️ **Fluent builder (`builder.go`)**
  - The builder is instantiated with a `Registry` instance (`builder.New(registry)`), making it decoupled from the provider registration process.
  - **Improved Type-Safety**: The builder now looks up **strongly-typed factories** (e.g., `RetrieverFactory`) from its `Registry`. This is a significant improvement but is not consistently applied (e.g., for client factories).
  - **Provider-Led Construction**: Complex construction logic has been moved into provider factories. However, this has introduced new problems, such as composite components requiring a callback to the full `BuilderAPI`, creating a leaky abstraction.
  - **Implicit Dependency Guessing**: The builder contains a fragile `resolveDependencies` method that attempts to guess component needs (e.g., inferring an embedder) instead of requiring explicit configuration.
- ⚠️ **Configuration (`config.go`)**
  - `Config` describes orchestrator selection, linear component slots, declarative `tools`, and provider-family defaults (`google`, `openai`, `groq`, `openaiCompatible`, `mangle`).
  - `NewBuilderFromYAML` expands environment variables, resolves every `path:"resolve"` field relative to the file, hydrates typed option structs via JSON marshal/unmarshal, and chains the relevant `With*` calls—including state providers. The optional `logging` block only swaps in the stdlib-backed logger; level/format values are parsed but not yet applied to zap.
  - `NewBuilderFromEnv` mirrors the YAML flow with `MKT_*` variables (`MKT_LLM_NAME`, `MKT_LLM_PARAMS`, etc.), expecting JSON parameter blobs. Paths inside those blobs are resolved relative to `cwd`.

---

## Execution Pipelines
- ⚠️ **Sandwich orchestrator (`pipeline/sandwich.go`)**
  - `Execute(ctx, sessionID, q)` decorates the logger with `request_id`, `pipeline`, and `session_id`. When a `StateProvider` is present it attempts to hydrate a `core.ConversationHistory` blob (stored as JSON bytes) before execution and persists the augmented history after a successful LLM call.
  - **Pre rules**: Calls `Rules.Evaluate(core.Pre, q, nil)`, records `manglekit.rules_pre_ms`, applies `RuleResult.Mutate`, and raises `ErrDenied` on disallowed requests.
  - **Retrieval**: Invokes `retrieve.Retriever.Retrieve` with `TopK` from options, records `retrieve_ms`, and caches `original_docs` plus retriever metadata in `Answer.Meta`. Empty results short-circuit with `ErrNoEvidence`.
  - **Rerank & fallback**: When configured, reranking captures `rerank_ms`, rebuilds citations from `rerank.ScoredDoc`, stores `best_score`, and applies `FallbackThreshold`, returning `ErrNoEvidence` if the score falls below the configured floor.
  - **LLM**: `prepareLlmRequest` flattens docs to passages and rewrites citations. `runLlm` merges `Query.Meta` (including conversation `history`) into the prompt data, forwards `MaxTokens`, captures `llm_ms`, and records token usage.
  - **Post rules**: Replays `Rules.Evaluate(core.Post, q, &answer)`, records `manglekit.rules_post_ms`, applies answer mutations, and bubbles denials.
- ❌ **Declarative orchestrator (`pipeline/declarative/orchestrator.go`)**
  - **NOTE: This feature is currently disabled due to the recent refactoring.** The build path is stubbed out and will return an error if invoked.

---

## Rules Engine (`internal/providers/mangle`) ⚠️
- **Initialization (`New`)**: Parses optional schemas through registered schema parsers (e.g., `jsonschema`, `rdf`), instantiates default and custom converters, decides between code-first vs file-first predicate declarations, loads `.dlog` / `.facts` files, stratifies the program, seeds a base fact store, and performs an initial evaluation.
- **Converters**: Built-ins include `QueryConverter` (tokens, versions, normalized text), `UserContextConverter` (metadata to `user_attribute/2` facts with optional IRI handling), and `DocumentConverter` (document IDs, text, metadata fan-out).
- **Evaluate (pre/post)**: Clones the base store, injects converter-derived facts, re-evaluates the program, and returns `RuleResult` with `Allowed`, `Reason`, `Mutate` (injecting `filters` and `expansion_terms`), and optional `SkippedStages`.
- **Post hook (`Post`)**: Accepts the query, evidence, and execution metadata; adds `retrieved_doc` facts; evaluates drop/redact/deny rules; applies regex-based redactions; filters documents; records applied rule metadata; and returns `core.PostRuleResult`.
- **Query**: Implements read-only fact queries by running the engine and manually unifying results into Go maps.

---

## Providers & Components
- ⚠️ **Retrieval**
  - ✅ `internal/providers/bm25`: Walks Markdown files (parsing YAML front matter), builds a TF-IDF vocabulary, applies Okapi BM25 scoring with a default `TopK=10`, and mirrors scores in document metadata.
  - ⚠️ `internal/providers/dense`: Couples an `ai.Embedder` with a `core.VectorStore`, embeds the query, forwards `req.Meta["filters"]` to the store’s `Search`, and requires both dependencies to be injected by the builder.
  - ⚠️ `internal/providers/hybrid`: Runs sparse and dense retrievers concurrently via `errgroup`, fuses results with Reciprocal Rank Fusion (`k=60`), and trims the fused list to the request’s `TopK`.
  - ⚠️ `internal/providers/retrievers/inmemory`: Mutex-protected map that ignores the query string and returns all stored docs (capped by `TopK`), while supporting `Upsert`/`Replace` for live updates.
- ⚠️ **Vector store**
  - ⚠️ `internal/vectorstores/localvec`: Boots Genkit LocalVec with a managed context, eagerly indexes Markdown files through `localvec.Index`, filters matches via stringified metadata equality, and provides `Close` to cancel the background Genkit context.
- ⚠️ **Rerank**
  - ⚠️ `internal/providers/rerank/cosine`: Shares the configured embedder across query/doc embeddings (concurrently via `errgroup`), computes cosine similarity with a custom float32 `sqrt`, sorts results descending, and respects either the options or per-request `TopK`.
- ⚠️ **Embedders**
  - ⚠️ `internal/embedders/openai`: Registers as `"openai-embedder"` and `"groq-embedder"`, builds embeddings through the OpenAI API with optional `Dimensions`, casts float64 responses to float32, and exposes a no-op `Close`.
  - ⚠️ `internal/embedders/google`: Registers `"google-embedder"`, wraps Genkit’s GoogleAI embedder (default `"embedding-001"`); the builder normalises this alias back to the `"google"` client family.
- ⚠️ **LLM clients**
  - ⚠️ `internal/providers/llm/google`: Uses Genkit `Model.Generate`, assembles prompts with `llm.PromptBuilder`, concatenates text parts, and returns usage counters keyed `"prompt"`/`"completion"`.
  - ⚠️ `internal/providers/llm/openai`: Targets OpenAI-compatible APIs (OpenAI/Groq), builds prompts via `PromptBuilder`, calls Chat Completions through `openai-go`, and reports a resource closer that shuts down idle HTTP transports.
- ✅ **Schema parsers**
  - ✅ `internal/providers/schemaparsers/jsonschema`: Converts JSON Schema definitions into fact predicates consumable by the rules engine.
  - ✅ `internal/providers/schemaparsers/rdf`: Loads Turtle RDF triples into `triple/3` facts with basic IRI handling.
- ✅ **State providers**
  - ✅ `internal/providers/state/inmemory`: Thread-safe map implementation of `core.StateProvider`, returning raw values and ignoring `Close`.
  - ✅ `internal/providers/state/redis`: Wraps `redis/v9`, expects state payloads as `[]byte`, and returns a close hook to release the client.
- ⚠️ **Mocks (`internal/providers/mock`)**: Supplies mock retriever, reranker, LLM, and tool implementations plus adapters for Datalog constants—used extensively in tests.
- ⚠️ **Catch-all import (`providers/all/all.go`)**: Blank-import convenience that registers every bundled provider family.

---

## Prompting (`llm/prompt.go`) ⚠️
- `PromptBuilder` caches compiled `text/template` instances, guarded by a RW mutex, and injects helper functions (`toJSON`, `join`, `truncate`).
- The default RAG template instructs the model to ground answers strictly in the provided context and to refuse when evidence is missing, but it expects each entry in `.documents` to expose a `.Text` field—`Sandwich.prepareLlmRequest` currently passes a `[]string`, so the default template fails unless callers override it.

---

## Applications & Samples
- ✅ **`apps/rdf-knowledge-base`**: CLI demo that loads RDF data via the Mangle rules engine, normalises user input into query metadata, runs `RuleSet.Evaluate(core.Pre, ...)`, and prints sheet associations plus canonical keyword expansions. Uses `godotenv` to source environment variables.

---

## Observability & Logging ✅
- `NewBuilder` always installs `internal/logger.StdLogger` so pipeline code never checks for nil. YAML `logging` blocks and `MKT_LOG_LEVEL` / `MKT_LOG_FORMAT` env vars currently just trigger the same fallback; level/format values are parsed but not yet applied to zap.
- `internal/logger.ZapAdapter` can wrap an injected `zap.SugaredLogger`, while the default `StdLogger` emits key/value pairs to stdout with shared fields inherited via `Logger.With`.
- Sandwich and declarative orchestrators attach `request_id`, `pipeline`, and flow/session metadata to scoped loggers; component failures and stage transitions log through the same interface.
- Metrics flow through `Meter.Record` with names such as `manglekit.rules_pre_ms`, `manglekit.retrieve_ms`, `manglekit.rerank_ms`, `manglekit.llm_ms`, and `manglekit.rules_post_ms`.
- Builders aggregate `ResourceClosers` from client factories and components and unwind them LIFO on orchestrator shutdown or build failure; `Tracer.StartSpan("manglekit.Execute")` wraps Sandwich runs when provided.

---

## Testing & Tooling ⚠️
- Unit tests currently cover the builder (`builder_test.go`), the sandwich and declarative orchestrators, the in-memory retriever, and both state providers. BM25, dense, hybrid, cosine, and rules providers remain untested.
- `builder_test.go` registers mock providers and option types to exercise YAML/env configuration paths.
- `Makefile` defines `fmt`, `lint`, `test`, `build`, and `run` targets, though the build/run targets point to a non-existent `./cmd/agent`.
- `go.mod` pins Firebase Genkit, Google Mangle, OpenAI-go, and supporting libraries; `go.sum` tracks module hashes.

---

## Known Gaps (machine-readable)
| Severity | Issue | File | Description |
| --- | --- | --- | --- |
| High | **Incomplete State Encapsulation** | typemap.go | The registry modifies global variables in `typemap.go`, preventing isolated instances of the framework. |
| High | **Leaky Builder Abstraction** | internal/providers/hybrid/hybrid.go | Composite components depend on the entire `BuilderAPI` instead of just the factories they need, creating tight coupling. |
| High | **Fragile Dependency Guessing** | builder.go | The builder's `resolveDependencies` method uses brittle, implicit logic to guess component needs instead of requiring explicit configuration. |
| High | CLI build targets | Makefile | `make build` and `make run` reference missing `./cmd/agent`, so default workflows fail. |
| High | Prompt template mismatch | llm/prompt.go, pipeline/sandwich.go | The default RAG template expects `.documents[*].Text`, but Sandwich passes a `[]string`, so default prompting panics unless callers override it. |
| High | Empty provider stubs | llm/google.go, llm/openai.go | Root-level files are empty, indicating incomplete or dead code paths that confuse navigation and coverage. |
| Medium | Logging config unused | config.go | YAML/env logging fields flip the logger on but never apply level/format or wire zap, leaving configuration knobs ineffective. |
| Medium | Fallback behaviour | pipeline/sandwich.go | `FallbackThreshold` is enforced only when a reranker is configured, so non-reranked pipelines always proceed to the LLM. |
| Medium | MaxTokens ignored | internal/providers/llm/openai.go, internal/providers/llm/google.go | LLM clients discard `req.MaxTokens`, so orchestrator defaults cannot constrain response length. |
| Medium | OCP Violation in Builder | builder.go | The main `Build` method still contains a `switch` statement for orchestrator types. |
| Low | Declarative state | pipeline/declarative/orchestrator.go | The declarative orchestrator is disabled, but if re-enabled, it does not use its configured state provider. |
| Low | Heuristic constants | internal/providers/hybrid/hybrid.go | Reciprocal Rank Fusion uses a hard-coded `k=60` without configuration hooks. |
| Low | Deprecated Code | core/types.go | `LocalvecOptions` is marked as deprecated but has not been removed from the core API. |


For detailed analysis and refactoring suggestions, please refer to the full [Code Review Document](./code-review.md).