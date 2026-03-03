[![Go](https://img.shields.io/badge/Go-1.25%2B-blue?logo=go)](https://golang.org) [![License](https://img.shields.io/badge/License-Apache_2.0-yellow)](LICENSE)

# Manglekit

**Manglekit** is the **Sovereign Neuro-Symbolic Logic Kernel** for Go.

It solves the **Stochastic Runtime Paradox** of modern AI: applications require **Deterministic Reliability** (strict protocols, type safety, logic), but LLMs are inherently **Probabilistic** (creative, non-deterministic).

Manglekit bridges this gap by formalizing the agent lifecycle into an **OODA Loop** (Observe, Orient, Decide, Verify, Act) protected by a **Zero-Trust Supervisor** architecture:
1.  **The Brain (Symbolic)**: The Datalog Engine and **Tiered GenePool** that handle verifiable reasoning and Shadow Audits.
2.  **The Planner (Neural)**: The Execution Runtime (Genkit) that drafts generative plans.
3.  **The Memory (Silo)**: A persistent BadgerDB storage layer for SPO facts and vector embeddings.

## Core Capabilities

1.  **OODA Loop Execution**: Orchestrates AI workflows using a structural Observe, Orient, Decide, Verify, Act pipeline.
2.  **Shadow Audit (Self-Correction)**: The *Verify* step mathematically proves AI-generated plans against Tier 0 Axioms in the GenePool using Datalog *before* execution. If a policy is violated, the loop self-corrects using real-time generative feedback.
3.  **The Silo (Persistent Knowledge)**: Native BadgerDB integration providing high-performance SPOg (Subject-Predicate-Object-Graph) quad indexing and vector storage for long-term memory.
4.  **Source-to-Knowledge Pipeline**: Built-in extractors capable of ingesting Markdown/Code and dynamically inducing Tier 2 Datalog policies.
5.  **Deep Observability**: Fully integrated OpenTelemetry tracing that links Genkit spans directly to logic rules, showing exactly *why* a decision was made.

## System Building Blocks

| Component | Role | Responsibility |
| :--- | :--- | :--- |
| **SDK** | **Client** | The entry point. Developers use `client.SupervisedAction()` to wrap capabilities. |
| **GenePool** | **Logic Store** | Datalog files (`.dl`) defining the Tier 0, 1, and 2 "Standard Operating Procedures". |
| **The Silo** | **Persistent Memory**| BadgerDB backed SPOg quad fact and vector storage. |
| **Supervisor** | **Interceptor** | The zero-trust gateway that enforces the GenePool on every action. |
| **Adapters** | **Drivers** | Universal adapters for LLMs (Genkit), Extractors, Tools (MCP), Functions, and Resilience. |

## Getting Started

### Installation

```bash
go get github.com/duynguyendang/manglekit-wip
```

### Quick Start

This example demonstrates the **Self-Correcting Loop**. We wrap an LLM capability in a "Supervised Action".

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit-wip/core"
	"github.com/duynguyendang/manglekit-wip/sdk"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()
	_ = godotenv.Load() // Load GOOGLE_API_KEY from .env

	// 1. Initialize Client from YAML Configuration
	client, err := sdk.NewClientFromFile(ctx, "mangle.yaml")
	if err != nil {
		log.Fatalf("Client Init Failed: %v", err)
	}
	defer client.Shutdown(ctx)

	// 2. Create a supervised action
	// The Supervisor automatically checks input/output against the GenePool.
	action := client.SupervisedAction(&myLLMAction{})

	// 3. Execute with governance
	result, err := action.Execute(ctx, core.NewEnvelope("Tell me a joke about security."))
	if err != nil {
		log.Fatalf("Task Failed: %v", err)
	}

	fmt.Printf("Result: %s\n", result.Payload)
}

// myLLMAction is a placeholder for your LLM capability
type myLLMAction struct{}

func (m *myLLMAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Your LLM logic here
	return core.Envelope{Payload: "Here's a joke..."}, nil
}
```

### Configuration File (`mangle.yaml`)

```yaml
# Policy configuration
policy:
  path: "${POLICY_PATH:-./policies/main.dl}"
  evaluation_timeout: 30

# Observability configuration
observability:
  enabled: true
  service_name: "${SERVICE_NAME:-manglekit-app}"
  log_level: "${LOG_LEVEL:-info}"

# Pre-defined Actions
actions:
  llm_google:
    type: llm
    provider: google
    options:
      model: gemini-pro
      temperature: 0.7
```

### Defining Policies (`main.dl`)

Manglekit uses **Datalog** to define governance logic. It's like SQL but for rules.

```prolog
// Allow requests by default
allow(Req) :- request(Req).

// The "Quality Control" Rule
// If the joke contains "password", deny it.
deny(Req) :-
    request(Req),
    req_payload(Req, Text),
    fn:contains(Text, "password").

violation_msg("Do not mention passwords in jokes.") :- deny(Req).
```

## Architecture

Manglekit is a **Sovereign Logic Kernel** built on four core layers:

### Layer 1: The Client (SDK)

*   **Role**: Orchestrates the entire governance flow
*   **Responsibilities**: Holds configuration, manages the Cognitive Loop, and coordinates observability.
*   **Entry Point**: `manglekit.NewClient()` initializes the kernel with policy rules.

### Layer 2: The Cognitive Loop (OODA)

*   **Role**: An intelligent orchestration layer that binds logic to execution.
*   **Lifecycle**: `Observe -> Orient -> Decide -> Verify -> Act`
    *   **Observe**: Ingest raw signals and extract logical quad facts (SPOg) and embeddings into The Silo.
    *   **Orient**: Align input context against The Silo and Tiered Policy Rules.
    *   **Decide**: Generate an execution plan via the LLM Driver.
    *   **Verify**: Mathematically prove the execution plan against Datalog GenePool policies (Shadow Audit).
    *   **Act**: Safely execute capability (Tool, API Call) through the Zero-Trust Supervisor.

### Layer 3: The Zero-Trust Supervisor (Interceptor)

*   **Role**: The mechanical port that physically blocks unverified Actions.
*   **Lifecycle**: `Trace -> Check Proof -> Emit Spans`
*   **Pattern**: Middleware / Decorator for execution protocols.

### Layer 4: The Brain (Memory & Logic Store)

*   **Role**: The deterministic reasoning and storage layer.
*   **Components**:
    *   **The Silo**: Persistent BadgerDB storage for metadata, vectors, and facts (Quads).
    *   **Tiered GenePool**: Segregates policies by trust level limits (Axioms, Governance, AI-induced).
    *   **Policy Solver**: Robust Datalog Evaluator natively supporting built-in comparison mapping and stratified execution.
*   **Guarantees**: Fast (microsecond latency), deterministic, testable, verifiable.

### Universal Adapters

Bridge external libraries into the kernel:

*   **`ai` Adapter**: Wraps Google Genkit models and embedders.
*   **`func` Adapter**: Wraps native Go functions as Actions.
*   **`mcp` Adapter**: Integrates Model Context Protocol (MCP) servers.
*   **`extractor` Adapter**: Performs semantic extraction using LLMs.
*   **`vector` Adapter**: Handles vector search and retrieval operations.
*   **`resilience` Adapter**: Provides Circuit Breaker functionality for failure resilience.

#### Resilience Adapter

The `resilience` adapter provides a zero-dependency Circuit Breaker that prevents failure amplification.

```go
import (
	"time"

	"github.com/duynguyendang/manglekit-wip/adapters/resilience"
	"github.com/duynguyendang/manglekit-wip/core"
)

func main() {
	var myAction core.Action

	config := resilience.CircuitBreakerConfig{
		FailureThreshold: 5,
		ResetTimeout:     30 * time.Second,
	}

	safeAction := resilience.NewCircuitBreaker(myAction, config)
	// If myAction fails repeatedly, safeAction returns resilience.ErrCircuitOpen
}
```

## Directory Structure

```
manglekit/
├── adapters/           # Drivers for External Systems (AI, MCP, Vector)
│   ├── ai/             # Google Genkit & LLM Adapters
│   ├── knowledge/       # N-Quads/RDF Knowledge Loaders
│   ├── mcp/             # Model Context Protocol Tools
│   └── resilience/      # Circuit Breaker
├── cmd/                 # CLI Tools
│   └── mkit/            # The 'mkit' Developer Utility
├── config/              # Configuration Loading
├── core/                # Public Interfaces & Types (Action, Envelope)
├── docs/                # Architecture Documentation
├── internal/            # Private Implementation
│   ├── engine/          # The Datalog Logic Engine (Solver, Runtime)
│   ├── supervisor/      # The Governance Interceptor
│   ├── genepool/        # Tiered Policy Management
│   └── ...
├── sdk/                 # The User-Facing API (Client, Loop)
└── examples/            # Runnable Demo Projects
```

## Error Handling

Manglekit provides structured error types for proper error categorization:

```go
import (
	"errors"
	"github.com/duynguyendang/manglekit-wip/core"
)

// Check for policy violations
if core.IsPolicyViolationError(err) {
    var pve *core.PolicyViolationError
    if errors.As(err, &pve) {
        log.Printf("Blocked by policy: %s at %s", pve.Tier, pve.RuleID)
    }
}
```

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

Apache 2.0
