[![Go](https://img.shields.io/badge/Go-1.24%2B-blue?logo=go)](https://golang.org) [![License](https://img.shields.io/badge/License-Apache_2.0-yellow)](LICENSE)

# Manglekit

**Manglekit** is the **Neuro-Symbolic Engine for Agents** in Go.

It adds a deterministic control plane to your probabilistic AI workflows. By combining a logic engine (Google Mangle) with an execution runtime, Manglekit allows you to define *how* an agent should behave using strict protocols, ensuring reliability without sacrificing flexibility.

Unlike heavy agent frameworks that dictate your architecture, Manglekit follows a **"Wrap, Don't Build"** philosophy. You bring your own capabilities (Genkit, LangChain, or raw APIs), and Manglekit wraps them in a **Supervised Runtime**—handling the complex state machine of retries, corrections, and routing for you.

## 🚀 Core Mechanics

1.  **Deterministic Steering**: Control LLM probabilities with Datalog logic. If the AI hallucinates logic errors, the engine corrects it.
2.  **Active Self-Correction**: Implements a native Teacher-Student loop. Errors are not just caught; they are fed back to the model for immediate recovery.
3.  **Logical Observability**: Automatic tracing of the *reasoning process*. See exactly which rule triggered a decision via OpenTelemetry.

## 🛠️ Key Features

* 🏗️ **Blueprint-Driven Logic**: Define protocols in `.dl` files. Logic is treated as data—hot-swappable and decoupled from your Go code.
* ⚡ **Supervised Actions**: Use `client.Manage()` to wrap any function. The runtime automatically handles the execution loop, context injection, and state management.
* 🪞 **Type-Safe Reflection**: Zero-config mapping of Go structs to Logic predicates. Your data models *are* your logic schema.
* 🔌 **Agnostic Driver System**: Works seamlessly with Google Genkit, MCP, or any custom provider via clean adapters.

## 🛠️ Getting Started

### Installation

```bash
go get github.com/duynguyendang/manglekit
```

### Quick Start

This example shows how to wrap a Genkit LLM model with Manglekit governance.

```go
package main

import (
    "context"
    "log"

    "github.com/duynguyendang/manglekit"
    "github.com/duynguyendang/manglekit/adapters/ai"
    "github.com/duynguyendang/manglekit/core"
    "github.com/firebase/genkit/go/plugins/googleai"
)

func main() {
    ctx := context.Background()

    // 1. Initialize Manglekit Client (The Kernel)
    // Loads policy rules from "policy.dl"
    client, err := manglekit.NewClient(ctx, manglekit.WithPolicyPath("policy.dl"))
    if err != nil {
        log.Fatal(err)
    }

    // 2. Initialize Genkit (The Driver)
    if err := googleai.Init(ctx, nil); err != nil {
        log.Fatal(err)
    }
    model := googleai.Model("gemini-1.5-flash")

    // 3. Create an Adapter
    // Wraps the Genkit model in a Manglekit Action
    llmAction := ai.NewGenkitAction("generate-content", model)

    // 4. Protect the Action
    // Wraps the adapter in a GuardedAction (Trace → AuthZ → Exec → Validate)
    safeAction := client.Protect(llmAction)

    // 5. Execute
    // The policy engine will authorize the input before Genkit is called
    input := core.NewEnvelope("Tell me a joke about security.")
    result, err := safeAction.Execute(ctx, input)
    if err != nil {
        log.Fatalf("Blocked by policy: %v", err)
    }

    log.Printf("Result: %v", result.Payload)
}
```

### Defining Policy (`policy.dl`)

Manglekit uses Datalog to define what is allowed.

```prolog
// Allow all requests by default
allow(Req) :- request(Req).

// Deny if the input contains "security" (just as an example)
deny(Req) :-
    request(Req),
    req_payload(Req, Text),
    fn:contains(Text, "security").
```

## 📦 Architecture

Manglekit v1.0 is a **Neuro-Symbolic AI Kernel** built on three core layers:

### Layer 1: The Kernel (Client)

*   **Role**: Orchestrates the entire governance flow
*   **Responsibilities**: Holds configuration, manages the Policy Engine, and coordinates observability
*   **Entry Point**: `manglekit.NewClient()` initializes the kernel with policy rules

### Layer 2: The Guard (GuardedAction)

*   **Role**: A transparent execution wrapper that enforces deterministic governance
*   **Lifecycle**: `Trace → Authorize → Execute → Validate`
    *   **Trace**: Emit OpenTelemetry span for observability
    *   **Authorize**: Pre-check using Datalog rules (deny unsafe inputs)
    *   **Execute**: Run the inner Action (LLM, DB query, Tool)
    *   **Validate**: Post-check using Datalog rules (reject unsafe outputs)
*   **Pattern**: Decorator wrapping any `core.Action`

### Layer 3: The Engine (Datalog Runtime)

*   **Role**: The deterministic reasoning layer
*   **Components**:
    *   **Solver**: Evaluates Datalog policies against facts
    *   **Reflector**: Automatically converts Go structs to Datalog facts (zero-config)
    *   **Knowledge Base**: Loads static RDF knowledge for reasoning
*   **Guarantees**: Fast (microsecond latency), deterministic, testable

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
package main

import (
	"time"

	"github.com/duynguyendang/manglekit/adapters/resilience"
	"github.com/duynguyendang/manglekit/core"
)

func main() {
	// Assume `myAction` is an existing core.Action (e.g., a Genkit action)
	var myAction core.Action

	// Configure the Circuit Breaker
	config := resilience.CircuitBreakerConfig{
		FailureThreshold: 5,                // Open circuit after 5 consecutive failures
		ResetTimeout:     30 * time.Second, // Wait 30s before probing (Half-Open)
	}

	// Wrap the action
	safeAction := resilience.NewCircuitBreaker(myAction, config)

	// Use safeAction normally
	// If myAction fails repeatedly, safeAction will fail-fast with resilience.ErrCircuitOpen
}
```

## Core Concepts

| Concept | Purpose | Example |
| :--- | :--- | :--- |
| **Action** | Universal interface for any operation | LLM call, DB query, API request |
| **Envelope** | Standardized data container | `{ID, Payload, Metadata}` |
| **GuardedAction** | Governance wrapper | `client.Protect(action)` |
| **Policy** | Datalog rules defining safety | `deny(Input) :- risk_score(Input, Score), Score > 8.` |
| **Decision** | Governance outcome | `ALLOW`, `DENY`, `RETRY`, `ROUTE` |

## 📂 Directory Structure

```text
.
├── adapters/           # Universal Adapters for external systems
│   ├── ai/             # Google Genkit AI models and embedders
│   ├── extractor/      # Semantic extraction adapter
│   ├── func/           # Native Go function wrapper
│   ├── mcp/            # Model Context Protocol adapter
│   ├── resilience/     # Circuit Breaker and resilience patterns
│   └── vector/         # Vector/retrieval adapters
├── cmd/                # CLI tools and executables
│   └── mkit/           # Manglekit CLI command tools
├── config/             # Configuration loading (YAML, schema validation)
├── core/               # Core interfaces and contracts
│   ├── action.go       # Universal Action interface
│   ├── envelope.go     # Data container for all operations
│   ├── logger.go       # Structured logging interface
│   ├── tracer.go       # OpenTelemetry integration
│   └── ...             # Additional core abstractions
├── docs/               # Architecture and design documentation
├── engine/             # Datalog Policy Engine
│   ├── solver.go       # Policy evaluation runtime
│   ├── reflection.go   # Go struct to Datalog facts converter
│   ├── memory/         # Memory store implementations
│   └── resources/      # Knowledge base management
├── examples/           # Runnable examples and demos
├── guard/              # The Guard lifecycle orchestrator
│   ├── guard.go        # GuardedAction (Trace → AuthZ → Exec → Validate)
│   └── trace.go        # Tracing utilities
├── internal/           # Internal utilities (not for public import)
│   ├── logger/         # Logger implementations (zap, stdout)
│   ├── telemetry/      # OTel infrastructure
│   └── util/           # Helper utilities
├── policies/           # Static policy assets
├── sdk/                # SDK entry points and orchestration
│   ├── sdk.go          # Main SDK initialization
│   ├── loop.go         # RunLoop execution engine
│   └── policy_generator.go  # Policy generation tools
├── manglekit.go        # Main library entry point
└── [config files]      # go.mod, mangle.yaml, Makefile, etc.
```

## 📚 Documentation

### Architecture & Design

*   **[docs/HLD.md](docs/HLD.md)**: High-level design explaining system boundaries and layering.

### Configuration & Deployment

*   **[docs/CONFIG.md](docs/CONFIG.md)**: Configuration guide with YAML setup and environment variables.
*   **[docs/TRACING.md](docs/TRACING.md)**: OpenTelemetry integration and span hierarchy.
*   **[docs/LOGGING.md](docs/LOGGING.md)**: Structured logging configuration and best practices.

### Design & Philosophy

*   **[docs/CSD.md](docs/CSD.md)**: Conceptual Solution Design and core philosophy.

### Contributing

*   **[AGENTS.md](AGENTS.md)**: Guide for AI agents working on this repo.
*   **[CONTRIBUTING.md](CONTRIBUTING.md)**: Guidelines for human contributors.

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📜 License

Apache 2.0
