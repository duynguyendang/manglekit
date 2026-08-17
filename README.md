[![Go](https://img.shields.io/badge/Go-1.25%2B-blue?logo=go)](https://golang.org) [![License](https://img.shields.io/badge/License-Apache_2.0-yellow)](LICENSE)

# Manglekit

**Manglekit** is the **Sovereign Neuro-Symbolic Logic Kernel** for Go.

It solves the **Stochastic Runtime Paradox** of modern AI: applications require **Deterministic Reliability** (strict protocols, type safety, logic), but LLMs are inherently **Probabilistic** (creative, non-deterministic).

Manglekit bridges this gap by formalizing the agent lifecycle into an **OODA Loop** (Observe, Orient, Decide, Verify, Act) protected by a **Zero-Trust Supervisor** architecture:
1.  **The Brain (Symbolic)**: The Datalog Engine and **Tiered GenePool** (the `.dl` policy set) that handle verifiable reasoning and fail-closed verification.
2.  **The Planner (Neural)**: The Execution Runtime (Genkit) that drafts generative plans.
3.  **The Memory (Silo)**: A persistent BadgerDB storage layer for SPO facts and vector embeddings.

---

## Quick Start

Three steps: create a client with a policy, define a typed action, run it.
Every execution passes the zero-trust supervisor's fail-closed pre-check.

**1. Define a policy (`policy.dl`)** — Datalog rules that gate execution:

```prolog
% Payload fields tagged with `mangle:"..."` become facts at pre-check.
Decl topic(Req, Value).

% Block jokes about passwords.
halt("Req", "do not tell jokes about passwords") :-
    action_operation("Req", "tell_joke"),
    topic(Req, "passwords").
```

**2. Write the skill (`main.go`)**:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/duynguyendang/manglekit/sdk"
)

type JokeRequest struct {
    Topic string `mangle:"topic"`
}

type JokeResponse struct {
    Joke string `mangle:"joke"`
}

func main() {
    ctx := context.Background()

    // Create the client with a Datalog policy blueprint.
    client, err := sdk.NewClient(ctx, sdk.WithPolicyPath("policy.dl"))
    if err != nil {
        log.Fatalf("client init: %v", err)
    }
    defer client.Shutdown(ctx)

    // Define a supervised, type-safe action.
    joke := sdk.Define(client, "tell_joke",
        func(ctx context.Context, in JokeRequest) (JokeResponse, error) {
            return JokeResponse{Joke: "Why do programmers prefer dark mode? Because light attracts bugs."}, nil
        })

    // Execute — the supervisor checks the policy before the handler runs.
    out, err := joke.Run(ctx, JokeRequest{Topic: "security"})
    if err != nil {
        log.Fatalf("blocked or failed: %v", err)
    }
    fmt.Println(out.Joke)
}
```

**3. Run it**:

```bash
go mod init example.com/joke && go mod tidy
go run .
```

Change `Topic` to `"passwords"` and the request is blocked with a
`core.PolicyViolationError` — before the handler ever runs.

> **Scaffold instead:** `mkit skill new <name>` generates this exact layout
> (`main.go` + `policy.dl` + a contract test) for you. Install with
> `make install-cli`.

### Configuration file (optional)

Instead of options, load a YAML config with `sdk.WithConfigFile("mangle.yaml")`:

```yaml
policy:
  path: "${POLICY_PATH:-./policies/main.dl}"
  evaluation_timeout: 30

observability:
  enabled: true
  service_name: "${SERVICE_NAME:-manglekit-app}"
  log_level: "${LOG_LEVEL:-info}"
```

---

## Core Capabilities

1.  **OODA Loop Execution**: Orchestrates AI workflows using a structural Observe, Orient, Decide, Verify, Act pipeline.
2.  **Shadow Audit (Self-Correction)**: The *Verify* step evaluates AI-generated plans against the Tier 0/1/2 GenePool policies using Datalog *before* execution, in a **fail-closed** manner. If a policy is violated, the action is blocked rather than allowed through.
3.  **The Silo (Persistent Knowledge)**: Native BadgerDB integration providing high-performance SPOg (Subject-Predicate-Object-Graph) quad indexing and vector storage for long-term memory.
4.  **Extractors**: Built-in extractors capable of ingesting Markdown/Code into structured data. (Dynamic rule induction is planned — see [ROADMAP.md](./ROADMAP.md).)
5.  **Deep Observability**: Fully integrated OpenTelemetry tracing that links Genkit spans directly to logic rules, showing exactly *why* a decision was made.

## System Building Blocks

| Component | Role | Responsibility |
| :--- | :--- | :--- |
| **SDK** | **Client** | The entry point. Developers use `client.Supervise()` (or `sdk.Define`) to wrap capabilities. |
| **GenePool** | **Logic Store** | Datalog files (`.dl`) defining the Tier 0, 1, and 2 "Standard Operating Procedures" enforced by the engine. |
| **The Silo** | **Persistent Memory**| BadgerDB backed SPOg quad fact and vector storage. |
| **Supervisor** | **Interceptor** | The zero-trust gateway that enforces the GenePool on every action. |
| **Adapters** | **Drivers** | Universal adapters for LLMs (Genkit), Extractors, Tools (MCP), Functions, and Resilience. |

---

## Learn more

| Topic | Where |
|---|---|
| Building OODA applications (phases, CognitiveFrame, memory, Genkit flows, middleware) | [docs/guides/ooda.md](./docs/guides/ooda.md) |
| Datalog engine capabilities (comparisons, negation, aggregation, arithmetic) | [docs/guides/datalog.md](./docs/guides/datalog.md) |
| Runnable examples (22 demos, one directory each) | [manglekit-examples](https://github.com/duynguyendang/manglekit-examples) |
| High-level design (layers, flows, governance) | [ARCHITECTURE.md](https://github.com/duynguyendang/manglekit) workspace docs |
| CLI reference (`eval`, `gen`, `inspect`, `kg`, `run`, `serve`, `skill`) | [cmd/mkit/README.md](./cmd/mkit/README.md) |

---

## Directory Structure

```
manglekit/
├── adapters/           # Drivers for External Systems (AI, MCP, Vector)
│   ├── ai/             # Google Genkit & LLM Adapters
│   ├── knowledge/      # N-Quads/RDF Knowledge Loaders
│   ├── mcp/            # Model Context Protocol Tools
│   └── resilience/     # Circuit Breaker
├── agents/             # Reference agent (Architect)
├── cmd/                # CLI Tools
│   └── mkit/           # The 'mkit' Developer Utility
├── config/             # Configuration Loading
├── core/               # Public Interfaces & Types (Action, Envelope)
├── docs/               # Guides (OODA, Datalog)
├── internal/           # Private Implementation
│   ├── engine/         # The Datalog Logic Engine (Solver, Runtime)
│   ├── supervisor/     # The Governance Interceptor
│   └── ...
├── multiagent/         # Multi-agent runtime (AgentSystem, workflows)
├── providers/          # LLM/embedder/memory provider plugins
└── sdk/                # The User-Facing API (Client, Loop)
    └── ooda/           # OODA Loop Implementation
```

Runnable demos live in the sibling
[manglekit-examples](https://github.com/duynguyendang/manglekit-examples)
repository.

---

## Architecture

Manglekit is a **Sovereign Logic Kernel** built on four core layers:

### Layer 1: The Client (SDK)

*   **Role**: Orchestrates the entire governance flow
*   **Responsibilities**: Holds configuration, manages the Cognitive Loop, and coordinates observability.
*   **Entry Point**: `sdk.NewClient()` initializes the kernel with policy rules.

### Layer 2: The Cognitive Loop (OODA)

*   **Role**: An intelligent orchestration layer that binds logic to execution.
*   **Lifecycle**: `Observe -> Orient -> Decide -> Verify -> Act`
    *   **Observe**: Ingest raw signals and extract logical quad facts (SPOg) and embeddings into The Silo.
    *   **Orient**: Align input context against The Silo and Tiered Policy Rules.
    *   **Decide**: Generate an execution plan via the LLM Driver.
    *   **Verify**: Evaluate the execution plan against Datalog GenePool policies (fail-closed Shadow Audit).
    *   **Act**: Safely execute capability (Tool, API Call) through the Zero-Trust Supervisor.

### Layer 3: The Zero-Trust Supervisor (Interceptor)

*   **Role**: The mechanical port that physically blocks unverified Actions.
*   **Pattern**: Middleware / Decorator for execution protocols. The
    pre-flight check is fail-closed: verifier errors and Tier-0/1 violations
    block execution.

### Layer 4: The Brain (Memory & Logic Store)

*   **Role**: The deterministic reasoning and storage layer.
*   **Components**:
    *   **The Silo**: Persistent BadgerDB storage for metadata, vectors, and facts (Quads).
    *   **Tiered GenePool**: The set of `.dl` policy files the engine loads by trust level (Axioms, Governance, User). AI-induced rule learning is planned — see ROADMAP.md.
    *   **Policy Solver**: Robust Datalog Evaluator supporting comparisons (`:ge`/`:le`/`:gt`/`:lt`), negation (`!`), aggregation (`fn:sum`/`fn:max`/`fn:min`/`fn:count`), arithmetic (`fn:mult`/`fn:div`/`fn:plus`/`fn:minus`), and stratified execution.
*   **Guarantees**: Fast (microsecond latency), deterministic, testable, verifiable.

### Universal Adapters

Bridge external libraries into the kernel:

*   **`ai` Adapter**: Wraps Google Genkit models and embedders.
*   **`func` Adapter**: Wraps native Go functions as Actions.
*   **`mcp` Adapter**: Integrates Model Context Protocol (MCP) servers.
*   **`extractor` Adapter**: Performs semantic extraction using LLMs.
*   **`vector` Adapter**: Handles vector search and retrieval operations.
*   **`resilience` Adapter**: Provides Circuit Breaker functionality for failure resilience.

```go
import (
    "time"

    "github.com/duynguyendang/manglekit/adapters/resilience"
    "github.com/duynguyendang/manglekit/core"
)

func wrap(myAction core.Action) core.Action {
    config := resilience.CircuitBreakerConfig{
        FailureThreshold: 5,
        ResetTimeout:     30 * time.Second,
    }
    // If myAction fails repeatedly, the wrapper returns resilience.ErrCircuitOpen
    return resilience.NewCircuitBreaker(myAction, config)
}
```

---

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

Apache 2.0
