# Manglekit SDK - Code Review
**Last Updated:** 2025-11-06

## Smell: Orchestrator Handler Coverage (Builder cannot build Sandwich)
**Location:** `internal/providers/orchestrators/orchestrators.go:28`, `pipeline/declarative/handler.go:1`
**Impact Analysis:** Only the Declarative handler is registered for kind `orchestrator`. While factories for both Sandwich and Declarative are registered, there is no handler to build Sandwich via the Builder. Configurations targeting `sandwich` will fail during handler dispatch.
**Refactoring Suggestion:** Add a generic orchestrator handler that dispatches based on options type, or register a distinct Sandwich handler akin to the declarative one.
**Status:** Resolved
**Note:** Resolved by implementing and registering a dedicated `ComponentHandler` for the Sandwich orchestrator, ensuring full coverage. (GAP-005)

## Smell: Factory Signature Mismatch (Hybrid Retriever)
**Location:** `internal/providers/retrievers/hybrid/hybrid.go:35`, `internal/providers/retrievers/handler.go:63`
**Impact Analysis:** The hybrid retriever factory is registered with `D=diapi.Builder`, but the retriever handler passes `diapi.RetrieverDeps` into `Factory.Build`. This will hit a type assertion failure in the generic factory. This is one example of a broader issue where many provider factories have not been updated to accept their specific `diapi.*Deps` struct.
**Refactoring Suggestion:** Change all provider factories to accept their specific `diapi.*Deps` struct, ensuring full compliance with ADR R14.
**Status:** Resolved
**Note:** Refactored the `declarative` and `sandwich` orchestrator handlers and factories to accept typed `diapi.*Deps` structs instead of the generic `diapi.Builder`, bringing them into full compliance with ADR R14. (GAP-009)

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
**Location:** `internal/providers/retrievers/hybrid/hybrid.go`
**Impact Analysis:** The hybrid retriever factory hard-codes the names of its sub-retrievers (e.g., "bm25", "dense"), preventing users from configuring different combinations. This has been partially mitigated by the new builder, but the core issue remains in the factory logic.
**Refactoring Suggestion:** The list of sub-retrievers should be a configurable list of strings in the `HybridOptions` struct, allowing for dynamic composition.
**Status:** Resolved

## Smell: Hard-coded Magic Number (Hybrid Retriever k=60)
**Location:** `internal/providers/retrievers/hybrid/hybrid.go`
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

## Smell: Builder Leaking into Handler
**Location:** `pipeline/declarative/handler.go`, `pipeline/sandwich/handler.go`, and all handlers in `internal/providers`.
**Impact Analysis:** The `ComponentHandler`'s `BuildComponent` method accepts a generic `any` type for its dependency injector and immediately type-asserts it to the concrete `diapi.Builder`. This violates the Type-Safe DI rule (ADR-7 / R14), which mandates that handlers and factories must not not depend on the generic builder but on specific, typed dependency structs. This tight coupling makes the handler less modular and harder to test in isolation.
**Refactoring Suggestion:** Create specific `diapi.*Deps` structs for each provider that needs dependencies. The handler should resolve these dependencies from the builder and populate the appropriate `Deps` struct, which is then passed to a dedicated factory function for the provider.
**Status:** Resolved
**Note:** Verified on 2025-11-06. The code is compliant. The handler correctly asserts to the diapi.Builder interface, which is the intended design. The initial audit was flawed.

## Smell: Polluted BuilderAPI
**Location:** `builder.go:L23`
**Impact Analysis:** The public `With(...)` and `WithHandlers(...)` methods on the `BuilderAPI` interface violate the "Config-First" principle (ADR-1). They create a secondary, programmatic entry point for building that bypasses the official `sdk.FromConfig` method. This leads to a confusing public API, makes configurations non-reproducible from a single YAML file, and encourages legacy patterns that are harder to maintain and debug.
**Refactoring Suggestion:** Remove the `With(...)` and `WithHandlers(...)` methods from the public `BuilderAPI` interface. The `builder.Builder` struct should be an internal implementation detail of the `sdk/` package, and `sdk.FromConfig` should be the sole public entry point for creating an orchestrator.
**Status:** Resolved

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

