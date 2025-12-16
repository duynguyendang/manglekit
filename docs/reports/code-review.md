# Manglekit Code Review: Deep Analysis & Findings

**Author:** Antigravity (Google Deepmind)
**Date:** 2025-12-16
**Version:** 1.0.0 (Genesis)
**Target:** `UGA-ARCH` (Universal Guarded Action)

---

## 1. Executive Summary

This review analyzes the Manglekit codebase against the **Universal Guarded Action (UGA)** architecture described in `docs/CONTEXT.md`. The system successfully implements a **Neuro-Symbolic Governance Kernel**, rigorously enforcing policy-as-code (Datalog) on AI interactions.

**Verdict:** The core architecture is **Sound, Modular, and Robust**. The separation of concerns between `SDK` (Orchestration), `Core` (Contracts), and `Internal` (Logic) is well-enforced.

However, several **critical technical debts** and **non-idiomatic patterns** were identified in the Datalog Engine implementation (`internal/engine`) that require attention before v1.0 GA.

---

## 2. Architectural Alignment (The "Good")

### 2.1 Strict Domain Boundaries (ADR 5)
The code strictly follows ADR 5. The `core/` directory is pristine, containing *only* interfaces and types. This serves as a perfect "Canonical Data Model" (CDM) for the system.
*   **Evidence:** `core/logic.go` defines `Action` and `TextGenerator`, while `core/governance.go` defines `Evaluator`. No implementation details leak into `core`.

### 2.2 The UGA Lifecycle (ADR 4)
The `SupervisedAction` in `internal/supervisor/supervisor.go` correctly implements the governance loop:
1.  **Trace**: OpenTelemetry spans are started automatically.
2.  **Assess (Pre-Check)**: Queries `infeasible(Req)` or `deny(Req)`.
3.  **Execute**: Calls the inner `Action.Execute`.
4.  **Reflect (Post-Check)**: Queries `infeasible(Resp)`.
5.  **Steer**: Evaluates `retry(Hint)` or `route(Target)`.

### 2.3 Explicit AI Adapter Pattern
The `adapters/ai` package implements a clean adapter for Genkit.
*   `adapters/ai/utils.go`: Correctly prioritizes **Native Genkit Structured Output** (`ai.WithOutputType(new(T))`), ensuring type safety for AI responses.
*   It includes a fallback for standard text completion, making it robust.

---

## 3. Critical Findings & Technical Debt (The "Bad")

### 3.1 Datalog Engine "God Object" & Panics (`internal/engine/solver.go`)
The `PolicyEngine` is doing too much: lifecycle management, fact conversion, query execution, and observability.
*   **Critical Issue**: The `New()` and `NewWithObservability()` constructors **PANIC** if `std.dl` fails to load.
    ```go
    // internal/engine/solver.go:100
    panic("manglekit: failed to load std.dl: " + err.Error())
    ```
    *Recommendation*: Libraries should strictly avoid panicking. Return `(*PolicyEngine, error)` instead.

### 3.2 Non-Idiomatic Error Handling
In `evaluateGate`, the code uses a fake error `fmt.Errorf("found")` to break out of the `QueryWithSolutions` callback.
```go
// internal/engine/solver.go:485
return fmt.Errorf("found") // Stop searching
```
*Recommendation*: This is fragile. `QueryWithSolutions` should ideally support a `bool` return to signal "stop", or the loop should be handled outside the callback.

### 3.3 Hardcoded Datalog Predicates
The engine relies on "Magic Strings" for predicates:
*   `"infeasible"`, `"deny"`, `"halt"`, `"retry"`, `"route"`, `"violation_msg"`.
*   *Risk*: A typo in `solver.go` will silently break governance without compilation errors.
*   *Recommendation*: Move these to `core/types.go` as constants (e.g., `core.PredHalt = "halt"`).

### 3.4 Inconsistent Tracing Implementation
In `internal/supervisor/supervisor.go`, there is a conflict between "Legacy" and "Auto" tracing.
*   The struct holds `g.tracer core.Tracer`.
*   But `Execute` ignores it and creates a **new global tracer**:
    ```go
    // internal/supervisor/supervisor.go:95
    tracer := otel.Tracer("manglekit")
    ```
*   *Impact*: Users passing a custom `TracerProvider` via `sdk.Client` might find their settings ignored in favor of the global `otel.Tracer`.

### 3.5 Missing Streaming Support
`adapters/ai/genkit.go` is missing the `Stream` implementation:
```go
func (g *genkitAdapter) Stream(...) {
    return nil, fmt.Errorf("streaming not implemented in genkit adapter yet")
}
```
*Impact*: This limits the framework's usability for Chatbot use cases where Time-To-First-Token (TTFT) matters.

---

## 4. Go Idioms & Code Smells

### 4.1 Swallowed Errors (`internal/engine/solver.go`)
The engine frequently ignores errors from `QueryWithSolutions`. While this might be intended to handle "no results found" gracefully, it also swallows legitimate runtime errors (e.g., database connection failure, query parsing errors).
```go
// internal/engine/solver.go:493
_ = e.runtime.QueryWithSolutions(...)
```
*Recommendation*: Handle errors explicitly. Distinguish between `ErrNoResults` and system errors.

### 4.2 Getter Naming Conventions
Several internal helpers violate Go's "no Get" convention for getters.
*   `GetPlannerRules()` -> `PlannerRules()`
*   `GetStdLib()` -> `StdLib()`
*   `GetProvider()` -> `Provider()` (or `LookupProvider`)

### 4.3 Loose Typing in Options
`core.GenerateOption` is defined as `func(o any)`.
```go
type GenerateOption func(o any)
```
*Issue*: `any` removes all type safety. Passing the wrong config struct will panic at runtime or arguably worse, be ignored silentl.
*Recommendation*: Define a concrete `GenerationConfig` struct and make the option `func(*GenerationConfig)`.

### 4.4 Unbounded Channels
The `Stream` interface returns `<-chan string`.
```go
Stream(ctx context.Context, prompt string) (<-chan string, error)
```
*Issue*: Without a clear contract on who closes the channel or a cancellation mechanism (context is present, which is good), this is a potential leak source.

---

## 5. Roadmap Recommendations

1.  **Refactor Engine Constructor**: Remove `panic` and return proper errors.
2.  **Constantize Predicates**: Move all Datalog predicate strings to `core/constants.go`.
3.  **Fix Tracing**: Use the injected `g.tracer` in Supervisor, or explicitly document why the Global implementation is preferred.
4.  **Implement Streaming**: Add `Stream()` support to `genkitAdapter`.
5.  **Sentinel Errors**: Replace `fmt.Errorf("found")` with a `var ErrFound = errors.New("found")` and check using `errors.Is`.
6.  **Fix Error Swallowing**: Properly handle errors in `solver.go`.
7.  **Type-Safe Options**: Refactor `GenerateOption` to use a concrete struct.

---

**Confidence Score**: 9/10
*The codebase is architecturaly sound but requires "polish" to be considered Production-Ready idiomatic Go.*
