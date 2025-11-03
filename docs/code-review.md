# Manglekit SDK - Code Review

## Smell: Orchestrator Handler Coverage (Builder cannot build Sandwich)
**Location:** `internal/providers/orchestrators/orchestrators.go:28`, `pipeline/declarative/handler.go:1`
**Impact Analysis:** Only the Declarative handler is registered for kind `orchestrator`. While factories for both Sandwich and Declarative are registered, there is no handler to build Sandwich via the Builder. Configurations targeting `sandwich` will fail during handler dispatch.
**Refactoring Suggestion:** Add a generic orchestrator handler that dispatches based on options type, or register a distinct Sandwich handler akin to the declarative one.
**Status:** Resolved
**Note:** Resolved by implementing and registering a dedicated `ComponentHandler` for the Sandwich orchestrator, ensuring full coverage. (GAP-005)

## Smell: Factory Signature Mismatch (Hybrid Retriever)
**Location:** `internal/providers/hybrid/hybrid.go:35`, `internal/providers/retrievers/handler.go:63`
**Impact Analysis:** The hybrid retriever factory is registered with `D=diapi.Builder`, but the retriever handler passes `diapi.RetrieverDeps` into `Factory.Build`. This will hit a type assertion failure in the generic factory.
**Refactoring Suggestion:** Change the hybrid factory to accept `diapi.RetrieverDeps` and implement `diapi.SubRetrieversDep` on `HybridOptions`; alternatively change the retriever handler to pass the builder (not recommended as it breaks the uniform DI contract).
**Status:** Resolved
**Note:** Resolved by refactoring the hybrid retriever's factory to correctly accept the `diapi.RetrieverDeps` struct provided by the handler, per ADR #7. (GAP-006)

## Smell: Arbitrary StateProvider Selection (Declarative)
**Location:** `pipeline/declarative/orchestrator.go:72-78`
**Impact Analysis:** The declarative orchestrator selects the first `StateProvider` from a map if present, which is non-deterministic and makes state backend choice implicit.
**Refactoring Suggestion:** Add a `stateProvider` field to declarative options (or make it part of a shared orchestrator options block) and perform explicit lookup.
**Status:** Resolved
**Note:** Resolved by adding an explicit `state_provider` field to the `declarative.Options` struct, allowing for deterministic selection. (GAP-007)

## Smell: Incomplete DI Interface
**Location:** `core/diapi/di.go`
**Impact Analysis:** The core `diapi.Builder` interface was missing getters for several component kinds, forcing handlers to perform unsafe type assertions or preventing them from resolving necessary dependencies.
**Refactoring Suggestion:** Add getters for all core component kinds to the `diapi.Builder` interface to provide a complete and safe dependency resolution surface for all handlers.
**Status:** Resolved
**Note:** Resolved by extending the `diapi.Builder` interface to include getters for all component kinds, completing the DI contract. (GAP-008)

## Smell: Arbitrary Selection of Singleton Components
**Location:** `pipeline/sandwich.go`
**Impact Analysis:** The `Sandwich` orchestrator arbitrarily selects the first available `RuleSet` and `StateProvider` from its dependency maps. If a user configures multiple components of these kinds, the behavior of the orchestrator will be non-deterministic and depend on map iteration order.
**Refactoring Suggestion:** Extend the `SandwichOptions` struct to include `ruleSet` and `stateProvider` string fields. The orchestrator factory should use these names to explicitly look up the required components, ensuring deterministic behavior.
**Status:** Resolved

## Smell: Hard-coded Dependencies in Factory (Hybrid Retriever)
**Location:** `internal/providers/hybrid/hybrid.go`
**Impact Analysis:** The hybrid retriever factory hard-codes the names of its sub-retrievers (e.g., "bm25", "dense"), preventing users from configuring different combinations. This has been partially mitigated by the new builder, but the core issue remains in the factory logic.
**Refactoring Suggestion:** The list of sub-retrievers should be a configurable list of strings in the `HybridOptions` struct, allowing for dynamic composition.
**Status:** Resolved

## Smell: Hard-coded Magic Number (Hybrid Retriever k=60)
**Location:** `internal/providers/hybrid/hybrid.go`
**Impact Analysis:** The Reciprocal Rank Fusion constant `k` is hard-coded to 60.0, preventing users from tuning the retriever's fusion algorithm for their specific use case.
**Refactoring Suggestion:** Expose `RRF_K` as a configurable `float64` field in the `HybridOptions` struct.
**Status:** Resolved

## Smell: Dead Code - Declarative Orchestrator
**Location:** `pipeline/declarative/`
**Impact Analysis:** The declarative orchestrator and its related components appear to be unused or untested in the main sandwich pipeline, representing dead code that increases maintenance overhead.
**Refactoring Suggestion:** Either fully integrate and test the declarative orchestrator as a first-class execution model or remove it from the codebase.
**Status:** Resolved

---
## Previously Resolved Smells

The following issues were identified in a previous review and have been resolved by the new handler-based builder and stage-based pipeline architecture.

