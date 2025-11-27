---
context_type: architecture_standard
project: manglekit
language: go
version: 3.0.0
last_updated: 2025-11-27T14:50:00Z
stability: stable
audience: humans_and_agents
---

# Manglekit: The Live Architecture Standard

This document is the canonical, single source of truth for the Manglekit SDK's architecture. It defines the non-negotiable rules, contracts, and patterns that govern the framework.

## 0. Implementation Snapshot (Current State)

```mermaid
graph TD
    subgraph "User Space"
        App[User Application]
        Config[config.yaml]
    end

    subgraph "Manglekit Kernel"
        Client[Client]
        Protect[Protect() API]
        
        App -->|NewClient| Client
        App -->|Protect(Action)| Protect
    end

    subgraph "The Guard"
        GA[GuardedAction]
        Lifecycle[Trace -> AuthZ -> Exec -> Validate]
        
        Protect --> GA
        GA --> Lifecycle
    end

    subgraph "Logic Engine"
        PE[PolicyEngine]
        Runtime[MangleRuntime]
        Reflect[Reflector]
        
        Lifecycle <--> PE
        PE --> Runtime
        PE --> Reflect
    end

    subgraph "Universal Adapters"
        AI[AI Adapter (Genkit)]
        Func[Func Adapter]
        Vector[Vector Adapter]
        
        Lifecycle -->|Delegate| AI
        Lifecycle -->|Delegate| Func
        Lifecycle -->|Delegate| Vector
    end
```

## 1. Architectural Overview

Manglekit v3.0 ("Genesis") is a **Universal AI Governance Kernel**. It abandons the complex "Builder/Registry" pattern of v0.x in favor of a streamlined **Composition Model**.

The core philosophy is **"Wrap, Don't Build"**. Manglekit does not construct your AI components; it *wraps* them. You bring your own execution engine (Genkit, LangChain, native Go code), and Manglekit wraps it in a **Guarded Action** that enforces policy, observability, and safety.

## 2. Dependency Rules (Non-Negotiable)

1.  **`manglekit` (root) is the entry point.** It orchestrates `guard`, `engine`, and `core`.
2.  **`guard` depends on `engine` and `core`.** It must NOT depend on concrete adapters.
3.  **`engine` depends on `core` and `google/mangle`.** It is the pure logic layer.
4.  **`adapters` depend on `core` and external drivers (Genkit).** They bridge the gap between the world and the kernel.
5.  **`core` has NO dependencies.** It defines the interfaces (`Action`, `Envelope`, `Logger`).

## 3. Core Contracts

### Primary Interfaces

-   **`core.Action`**: The universal interface for *anything* that does work.
    ```go
    type Action interface {
        Execute(ctx context.Context, input Envelope) (Envelope, error)
        Metadata() ActionMetadata
    }
    ```
-   **`core.Envelope`**: The standardized container for data moving through the kernel.
    ```go
    type Envelope struct {
        ID       uuid.UUID
        Payload  any
        Metadata map[string]string
    }
    ```
-   **`core.Logger`**: The structured logging interface used throughout the kernel.

### Logic Interfaces

-   **`engine.PolicyEngine`**: The high-level coordinator for governance checks.
-   **`engine.Reflector`**: The system that converts Go structs into Datalog facts.

## 4. The "Guarded Action" Lifecycle

Every protected operation undergoes a strict, immutable lifecycle managed by `guard.GuardedAction`:

1.  **Trace Start**: A new OpenTelemetry span is created.
2.  **Context Injection**: Logger and Trace ID are injected into `context.Context`.
3.  **Authorization (Pre-Check)**:
    *   Input Payload is reflected to Datalog facts.
    *   `deny(Input)` rule is evaluated.
    *   **BLOCK**: If denied, execution halts with `PolicyViolationError`.
4.  **Execution**: The inner `core.Action` (Adapter) is executed.
5.  **Validation (Post-Check)**:
    *   Output Payload is reflected to Datalog facts.
    *   `violation(Output)` rule is evaluated.
    *   **BLOCK**: If violated, result is discarded.
6.  **Trace End**: Span is closed, outcome recorded.

## 5. Universal Adapters

Instead of "Providers" and "Factories", Manglekit v3 uses **Adapters**.

*   **`adapters/ai`**: Wraps Google Genkit `ai.Model` and `ai.Embedder`.
*   **`adapters/vector`**: Wraps Genkit Retrievers.
*   **`adapters/func`**: Wraps any Go function `func(ctx, In) (Out, error)`.

## 6. Configuration

Configuration is handled by the `config` package, which loads from YAML.

*   **Policy Path**: Location of `.dl` files.
*   **Observability**: OTel endpoint, service name.
*   **Env Vars**: Supported via `${VAR}` syntax in YAML.

## 7. Observability

*   **Tracing**: Native OpenTelemetry integration. Every Action gets a span.
*   **Logging**: Structured logging injected into Context.
*   **Lineage**: (Planned) Automatic tracking of Input ID -> Output ID derivation.

## 8. Known Gaps

The codebase is **Stable (v3.0.0)**.

**Status Legend:**
- **✅ RESOLVED**: Feature works out-of-the-box.
- **⚠️ PARTIALLY RESOLVED**: Foundation exists, refinement needed.
- **❌ OPEN**: Blocking issue.

### GAP-001: Lineage Tracking — ⚠️ PARTIALLY RESOLVED
- **Description**: The `LineageTracker` component mentioned in HLD is not yet fully implemented as a standalone subsystem.
- **Status**: Basic lineage (Trace ID propagation) works via OTel. Explicit data lineage graph is pending.

## 9. Changelog

-   **2025-11-27**: **Genesis Release (v3.0.0)**. Complete re-architecture. Removed Builder/Registry. Introduced Client/Guard/Engine model.
-   **2025-11-25**: **Smart Router**: (Legacy v0.x) Implemented dynamic dispatch.
-   **2025-11-20**: **Reflection**: Added `core/reflection`.
