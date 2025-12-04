---
context_type: architecture_snapshot
project: manglekit
language: go
version: 1.1
last_updated: 2025-12-05T15:00:00Z
stability: stable
audience: humans_and_agents
---

# Manglekit: Source Code Snapshot

This document provides a strictly factual, deep-dive technical snapshot of the Manglekit (Genesis v3) codebase. It details where files are, how functions connect, and current implementation details.

## 1. The High-Level Directory Map

```text
manglekit/
├── sdk/                  # [KERNEL] The entry point and orchestration layer
│   ├── sdk.go            # Client struct, initialization, and RegisterAction
│   ├── action.go         # Typed Action definitions (Define, DefineDynamic)
│   ├── loop.go           # Semantic State Machine (ExecuteByName, RunLoop)
│   ├── context.go        # Context helpers (WithFact, ContextFacts)
│   ├── helpers.go        # Convenience helpers (Must)
│   ├── options.go        # Functional options for Client
│   ├── options_ext.go    # Extended ExecutionOptions (WithMetadataMap)
│   ├── tracing.go        # Tracing setup (WithStdoutTracer)
│   ├── types.go          # Type aliases
│   └── policy_generator.go # Policy Copilot (Natural Language -> Datalog)
├── guard/                # [GOVERNANCE] The interception layer
│   ├── guard.go          # GuardedAction decorator (Trace -> AuthZ -> Exec -> Validate)
│   └── guard_test.go     # Governance tests
├── engine/               # [LOGIC] The Datalog reasoning core
│   ├── solver.go         # PolicyEngine (High-level coordinator)
│   ├── runtime.go        # MangleRuntime (Low-level Datalog wrapper)
│   ├── evaluator.go      # Evaluator (Lightweight single-rule checker)
│   ├── reflection.go     # ToFacts (Struct -> Fact conversion)
│   ├── flattener.go      # Flatten (JSON -> Graph conversion)
│   ├── memory/           # Memory Store implementations
│   │   └── volatile.go   # In-Memory VolatileStore
│   └── resources/        # Knowledge Base (RDF/Turtle loading)
├── core/                 # [PUBLIC API] Interfaces and shared types (No dependencies)
│   ├── action.go         # Action interface
│   ├── envelope.go       # Envelope struct
│   ├── logger.go         # Logger interface
│   ├── logger_context.go # Context helpers (LoggerFromContext)
│   ├── tracer.go         # Tracer interface
│   ├── memory.go         # MemoryStore interface (Chat History)
│   ├── state.go          # StateProvider interface (Generic State)
│   ├── context_lineage.go # Context helpers for Lineage (ParentID)
│   ├── errors.go         # Standard error definitions (PolicyViolation)
│   ├── constants.go      # System constants (Metadata keys, Decisions)
│   └── types.go          # Shared types (Message, Query, Answer)
├── adapters/             # [DRIVERS] Bridges to external systems
│   ├── ai/               # Google Genkit AI models
│   ├── func/             # Native Go function wrapper
│   ├── mcp/              # Model Context Protocol tools (Action & Loader)
│   ├── vector/           # Vector database retrievers
│   └── extractor/        # Structured data extraction
├── config/               # Configuration loading
│   ├── schema.go         # Config struct definitions (YAML mapping)
│   └── loader.go         # Viper-based config loading logic with Env Expansion
├── internal/             # [PRIVATE] Implementation details
│   ├── logger/           # Logger adapters (Zap, Stdout)
│   ├── telemetry/        # OTel tracing setup
│   ├── statehelper/      # State manipulation helpers
│   ├── tools/            # CLI internal tools (RuleGen)
│   ├── testproviders/    # Test-only providers
│   └── util/             # Utilities (Schema validation)
└── cmd/                  # CLI tools (mkit)
    └── mkit/             # Main entry point
```

## 2. Core Component Analysis

### 2.1 Engine (`engine/`)
The brain of the system. It translates Go objects into Datalog facts and evaluates policies.
*   **Key Structs**:
    *   `PolicyEngine` (`engine/solver.go`): The main facade. Manages `MangleRuntime`, `Tracer`, and `Logger`.
    *   `MangleRuntime` (`engine/runtime.go`): Wraps `google/mangle`. Handles parsing, stratification (`strata`), and query execution. Also strips comments (`//`, `%`).
    *   `VolatileStore` (`engine/memory/volatile.go`): In-memory implementation of `core.MemoryStore` for transient sessions.
