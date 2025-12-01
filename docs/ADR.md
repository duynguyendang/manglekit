
# ADR — Consolidated Architecture Decisions for Manglekit Core

**Status:** Accepted  
**Scope:** Core SDK (Kernel, Guard, Engine, Adapters)  
**Period:** Oct 2025 – Nov 2025  
**Audience:** Core maintainers, provider/orchestrator authors, contributors  
**Last Updated:** 2025-11-27

---

## Quick Reference

| ADR | Title | Status | Key Decision |
|-----|-------|--------|--------------|
| 1 | Config-First & Declarative Architecture | Accepted | YAML/ENV are first-class inputs; config→builder bridge decouples parsing from construction |
| 2 | Observability & Lifecycle as First-Class | Accepted | Unified logging/observability contracts; graceful shutdown for all resources |
| 3 | Context Propagation Throughout SDK | Accepted | `context.Context` mandatory across all factories and runtime calls |
| 4 | Generic, Type-Safe Registry & Builder | **Superseded** | (Replaced by ADR 12) One generic registry; providers self-identify via Options type |
| 5 | Orchestrator Modernization | **Superseded** | (Replaced by ADR 12) Refactor into stage-based pipeline consuming typed `Resolved` dependencies |
| 6 | Testing & DX Uplift | Accepted | Increase coverage; adopt typed factories and `Resolved` in tests |
| 7 | Per-Kind Handlers and Typed DI Enforcement | **Superseded** | (Replaced by ADR 12) Strict separation: handlers encapsulate build logic |
| 8 | Static Architecture Rules & Tooling | Accepted | Codify layering, registration, DI, and observability rules via static checks |
| 9 | Remediation Plan for Current Gaps | Completed | Verified compliance with ADR R14; marked "Builder Leaking into Handler" as resolved |
| 10 | Dual-Path Build Architecture | **Superseded** | (Replaced by ADR 12) Support both `sdk.Load()` and `sdk.NewBuilder()` |
| 11 | DependencyResolver Pattern | **Superseded** | (Replaced by ADR 12) Handlers delegate dependency resolution to resolvers |
| 12 | **Universal AI Governance Kernel** | **Accepted** | **Complete Re-Architecture**: Replace Builder/Registry with Client/Guard/Engine composition. |

---

## Table of Contents

