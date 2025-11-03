# Manglekit SDK - Code Review

## Smell: Orchestrator Handler Coverage (Builder cannot build Sandwich)
**Location:** `internal/providers/orchestrators/orchestrators.go:28`, `pipeline/declarative/handler.go:1`
**Impact Analysis:** Only the Declarative handler is registered for kind `orchestrator`. While factories for both Sandwich and Declarative are registered, there is no handler to build Sandwich via the Builder. Configurations targeting `sandwich` will fail during handler dispatch.
**Refactoring Suggestion:** Add a generic orchestrator handler that dispatches based on options type, or register a distinct Sandwich handler akin to the declarative one.
**Status:** Resolved
**Note:** Resolved by implementing and registering a dedicated `ComponentHandler` for the Sandwich orchestrator, ensuring full coverage. (GAP-005)

## Smell: Factory Signature Mismatch (Hybrid Retriever)
**Location:** `internal/providers/hybrid/hybrid.go:35`, `internal/providers/retrievers/handler.go:63`
**Impact Analysis:** The hybrid retriever factory is registered with `D=diapi.Builder`, but the retriever handler passes `diapi.RetrieverDeps` into `Factory.Build`. This will hit a type assertion failure in the generic factory. This is one example of a broader issue where many provider factories have not been updated to accept their specific `diapi.*Deps` struct.
**Refactoring Suggestion:** Change all provider factories to accept their specific `diapi.*Deps` struct, ensuring full compliance with ADR R14.
**Status:** Open
**Note:** This issue is the subject of the final DI refactoring phase. (GAP-009)

## Smell: Registry Integrity
**Location:** `providers/all/all.go`, `internal/providers/**/register.go`
**Impact Analysis:** The accidental deletion and subsequent restoration of provider `register.go` files highlighted a process gap. Without these files, providers are not registered with the central registry, making them unavailable to the builder and causing the application to fail at startup with a "component not found" error.
**Refactoring Suggestion:** Ensure all provider directories contain a `register.go` file that correctly calls `manglekit.Register` and is included in the build. Add a CI check to verify that provider packages contain this file.
**Status:** Resolved
**Note:** The missing `register.go` files were restored, resolving the immediate build failure.

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

---
## Open Architectural Smells

## Smell: Polluted BuilderAPI
**Location:** `builder.go:L23`
**Impact Analysis:** The public `With(...)` and `WithHandlers(...)` methods on the `BuilderAPI` interface violate the "Config-First" principle (ADR-1). They create a secondary, programmatic entry point for building that bypasses the official `sdk.FromConfig` method. This leads to a confusing public API, makes configurations non-reproducible from a single YAML file, and encourages legacy patterns that are harder to maintain and debug.
**Refactoring Suggestion:** Remove the `With(...)` and `WithHandlers(...)` methods from the public `BuilderAPI` interface. The `builder.Builder` struct should be an internal implementation detail of the `sdk/` package, and `sdk.FromConfig` should be the sole public entry point for creating an orchestrator.
**Status:** Open

## Smell: Legacy Registration Pattern
**Location:** `providers/all/all.go:L17`
**Impact Analysis:** The `ComponentHandlers()` function is a remnant of the legacy programmatic building pattern. It is designed to be used with the now-prohibited `builder.WithHandlers(...)` method. Its existence is confusing for new developers, as it suggests an alternative, non-standard way of initializing the framework that contradicts the "Config-First" architecture.
**Refactoring Suggestion:** Delete the `ComponentHandlers()` function and the `providers/all/all.go` file entirely. The `sdk.Load` function should be responsible for registering the necessary production handlers directly.
**Status:** Resolved

## Smell: Non-Deterministic Type Resolution
**Location:** `builder.go:L302`
**Impact Analysis:** The `FromConfig` function iterates over the `b.registry.OptionsTypeToName` map to find the `reflect.Type` for a given component type string. Go map iteration order is not guaranteed. In the unlikely but possible scenario where two different registered types share the same name and kind string, the builder could select a different one on subsequent runs, leading to non-deterministic behavior.
**Refactoring Suggestion:** The registry should be redesigned to provide a deterministic lookup, for example by using a struct that can be sorted or a more robust mapping that prevents ambiguous entries at registration time. The lookup should not rely on a `for...range` loop over a map.
**Status:** Resolved

## Smell: Non-Deterministic Reranking Tie-Breaking
**Location:** `internal/providers/hybrid/hybrid.go:L161`
**Impact Analysis:** The hybrid retriever's `Retrieve` method iterates over a map of document scores (`scores`) to build the final list of documents. While this list is subsequently sorted by score, the relative order of documents with identical Reciprocal Rank Fusion (RRF) scores is not guaranteed because the initial iteration order is random. This violates the determinism principle (ADR-15) and can lead to inconsistent results for the same query.
**Refactoring Suggestion:** After sorting by score, add a secondary, stable sort criterion, such as the document ID, to ensure a deterministic final order. For example: `sort.SliceStable(finalDocs, ...)` followed by another sort on the ID for tie-breaking.
**Status:** Resolved

## Smell: Builder Leaking into Handler
**Location:** `pipeline/sandwich/handler.go:L33`
**Impact Analysis:** The `sandwichHandler`'s `BuildComponent` method accepts a generic `any` type for its dependency injector and immediately type-asserts it to the concrete `diapi.Builder`. This violates the Type-Safe DI rule (ADR-7 / R14), which mandates that handlers and factories must not depend on the generic builder but on specific, typed dependency structs. This tight coupling makes the handler less modular and harder to test in isolation.
**Refactoring Suggestion:** Create a new `diapi.SandwichDeps` struct that explicitly lists all the dependencies the sandwich orchestrator needs (e.g., `Retriever`, `LLMClient`, `Reranker`). The handler should resolve these dependencies from the builder and populate the `SandwichDeps` struct, which is then passed to a dedicated factory function for the orchestrator.
**Status:** Resolved
