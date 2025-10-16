# Code Review: Decoupling Config Loading from Builder

**Author:** Jules, Senior Go Architect
**Date:** 2025-10-16
**Status:** Complete

---

## 1. Summary of Changes

This refactoring addresses a critical Single Responsibility Principle (SRP) violation in the Manglekit SDK's initialization process. Previously, the `builder.go` and the root `config.go` files were tightly coupled, with the builder being responsible for both loading/parsing configuration (from YAML/env) and wiring the application's dependencies.

The key changes are:
1.  **Introduced a dedicated `config` package:** This package is now solely responsible for defining the configuration schema (`config.Config`), loading it from sources, normalizing defaults, and performing validation.
2.  **Created a single, clean entrypoint `NewBuilderFromConfig`:** This function, located in `from_config.go`, acts as the sole bridge between a validated configuration and the builder. Its only job is to translate the `config.Config` struct into a series of `With...` calls on the builder.
3.  **Removed Legacy Code:** The old `NewBuilderFromYAML` and `NewBuilderFromEnv` functions have been deleted, and the `Builder` struct has been stripped of all configuration-loading responsibilities.
4.  **Enforced a One-Way Dependency:** The `config` package has zero knowledge of the `builder` or any other part of the application. Only the `builder` imports `config`, ensuring a clean, acyclic dependency graph.

## 2. Code Smell Analysis: SRP Violation

The primary code smell addressed was the **Single Responsibility Principle (SRP) Violation**.

### Before Refactoring:

-   `config.go` contained `NewBuilderFromYAML` and `NewBuilderFromEnv`.
-   These functions would:
    1.  Read a file or environment variables.
    2.  Parse YAML/JSON.
    3.  Instantiate a `Builder`.
    4.  Use reflection and `json.Unmarshal` to dynamically create provider `Options` structs.
    5.  Call the builder's `With...` methods to wire up the components.
    6.  Resolve relative file paths within the configuration.

This gave the configuration loading logic intimate knowledge of the builder's internal structure and made the entire process brittle and hard to test.

### After Refactoring:

The responsibilities are now clearly separated:

-   **`config` package:** What is the configuration? (Schema, defaults, validation)
-   **`from_config.go`:** How does a configuration map to a builder? (Translation logic)
-   **`builder.go`:** How are components wired together programmatically? (Fluent API)

This separation makes the system more modular, easier to understand, and significantly easier to test in isolation.

## 3. Config vs. Builder SRP Checklist

This checklist should be used for future PRs to ensure the clean separation is maintained:

-   [x] **Is all configuration schema defined in the `config` package?**
-   [x] **Is all loading (YAML/env) logic contained within the `config` package?**
-   [x] **Are all default values applied within `config.Normalize()`?**
-   [x] **Is all semantic validation performed within `config.Validate()`?**
-   [x] **Does the `builder` only accept fully-formed, typed `Options` structs?**
-   [x] **Is `NewBuilderFromConfig` the *only* function that translates a `Config` object into builder calls?**
-   [x] **Does the `config` package have any `import` statements pointing to the `builder` or other application packages?** (This should always be **false**).

This refactoring establishes a robust and maintainable foundation for the Manglekit SDK's configuration system.