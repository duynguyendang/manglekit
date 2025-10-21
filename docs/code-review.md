# Manglekit SDK - Code Review

## Smell: Arbitrary Selection of Singleton Components
**Location:** `pipeline/sandwich.go`
**Impact Analysis:** The `Sandwich` orchestrator arbitrarily selects the first available `RuleSet` and `StateProvider` from its dependency maps. If a user configures multiple components of these kinds, the behavior of the orchestrator will be non-deterministic and depend on map iteration order.
**Refactoring Suggestion:** Extend the `SandwichOptions` struct to include `ruleSet` and `stateProvider` string fields. The orchestrator factory should use these names to explicitly look up the required components, ensuring deterministic behavior.
**Status:** Open

## Smell: Hard-coded Dependencies in Factory (Hybrid Retriever)
**Location:** `internal/providers/hybrid/hybrid.go`
**Impact Analysis:** The hybrid retriever factory hard-codes the names of its sub-retrievers (e.g., "bm25", "dense"), preventing users from configuring different combinations. This has been partially mitigated by the new builder, but the core issue remains in the factory logic.
**Refactoring Suggestion:** The list of sub-retrievers should be a configurable list of strings in the `HybridOptions` struct, allowing for dynamic composition.
**Status:** Open

## Smell: Hard-coded Magic Number (Hybrid Retriever k=60)
**Location:** `internal/providers/hybrid/hybrid.go`
**Impact Analysis:** The Reciprocal Rank Fusion constant `k` is hard-coded to 60.0, preventing users from tuning the retriever's fusion algorithm for their specific use case.
**Refactoring Suggestion:** Expose `RRF_K` as a configurable `float64` field in the `HybridOptions` struct.
**Status:** Open

## Smell: Dead Code - Declarative Orchestrator
**Location:** `pipeline/declarative/`
**Impact Analysis:** The declarative orchestrator and its related components appear to be unused or untested in the main sandwich pipeline, representing dead code that increases maintenance overhead.
**Refactoring Suggestion:** Either fully integrate and test the declarative orchestrator as a first-class execution model or remove it from the codebase.
**Status:** Open

---
## Resolved Smells

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