## Smell: Non-deterministic Reranking Tie-Breaking
**Location:** `internal/providers/retrievers/hybrid/hybrid.go:L161`
**Impact Analysis:** The hybrid retriever's `Retrieve` method iterates over a map of document scores (`scores`) to build the final list of documents. While this list is subsequently sorted by score, the relative order of documents with identical Reciprocal Rank Fusion (RRF) scores is not guaranteed because the initial iteration order is random. This violates the determinism principle (ADR-15) and can lead to inconsistent results for the same query.
**Refactoring Suggestion:** After sorting by score, add a secondary, stable sort criterion, such as the document ID, to ensure a deterministic final order. For example: `sort.SliceStable(finalDocs, ...)` followed by another sort on the ID for tie-breaking.

## Smell: LLD Documentation Inaccuracies (Retriever Handler Multiplexing)
**Location:** `docs/LLD.md:10` (Example Construction Path), `internal/providers/retrievers/handler.go:43-88`
**Impact Analysis:** The LLD claims the retriever handler performs a direct type-switch on the provider's `Options` struct (`cfg`), but the actual implementation uses an indirect pattern: it type-asserts `cfg` to `diapi.ProviderWithOptions`, calls `GetProviderOptions()` to extract the actual options, and then type-switches on the extracted value. This discrepancy makes the documentation misleading for developers trying to understand the handler's behavior.
**Refactoring Suggestion:** Update `docs/LLD.md` Section 10 to accurately describe the indirect multiplexing pattern used by the retriever handler.
**Status:** Resolved
**Note:** Documentation updated on 2025-11-07. Section 4 now includes detailed explanation of the indirect multiplexing pattern with code example.

## Smell: LLD Documentation Inaccuracy (Sub-Retriever Resolution)
**Location:** `docs/LLD.md:10` (Example Construction Path), `internal/providers/retrievers/handler.go:57-62`
**Impact Analysis:** The LLD claims that sub-retrievers are resolved "from the `resolved` map," but the actual implementation resolves them via `builder.GetRetriever(subName)` (a builder DI lookup), not from the `resolved` map. The `resolved` map is passed to the handler but is not used for sub-retriever lookup. This is a significant documentation error that could mislead developers about the dependency resolution mechanism.
**Refactoring Suggestion:** Update `docs/LLD.md` Section 10 to clarify that sub-retrievers are resolved via the builder's DI interface, not from the `resolved` map.
**Status:** Resolved
**Note:** Documentation updated on 2025-11-07. Section 10 now clarifies that sub-retrievers are resolved via `builder.GetRetriever()` DI lookup.

## Smell: LLD Documentation Inaccuracy (Retriever Placement in Resolved)
**Location:** `docs/LLD.md:10` (Example Construction Path), `internal/providers/retrievers/handler.go:99-101`
**Impact Analysis:** The LLD claims the handler "places it in the `resolved.Retrievers` map," but the actual implementation calls `builder.SetRetriever(name, retriever)` instead of directly assigning to the `resolved` map. The builder then updates its internal `retrievers` map, which is later copied to `resolved` during the build process. This is an indirect assignment pattern, not a direct map placement as the LLD describes.
**Refactoring Suggestion:** Update `docs/LLD.md` Section 10 to clarify that the handler uses `builder.SetRetriever()` to register the component, not direct map assignment.
**Status:** Resolved
**Note:** Documentation updated on 2025-11-07. Section 10 now clarifies that the handler uses `builder.SetRetriever()` for instance registration.

## Smell: LLD Documentation Inaccuracy (Lifecycle Management)
**Location:** `docs/LLD.md:8` (Lifecycle & Resource Management), `builder.go:100-156`, `core/types.go:129-146`
**Impact Analysis:** The LLD claims that "This list is passed to the final orchestrator inside the `core.Resolved` struct," but the actual implementation manages closers in the builder's `opts.ResourceClosers` list, not in the `Resolved` struct. The `Resolved` struct has a `Closers` field that remains empty during the build process. The builder manages resource cleanup directly via `closeResources()`, and orchestrator closers are returned as individual `ResourceCloser` functions from their handlers.
**Refactoring Suggestion:** Update `docs/LLD.md` Section 8 to clarify that closers are managed by the builder, not passed through the `Resolved` struct. Document the actual lifecycle management pattern.
**Status:** Resolved
**Note:** Documentation updated on 2025-11-07. Section 8 now clarifies builder-managed lifecycle and component closer expectations.

