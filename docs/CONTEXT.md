# Manglekit Project Context

_Last updated: 2025-05-30_

---

## Overview
Manglekit is a Go 1.24 SDK for policy-aware retrieval-augmented generation. It wraps Google’s Mangle rules engine around Genkit-powered retrieval, reranking, and LLM stages so every answer runs through the Sandwich sequence **(Mangle-Pre → Retrieval → optional Rerank → LLM → Mangle-Post)**. The library targets explainable copilots that must cite sources, enforce policy filters, and remain configurable at runtime.

---

## Implementation Snapshot
- SDK entry points live in `builder.go`/`config.go` (fluent `With*`, YAML/env loaders) and `sdk.go` (Sandwich constructor guarded by `core.ErrInvalidOptions`).
- Domain contracts in `core/` cover `Doc`, `Query`, `Answer`, `Citation`, options, rule interfaces (`RuleSet`, `FlowController`, `PostRuleEvaluator`), and `Observability` hooks (logger/tracer/meter).
- Pipelines include `pipeline/sandwich.go` (pre rules, retrieval, optional rerank, fallback threshold, LLM call, post rules, metrics, LIFO `ResourceClosers`) and `pipeline/declarative/orchestrator.go` (Datalog-defined flows with shared context map, stage skipping, and denial propagation).
- Providers under `internal/` back the standard stack: BM25/dense/hybrid/in-memory retrievers, cosine reranker, Google/OpenAI embedders, localvec vector store, Google/OpenAI/Groq LLM clients, Mangle rules engine, and schema parsers (JSON Schema, RDF); `providers/all` registers the bundle.
- Rule converters (`internal/providers/mangle/converters`) turn queries, documents, and user context into facts; schema loaders (`internal/providers/schemaparsers`) seed program facts; `core.MangleOptions` toggles file-first vs code-first behaviour.
- Examples (`examples/01-basic-rag` … `examples/09-genkit-tool`) and the `apps/rdf-knowledge-base` CLI demonstrate sandwich, declarative, and rules-only usage; they rely on `.env` keys and provider registration via blank imports.
- Tests cover the Sandwich orchestrator (`pipeline/sandwich_test.go`) and all core providers (BM25, dense, hybrid, cosine reranker, Google embedder, Mangle rules); builder/declarative end-to-end tests remain on the backlog.

---

## HLD Feature Status
**HLD 1.2 Architectural Principles**
- Done — SDK-first, service-optional: the repository centres on the Go library with optional thin CLIs (`apps/rdf-knowledge-base`).
- Done — Sandwich by default: `sdk.New` and `builder.Build` default to `pipeline/sandwich` guarded by Mangle stages.
- Done — Provider-agnostic extensibility: `registry.go` plus `providers/all` let callers swap constructors without touching pipelines.
- Done — Fail fast construction: `builder.go` resolves configs, instantiates external clients, and aborts `Build()` on misconfiguration.
- Done — Stateless pipelines & explicit resources: orchestrators keep no mutable state and rely on `core.ResourceCloser` slices.
- Done — Declarative hooks everywhere: `pipeline/declarative` + the Mangle `FlowController` expose Datalog-driven orchestration.

**HLD 3.1 Builder, Configuration, and Registry**
- Done — NewBuilder `With*` typed options: `builder.go` records provider names plus typed configs for every component.
- Done — `NewBuilderFromYAML`: `config.go` expands env vars, resolves `path:"resolve"` fields, and seeds the builder state.
- Done — Provider configs instantiate clients up front: `resolveProviderConfig` wires Google/OpenAI/Groq clients with closers.
- Done — Providers register via `Register*` with enforced signatures; builders type-assert before invocation.
- Done — Dependency ordering is automatic: `buildComponents` constructs embedder → vector store → retrievers → reranker → rules → LLM.
- Done — `Build()` emits populated `core.Options` (TopK, MaxTokens, FallbackThreshold, Observability hooks).

**HLD 3.2 Standard Provider Set**
- Done — Retrievers (in-memory, BM25, dense, hybrid) are implemented and registered under `internal/providers/...`.
- Done — Cosine reranker backed by the shared embedder lives in `internal/providers/rerank/cosine`.
- Done — Google and OpenAI embedders reside in `internal/embedders/{google,openai}` with typed registration.
- Done — Localvec vector store (`internal/vectorstores/localvec`) injects the embedder and indexes corpora on disk.
- Done — LLM clients for Google, OpenAI, and Groq wrap Genkit/openai-go in `internal/providers/llm`.
- Done — Mangle rules engine provides `RuleSet` and `FlowController` via `internal/providers/mangle`.
- Done — Schema parsers for JSON Schema and RDF are available under `internal/providers/schemaparsers`.
- Done — Custom providers can register alternative constructors through the public `Register*` helpers.