*   **Key Functions**:
    *   `Authorize(ctx, meta, input)`: Pre-check hook. Evaluates `deny("Req")`.
    *   `Validate(ctx, meta, output)`: Post-check hook. Evaluates `deny("Output")` and checks for `violation_msg(Msg)`.
    *   `EvaluateSteering(ctx, input)`: Determines next step (`RETRY` with `correction`, `ROUTE` with `next_step`).
    *   `LoadFacts(facts)`: Injects dynamic facts at runtime.
    *   `LoadKnowledge(path)`: Loads static RDF knowledge from Turtle files.
    *   `LoadFromString(rule)`: Parses and loads a single Datalog rule from a string.
    *   `ExecuteQuery(ctx, facts, query)`: Runs a raw Datalog query with tracing.
    *   `ToFacts(id, input)` (`engine/reflection.go`): Reflectively converts Go Structs to `predicate(id, val)` facts (Typed Mode).
    *   `Flatten(id, input)` (`engine/flattener.go`): Recursively converts JSON/Maps to graph facts (`json_link`, `json_val`) (Dynamic Mode).

### 2.2 SDK (`sdk/`)
The user-facing API and orchestration kernel.
*   **Key Structs**:
    *   `Client` (`sdk/sdk.go`): Holds the `PolicyEngine`, `Registry`, and `MemoryStore`.
    *   `Action[In, Out]` (`sdk/action.go`): Typed wrapper for `core.Action`.
*   **Key Functions**:
    *   `NewClient`: Initializes with functional options.
    *   `NewClientFromConfig`: Initializes from YAML.
    *   `Must`: Panics on initialization error.
    *   `Define[In, Out]`: Registers a typed Action (Typed Mode, `TypeStruct`).
    *   `DefineDynamic`: Registers a dynamic Action for `map[string]any` (Dynamic Mode, `TypeJSON`).
    *   `WithFact(ctx, key, val)`: Injects request-scoped facts into the context.
    *   `Protect(Action)`: Wraps an Action with a `GuardedAction`.
    *   `ExecuteByName`: Entry point for the Semantic State Machine (`sdk/loop.go`).
    *   `WithStdoutTracer`: ClientOption for enabling console tracing (`sdk/tracing.go`).
    *   `NewPolicyGenerator`: Creates a `Generator` to translate natural language to Datalog rules.
*   **Configuration**:
    *   `ClientOption`: `WithPolicyPath`, `WithFailureMode`, `WithLogger`, `WithMemory`.
    *   `ExecuteOption`: `WithSessionID` (Persist), `WithTransientMemory` (Volatile), `WithMetadata`, `WithMetadataMap`.

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
    *   `MCPAction` (`adapters/mcp/action.go`): Wraps an MCP Tool. Checks `initError` for availability.
    *   `MCPLoader` (`adapters/mcp/loader.go`): Discovers tools from MCP servers. Implements **Driver Resilience**:
        *   **FailOnStartup=true**: Returns error on connection failure.
        *   **FailOnStartup=false**: Logs warning, returns "Unhealthy" actions (Soft Failure).
    *   `ExtractorAction` (`adapters/extractor/adapter.go`): Uses an LLM to extract JSON.
    *   `Wrapper` (`adapters/func/wrapper.go`): Wraps a native Go function (`ToolFunc`).
    *   `GenkitModel` (`adapters/ai/genkit_model.go`): Wraps `ai.Model`.
    *   `GenkitRetrieverAdapter` (`adapters/vector/genkit_retriever.go`): Wraps a Genkit retriever.

### 2.5 Config (`config/`)
Handles configuration loading and validation.
*   **Key Functions**:
    *   `Load(path)`: Reads YAML and expands environment variables (`${VAR}`).
    *   `applyDefaults`: Sets default values (e.g., ServiceName="manglekit-app").

### 2.6 Internal (`internal/`)
Private implementation details not exposed in the public API.
*   **Logger**: `zap_adapter.go` (Zap) and `std_logger.go` (Standard Lib).
*   **Telemetry**: `otel.go` handles OpenTelemetry provider registration.
*   **Utils**: `schema/` contains JSON schema generation and validation logic.
*   **Tools**: `rulegen/` for CLI rule generation.

## 3. The Critical Path: `ExecuteByName`

Tracing the execution flow of a request through the system:

1.  **Entry**: User calls `client.ExecuteByName(ctx, "myAction", input)` in `sdk/loop.go`.
2.  **State Machine**: Calls `runLoopInternal`.
3.  **Lookup**: `runLoopInternal` retrieves the `core.Action` from `c.registry["myAction"]`.
4.  **Feedback Injection**:
    *   If `RETRY` occurred, injects `prev_feedback` (History).
    *   If `PolicyViolationError` occurred, injects `mangle_feedback` (Teacher-Student Protocol).
5.  **Guard Interception**: The retrieved Action is a `guard.GuardedAction`. `Execute()` is called (`guard/guard.go`).
6.  **Tracing**: `GuardedAction` starts an OTel span `Action.myAction`.
7.  **Authorization**: `GuardedAction` calls `engine.Authorize` (`engine/solver.go`).
    *   Input is converted to facts via `ToFacts` (Struct) or `Flatten` (JSON).
    *   `runtime.ExecuteQuery` checks for `deny("Req")`.
