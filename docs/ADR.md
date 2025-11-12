
# ADR — Consolidated Architecture Decisions for Manglekit Core

**Status:** Accepted  
**Scope:** Core SDK (registry, builder, orchestrators, providers, config bridge)  
**Period:** Oct 2025 – Nov 2025  
**Audience:** Core maintainers, provider/orchestrator authors, contributors  
**Last Updated:** 2025-11-07

---

## Quick Reference

| ADR | Title | Status | Key Decision |
|-----|-------|--------|--------------|
| 1 | Config-First & Declarative Architecture | Accepted | YAML/ENV are first-class inputs; config→builder bridge decouples parsing from construction |
| 2 | Observability & Lifecycle as First-Class | Accepted | Unified logging/observability contracts; graceful shutdown for all resources |
| 3 | Context Propagation Throughout SDK | Accepted | `context.Context` mandatory across all factories and runtime calls |
| 4 | Generic, Type-Safe Registry & Builder | Accepted | One generic registry; providers self-identify via Options type; spec-driven builder |
| 5 | Orchestrator Modernization (Stage-Based, Typed) | Accepted | Refactor into stage-based pipeline consuming typed `Resolved` dependencies |
| 6 | Testing & DX Uplift | Accepted | Increase coverage; adopt typed factories and `Resolved` in tests |
| 7 | Per-Kind Handlers and Typed DI Enforcement | Accepted | Strict separation: handlers encapsulate build logic; factories accept typed deps only |
| 8 | Static Architecture Rules & Tooling | Accepted | Codify layering, registration, DI, and observability rules via static checks |
| 9 | Remediation Plan for Current Gaps | Completed | Verified compliance with ADR R14; marked "Builder Leaking into Handler" as resolved |
| 10 | Dual-Path Build Architecture (Programmatic & Config-First) | Accepted | Support both `sdk.Load()` (production) and `sdk.NewBuilder()` (testing/advanced) |
| 11 | DependencyResolver Pattern for Extensible Handlers | Accepted | Handlers delegate dependency resolution to registered, type-matched resolvers; no switch statements |

---

## Table of Contents