## Smell: Monolithic Build Logic (Violation of Open/Closed)
**Location:** `builder.go` (the `specTable` function)
**Impact Analysis:** The builder's `specTable` centralized all component creation logic, violating the Open/Closed Principle.
**Refactoring Suggestion:** Abstract the build logic into a `ComponentHandler` interface.
**Status:** Resolved

## Smell: Non-Deterministic Orchestrator
**Location:** `pipeline/sandwich.go`
**Impact Analysis:** The default orchestrator arbitrarily picked the first available component from its dependency maps.
**Refactoring Suggestion:** The `Sandwich` orchestrator should be configured with the specific names of the components it should use.
**Status:** Resolved

## Smell: Magic Strings for Execution Context
**Location:** `pipeline/`
**Impact Analysis:** Using magic strings as keys to pass data between pipeline stages was error-prone and lacked type safety.
**Refactoring Suggestion:** Introduce a strongly-typed `PipelineContext` struct to pass data between stages.
**Status:** Resolved

## Smell: Hard-Coded Default Orchestrator
**Location:** `builder.go`
**Impact Analysis:** The builder defaulted to a specific orchestrator, coupling it to one implementation.
**Refactoring Suggestion:** Remove the hard-coded default and return an error if no orchestrator is explicitly configured.
**Status:** Resolved

## Smell: Redundant Builder API (WithKind)
**Location:** `builder.go`
**Impact Analysis:** The builder had a legacy `WithKind` method that bypassed the type-safe `With` method.
**Refactoring Suggestion:** Deprecate and remove the `WithKind` method.
**Status:** Resolved

## Smell: Implicit Dependency Resolution
**Location:** `builder.go`
**Impact Analysis:** The builder's reliance on the "last-built" component for dependency injection was fragile and order-dependent.
**Refactoring Suggestion:** Components should declare their dependencies by name in their `Options` struct. The builder should resolve these dependencies explicitly. (Partially resolved, as named resolution is now possible but not universally enforced).
**Status:** Resolved

## Smell: Broken Resource Cleanup Lifecycle
**Location:** `builder.go` and `core/`
**Impact Analysis:** Components with resources that need explicit closing were not being cleaned up properly.
**Refactoring Suggestion:** Ensure all components that manage resources implement a `Close()` method and that the builder correctly collects and calls these methods.
**Status:** Resolved

## Smell: Type Assertions in Core Component Factories
**Location:** `builder.go`
**Impact Analysis:** Using `any` and runtime type assertions for dependency injection in factories was brittle.
**Refactoring Suggestion:** Use the strongly-typed `diapi` structs for all dependency injection.
**Status:** Resolved

## Smell: Missing Core DI Component
**Location:** `core/diapi/di.go`
**Impact Analysis:** The `diapi` package, which defines the dependency injection contract, is missing a `CoreDeps` struct. This struct is a key component of the modern, typed registration pattern (`manglekit.Register[T, D, O]`) as it provides a standard way to inject core framework services (like `Observability`) into component factories. Without it, factories that need these services have no clean way to receive them, leading to inconsistent dependency injection.
**Refactoring Suggestion:** Define and export a `diapi.CoreDeps` struct in `core/diapi/di.go` containing fields for core services like `Obs core.Observability`. Update the generic `manglekit.Register` function to inject this struct into all component factories.
**Status:** Open

## Smell: Legacy Registration Pattern
**Location:** `providers/all/all.go`
**Impact Analysis:** The `providers/all/all.go` file uses a `ComponentHandlers()` function to manually collect and register a list of `core.ComponentHandler` implementations. This pattern is a holdover from the programmatic, builder-first architecture. It is incompatible with the modern "Config-First" approach, which relies on a pre-populated registry of provider factories (`manglekit.Register`) that the `sdk.FromConfig` function uses for dynamic component building. This legacy function creates architectural ambiguity and is redundant.
**Refactoring Suggestion:** Delete the `ComponentHandlers()` function. Create a new `all.Register(r *manglekit.Registry)` function that calls the individual `Register` function for each production-ready provider, populating the registry with all necessary types, factories, and handlers for the `sdk.FromConfig` workflow.
**Status:** Open

## Smell: Polluted BuilderAPI
**Location:** `builder.go`
**Impact Analysis:** The `BuilderAPI` interface includes methods like `With(...)` and `WithHandlers(...)`. These methods support a programmatic, fluent-style of building that is at odds with the "Config-First" architectural principle mandated by ADR.md. The one true entry point for the modern architecture should be `FromConfig`. The presence of these legacy methods pollutes the builder's public interface, creates confusion about the intended usage pattern, and increases the maintenance surface.
**Refactoring Suggestion:** Remove the `With(...)` and `WithHandlers(...)` methods from the `BuilderAPI` interface. Refactor the `Builder` struct to be an internal implementation detail of the `sdk.FromConfig` function, removing its export and hiding the programmatic building capabilities from the public API.
**Status:** Open