8.  **Execution**: If authorized, `GuardedAction` calls `inner.Execute()` (the Adapter).
    *   e.g., `MCPAction` calls `tool.RunRaw`.
9.  **Validation**: `GuardedAction` calls `engine.Validate` (`engine/solver.go`).
    *   Output is converted to facts.
    *   `runtime.ExecuteQuery` checks for `deny("Output")`.
    *   If denied, queries `violation_msg(Msg)` to return a structured `PolicyViolationError`.
10. **Steering**: `GuardedAction` calls `engine.EvaluateSteering`.
    *   Checks for `correction` (RETRY) or `next_step` (ROUTE).
11. **Loop**: `runLoopInternal` receives the result.
    *   If `RETRY`: Loops again with feedback.
    *   If `PolicyViolationError`: Catch error, set `mangle_feedback`, and RETRY (Teacher-Student).
    *   If `ROUTE`: Loops again with new action.
    *   If `ALLOW`: Returns result to user.

## 4. Data Structures & State

### 4.1 The Envelope (`core/envelope.go`)
The standard container for all data moving through the kernel.
*   `ID` (uuid.UUID): Unique identifier for the message.
*   `Payload` (any): The actual data (struct, string, map).
*   `Metadata` (map[string]string): Control plane signals (`decision`, `latency`, `trace_id`).
*   `SecurityLabels` ([]string): Taint tags (e.g., "secret", "pii") for information flow control.
*   `ContentType` (string): "STRUCT" (Typed) or "JSON" (Dynamic).

### 4.2 Reflection & Flattening
*   **Typed Mode**: Uses `engine.Reflector` (`engine/reflection.go`) to flatten structs into `field(ID, Val)` facts.
    *   **Function**: `ToFacts(id string, input any)`.
    *   **Output**: `field_name("ID", "Value")`. Optimized for Go structs.
*   **Dynamic Mode**: Uses `engine.Flattener` (`engine/flattener.go`) to traverse JSON trees.
    *   **Function**: `Flatten(id string, input any)`.
    *   **Output**: Graph facts `json_link(Parent, Key, Child)`, `json_str/num/bool`. Optimized for nested JSON.

### 4.3 Memory & Context
*   **Lineage**: `context.Context` carries the Parent ID via `core.WithParentID` / `core.GetParentID` (`core/context_lineage.go`).
*   **Facts**: `context.Context` carries request-scoped facts via `sdk.WithFact`.
*   **Logging**: `context.Context` carries the Logger via `core.LoggerWithContext`.
*   **Session History**: Managed by `core.MemoryStore` interface.
    *   `VolatileStore` (`engine/memory/volatile.go`): In-memory map for transient sessions.
    *   `NoOpStore`: Default stateless behavior.
*   **Generic State**: Managed by `core.StateProvider` interface (`core/state.go`).
    *   Allows storing arbitrary session data beyond chat history.

## 5. Technical Debt & Gaps

### 5.1 Missing Implementations
*   **Lineage Tracker**: The `LineageTracker` component is partially implemented via OTel and metadata, but lacks a dedicated graph store or query API.
*   **Action Configuration**: The `actions` section in `mangle.yaml` is defined in the schema but ignored by the loader. Actions must be registered programmatically.
*   **Config Validation**: `config.Load` performs basic YAML parsing. Semantic validation (using `internal/util/schema`) is not yet integrated into the startup flow.

### 5.2 Hardcoded / Temporary Logic
*   **Max Steps**: `runLoopInternal` has a hardcoded limit of `10` steps (`sdk/loop.go`).
*   **Magic Strings**: Predicate names (`deny`, `correction`, `next_step`) are hardcoded in `engine/solver.go`.

### 5.3 Panics
*   **`sdk.Must`**: Explicitly designed to panic on initialization errors.
*   **Examples**: Demo code in `examples/` frequently uses `panic(err)`.
*   **Safety**: `guard` and `logger` are tested to ensure they do *not* panic, but `adapters` (specifically `ai`) have been flagged in audits for potential nil pointer panics if not initialized correctly.

### 5.4 TODOs
*   A scan of the codebase reveals **0** explicit `TODO` or `FIXME` markers.

## 6. Development & Build

### 6.1 Build System (`Makefile`)
*   `make all`: Runs `fmt`, `lint`, `build`, `test`.
*   `make test`: Runs unit tests (`go test ./... -v`).
*   `make lint`: Runs `golangci-lint`.
*   `make install-cli`: Installs the `mkit` binary.

