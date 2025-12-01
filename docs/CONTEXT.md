---
context_type: architecture_standard
project: manglekit
language: go
version: 1.0
last_updated: 2025-12-01T12:00:00Z
stability: stable
audience: humans_and_agents
---

# Manglekit: Source Code Snapshot

This document provides a strictly factual, deep-dive technical snapshot of the Manglekit (Genesis v3) codebase. It details where files are, how functions connect, and current implementation details.

## 1. The High-Level Directory Map

```text
manglekit/
├── sdk/                  # [KERNEL] The entry point and orchestration layer
│   ├── sdk.go            # Client struct, initialization, and Protect() API
│   └── loop.go           # Semantic State Machine (RunLoop) implementation
├── guard/                # [GOVERNANCE] The interception layer
│   └── guard.go          # GuardedAction decorator (Trace -> AuthZ -> Exec -> Validate)
├── engine/               # [LOGIC] The Datalog reasoning core
│   ├── solver.go         # PolicyEngine (High-level coordinator)
│   ├── runtime.go        # MangleRuntime (Low-level Datalog wrapper)
│   └── reflection.go     # ToFacts (Struct -> Fact conversion)
├── core/                 # [PUBLIC API] Interfaces and shared types (No dependencies)
│   ├── action.go         # Action interface
│   └── envelope.go       # Envelope struct
├── adapters/             # [DRIVERS] Bridges to external systems
│   ├── ai/               # Google Genkit AI models
│   ├── mcp/              # Model Context Protocol tools
│   ├── vector/           # Vector database retrievers
│   └── extractor/        # Structured data extraction
├── config/               # Configuration loading (YAML -> Struct)
└── cmd/                  # CLI tools (mkit)
```

## 2. Core Component Analysis

### 2.1 Engine (`engine/`)
The brain of the system. It translates Go objects into Datalog facts and evaluates policies.
*   **Key Structs**:
    *   `PolicyEngine` (`engine/solver.go`): The main facade. Manages `MangleRuntime`, `Tracer`, and `Logger`.
    *   `MangleRuntime` (`engine/runtime.go`): Wraps `google/mangle`. Handles parsing, stratification (`strata`), and query execution.
*   **Key Functions**:
    *   `Authorize(ctx, meta, input)`: Pre-check hook. Evaluates `deny("Req")`.
    *   `Validate(ctx, meta, output)`: Post-check hook. Evaluates `deny("Output")`.
    *   `EvaluateSteering(ctx, input)`: Determines next step (`RETRY`, `ROUTE`).
    *   `ToFacts(id, input)` (`engine/reflection.go`): Reflectively converts structs to `predicate(id, val)` facts.

### 2.2 SDK (`sdk/`)
The user-facing API and orchestration kernel.
*   **Key Structs**:
    *   `Client` (`sdk/sdk.go`): Holds the `PolicyEngine`, `Registry`, and `MemoryStore`.
*   **Key Functions**:
    *   `NewClientFromConfig`: Initializes the system from YAML.
    *   `Protect(Action)`: Wraps an Action with a `GuardedAction`.
    *   `ExecuteByName`: Entry point for the Semantic State Machine.
    *   `Call[Out]`: Generic helper for typed execution.

### 2.3 Guard (`guard/`)
The enforcement layer. It ensures no Action runs without policy checks.
*   **Key Structs**:
    *   `GuardedAction` (`guard/guard.go`): Implements `core.Action`. Wraps an inner Action.
*   **Key Logic**:
    *   `Execute`: Implements the `Trace -> Authorize -> Exec -> Validate -> Steer` lifecycle.
    *   `shouldBlock(err)`: Determines if execution should halt based on `FailureMode` (Open/Closed).

### 2.4 Adapters (`adapters/`)
Concrete implementations of `core.Action`.
*   **Key Structs**:
    *   `MCPAction` (`adapters/mcp/action.go`): Wraps an MCP Tool.
    *   `ExtractorAction` (`adapters/extractor/adapter.go`): Uses an LLM to extract JSON.
    *   `FuncAction` (`adapters/func/wrapper.go`): Wraps a native Go function.
    *   `GenkitModel` (`adapters/ai`): Wraps `ai.Model` (implied).

## 3. The Critical Path: `ExecuteByName`

Tracing the execution flow of a request through the system:

1.  **Entry**: User calls `client.ExecuteByName(ctx, "myAction", input)` in `sdk/sdk.go`.
2.  **State Machine**: Calls `runLoopInternal` in `sdk/loop.go`.
3.  **Lookup**: `runLoopInternal` retrieves the `core.Action` from `c.registry["myAction"]`.
4.  **Guard Interception**: The retrieved Action is a `guard.GuardedAction`. `Execute()` is called (`guard/guard.go`).
5.  **Tracing**: `GuardedAction` starts an OTel span `Action.myAction`.
6.  **Authorization**: `GuardedAction` calls `engine.Authorize` (`engine/solver.go`).
    *   Input is converted to facts via `ToFacts`.
    *   `runtime.ExecuteQuery` checks for `deny("Req")`.
