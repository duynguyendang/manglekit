# Manglekit SDK - Code Review

## Smell: Broken Resource Cleanup Lifecycle
**Location:** `builder.go` and `core/`
**Impact Analysis:** Components with resources that need explicit closing (e.g., database connections) may not be cleaned up properly on shutdown, leading to resource leaks.
**Refactoring Suggestion:** Ensure all components that manage resources implement a `Close()` method and that the builder correctly collects and calls these methods during the application's graceful shutdown sequence.
**Status:** Open

## Smell: Type Assertions in Core Component Factories
**Location:** `builder.go`
**Impact Analysis:** Using `any` and runtime type assertions for dependency injection in factories is brittle and bypasses compile-time safety checks, leading to potential runtime panics.
**Refactoring Suggestion:** Use the strongly-typed `diapi` structs for all dependency injection. Avoid `any` and type assertions in factory `Build` methods.
**Status:** Open

## Smell: Dependency Injection Bypass
**Location:** `builder.go`
**Impact Analysis:** Some components may be bypassing the formal dependency injection mechanism, making the system harder to reason about, test, and maintain.
**Refactoring Suggestion:** Enforce that all inter-component dependencies are resolved via the `diapi` structs and the builder. Forbid direct instantiation of dependencies within a factory.
**Status:** Open

## Smell: Hard-coded Dependencies in Factory (Hybrid Retriever)
**Location:** `internal/providers/hybrid/hybrid.go`
**Impact Analysis:** The hybrid retriever factory hard-codes the names of its sub-retrievers (e.g., "bm25", "dense"), preventing users from configuring different combinations.
**Refactoring Suggestion:** The list of sub-retrievers should be a configurable list of strings in the `HybridOptions` struct, allowing for dynamic composition.
**Status:** Open

## Smell: Hard-coded Magic Number (Hybrid Retriever k=60)
**Location:** `internal/providers/hybrid/hybrid.go`
**Impact Analysis:** The Reciprocal Rank Fusion constant `k` is hard-coded to 60.0, preventing users from tuning the retriever's fusion algorithm for their specific use case.
**Refactoring Suggestion:** Expose `RRF_K` as a configurable `float64` field in the `HybridOptions` struct.
**Status:** Open

## Smell: Dead Code - Declarative Orchestrator
**Location:** `pipeline/declarative/`
**Impact Analysis:** The declarative orchestrator and its related components may be unused or untested, representing dead code that increases maintenance overhead.
**Refactoring Suggestion:** Either fully integrate and test the declarative orchestrator as a first-class execution model or remove it from the codebase.
**Status:** Open

## Smell: Magic Strings for Execution Context
**Location:** `pipeline/`
**Impact Analysis:** Using magic strings as keys to pass data between pipeline stages is error-prone and lacks type safety, leading to potential runtime errors that are hard to debug.
**Refactoring Suggestion:** Introduce a strongly-typed `PipelineContext` struct to pass data between stages, providing compile-time safety and better discoverability.
**Status:** Open

## Smell: Implicit Dependency Resolution
**Location:** `builder.go`
**Impact Analysis:** The builder's reliance on the "last-built" component for dependency injection is fragile and order-dependent.
**Refactoring Suggestion:** Components should declare their dependencies by name in their `Options` struct. The builder should resolve these dependencies explicitly.
**Status:** Open

## Smell: Monolithic Build Logic (Violation of Open/Closed)
**Location:** `builder.go` (the `specTable` function)
**Impact Analysis:** The builder's `specTable` centralizes all component creation logic, violating the Open/Closed Principle.
**Refactoring Suggestion:** Abstract the build logic into a `ComponentHandler` interface. Each provider package would register a handler for its kind.
**Status:** Open

## Smell: Non-Deterministic Orchestrator
**Location:** `pipeline/sandwich.go`
**Impact Analysis:** The default orchestrator arbitrarily picks the first available component from its dependency maps, leading to unpredictable behavior.
**Refactoring Suggestion:** The `Sandwich` orchestrator should be configured with the specific names of the components it should use.
**Status:** Open

## Smell: Hard-Coded Pipeline Parameters (Tool Adapters)
**Location:** `core/tool_adapters.go`
**Impact Analysis:** Tool adapters have hard-coded parameters (e.g., `TopK`), which should be configurable at the step level.
**Refactoring Suggestion:** Allow passing a generic `Params map[string]any` to tool steps in the declarative orchestrator configuration.
**Status:** Open

## Smell: Hard-Coded Default Orchestrator
**Location:** `builder.go`
**Impact Analysis:** The builder defaults to a specific orchestrator, coupling it to one implementation. The choice should be explicit.
**Refactoring Suggestion:** Remove the hard-coded default and return an error if no orchestrator is explicitly configured.
**Status:** Open

## Smell: Redundant Builder API (WithKind)
**Location:** `builder.go`
**Impact Analysis:** The builder has a legacy `WithKind` method that bypasses the type-safe `With` method, creating an inconsistent API.
**Refactoring Suggestion:** Deprecate and remove the `WithKind` method, refactoring the config loader to use the type-safe `With` method exclusively.
**Status:** Open
