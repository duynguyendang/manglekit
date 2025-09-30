# Refactoring Opportunities

This document captures potential refactoring targets discovered during the current review. The intent is to guide future
iterations that make the orchestration pipeline easier to extend and reason about.

## 1. `internal/orchestrator/orchestrator.go`

* **Encapsulate flow phases.** `RunFlow` currently orchestrates intent parsing, preprocessing, retrieval, reranking,
  post-processing, fallback selection, and LLM generation in a single 100+ line function. Breaking the method into clearly
  named helpers (for example `parseIntent`, `expandQuery`, `retrieveAndRerank`, `generateAnswer`) would reduce cyclomatic
  complexity and make it easier to unit test each phase independently.【F:internal/orchestrator/orchestrator.go†L54-L133】
* **Guard against missing dependencies.** The struct allows `retriever` and `reranker` to be nil, but `RunFlow` assumes they
  are always present. Adding constructor checks (similar to the existing Genkit/LLM validation) or defensive errors before
  use would prevent panics when wiring the orchestrator in different environments.【F:internal/orchestrator/orchestrator.go†L33-L94】【F:internal/orchestrator/orchestrator.go†L104-L118】
* **Clarify metadata handling.** `addExplanations` type-asserts `response.Metadata` to a `map[string]any`, but the field is
  declared as `interface{}`. Consider normalising this field (e.g. by introducing a dedicated `ResponseMetadata` struct) so
  callers do not need type assertions and to guarantee thread-safety when the same response is shared across goroutines.【F:internal/orchestrator/orchestrator.go†L137-L170】【F:internal/types/types.go†L45-L72】
* **Capture reusable fallback config.** The fallback answer is hard-coded inside `RunFlow`. Extracting it into the config (or a
  dedicated helper) would let operators adjust fallback copy and metadata without touching code. It also removes the need to
  duplicate the literal in tests.【F:internal/orchestrator/orchestrator.go†L118-L133】

## 2. `internal/llm/gateway.go`

* **Avoid repeated template parsing.** `buildPrompt` recomputes `countWords(fmt.Sprintf(template, "", userPrompt))` for every
  invocation even when the template and prompt are unchanged. Caching the base token cost or precomputing the formatted
  header once per request would reduce allocations and simplify testing of token budgets.【F:internal/llm/gateway.go†L58-L109】
* **Model initialisation is tightly coupled to providers.** `New` switches on the provider string and mutates Genkit state when
  Ollama is selected. Consider extracting provider-specific builders behind a `ProviderFactory` interface so you can add
  vendors (e.g., Vertex AI) without touching `gateway.go`. This also makes it easier to inject mock models in tests.【F:internal/llm/gateway.go†L24-L73】
* **Stream handling is lossy.** The `Generate` callback only inspects `chunk.Content[0]`, ignoring multi-part content and role
  metadata. Refactoring to iterate over the slice (and respecting structured parts) would make the gateway resilient to
  richer Genkit responses.【F:internal/llm/gateway.go†L75-L94】

## 3. `internal/retrieval/hybrid.go`

* **Deduplication order is non-deterministic.** The current merge preserves the order of the concatenated slices, which means
  BM25 results always outrank dense ones even when scores suggest otherwise. Introducing a scoring structure (e.g. a
  `Chunk` wrapper with source metadata) would let you normalise scores and perform a consistent merge strategy.【F:internal/retrieval/hybrid.go†L33-L69】
* **Result container differs from package types.** `HybridRetriever.Retrieve` returns `[]string`, while the rest of the system
  operates on `*types.Chunk`. Aligning the return type eliminates the need to reconstruct chunks later and keeps metadata and
  scores intact through the pipeline.【F:internal/retrieval/hybrid.go†L11-L69】【F:internal/types/types.go†L25-L44】

## 4. Cross-cutting ideas

* **Surface tracing hooks.** Each major component logs progress, but tracing IDs and timings are not propagated. Introducing a
  lightweight trace struct (or leveraging `context.Context` values) would provide observability without coupling to Zap
  directly.
* **Strengthen configuration validation.** Several config structs are passed through without validation (e.g. negative `topK`
  values). Centralising validation in a `Validate()` method per config type would fail fast during startup and reduce the
  number of defensive checks needed deeper in the pipeline.【F:internal/types/types.go†L61-L88】

These changes aim to make the sandwich orchestration easier to extend while preserving its current behaviour.
