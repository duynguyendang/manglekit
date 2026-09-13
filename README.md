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

> **Loops are opt-in (ADR-004):** since v0.10 the OODA cognitive runtime is
> the extension package `github.com/duynguyendang/manglekit/x/ooda` — import
> `x/ooda` for loops (`ooda.Run` / `ooda.RunOODA`), `x/oodaflow` to run them
> as Genkit flows. It moved from `sdk/ooda` with the same exported names; the
> SDK itself ships governance-only and never imports `x/`.

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
2.  **Shadow Audit (Fail-Closed Governance)**: The gate evaluates every
    action's facts against the tiered policy (T0–T3) using Datalog *before*
    execution — and reflects on the output *after* it. Both checks are
    **fail-closed**: a broken verifier blocks the action. Violations at T0/T1
    block; rules tagged T2/T3 are advisory (logged, not blocking) — tier
    semantics are real, so learned playbook rules cannot silently hard-block
    production traffic. A blocking-tier violation surfaces to the caller as a
    structured `core.PolicyViolationError` — before the handler ever runs.
3.  **The Silo (Persistent Knowledge)**: Native BadgerDB integration providing high-performance SPOg (Subject-Predicate-Object-Graph) quad indexing and vector storage for long-term memory.
4.  **Rule Learning**: Extractors ingest Markdown/code into structured data. Offline rule induction ships today as `mkit gen` (Teacher-Student loop with syntax validation); the optional `x/genes` extension packages signed learned rules that enter enforcement only through the official policy channel (`LoadPolicy`), with hard tiers requiring an explicit human-review opt-in.
5.  **Deep Observability**: Fully integrated OpenTelemetry tracing that links Genkit spans directly to logic rules, showing exactly *why* a decision was made.
6.  **The Kernel Eats Its Own Dogfood**: manglekit learns policy from its own sources. A deterministic scanner induces *signed* advisory genes (`x/genes`, tiers T3/T2) from shipped code; the induced pool is a committed, human-reviewed artifact; CI (`policy-hygiene.yml`) fails any PR whose code changes what the repo teaches itself without a re-reviewed pool. Machine-derived policy reaches enforcement through exactly one door — the policy channel — and hard tiers require an explicit human opt-in.

    ```text
    code signals ─▶ signed T3 gene ─▶ reviewed pool ─▶ human --confirm ─▶ T1
    (learn_from_code, manglekit-examples)               (this repo gated by it)
    ```

## System Building Blocks

| Component | Role | Responsibility |
| :--- | :--- | :--- |
| **SDK** | **Client** | The entry point. Developers use `client.Supervise()` (or `sdk.Define`) to wrap capabilities. |
| **GenePool** | **Logic Store** | Datalog files (`.dl`) defining the Tier 0–3 "Standard Operating Procedures" enforced by the engine. (Terminology for the policy set — tiers are enforced natively; learned-rule packaging lives in `x/genes`.) |
| **The Silo** | **Persistent Memory**| BadgerDB backed SPOg quad fact and vector storage. |
| **Supervisor** | **Interceptor** | The zero-trust gateway that enforces the GenePool on every action. |
| **Adapters** | **Drivers** | Universal adapters for LLMs (Genkit), Extractors, Tools (MCP), Functions, and Resilience. |
| **`x/` extensions** | **Optional layers** | Public extensions the core never imports: `x/ooda` (OODA cognitive-loop runtime), `x/oodaflow` (OODA-as-Genkit-flow bridge), `x/agents` (reference Architect agent), `x/east` (EAST-steered generation) and `x/genes` (signed learned-rule packaging). |

---

## Learn more