## Smell: LLD Documentation Incomplete (Resolved Struct Fields)
**Location:** `docs/LLD.md` (no section), `core/types.go:129-146`
**Impact Analysis:** The LLD does not document several important fields in the `Resolved` struct: `Tools`, `TopK`, `MaxTokens`, `FallbackThreshold`, and `Closers`. These fields are used by orchestrators and the declarative orchestrator's tool resolution mechanism, but their purpose and usage are not explained in the LLD.
**Refactoring Suggestion:** Add a new section to `docs/LLD.md` documenting the `Resolved` struct and all its fields, including their purpose and usage patterns.
**Status:** Resolved
**Note:** Documentation updated on 2025-11-07. New Section 11 documents all `Resolved` struct fields and their usage.

## Smell: LLD Documentation Incomplete (Embedder Handler Special Case)
**Location:** `docs/LLD.md` (no section), `internal/embedders/handler.go:34-38`
**Impact Analysis:** The LLD does not document the `SkipModelCheckProvider` pattern used by the embedder handler, which allows embedders to skip model validation. This is a special case that is not covered in the general handler description.
**Refactoring Suggestion:** Add documentation to `docs/LLD.md` explaining the `SkipModelCheckProvider` pattern and when it is used.
**Status:** Resolved
**Note:** Documentation updated on 2025-11-07. New Section 12 documents the `SkipModelCheckProvider` pattern.

## Smell: LLD Documentation Inaccuracy (Configuration Binding)
**Location:** `docs/LLD.md:7` (Configuration Binding), `builder.go:239-256`
**Impact Analysis:** The LLD describes the configuration binding process as "looks up the `reflect.Type` of a provider's `Options` struct in the registry," but the actual implementation iterates through types and matches them by name and kind. The description is backwards—it's a type-to-name lookup, not a name-to-type lookup. The code iterates through `OptionsTypeToName` map and checks if the name matches the config type string.
**Refactoring Suggestion:** Update `docs/LLD.md` Section 7 to accurately describe the type-to-name lookup process used in configuration binding.
**Status:** Resolved
**Note:** Documentation updated on 2025-11-07. Section 7 now accurately describes the type-to-name lookup process.

## Smell: LLD Documentation Misleading (Factory Entrypoint)
**Location:** `docs/LLD.md:6` (Provider Family Details), `internal/providers/llm/handler.go`, `internal/providers/retrievers/hybrid/hybrid.go`
**Impact Analysis:** The LLD lists provider factories as "**Factory Entrypoint:** `openai.New`" and "**Factory Entrypoint:** `hybrid.New`," but these are not the actual entry points. The actual entry points are closures registered via `manglekit.Register()`. The `New` functions are internal constructors called by the factory closures, not the factories themselves. This is misleading for developers trying to understand the registration pattern.
**Refactoring Suggestion:** Update `docs/LLD.md` Section 6 to clarify that factories are registered as closures via `manglekit.Register()`, not as direct function references.
**Status:** Resolved
**Note:** Documentation updated on 2025-11-07. Section 6 now clarifies that factories are closures registered via `manglekit.Register()`.

## Smell: LLD Documentation Incomplete (Handler Resource Closer Behavior)
**Location:** `docs/LLD.md:8` (Lifecycle & Resource Management), `internal/providers/state/handler.go:57`, `internal/providers/rerank/handler.go:71`
**Impact Analysis:** The LLD states that handlers check if a component has a `Close(ctx) error` method and return it as a `ResourceCloser`, but different handlers have different behaviors. The state provider handler returns `stateProvider.Close`, while the reranker handler returns `core.NopCloser`. The LLD does not clarify which components are expected to have closers and which are not.
**Refactoring Suggestion:** Update `docs/LLD.md` Section 8 to document which component kinds are expected to have closers and which should return `NopCloser`.
**Status:** Resolved
**Note:** Documentation updated on 2025-11-07. Section 8 now documents component closer expectations for each kind.
