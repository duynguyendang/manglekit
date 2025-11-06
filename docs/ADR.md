# ADR — Consolidated Architecture Decisions for Manglekit Core

**Status:** Accepted
**Scope:** Core SDK (registry, builder, orchestrators, providers, config bridge)
**Period:** Oct 2025
**Audience:** Core maintainers, provider/orchestrator authors, contributors
**Last Updated:** 2025-11-06

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
* Warm-up/caching policies for a heavy client.
* Performance budgets & benchmarks for stage pipelines.

---

## 7) Per-Kind Handlers and Typed DI Enforcement (2025‑10‑23)

### Context

As the generic registry and builder landed, providers still varied in how they received dependencies. Some factories accepted the builder directly; orchestrator construction logic was sometimes embedded instead of routed through kind handlers.

### Decision

Adopt a strict separation of responsibilities:

* Per-kind `core.ComponentHandler` encapsulates build logic for that kind (assembling `diapi.*Deps`, calling the typed factory, storing the result on `Resolved`).
* Provider factories MUST accept typed deps (`diapi.*Deps`) and MUST NOT accept the builder.
* Every orchestrator must have a matching handler to be buildable via the builder.

### Rationale

* Keeps factories pure and testable; avoids leaking builder concerns into providers.
* Centralizes DI in handlers, enabling static validation and consistent lifecycle handling.
* Ensures the builder constructs all kinds, including orchestrators, deterministically.

### Consequences

* Some providers (e.g., hybrid retriever) require refactoring factory signatures to typed deps.
* An orchestrator handler is required for Sandwich (currently missing), otherwise Sandwich cannot be built via the builder.

### Migration

* Refactor factories to use `diapi.*Deps`; remove any `diapi.Builder` parameters.
* Add an orchestrator handler for Sandwich mirroring the Declarative handler pattern.

---

## 8) Static Architecture Rules & Tooling (2025‑10‑23)

### Context

To keep the codebase aligned with the architecture, we introduced static rules under `docs/rules/manglekit-arch.yml` and corresponding guidance in `AGENTS.md`.

### Decision

Codify rules for layering, registration, DI, and observability/lifecycle via static checks:

* R2: forbid `init()` in providers (explicit registration only).
* R3: pipeline must not import concrete providers.
* R10: discourage magic numbers/names in hybrid.
* R13: core must not import providers, pipeline, or root.
* R14: factories must not accept `diapi.Builder` (typed deps only).
* R15: declarative must not pick the first state provider (require config).
* R18: no direct stdout logging in prod paths (use `core.Logger`).
* R19: providers/orchestrators must not parse env directly (bind via config).

### Rationale

Automated enforcement reduces drift and speeds reviews. Agents can rely on clear failure signals to correct violations.

### Consequences

Short-term rule violations will surface (e.g., hybrid factory); they document the remaining migration work.

### Migration

Fix flagged findings as part of ongoing refactors; adjust severities as needed when transitioning.

---

## 9) Remediation Plan for Current Gaps (2025‑11‑06) - COMPLETED

### Context
An architectural audit on 2025-11-05 incorrectly flagged a violation ("Builder Leaking into Handler"). A subsequent verification on 2025-11-06 confirmed that all component handlers are, in fact, compliant with the architectural rule (ADR R14). They correctly use the `diapi.Builder` interface, not the concrete `*builder.Builder` type. The initial audit was flawed.

### High-Priority Items
1.  **[COMPLETED]** **Remediate all instances of "Builder Leaking into Handler" (ADR R14).** Verification confirmed this was not a valid issue. The code is compliant.

### Acceptance
- All provider and pipeline handlers were verified to be compliant.
- The `Builder Leaking into Handler` smell in `code-review.md` is marked `Resolved`.
- `CONTEXT.md` (GAP-001) is updated to `Resolved`.
- The `stability` frontmatter in `CONTEXT.md` and `LLD.md` has been changed from `unstable` to `stable`.
---

## 10) Hybrid Build Architecture (Programmatic & Config-First)

### Title

ADR-010: Hybrid Build Architecture (Programmatic & Config-First)

### Status

Proposed

### Context

ADR 1 established a "Config-First" architecture to ensure reproducibility and stability, removing the old, confusing programmatic builder. This rigidity makes advanced testing, TDD, and dynamic embedding scenarios unnecessarily complex. We need to re-introduce the flexibility of programmatic building, but in a way that is explicit, controlled, and does not violate the spirit of ADR 1 (which is about safe production deployments).

### Decision

We will officially support two distinct (and separate) entry points for building a Manglekit orchestrator:

*   **Default Path (Production & Standard Use):** `sdk.Load(ctx, configData)` remains the primary, recommended entry point. This path guarantees "Config-First" (ADR 1) compliance, ensuring reproducibility from a single YAML file.
*   **Advanced Path (Testing & Advanced Embedding):** We will introduce a new, explicit function in the `sdk` package: `sdk.NewBuilder()`. This function returns a `manglekit.Builder` instance. This Builder instance will expose the programmatic methods (e.g., `builder.With(opts)`, `builder.Register(handler)`) that were previously internal or removed. The user is then responsible for calling `builder.Build(ctx)` manually.

### Rationale

This hybrid approach provides the best of both worlds:

*   **Stability (ADR 1):** The default `sdk.Load` path is simple, safe, and guarantees reproducible builds from config.
*   **Flexibility (This ADR):** The explicit `sdk.NewBuilder()` path provides a necessary "escape hatch" for advanced developers and TDD, without confusing the standard user.
*   **Clear API:** By separating the two entry points (`Load` vs. `NewBuilder`), we avoid the "Polluted BuilderAPI" smell of the past. The user makes a conscious choice.

### Consequences

*   The `manglekit.Builder` struct and its methods (`With`, `Register`, `Build`) will need to be exposed via a public interface returned by `sdk.NewBuilder()`.
*   The `sdk/` package will now export `NewBuilder()`.
*   Documentation (`README.md`, `AGENTS.md`) must be updated to explain when to use `sdk.Load` (for production) vs. `sdk.NewBuilder` (for testing/advanced use).
*   The (future) "DAG" (Dependency Graph) refactor will become easier to implement, as `sdk.NewBuilder` will be the natural entry point for programmatic dependency registration.
*   This decision supersedes any previous interpretation of ADR 1 that forbade all programmatic building.