7.  **Execution**: If authorized, `GuardedAction` calls `inner.Execute()` (the Adapter).
    *   e.g., `MCPAction` calls `tool.RunRaw`.
8.  **Validation**: `GuardedAction` calls `engine.Validate` (`engine/solver.go`).
    *   Output is converted to facts.
    *   `runtime.ExecuteQuery` checks for `deny("Output")`.
9.  **Steering**: `GuardedAction` calls `engine.EvaluateSteering`.
    *   Checks for `correction` or `next_step` facts.
10. **Loop**: `runLoopInternal` receives the result.
    *   If `RETRY`: Loops again with feedback.
    *   If `ROUTE`: Loops again with new action.
    *   If `ALLOW`: Returns result to user.

## 4. Data Structures & State

### 4.1 The Envelope (`core/envelope.go`)
The standard container for all data moving through the kernel.
*   `ID` (uuid.UUID): Unique identifier for the message.
*   `Payload` (any): The actual data (struct, string, map).
*   `Metadata` (map[string]string): Control plane signals (`decision`, `latency`, `trace_id`).
*   `SecurityLabels` ([]string): Taint tags (e.g., "secret", "pii") for information flow control.

### 4.2 Reflection (`engine/reflection.go`)
*   **Function**: `ToFacts(id string, input any) ([]string, error)`
*   **Logic**: Recursively walks Go structs/maps/slices.
*   **Output**: Generates Datalog facts like `field_name("ID", "Value")`.
*   **Tagging**: Supports `mangle:"predicate_name"` struct tags.

### 4.3 Memory & Context
*   **Lineage**: `context.Context` carries the Parent ID via `core.WithParentID` / `core.GetParentID`.
*   **Logging**: `context.Context` carries the Logger via `core.LoggerWithContext`.
*   **Session History**: Managed by `core.MemoryStore` interface.
    *   `VolatileStore` (`engine/memory/volatile.go`): In-memory map for transient sessions.
    *   `NoOpStore`: Default stateless behavior.

## 5. Technical Debt & Gaps

### 5.1 Missing Implementations
*   **Lineage Tracker**: The `LineageTracker` component is partially implemented via OTel and metadata, but lacks a dedicated graph store or query API.
*   **Action Configuration**: The `actions` section in `mangle.yaml` is defined in the schema but ignored by the loader. Actions must be registered programmatically.

### 5.2 Hardcoded / Temporary Logic
*   **Max Steps**: `runLoopInternal` has a hardcoded limit of `10` steps (`sdk/loop.go`).
*   **Magic Strings**: Predicate names (`deny`, `correction`, `next_step`) are hardcoded in `engine/solver.go`.

### 5.3 Panics
*   **`sdk.Must`**: Explicitly designed to panic on initialization errors.
*   **Examples**: Demo code in `examples/` frequently uses `panic(err)`.
*   **Safety**: `guard` and `logger` are tested to ensure they do *not* panic, but `adapters` (specifically `ai`) have been flagged in audits for potential nil pointer panics if not initialized correctly.

### 5.4 TODOs
*   A scan of the codebase reveals **0** explicit `TODO` or `FIXME` markers.

## 6. Changelog

-   **2025-11-29**: **Final Architecture Migration**. Moved root files (`manglekit.go`, `run_loop.go`) to `sdk/`. Renamed `engine/policy.go` to `engine/solver.go`. Consolidated `policies/` directory. `sdk` is now the primary entry point.
-   **2025-11-29**: **Architecture Cleanup**. Refactored `policy/` directory. Moved `evaluator` to `engine/` and `generator` to `sdk/`. `policy/` now strictly contains static assets.
-   **2025-11-29**: **Memory Subsystem**. Implemented "Stateless-by-Default" architecture. Added `MemoryMode` to `RunLoop` and `VolatileStore` for transient history.
-   **2025-11-28**: **Knowledge Integration**. The Engine now supports loading static knowledge from RDF Turtle files as Datalog facts. `ToFacts` reflection updated for standard Datalog predicates.
-   **2025-11-27**: Complete re-architecture. Removed Builder/Registry. Introduced Client/Guard/Engine model.
-   **2025-11-25**: **Smart Router**: (Legacy v0.x) Implemented dynamic dispatch.
-   **2025-11-20**: **Reflection**: Added `core/reflection`.