1. [Universal AI Governance Kernel (ADR 12)](#universal-ai-governance-kernel-adr-12)nance-kernel-adr-12)
2. [Foundation Layer (ADRs 1–3)](#foundation-layer)
3. [Legacy Architecture (ADRs 4–5, 7, 10-11)](#legacy-architecture-superseded)
4. [Enforcement & Refinement (ADRs 6–8)](#enforcement--refinement)
5. [Appendix: Status & Remediation (ADR 9)](#appendix-status--remediation)

---

## Universal AI Governance Kernel (ADR 12)

### ADR 12: Universal AI Governance Kernel

#### Status
Accepted (v1.0)

#### Context
The v0.x architecture (ADRs 4, 5, 7, 10, 11) relied on a complex "Builder/Registry" pattern. While it provided type safety, it suffered from:
1.  **High Cognitive Load**: Developers had to understand Handlers, Factories, Resolvers, and the Registry just to add a simple component.
2.  **Rigidity**: The "Build Order" was hard-coded, making it difficult to create custom pipelines.
3.  **Conflation**: The Builder mixed *configuration parsing* with *object construction*.
4.  **Vendor Lock-in**: The framework felt like a "walled garden" where everything had to be a registered provider.

#### Decision
**Abandon the Builder/Registry pattern entirely.** Move to a **Composition-based** architecture centered on the **Guarded Action**.

1.  **The Kernel (`manglekit.Client`)**: A lightweight coordinator that holds the Policy Engine and Observability stack.
2.  **The Guard (`Protect()`)**: A single API that wraps *any* `core.Action` with governance (Trace -> AuthZ -> Exec -> Validate).
3.  **Universal Adapters**: Instead of "Providers", we have simple adapters that convert external libraries (Genkit, native Go functions) into `core.Action`.
4.  **Logic Engine**: A dedicated subsystem (`engine`) for Datalog reasoning and reflection.

#### Rationale
*   **Simplicity**: "Wrap, Don't Build." The framework no longer constructs your objects; it governs them.
*   **Flexibility**: Users can bring any execution engine (Genkit, LangChain, raw HTTP) and simply wrap it.
*   **Separation of Concerns**:
    *   **User**: Constructs the object (e.g., `genkit.NewModel`).
    *   **Manglekit**: Wraps it (`client.Protect(model)`).
*   **Zero-Config Reflection**: The new `core/reflection` engine allows governance over arbitrary Go structs without manual mapping.

#### Consequences
*   **Deleted**: `Builder`, `Registry`, `ComponentHandler`, `Factory` interfaces.
*   **Introduced**: `Client`, `GuardedAction`, `MangleRuntime`.
*   **Supersedes**: ADRs 4, 5, 7, 10, 11 are now obsolete history.

---

## Foundation Layer

### ADR 1: Config-First & Declarative Architecture

#### Context
Early versions mixed runtime construction with ad-hoc configuration.

#### Decision
Adopt a **config-first** stance: YAML/ENV are first-class inputs.

#### Status
**Retained**. The `config` package still loads YAML, but instead of feeding a Builder, it configures the `Client` (Policy path, Observability settings).

---

### ADR 2: Observability & Lifecycle as First-Class

#### Context
Resource leaks and inconsistent logging.

#### Decision
Unify logging/observability contracts and implement graceful shutdown.

#### Status
**Retained**. `core.Logger` and `core.Tracer` are central to the `Client` and `GuardedAction`.

---

### ADR 3: Context Propagation Throughout SDK

#### Context
APIs ignored `context.Context`.

#### Decision
Make `context.Context` explicit and mandatory.

#### Status
**Retained**. `Execute(ctx, ...)` is the standard signature.

---

## Legacy Architecture (Superseded)

*The following ADRs describe the v0.x "Builder/Registry" architecture. They are kept for historical context but are no longer active in v1.0.*

### ADR 4: Generic, Type-Safe Registry & Builder (Superseded)
**Replaced by ADR 12.** The Registry and Builder have been removed in favor of direct composition.

### ADR 5: Orchestrator Modernization (Superseded)
**Replaced by ADR 12.** Orchestrators are now just `core.Action` implementations that chain other Actions. The "Sandwich" pattern is now a legacy implementation detail, not a framework constraint.

### ADR 7: Per-Kind Handlers and Typed DI Enforcement (Superseded)
**Replaced by ADR 12.** Dependency Injection is now the user's responsibility (standard Go constructor injection). The framework no longer manages DI.

### ADR 10: Dual-Path Build Architecture (Superseded)
**Replaced by ADR 12.** There is now only one path: Initialize `Client`, then `Protect()` your actions.

### ADR 11: DependencyResolver Pattern (Superseded)
**Replaced by ADR 12.** Complex resolution logic is no longer needed as users construct their own dependencies.

---

## Enforcement & Refinement

### ADR 6: Testing & DX Uplift

#### Context
Brittle tests.

#### Decision
Increase test coverage; adopt typed contracts.

#### Status
**Retained**. Testing against `core.Action` interface is the standard.

---

### ADR 8: Static Architecture Rules & Tooling

#### Context
Need to keep codebase aligned.

#### Decision
Codify rules via static checks.

#### Status
**Retained**, but rules updated to reflect v1.0 architecture (e.g., "No Builder" rule).

---

## Appendix: Status & Remediation

### ADR 9: Remediation Plan for Current Gaps (Completed)

#### Context
Addressed v0.x gaps (Builder leaking, etc.).

#### Status
**Completed** (Historical).

---

## Resulting System (v1.0 Snapshot)

*   **Kernel**: `manglekit.Client` (Policy + Observability).
*   **Guard**: `GuardedAction` (The "Sandwich" of governance).
*   **Engine**: `MangleRuntime` (Datalog) + `Reflector` (Go Structs -> Facts).
*   **Adapters**: `adapters/ai`, `adapters/func`, `adapters/vector`.
*   **Config**: YAML loads policy/observability, not component wiring.
