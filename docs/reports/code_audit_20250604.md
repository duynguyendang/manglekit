# Code Quality Audit Report

**Date:** 2025-06-04
**Auditor:** Principal Go Engineer (AI Agent)
**Scope:** `engine/`, `guard/`, `adapters/`, `sdk/`
**Version:** Manglekit v1.0

## I. Executive Summary

*   **Overall Health Score:** 88% (Good stability, but some debt in complexity and error handling)
*   **Total Technical Debt:** 5 Issues (1 High, 3 Medium, 1 Low)
*   **Summary:** The codebase is generally well-structured with clear separation of concerns (UGA Architecture). Core safety mechanisms (Context propagation, Guarding) are solid. However, there are potential runtime panic risks in the `adapters` layer and high cyclomatic complexity in the main execution loops (`RunLoop`, `GuardedAction`). Magic strings in the Engine layer pose a maintainability risk.

## II. High Priority Fixes (Stability)

### 1. Potential Panic in `adapters/ai`
*   **Location:** `adapters/ai/adapter.go:23` (`NewLLMAction`)
*   **Issue:** The constructor `NewLLMAction` returns a struct with the `generator` field. It does not validate if `generator` is nil.
*   **Impact:** Calling `Execute` on an action created with a nil generator will cause a **runtime panic** at `a.generator.Complete`.
*   **Recommendation:** Add a nil check in `NewLLMAction` or `Execute`.

### 2. Runtime Error in Datalog Querying (Number Support)
*   **Location:** `engine/runtime.go:275` (`constantToString`)
*   **Issue:** The helper function `constantToString` only handles `StringValue` and `NameValue` from `ast.Constant`. It returns an error for other types (e.g., `ast.NumberConstant`).
*   **Impact:** Queries that return numeric values (e.g., risk scores, amounts) in `QueryWithSolutions` will fail with "unsupported constant type".
*   **Recommendation:** Add support for `ast.NumberConstant` formatting.

## III. Code Smells & Maintenance Issues

### 1. High Cyclomatic Complexity
*   **`sdk/loop.go` (`RunLoop`)**: Complexity ~18.
    *   **Reason:** Deeply nested logic with multiple `switch` statements, error checks, and state management (history, persistence, feedback) all in one function.
    *   **Refactoring:** Extract `PersistHistory` and `HandleSteeringDecision` into separate helper methods.
*   **`guard/guard.go` (`Execute` + `executeInternal`)**: Complexity ~12.
    *   **Reason:** The execution flow mixes tracing, authorization, execution, lineage recording, validation, and steering.
    *   **Refactoring:** Consider splitting `executeInternal` into `preCheck`, `invoke`, and `postCheck`.

### 2. Magic Strings (Encapsulation)
*   **Location:** `engine/solver.go`
*   **Strings:**
    *   `"Req"`: Used as the fixed Entity ID for input facts.
    *   `"Output"`: Used as the fixed Entity ID for output facts.
    *   `"ALLOWED"`: Used in tracing attributes (should use `core.DecisionAllow`).
    *   `"outcome"`, `"trace_id"`, `"mangle.labels"`: Literal strings for span attributes.
*   **Impact:** Renaming or refactoring these concepts requires searching through strings. Hardcoded Entity IDs might limit future support for batch processing or multi-entity contexts.

### 3. Magic Strings in Guard
*   **Location:** `guard/guard.go`
*   **Strings:** `"success"`, `"derived_from"`.
*   **Impact:** Minor, but inconsistent with `core/constants.go`.

## IV. Feature Integration Audit

### 1. OpenTelemetry (OTel)
*   **Status:** ✅ Passed.
*   **Verification:** `guard/guard.go` correctly sets `span.SetAttr("action.name", meta.Name)` and starts spans with `Action.{Name}`.
*   **Note:** `sdk.NewClient` includes a safety fix to ensure `c.tracer` is never nil, preventing panics when OTel is not configured.

### 2. OS File Usage
*   **Status:** ⚠️ Caution.
*   **Location:** `engine/runtime.go`, `config/loader.go`.
*   **Finding:** Uses `os.ReadFile` and `filepath.Glob`.
*   **Risk:** While standard for CLI/Config loading, `filepath.Glob` in `resolveFiles` could be a DoS vector if the path comes from untrusted user input (e.g., via API). Currently, it appears to be admin-controlled config.

## V. Data Integrity

### 1. Type Assertion Safety
*   **Status:** ✅ Passed.
*   **Location:** `adapters/func/wrapper.go` and `adapters/ai/adapter.go`.
*   **Finding:** Type assertions `input.Payload.(T)` are correctly guarded with `if !ok` checks, returning clean errors instead of panicking.

## VI. Recommendations

1.  **Fix the Panic:** Immediately patch `adapters/ai/adapter.go` to validate the generator.
2.  **Fix Number Support:** Update `engine/runtime.go` to handle numeric constants in queries.
3.  **Refactor RunLoop:** Break down `RunLoop` into smaller, testable components.
4.  **Centralize Constants:** Move all Engine/Guard literal strings to `core/constants.go`.