**HLD 3.3 Orchestrators**
- Done — Sandwich orchestrator enforces Pre → Retrieve → optional Rerank → Fallback → LLM → Post, records metrics, and closes resources in LIFO order (`pipeline/sandwich.go`).
- Done — Declarative orchestrator queries `flow_stage`/`stage_tool` facts, shares execution context, honours stage skips, and dispatches named tools (`pipeline/declarative`).

**HLD 3.4 Rules Engine & Fact Management**
- Done — `core.MangleOptions` captures rule paths, schema sources, converters, and the FileFirst toggle.
- Done — Default and custom converters load via `internal/providers/mangle/converters`.
- Done — Schema sources turn into facts and declarations through `parseSchemas` and registered parsers.
- Done — Rules are stratified once via Mangle analysis and each evaluation clones the base fact store before executing.
- Done — `ruleSet` implements `FlowController`, powering both rule stages and declarative flow queries.

**HLD 3.5 Retrieval, Embedding, Ranking, and Generation**
- Done — BM25 retriever indexes Markdown with front matter metadata and surfaces scores (`internal/providers/bm25`).
- Done — Dense retriever embeds queries, applies metadata filters, and delegates to `core.VectorStore`.
- Done — Hybrid retriever executes sparse and dense searches in parallel and fuses via Reciprocal Rank Fusion.
- Done — Localvec vector store persists embeddings and honours metadata filters during search.
- Done — Cosine reranker embeds query/docs concurrently and emits scored citations.
- Done — LLM clients use `llm.PromptBuilder`, call provider SDKs, and expose token usage in `llm.Response`.
- Done — Embedders are reused across retrievers and rerankers through builder-managed dependency injection.

**HLD 3.6 Observability & Lifecycle**
- Done — `core.Observability` supplies logger/tracer/meter hooks consumed across the pipelines.
- Done — `ResourceClosers` are appended during client creation and invoked in LIFO order by `Orchestrator.Close`.
- Done — `Answer.Meta` captures retrieve_ms, llm_ms, best_score, rule results, and token usage for audits.

**HLD 4 Usage Patterns**
- Done — Sandwich SDK mode demonstrated in `examples/01-basic-rag` with `providers/all` and builder chaining.
- Done — Declarative flow mode exercised via `examples/04-declarative-flow` (`orchestrator.type: declarative`, Datalog flows).
- Done — Rules-first utilities illustrated by `apps/rdf-knowledge-base`, which instantiates the rules provider directly.

**HLD 5 Non-Functional Requirements**
- Done — Performance: retrievers/rerankers use `errgroup` for concurrency and pipelines remain stateless for low latency.
- Done — Scalability: orchestrators avoid mutable state; external stores and LLMs scale independently.
- Done — Security & compliance: Mangle post-rules drop/redact evidence and secrets flow from env-backed provider configs.
- Done — Observability: metrics (`manglekit.*`), structured logs, and token usage propagate through response metadata.

---

## Known Gaps and Risks
- Context propagation: dense retrieval, cosine rerank, and LLM providers use `context.Background()`, so caller cancellations/timeouts do not reach external APIs.
- Localvec lifecycle: `internal/vectorstores/localvec` calls `genkit.Init` per build without registering a `ResourceCloser`, leaving background goroutines alive after `Close`.
- Fallback semantics: when no reranker is configured, `best_score` stays 0 and any positive `FallbackThreshold` yields `core.ErrNoEvidence` instead of a deterministic fallback message.
- Declarative dependency inference: `getToolDependencies` treats every string parameter as a tool reference, risking build-order issues for configs containing literal strings.

---

## Roadmap
- Propagate contexts through dense retrieval, reranking, and LLM clients to support cancellation and deadlines.
- Register `ResourceClosers` (or reuse handles) for localvec/genkit resources to guarantee clean shutdowns.
- Refine fallback threshold handling so sandwich pipelines emit deterministic fallback text when rerankers are absent.
- Tighten declarative tool dependency parsing (e.g., explicit dependency keys) to avoid false-positive dependency cycles.
- Add end-to-end tests covering builder YAML/env flows and declarative orchestrator execution.
