---
context_type: codebase_overview
project: manglekit
language: go
version: 2025.10
last_updated: 2025-10-15
---

# Manglekit Project Context

## Architectural Integrity Summary
The Manglekit SDK is in a **partially stable** state. A major refactoring effort successfully resolved several critical architectural flaws, notably by eliminating global state in the registry and improving adherence to the Open/Closed principle in the builder. The architecture is now based on a much cleaner, factory-based pattern for component instantiation.

However, this refactoring has left behind significant and subtle code smells. The system's type safety is compromised by the inconsistent use of `any` to break package dependency cycles. The builder's API is inconsistent, and its configuration mechanisms are tightly coupled and not fully extensible. Key components, like the hybrid retriever, suffer from configuration flaws that limit their utility. The declarative orchestrator remains disabled.

While the foundation is stronger, the SDK is not yet stable. The remaining issues impact developer experience, maintainability, and type safety, and must be addressed before the SDK can be considered robust.

## Core Building Blocks
- ✅ **Registry (`registry.go`)**: The `Registry` is now a fully encapsulated instance, a major improvement. It correctly uses strongly-typed factory functions for most components.
- ⚠️ **Registry Smells**: The `ClientFactories` map still uses `any`, and its factory signature is inconsistent with all others, representing a significant remaining type-safety hole. The registry also contains a dead/unused `Options` field.
- ✅ **Providers (`providers/`)**: The provider registration builder (`providers.NewSet()`) offers a clean, fluent API for developers.
- ✅ **Core types (`core/types.go`)**: The core data structures are well-defined.
- ⚠️ **Core Smells**: The `Orchestrator` interface and `Options` struct use `any` to avoid import cycles, which sacrifices compile-time safety and is a symptom of poor package boundaries.

## Orchestrator Construction
- ⚠️ **Fluent builder (`builder.go`)**
  - The builder is now "dumber" and correctly uses factories, which is a major step forward. The `SubRetrieverBuilder` is a good example of the Interface Segregation Principle.
  - **Inconsistent API**: The builder's `With...` methods are inconsistent (e.g., `WithEmbedder` can take an instance, others cannot).
  - **Opaque Internals**: The builder's internal `clients` map for dependency injection is not exposed via a public method, making it unusable.
  - **Dead Code**: Contains an unused `embedderAlias` map.
- ❌ **Configuration (`config.go`)**
  - **Tight Coupling**: `NewBuilderFromYAML` and `NewBuilderFromEnv` are tightly coupled to the builder, violating SRP. They are responsible for both parsing config and instantiating the builder.
  - **Duplicated Logic**: The logic for processing components is duplicated across both functions.
  - **OCP Violation**: Uses a `switch` statement to configure components, which is not easily extensible.

## Execution Pipelines
- ⚠️ **Sandwich orchestrator (`pipeline/sandwich.go`)**
  - **SRP Violation**: The `Execute` method is a "god method" with mixed responsibilities (logging, tracing, state management, orchestration), making it hard to maintain.
  - **Magic Strings**: The code is littered with raw string literals for `Meta` map keys, which is error-prone.
- ❌ **Declarative orchestrator (`pipeline/declarative/orchestrator.go`)**
  - **Disabled**: This feature is currently disabled and stubbed out.

## Known Gaps & Critical Risks (machine-readable)

This table is the authoritative list of open issues, synchronized with `docs/code-review.md`.

| Severity | Issue | File(s) | Description |
| --- | --- | --- | --- |
| High | **Tight Coupling in Configuration** | `config.go` | `NewBuilderFromYAML` and `NewBuilderFromEnv` are coupled to the builder, violating SRP and duplicating logic. |
| High | **Interface Pollution & Type Safety** | `core/types.go` | The `Orchestrator` interface and `Options` struct use `any` to break import cycles, sacrificing compile-time safety. |
| High | **God Method & Magic Strings** | `pipeline/sandwich.go` | The `Execute` method has too many responsibilities (SRP violation), and the code uses brittle string literals for metadata keys. |
| Medium | **Inconsistent Factory Signatures** | `registry.go` | The `ClientFactories` map uses `any` and a custom signature, creating a type-safety hole in the registry. |
| Medium | **Inconsistent Builder API** | `builder.go` | The `With...` methods are inconsistent (some accept instances, others don't), and the client injection mechanism is unusable. |
| Medium | **Hybrid Retriever Configuration Flaws** | `internal/providers/hybrid/hybrid.go` | A key algorithm parameter (`k=60.0`) is hard-coded, and it's impossible to configure the sub-retrievers. |
| Low | **Dead Code** | `builder.go`, `registry.go` | The codebase contains unused variables (`embedderAlias`) and struct fields (`Registry.Options`) from past refactors. |
| Low | **Disabled Declarative Orchestrator** | `pipeline/declarative/orchestrator.go` | The declarative orchestrator, a key feature in the HLD, remains disabled. |

---

### Review Summary — 2025-10-15
- **Total smells identified:** 11
- **Resolved:** 4
- **Remaining:** 7
- **Key risks:**
  - **Type Safety:** The pervasive use of `any` in core interfaces and the registry undermines the benefits of Go's type system.
  - **Maintainability:** The tight coupling in `config.go` and the SRP violations in `pipeline/sandwich.go` make the code difficult to test and maintain.
  - **Extensibility:** The configuration flaws in the hybrid retriever and the OCP violation in `config.go` make it hard for users to extend or properly configure the system.