### 6.2 Key Dependencies (`go.mod`)
*   **AI Framework**: `github.com/firebase/genkit/go` (v1.2.0)
*   **Logic Engine**: `github.com/google/mangle` (v0.3.0)
*   **Observability**: `go.opentelemetry.io/otel` (v1.38.0)
*   **Validation**: `github.com/santhosh-tekuri/jsonschema/v5`
*   **Kubernetes**: `k8s.io/api` (v0.34.2)

## 7. CLI Tools (`cmd/mkit`)

The `mkit` CLI facilitates neuro-symbolic AI governance.
*   **Location**: `cmd/mkit/main.go`
*   **Commands**:
    *   `gen`: Generate rules or policies.
    *   `inspect`: Inspect data schemas or policy states.

## 8. Reference Examples (`examples/`)

| Example | Key Concepts |
| :--- | :--- |
| `dynamic_pricing` | **Numeric Logic**, Tracing, Inventory Management |
| `fintech_approval` | **Recursive Logic**, Math, Credit Scoring |
| `sre_guardrail` | **Kubernetes**, Safety Policies, Pod Validation |
| `policy-copilot` | **Natural Language to Datalog**, Policy Generation |
| `rag_flow` | **RAG**, Vector Retrieval, Knowledge Integration |
| `steering` | **Control Flow**, `next_step` logic, Routing |
| `taint_demo` | **Information Flow**, Security Labels, Taint Tracking |
| `extractor_demo` | **Structured Output**, JSON Extraction |

## 9. Changelog

-   **2025-12-05**: **Synchronization Audit**. Updated `CONTEXT.md` to match exact file structure and API surface. Added `sdk/action.go` (Define/DefineDynamic), `sdk/tracing.go`, and `engine/memory/volatile.go`. Clarified `ExecuteByName` location (`sdk/loop.go`).
-   **2025-12-05**: **Resilience Update**. Documented `MCPLoader` Soft Failure (Health Check) pattern and moved it from Technical Debt to Adapters.
-   **2025-12-05**: **Feature Sync**. Documented `sdk.WithFact` and `sdk.Must`. Added `engine.LoadFromString` and `ExecuteQuery`.
-   **2025-12-05**: **Teacher-Student Protocol**. Updated "Critical Path" to explicitly document `PolicyViolationError` handling and `mangle_feedback` injection in `runLoopInternal`.
-   **2025-12-05**: **Dual-Mode Input Strategy**. Implemented `ContentType` (STRUCT/JSON) routing in the Engine. Added `engine/flattener.go` for recursive JSON graph fact generation. Updated SDK with `Define` (Typed) and `DefineDynamic` (JSON).
-   **2025-12-05**: **Dev & Examples**. Added Build System, CLI, and Reference Examples sections to provide a complete operational picture.
-   **2025-12-05**: **Tech Debt Update**. Added notes on missing Config Validation and MCP startup error handling to Technical Debt section.
-   **2025-12-05**: **Internal Sync**. Added `internal/` directory structure (Logger, Telemetry, Utils) to map and component analysis.
-   **2025-12-05**: **Context Sync**. Updated `CONTEXT.md` to reflect `core/context_lineage.go`, `core/errors.go`, and `adapters/mcp/loader.go` details. Added `config` section.
-   **2025-12-05**: **Validation Suite**. Added comprehensive examples: `dynamic_pricing` (Tracing/Numeric Logic), `fintech_approval` (Recursive/Math), and `sre_guardrail` (K8s/Safety). Validated sub-10ms latency and native predicate support.
-   **2025-12-04**: **Reflector 2.0**. Enhanced `engine/reflection.go` to support deep traversal of Go Maps (keys as arguments), K8s-style JSON tags, and **native numeric types** (unquoted). Added `LoadFacts` to `PolicyEngine`.
-   **2025-11-29**: **Final Architecture Migration**. Moved root files (`manglekit.go`, `run_loop.go`) to `sdk/`. Renamed `engine/policy.go` to `engine/solver.go`. Consolidated `policies/` directory. `sdk` is now the primary entry point.
-   **2025-11-29**: **Architecture Cleanup**. Refactored `policy/` directory. Moved `evaluator` to `engine/` and `generator` to `sdk/`. `policy/` now strictly contains static assets.
-   **2025-11-29**: **Memory Subsystem**. Implemented "Stateless-by-Default" architecture. Added `MemoryMode` to `RunLoop` and `VolatileStore` for transient history.
-   **2025-11-28**: **Knowledge Integration**. The Engine now supports loading static knowledge from RDF Turtle files as Datalog facts. `ToFacts` reflection updated for standard Datalog predicates.
-   **2025-11-27**: Complete re-architecture. Removed Builder/Registry. Introduced Client/Guard/Engine model.
-   **2025-11-25**: **Smart Router**: (Legacy v0.x) Implemented dynamic dispatch.
-   **2025-11-20**: **Reflection**: Added `core/reflection`.