1. [Foundation Layer (ADRs 1–3)](#foundation-layer)
2. [Core Architecture (ADRs 4–5)](#core-architecture)
3. [Enforcement & Refinement (ADRs 6–8)](#enforcement--refinement)
4. [Advanced Patterns (ADRs 10–11)](#advanced-patterns)
5. [Appendix: Status & Remediation (ADR 9)](#appendix-status--remediation)
6. [Known Trade-offs & Risks](#known-trade-offs--risks)
7. [What's Next (Future ADRs)](#whats-next-future-adrs)

---

## Foundation Layer

### ADR 1: Config-First & Declarative Architecture

#### Context

Early versions mixed runtime construction with ad-hoc configuration, making examples brittle and pipelines hard to reproduce/migrate.

#### Decision

Adopt a **config-first** stance: YAML/ENV become first-class inputs. Introduce a thin **config→builder bridge** (`sdk.FromConfig`), which validates/normalizes config and then calls the builder with typed options.

#### Rationale

* **Reproducibility:** Pipelines can be checked in, diffed, and promoted across environments.
* **Separation of Concerns:** Parsing/validation is decoupled from construction.
* **Operational Safety:** Configuration is the single source of truth for production deployments.

#### Consequences

* Config parsing and path resolution are decoupled from the builder.
* Examples and docs center on declarative usage.
* A minimal `DecodeOptions` utility maps config maps → typed options.

#### Migration

* Move any parsing from the builder into the config bridge.
* Keep the builder free of YAML/ENV specifics.

**See Also:** [ADR 10](#adr-10-dual-path-build-architecture-programmatic--config-first) (Dual-Path Architecture)

---

### ADR 2: Observability & Lifecycle as First-Class

#### Context

Resource leaks and inconsistent logging made debugging and operations difficult.

#### Decision

Unify logging/observability contracts and implement **graceful shutdown** for local/indexed stores and API clients. Resource closers are tracked centrally by the builder.

#### Rationale

* **Predictable Logs:** Uniform logging across modules via `core.Logger`.
* **Safe Teardown:** Tests, CLIs, and long-running services can cleanly release resources.
* **Operational Visibility:** Structured logging enables better observability and debugging.

#### Consequences

* Standard "closer" pattern and structured logging defaults.
* Builders register closers when components are created.
* All providers must implement (or explicitly opt out of) closer semantics.

#### Migration

* Ensure every provider implements (or clearly opts out of) closer semantics.
* Replace ad-hoc loggers with the unified `core.Logger` interface.

**See Also:** [`docs/LOGGING.md`](docs/LOGGING.md)

---

### ADR 3: Context Propagation Throughout SDK

#### Context

Some APIs ignored `context.Context`, blocking cancellation, deadlines, and tracing.

#### Decision

Make `context.Context` explicit and mandatory across all factories and runtime calls.

#### Rationale

* **Cancellation & Deadlines:** Enables proper timeout and cancellation handling.
* **Tracing:** Aligns with modern Go service patterns (OpenTelemetry, etc.).
* **Predictable Behavior:** Better control under load and during shutdown.

#### Consequences

* Touches many signatures; simplifies test control and instrumentation.
* More predictable behavior under load and during shutdown.

#### Migration

* Thread `ctx` through all call chains (providers, orchestrators, stores).

---

## Core Architecture

### ADR 4: Generic, Type-Safe Registry & Builder

#### Context

The original builder/registry had per-type maps, per-type `With…` methods, and `build…` functions with duplicated wiring. Orchestrators consumed an `any` options blob, causing type erasure and runtime casts.

#### Decision (4 parts)

##### 4.1 Generic Registry

* Replace per-type registries with **one generic registry** of typed factories.
* Factory signature is standardized: `func(ctx context.Context, deps D, cfg O) (T, error)`.
* Registration uses generics; the registry holds a catalog keyed by **Kind** (llm, retriever, vector_store, …).

##### 4.2 Provider Self-Identification

* Each provider's **Options** type implements:

  ```go
  type ProviderOptions interface {
      ProviderName() string   // e.g., "openai-chat"
      ProviderKind() Kind     // e.g., KindLLM
  }
  ```

* Registration takes an **options sample** instead of a string literal:
  ```go
  manglekit.Register(registry, openai.Options{}, openai.NewFactory())
  ```
  (No magic strings; name/kind come from the type.)

##### 4.3 Generic Builder (DRY) with a Spec Table

* Collapse all per-type `With…`/`build…` into a single `With(opts any)` and a **data-driven build loop**.
* A **spec table** defines dependency order and assignment:
  ```
  embedder → vector_store → retriever → reranker → rules → llm → state_provider
  ```
* Hybrid retrievers are supported via a typed `BuildRetriever` hook (using the registry).

**Spec Table Example:**
```
Kind: embedder
  ↓ (required by)
Kind: vector_store
  ↓ (required by)
Kind: retriever
  ↓ (required by)
Kind: reranker
  ↓ (required by)
Kind: rules
  ↓ (required by)
Kind: llm
  ↓ (required by)
Kind: state_provider
```

##### 4.4 Typed Orchestrator Inputs

* Replace `core.Options any` with a **typed `Resolved` struct** containing:
  ```go
  type Resolved struct {
      Retriever      core.Retriever
      VectorStore    core.VectorStore
      Reranker       core.Reranker
      Rules          core.Rules
      LLM            core.LLM
      Embedder       core.Embedder
      StateProvider  core.StateProvider
      Observability  *core.Observability
  }
  ```
* Orchestrator factories now consume `Resolved` directly—no runtime casts.

#### Rationale

* **Eliminates Duplication:** Enforces Open/Closed Principle; adding a provider requires only registering one typed factory.
* **Compile-Time Guarantees:** Type-safety end-to-end (providers, builder, orchestrators).
* **Streamlined DX:** Developers add a provider by registering one typed factory—no edits to core builder/registry.

#### Consequences

* **Massive** reduction in builder boilerplate.
* Type-safety end-to-end (providers, builder, orchestrators).
* Clean plugin surface for third parties.
* Minimal reflection for options type lookups (acceptable, limited to wiring).

#### Migration

* Update all providers to implement `ProviderOptions` on their Options type and to accept **typed** `cfg` in factories.
* Register with `manglekit.Register(registry, OptionsSample{}, Factory)`.
* Remove per-type `With…` and `build…` from the builder; use `With(opts)` and spec-driven `buildAll`.
* Update orchestrators to accept `Resolved`.

**See Also:** [`core/factory.go`](core/factory.go), [`registry.go`](registry.go), [`builder.go`](builder.go)

---

### ADR 5: Orchestrator Modernization (Stage-Based, Typed)

#### Context

The "Sandwich" orchestrator started monolithic, with `any`-based accessors and inlined logic.

#### Decision

Refactor into a **stage-based pipeline** consuming typed `Resolved` dependencies (retrieval, rerank, rules, LLM calls, state), removing casts and allowing easier unit tests and targeted changes.

#### Rationale

* **Composability:** Each stage is a focused, testable unit.
* **Type Safety:** Strict interface boundaries with typed inputs/outputs.
* **Extensibility:** Easier to add new stages or swap implementations.

#### Consequences

* Clear separation of retrieval/rerank/generation/state logic.
* Easier to add new stages or swap implementations.
* Orchestrator factories must accept `Resolved` (not `any`).

#### Migration

* Move stage logic into small, focused units; wire via the orchestrator using `Resolved`.
* Implement a handler for each orchestrator (e.g., `sandwich.Handler`, `declarative.Handler`).

**See Also:** [`pipeline/sandwich/`](pipeline/sandwich/), [`pipeline/declarative/`](pipeline/declarative/)

---

## Enforcement & Refinement

### ADR 6: Testing & DX Uplift

#### Context

Brittle tests, many fixtures bound to legacy APIs, and gaps in coverage.

#### Decision

Increase test coverage across config, builder, providers, and orchestrators; remove obsolete tests; adopt typed factories and `Resolved` in tests; unify logging to reduce noise.

#### Rationale

* **Confidence in Refactors:** Faster iteration and safer changes.
* **Easier Onboarding:** Smaller, more targeted tests are easier to understand.

#### Consequences

* Smaller, more targeted tests.
* Fewer integration-only test paths.
* Tests use typed options/factories instead of `any`.

#### Migration

* Replace legacy tests that assert `any` contents with typed expectations.
* Prefer table-driven tests with typed options/factories.
* Use `sdk.LoadWithRegistry` for DI integration tests (see [`AGENTS.md`](AGENTS.md) §15 for test patterns).

**See Also:** [`AGENTS.md`](AGENTS.md) §15 (Provider Test Architecture)

---

### ADR 7: Per-Kind Handlers and Typed DI Enforcement

#### Context

As the generic registry and builder landed, providers still varied in how they received dependencies. Some factories accepted the builder directly; orchestrator construction logic was sometimes embedded instead of routed through kind handlers.

#### Decision

Adopt a strict separation of responsibilities:

* Per-kind `core.ComponentHandler` encapsulates build logic for that kind (assembling `diapi.*Deps`, calling the typed factory, storing the result on `Resolved`).
* Provider factories **MUST** accept typed deps (`diapi.*Deps`) and **MUST NOT** accept the builder.
* Every orchestrator must have a matching handler to be buildable via the builder.

#### Rationale

* **Pure Factories:** Keeps factories testable; avoids leaking builder concerns into providers.
* **Centralized DI:** Handlers encapsulate dependency injection, enabling static validation and consistent lifecycle handling.
* **Deterministic Building:** Ensures the builder constructs all kinds, including orchestrators, deterministically.

#### Consequences

* Some providers (e.g., hybrid retriever) require refactoring factory signatures to typed deps.
* An orchestrator handler is required for Sandwich (currently implemented in [`pipeline/sandwich/handler.go`](pipeline/sandwich/handler.go)).

#### Migration

* Refactor factories to use `diapi.*Deps`; remove any `diapi.Builder` parameters.
* Add an orchestrator handler for each orchestrator mirroring the Declarative handler pattern.

**See Also:** [`core/handler.go`](core/handler.go), [`core/diapi/di.go`](core/diapi/di.go)

---

### ADR 8: Static Architecture Rules & Tooling

#### Context

To keep the codebase aligned with the architecture, we introduced static rules under [`docs/rules/manglekit-arch.yml`](docs/rules/manglekit-arch.yml) and corresponding guidance in [`AGENTS.md`](AGENTS.md).

#### Decision

Codify rules for layering, registration, DI, and observability/lifecycle via static checks:

* **R2:** Forbid `init()` in providers (explicit registration only).
* **R3:** Pipeline must not import concrete providers.
* **R10:** Discourage magic numbers/names in hybrid.
* **R13:** Core must not import providers, pipeline, or root.
* **R14:** Factories must not accept `diapi.Builder` (typed deps only).
* **R15:** Declarative must not pick the first state provider (require config).
* **R18:** No direct stdout logging in prod paths (use `core.Logger`).
* **R19:** Providers/orchestrators must not parse env directly (bind via config).

#### Rationale

Automated enforcement reduces drift and speeds reviews. Agents can rely on clear failure signals to correct violations.

#### Consequences

Short-term rule violations will surface (e.g., hybrid factory); they document the remaining migration work.

#### Migration

Fix flagged findings as part of ongoing refactors; adjust severities as needed when transitioning.

**See Also:** [`docs/rules/manglekit-arch.yml`](docs/rules/manglekit-arch.yml)

---

## Advanced Patterns

### ADR 10: Dual-Path Build Architecture (Programmatic & Config-First)

#### Status

Accepted

#### Context

ADR 1 established a "Config-First" architecture to ensure reproducibility and stability, removing the old, confusing programmatic builder. This rigidity makes advanced testing, TDD, and dynamic embedding scenarios unnecessarily complex. We need to re-introduce the flexibility of programmatic building, but in a way that is explicit, controlled, and does not violate the spirit of ADR 1 (which is about safe production deployments).

#### Decision

We will officially support two distinct (and separate) entry points for building a Manglekit orchestrator:

* **Default Path (Production & Standard Use):** `sdk.Load(ctx, configData)` remains the primary, recommended entry point. This path guarantees "Config-First" (ADR 1) compliance, ensuring reproducibility from a single YAML file.
* **Advanced Path (Testing & Advanced Embedding):** We will introduce a new, explicit function in the `sdk` package: `sdk.NewBuilder()`. This function returns a `manglekit.Builder` instance. This Builder instance will expose the programmatic methods (e.g., `builder.With(opts)`, `builder.Register(handler)`) that were previously internal or removed. The user is then responsible for calling `builder.Build(ctx)` manually.

#### Decision Matrix: When to Use Each Path

| Scenario | Entry Point | Rationale |
|----------|-------------|-----------|
| Production deployment | `sdk.Load(ctx, configData)

` | Guaranteed reproducibility from YAML; no surprises in production |
| Testing & TDD | `sdk.NewBuilder()` | Explicit escape hatch; full programmatic control for advanced scenarios |
| Dynamic embedding | `sdk.NewBuilder()` | Allows runtime composition without config files |

#### Rationale

This hybrid approach provides the best of both worlds:

* **Stability (ADR 1):** The default `sdk.Load` path is simple, safe, and guarantees reproducible builds from config.
* **Flexibility (This ADR):** The explicit `sdk.NewBuilder()` path provides a necessary "escape hatch" for advanced developers and TDD, without confusing the standard user.
* **Clear API:** By separating the two entry points (`Load` vs. `NewBuilder`), we avoid the "Polluted BuilderAPI" smell of the past. The user makes a conscious choice.

#### Consequences

* The `manglekit.Builder` struct and its methods (`With`, `Register`, `Build`) will need to be exposed via a public interface returned by `sdk.NewBuilder()`.
* The `sdk/` package will now export `NewBuilder()`.
* Documentation (`README.md`, `AGENTS.md`) must be updated to explain when to use `sdk.Load` (for production) vs. `sdk.NewBuilder` (for testing/advanced use).
* The (future) "DAG" (Dependency Graph) refactor will become easier to implement, as `sdk.NewBuilder` will be the natural entry point for programmatic dependency registration.
* This decision supersedes any previous interpretation of ADR 1 that forbade all programmatic building.

---

### ADR 11: DependencyResolver Pattern for Extensible Handlers

#### Status

Accepted

#### Context

Component handlers often must dispatch to different construction paths based on provider options. For example, the retriever handler handles three different kinds of retrievers:

1. **Hybrid Retrievers** — Depend on sub-retrievers (e.g., `HybridOptions` implements `diapi.SubRetrieversDep`)
2. **Dense Retrievers** — Depend on an embedder and vector store (e.g., `DenseOptions` implements `diapi.EmbedderDep` and `diapi.VectorStoreDep`)
3. **Other Retrievers** — Have no special dependencies beyond `CoreDeps` (e.g., `BM25Options`)

A naive implementation would use a large type-switch statement:

```go
switch opts := cfg.(type) {
case diapi.SubRetrieversDep:
    // Handle hybrid
case diapi.EmbedderDep:
    // Handle dense
default:
    // Handle noop
}
```

This violates the **Open/Closed Principle**: adding a new retriever type requires modifying the handler. It also makes the handler rigid and tightly coupled to specific option types.

#### Decision

Introduce a **DependencyResolver** pattern:

1. **Define the `DependencyResolver` interface** (in `core/diapi/di.go`):
   ```go
   type DependencyResolver interface {
       Matches(opts any) bool
       Resolve(ctx context.Context, builderDI any, cfg any) (any, error)
   }
   ```

2. **Create a `ResolverRegistry`** (in `core/diapi/resolvers.go`) to manage a collection of resolvers:
   ```go
   type ResolverRegistry struct {
       resolvers map[core.Kind][]DependencyResolver
   }
   
   func (r *ResolverRegistry) Register(kind core.Kind, resolver DependencyResolver)
   func (r *ResolverRegistry) Resolve(ctx context.Context, kind core.Kind, 
       builderDI any, cfg any) (any, error)
   ```

3. **Implement built-in resolvers** for each supported pattern:
   - `SubRetrieverResolver` — Matches `diapi.SubRetrieversDep`; resolves sub-retrievers and builds `RetrieverDeps`
   - `DenseRetrieverResolver` — Matches `diapi.EmbedderDep` + `diapi.VectorStoreDep`; builds `DenseRetrieverDeps`
   - `NoopRetrieverResolver` — Catch-all; builds `NoopDeps`

4. **Refactor handlers to delegate** to the resolver:
   ```go
   func (h *Handler) BuildComponent(...) (core.ResourceCloser, error) {
       deps, err := h.resolver.Resolve(ctx, core.KindRetriever, builderDI, opts)
       if err != nil {
           return nil, fmt.Errorf("failed to resolve dependencies: %w", err)
       }
       // Factory receives fully-resolved deps
       built, err := f.Build(ctx, deps, cfg)
       // ...
   }
   ```

#### Rationale

This pattern provides several benefits:

* **Open/Closed Principle:** New retriever types can be supported by registering new resolvers *without modifying the handler*.
* **Extensibility:** Users or extensions can register custom resolvers to handle new provider patterns.
* **Type Safety:** Each resolver is responsible for matching options to a specific dependency pattern; no brittle type-switches.
* **Clarity:** Each resolver encapsulates one construction pattern; the intent is clear.
* **Testability:** Resolvers can be tested in isolation without a full handler.

#### Implementation Details

**Resolver Matching Strategy:**

Resolvers are tried in registration order. The first resolver that returns `true` from `Matches(opts)` is used to resolve dependencies. The matching strategy can be:

- **Marker Interface Check:** Does `opts` implement interface X? (e.g., `SubRetrieversDep`)
- **Field Presence Check:** Does `opts` have a required field?
- **Config Value Check:** Is a config string non-empty?

**Registration Order:** The handler registers resolvers in **priority order**. For retrievers:

1. `SubRetrieverResolver` — Most specific (sub-retrievers are rare)
2. `DenseRetrieverResolver` — General case (dense retrievers are common)
3. `NoopRetrieverResolver` — Catch-all (any other retriever type)

**Error Handling:** If no resolver matches, an error is returned. If a resolver matches but resolution fails (e.g., a sub-retriever is not found), that error is propagated with context.

#### Consequences

* Handlers are now **stable**; adding new provider types does not require handler changes.
* Resolvers can be **independently tested** without spinning up a full handler/builder.
* The pattern is **reusable** across all component kinds (already applied to retrievers; extensible to other kinds).
* Handlers become **easier to understand** (single entry point; delegation to resolver).

#### Migration

This pattern is already implemented for retrievers in:

- `core/diapi/di.go` — Interface definition
- `core/diapi/resolvers.go` — Registry and built-in resolvers
- `internal/providers/retrievers/handler.go` — Handler using the registry

To extend this pattern to other component kinds (e.g., LLMs, embedders), follow the same steps:

1. Define resolver interfaces as `diapi` marker interfaces (e.g., `APIClientDep`, `ConfigParamDep`).
2. Implement resolvers in the component handler package (e.g., `internal/providers/llm/resolvers.go`).
3. Refactor the handler to use `ResolverRegistry`.

#### Example: Adding a New Retriever Type

```go
// 1. Define marker interface (in core/diapi/di.go)
type CustomRetrieverDep interface {
    GetCustomDependency() string
}

// 2. Implement resolver
type CustomRetrieverResolver struct{}

func (r *CustomRetrieverResolver) Matches(opts any) bool {
    _, ok := opts.(CustomRetrieverDep)
    return ok
}

func (r *CustomRetrieverResolver) Resolve(ctx context.Context, builderDI any, cfg any) (any, error) {
    // Resolve custom dependencies...
    return RetrieverDeps{...}, nil
}

// 3. Register in handler
h.resolver.Register(core.KindRetriever, &CustomRetrieverResolver{})
```

---

## Appendix: Status & Remediation

### ADR 9: Remediation Plan for Current Gaps (2025-11-06) - COMPLETED

#### Context

An architectural audit on 2025-11-05 incorrectly flagged a violation ("Builder Leaking into Handler"). A subsequent verification on 2025-11-06 confirmed that all component handlers are, in fact, compliant with the architectural rule (ADR R14). They correctly use the `diapi.Builder` interface, not the concrete `*builder.Builder` type. The initial audit was flawed.

#### High-Priority Items

1. **[COMPLETED]** **Remediate all instances of "Builder Leaking into Handler" (ADR R14).** Verification confirmed this was not a valid issue. The code is compliant.

#### Acceptance

- All provider and pipeline handlers were verified to be compliant.
- All handler-factory pairs now use properly typed `diapi.*Deps` structs.
- [`docs/CONTEXT.md`](docs/CONTEXT.md) (GAP-001) is updated to `Resolved`.
- The `stability` frontmatter in [`docs/CONTEXT.md`](docs/CONTEXT.md) and [`docs/LLD.md`](docs/LLD.md) has been changed from `unstable` to `stable`.

---

## Known Trade-offs & Risks

* Requires Go generics and minimal reflection for type→name/kind mapping.
* Breaking API (per-type `With…`/`build…` removed).
* Orchestrator factories must be migrated to typed `Resolved`.
* ADR 10 introduces dual entry points, which could confuse users if not documented clearly.

---

## What's Next (Future ADRs)

* Build-graph introspection & visualization.
* Pluggable runtime (Go plugin/WASM) & sandboxing.
* Options schema validation + JSON-Schema export.
* Warm-up/caching policies for a heavy client.
* Performance budgets & benchmarks for stage pipelines.
* DAG (Dependency Graph) refactor for advanced programmatic building.

---

## Resulting System (Snapshot)

* **One Registry** (generic, typed) for all components.
* **One Builder** (generic, spec-driven) that constructs pipelines in dependency order.
* **Providers self-identify** via their Options type; no string literals when registering.
* **Orchestrators** receive **typed dependencies** (`Resolved`) and run **stage-based** flows.
* **Config bridge** cleanly translates YAML/ENV → typed Options → builder.
* **Dual-path entry points:** `sdk.Load()` for production, `sdk.NewBuilder()` for advanced use.
* **Observability & lifecycle** standardized; **context propagation** is universal.
* **Testing & DX** significantly improved.
* **Static rules** enforce architecture compliance automatically.