| Topic | Where |
|---|---|
| Building OODA applications (phases, CognitiveFrame, memory, Genkit flows, middleware) | [docs/guides/ooda.md](./docs/guides/ooda.md) |
| Datalog engine capabilities (comparisons, negation, aggregation, arithmetic) | [docs/guides/datalog.md](./docs/guides/datalog.md) |
| Runnable examples (22 demos under six domain folders — incl. `skill_learning` and `learn_from_code`) | [manglekit-examples](https://github.com/duynguyendang/manglekit-examples) |
| Governance features, proven running (tiers vs binary gates, explainable denies, hot reload, streaming coverage, CI exit codes, signed genes) | [examples — Proof points](https://github.com/duynguyendang/manglekit-examples#proof-points) |
| High-level design (layers, flows, governance) | [ARCHITECTURE.md](https://github.com/duynguyendang/manglekit) workspace docs |
| CLI reference (`eval`, `gen`, `check`, `inspect`, `kg`, `run`, `serve`, `skill`) | [cmd/mkit/README.md](./cmd/mkit/README.md) |

---

## Directory Structure

```
manglekit/
├── adapters/           # Drivers for External Systems
│   ├── ai/             # Google Genkit bridge (actions, streaming gate, middleware)
│   ├── extractor/      # LLM-driven structured extraction into Go types
│   ├── func/           # Plain Go functions → supervised Actions
│   ├── knowledge/      # N-Quads/N-Triples/TTL knowledge loaders
│   ├── mcp/            # Model Context Protocol tools (policy-gated)
│   ├── resilience/     # Circuit breaker
│   ├── storage/        # BadgerDB quads (MEB bridge), session stores
│   └── vector/         # Vector store + Genkit retriever
├── cmd/                # CLI Tools
│   └── mkit/           # The 'mkit' Developer Utility
├── config/             # Configuration Loading (mangle.yaml)
├── core/               # Public Interfaces & Types (Action, Envelope, Errors)
├── docs/               # Guides (OODA, Datalog)
├── internal/           # Private Implementation
│   ├── engine/         # The Datalog Logic Engine (Solver, Runtime)
│   ├── supervisor/     # The Governance Interceptor
│   └── ...
├── multiagent/         # Multi-agent runtime (AgentSystem, workflows)
├── providers/          # LLM/embedder/memory provider plugins
├── scenario/           # BDD-style policy-scenario harness
├── sdk/                # The User-Facing API (Client, Options)
│   └── ports/          # Extension contracts (TransientStore, ReasoningPort, …)
├── testutil/           # Deterministic mocks for consumer test suites
└── x/                  # Optional public extensions (core never imports x/)
    ├── agents/         # Reference agent (Architect) + toolkit
    ├── east/           # EAST (v4) generation steering
    ├── genes/          # Signed learned-rule packaging → policy channel
    ├── ooda/           # OODA loop runtime (frame, chassis, registry)
    └── oodaflow/       # OODA-as-Genkit-flow bridge
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
    The loop ships as the opt-in extension `x/ooda` (ADR-004, v0.10);
    the governance core (Layers 1, 3, 4) works without it.
*   **Lifecycle**: `Observe -> Orient -> Decide -> Verify -> Act`
    *   **Observe**: Ingest raw signals and extract logical quad facts (SPOg) and embeddings into The Silo.
    *   **Orient**: Align input context against The Silo and Tiered Policy Rules.
    *   **Decide**: Generate an execution plan via the LLM Driver.
    *   **Verify**: Evaluate the execution plan against Datalog GenePool policies (fail-closed Shadow Audit).
    *   **Act**: Safely execute capability (Tool, API Call) through the Zero-Trust Supervisor.

### Layer 3: The Zero-Trust Supervisor (Interceptor)

*   **Role**: The mechanical port that physically blocks unverified Actions.
*   **Pattern**: Middleware / Decorator for execution protocols. Both gates
    are **fail-closed**: verifier/engine errors always block (`SupervisorError`);
    policy denies block at Tier-0/1 (`PolicyViolationError`), while
    explicitly-tagged Tier-2/3 rules stay advisory. Payload facts are
    resource-capped, and every decision carries an audit trail
    (`Explain`/`--explain`).

### Layer 4: The Brain (Memory & Logic Store)

*   **Role**: The deterministic reasoning and storage layer.
*   **Components**:
    *   **The Silo**: Persistent BadgerDB storage for metadata, vectors, and facts (Quads).
    *   **Tiered policy ("GenePool")**: The set of `.dl` policy files the engine loads by trust level (T0 axiom, T1 governance, T2 playbook, T3 user). Learned rules reach enforcement only as policy source: offline `mkit gen` today, signed `x/genes` packaging with human-review promotion, or app-side runtime adaptation via `ooda.Memory` (see the `skill_learning` example).
    *   **Policy Solver**: Deterministic Datalog evaluator — comparisons (`:ge`/`:le`/`:gt`/`:lt`), negation (`!`), aggregation (`fn:sum`/`fn:max`/`fn:min`/`fn:group_by`), stratified execution, external Go predicates, temporal facts, EXPLAIN proofs, and hot reload (engine builtins survive reloads).
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
