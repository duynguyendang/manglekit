# ADR — Consolidated Architecture Decisions for Manglekit Core

**Status:** Accepted
**Scope:** Core SDK (registry, builder, orchestrators, providers, config bridge)
**Period:** Oct 2025
**Audience:** Core maintainers, provider/orchestrator authors, contributors

---

## 1) Architecture becomes **Config-First & Declarative**

### Context

Early versions mixed runtime construction with ad-hoc configuration, making examples brittle and pipelines hard to reproduce/migrate.

### Decision

Adopt a **config-first** stance: YAML/ENV become first-class inputs. Introduce a thin **config→builder bridge** (`from_config`), which validates/normalizes config and then calls the builder with typed options.

### Rationale

* Reproducibility (pipelines can be checked in, diffed, and promoted).
* Cleaner separation of concerns: parsing/validation vs. construction.

### Consequences

* Config parsing and path resolution are decoupled from the builder.
* Examples and docs center on declarative usage.
* A minimal `DecodeOptions` utility maps config maps → typed options.

### Migration

* Move any parsing from the builder into the config bridge.
* Keep the builder free of YAML/ENV specifics.

---

## 2) **Observability & Lifecycle** are First-Class

### Context

Resource leaks and inconsistent logging made debugging and operations difficult.

### Decision

Unify logging/observability contracts and implement **graceful shutdown** for local/indexed stores and API clients. Resource closers are tracked centrally by the builder.

### Rationale

* Predictable, uniform logs across modules.
* Safe teardown for tests, CLIs, and long-running services.

### Consequences

* Standard “closer” pattern and structured logging defaults.
* Builders register closers when components are created.

### Migration

* Ensure every provider implements (or clearly opts out of) closer semantics.
* Replace ad-hoc loggers with the unified interface.

---

## 3) **Context Propagation** throughout the SDK

### Context

Some APIs ignored `context.Context`, blocking cancellation, deadlines, and tracing.

### Decision

Make `context.Context` explicit and mandatory across all factories and runtime calls.

### Rationale

* Enables cancellation, deadlines, and tracing.
* Aligns with modern Go service patterns.

### Consequences

* Touches many signatures; simplifies test control and instrumentation.
* More predictable behavior under load and during shutdown.

### Migration

* Thread `ctx` through all call chains (providers, orchestrators, stores).

---

## 4) The Core Shift: **Generic, Type-Safe Registry & Builder**

### Context

The original builder/registry had per-type maps, per-type `With…` methods, and `build…` functions with duplicated wiring; orchestrators consumed an `any` options blob, causing type erasure and runtime casts.

### Decision (4 parts)

**4.1 Generic Registry**

* Replace per-type registries with **one generic registry** of typed factories.
* Factory signature is standardized: `func(ctx, deps, cfg) (T, error)`.
* Registration uses generics; the registry holds a catalog keyed by **Kind** (llm, retriever, vector_store, …).

**4.2 Provider Self-Identification**

* Each provider’s **Options** type implements:

  ```go
  type ProviderOptions interface {
      ProviderName() string   // e.g., "openai-chat"
      ProviderKind() Kind     // e.g., KindLLM
  }
  ```
* Registration takes an **options sample** instead of a string literal:
  `Register(registry, OptionsSample{}, Factory)`.
  (No magic strings; name/kind come from the type.)

**4.3 Generic Builder (DRY) with a Spec Table**

* Collapse all per-type `With…`/`build…` into a single `With(opts any)` and a **data-driven build loop**.
* A **spec table** defines dependency order and assignment (e.g., embedder → vector store → retriever → reranker → rules → LLM → state provider).
* Hybrid retrievers are supported via a typed `BuildRetriever` hook (using the registry).

**4.4 Typed Orchestrator Inputs**

* Replace `core.Options any` with a **typed `Resolved` struct** (Retriever, VectorStore, Reranker, Rules, LLM, Embedder, StateProvider, Observability, etc.).
* Orchestrator factories now consume `Resolved` directly—no runtime casts.

### Rationale

* Eliminates duplication, enforces Open/Closed Principle, and maximizes compile-time guarantees.
* Streamlines developer experience (DX): add a provider by registering one typed factory—no edits to core builder/registry.

### Consequences

* **Massive** reduction in builder boilerplate.
* Type-safety end-to-end (providers, builder, orchestrators).
* Clean plugin surface for third parties.
* Slight reflection for options type lookups (acceptable, limited to wiring).

### Migration

* Update all providers to implement `ProviderOptions` on their Options type and to accept **typed** `cfg` in factories.
* Register with `Register(registry, OptionsSample{}, Factory)`.
* Remove per-type `With…` and `build…` from the builder; use `With(opts)` and spec-driven `buildAll`.
* Update orchestrators to accept `Resolved`.

---

## 5) **Orchestrator Modernization** (Stage-Based, Typed)

### Context

The “Sandwich” orchestrator started monolithic, with `any`-based accessors and inlined logic.

### Decision

Refactor into a **stage-based pipeline** consuming typed `Resolved` dependencies (retrieval, rerank, rules, LLM calls, state), removing casts and allowing easier unit tests and targeted changes.

### Rationale

* Composability and testability.
* Strict interface boundaries with typed inputs/outputs.

### Consequences

* Clear separation of retrieval/rerank/generation/state logic.
* Easier to add new stages or swap implementations.

### Migration

* Move stage logic into small, focused units; wire via the orchestrator using `Resolved`.

---

## 6) **Testing & DX Uplift**

### Context

Brittle tests, many fixtures bound to legacy APIs, and gaps in coverage.

### Decision

Increase test coverage across config, builder, providers, and orchestrators; remove obsolete tests; adopt typed factories and `Resolved` in tests; unify logging to reduce noise.

### Rationale

* Confidence in refactors and faster iteration.
* Easier contributor onboarding.

### Consequences

* Smaller, more targeted tests.
* Fewer integration-only test paths.

### Migration

* Replace legacy tests that assert `any` contents with typed expectations.
* Prefer table-driven tests with typed options/factories.

---

## Resulting System (Snapshot)

* **One Registry** (generic, typed) for all components.
* **One Builder** (generic, spec-driven) that constructs pipelines in dependency order.
* **Providers self-identify** via their Options type; no string literals when registering.
* **Orchestrators** receive **typed dependencies** (`Resolved`) and run **stage-based** flows.
* **Config bridge** cleanly translates YAML/ENV → typed Options → builder.
* **Observability & lifecycle** standardized; **context propagation** is universal.
* **Testing & DX** significantly improved.

---

## Known Trade-offs & Risks

* Requires Go generics and minimal reflection for type→name/kind mapping.
* Breaking API (per-type `With…`/`build…` removed).
* Orchestrator factories must be migrated to typed `Resolved`.

---

## What’s Next (Future ADRs)

* Build-graph introspection & visualization.
* Pluggable runtime (Go plugin/WASM) & sandboxing.
* Options schema validation + JSON-Schema export.
* Warm-up/caching policies for heavy clients.
* Performance budgets & benchmarks for stage pipelines.

---
