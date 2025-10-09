# Manglekit Project Context

_Last updated: 2025-05-29_

---

## Overview
Manglekit delivers a Go 1.21 SDK for controlled retrieval-augmented generation. It couples Google’s Mangle rules engine with semantic retrieval and LLM orchestration using the Sandwich sequence: **Mangle‑Pre → Retrieval → (optional) Rerank → LLM → Mangle‑Post**. The SDK targets policy-compliant internal copilots where answers must remain explainable, cite sources, and respect tenant/privacy boundaries.

Key design principles (see `docs/LLD.md` for detail):
- Single-import ergonomics via `manglekit.New` or the fluent `Builder`.
- Providers self-register through `registry.go` with typed constructors.
- Pipelines are pluggable (Sandwich default, declarative orchestrator optional).
- Observability hooks (logger/tracer/meter) are optional but first-class.

---

## Current Implementation (code state)
- **Entrypoints**
  - `sdk.New(core.Options)` validates required components and instantiates the Sandwich orchestrator.
  - `builder.BuilderAPI` configures components fluently or from YAML (`config.NewBuilderFromYAML`), resolving dependencies (embedders, vector stores, API clients) and building orchestrators.
- **Core Types (`core/`)**
  - Contracts for `Doc`, `Query`, `Answer`, `Citation`, `Options`, and error sentinels (`ErrInvalidOptions`, `ErrNoEvidence`, `ErrDenied`).
  - Rule abstractions: `RuleSet`, `FlowController`, `RuleResult`, `FactConverter`, `SchemaParser`.
- **Pipelines**
  - `pipeline/sandwich.go` executes pre-rules, retrieval, optional rerank (`rerank.Reranker`), confidence fallback, LLM call, and post-rules. Observability metrics include `retrieve_ms`, `llm_ms`, and best score. Answer metadata retains `original_docs` for post-rule inspection.
  - `pipeline/declarative/orchestrator.go` interprets flow definitions stored as Mangle facts (`flow_stage/3`, `stage_tool/2`). Pre-rules can mutate the query and skip stages dynamically.
- **Providers (registered under `internal/…`)**
  - Retrieval: BM25 over markdown corpora, dense (embedder + vector store), hybrid (RRF fusion), in-memory (updatable map).
  - Vector store: Genkit LocalVec implementation (`internal/vectorstores/localvec`).
  - Reranker: Cosine similarity leveraging shared embedder vectors.
  - LLM: OpenAI/Groq (chat completions) and Google (Genkit `googlegenai`).
  - Rules: Mangle FlowController with default converters (query, user context, documents) and schema parser integrations (JSON Schema, RDF).
  - Embedders: Google GenAI and OpenAI/Groq embeddings.
  - All providers are registered by importing `github.com/duynguyendang/manglekit/providers/all`.
- **Prompt Handling**
  - `llm.PromptBuilder` caches Go templates and ships with `DefaultRAGTemplate`.
- **Configuration**
  - `config.Config` models orchestrator, components, provider families, and pipeline defaults (`TopK`, `MaxTokens`, `FallbackThreshold`).
  - `resolvePathsInStruct` resolves `path:"resolve"` tagged fields relative to the YAML file.
- **Examples / Tooling**
  - `examples/05-chat-with-data`: Sandwich pipeline driven by YAML config (`config.yaml`), Mangle rules (`rules/kb.facts`, `rules/retrieval.dlog`), and Google LLM.
  - `cmd/agent/main.go`: Demonstrates programmatic builder usage plus Genkit HTTP server (`/answer` flow). Requires `GOOGLE_API_KEY` (embedder/LLM) and `OPENAI_API_KEY` when targeting OpenAI models.
- **Testing**
  - Unit tests cover Sandwich orchestrator behaviour and provider contracts (BM25, dense, hybrid, cosine reranker, Mangle ruleset). Coverage for localvec, HTTP handlers, and Genkit flows is pending.

---

## Known Gaps and Risks
- **Serialization**: `core.Citation.Score` struct tag typo (`json_`) prevents scores from appearing in JSON payloads.
- **Embedders**: `internal/embedders/{google,openai}.Register` functions are placeholders that panic; Genkit registry integration is incomplete.
- **Context propagation**: Several providers invoke external APIs with `context.Background()`; callers cannot cancel long-running operations.
- **Fallback semantics**: Without a reranker, Sandwich fallback compares a zero best score against the threshold, triggering `ErrNoEvidence`.
- **LocalVec lifecycle**: `localvec.New` re-initializes Genkit/localvec for each build; no shared shutdown or reuse strategy.
- **Declarative tooling**: Dependency inference treats any string parameter as a potential tool reference, which may cause false positives.
- **Observability wiring**: Defaults to stdout logging when `Observability.Logger` is nil; `internal/logger` helper is not wired.
- **Missing features**: Ingestion pipeline, config-from-env helper, richer HTTP service (auth, validation, ingestion endpoint), advanced rule explanations/redaction, retry/backoff/caching strategies.

---

## Roadmap (proposed)
1. **Reliability**: Fix `Citation.Score` tag, adopt caller contexts/timeouts, wire structured logging via `internal/logger`.
2. **Provider robustness**: Complete embedder `Register` implementations; add graceful shutdown for Genkit/localvec resources.
3. **Productization**: Ship ingestion API, FromEnv config builder, HTTP middleware (auth, rate limiting, sanitization).
4. **Rules & Explainability**: Extend Mangle converters for richer explanations, persistent fact stores, PII redaction policies.
5. **Testing**: Add integration tests for localvec path, declarative orchestrator flows, and HTTP surfaces; measure performance with representative corpora.

---

Use this document as the ground truth snapshot of architecture and implementation status. Update it alongside significant code or design changes (and keep `docs/LLD.md` for deeper technical reference, `docs/HLD.md` for system view, `docs/CSD.md` for business alignment). 
