[![Go](https://img.shields.io/badge/Go-1.24%2B-blue?logo=go)](https://golang.org) [![License](https://img.shields.io/badge/License-Apache_2.0-yellow)](LICENSE)

# Manglekit

**Manglekit** is the **Neuro-Symbolic Engine for Go**.

It solves the **Stochastic Runtime Paradox** of modern AI: applications require **Deterministic Reliability** (strict protocols, type safety, logic), but LLMs are inherently **Probabilistic** (creative, non-deterministic).

Manglekit bridges this gap with a **Dual-Brain Architecture**:
*   **The Left Brain (Symbolic)**: The Logic Engine (Google Mangle) that holds the **Blueprints** and performs reasoning.
*   **The Right Brain (Neural)**: The Execution Runtime (Genkit) that handles **Intuition** and generation.

It acts as a **Middleware Engine** following the *"Wrap, Don't Build"* philosophy. You bring your capabilities (Genkit flows, tools), and Manglekit wraps them in a **Supervisor** shell that handles safety, self-correction, and flow control.

## 🚀 Core Capabilities

1.  **Neuro-Symbolic Steering**: Use logic predicates to dynamically control execution flow (e.g., `next_step("escalate") :- output.confidence < 0.9.`).
2.  **Self-Correcting Loop**: Implements the **Teacher-Student Protocol**. If the AI violates a Blueprint, the engine feeds the error back as a "Correction" prompt, allowing the agent to fix itself in real-time.
3.  **Zero-Config Reflection**: **Type-Safety** at the boundary. Your Go structs *are* the schema, automatically mapped to Datalog facts for logical reasoning.
4.  **Logical Observability**: Trace the *reasoning* process. OpenTelemetry spans show exactly which **Logic Rule** triggered a decision.

## 🛠️ System Building Blocks

| Component | Role | Responsibility |
| :--- | :--- | :--- |
| **SDK** | **Client** | The entry point. Developers use client.Protect() to wrap capabilities. |
| **Blueprint** | **The Logic Store** | Datalog files (`.dl`) defining the "Standard Operating Procedures". |
| **Supervisor** | **The Interceptor** | The middleware that enforces the Blueprint on every action. |
| **Integrations** | **The Drivers** | Universal adapters for LLMs (Genkit), Tools (MCP), and Functions. |

## ⚡ Getting Started

### Installation

```bash
go get github.com/duynguyendang/manglekit
```

### Quick Start

This example demonstrates the **Self-Correcting Loop**. We wrap a Genkit model in a "Supervised Action".

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/adapters/ai"
)

func main() {
	ctx := context.Background()

	// 1. Initialize Manglekit Client (The Kernel)
	// Loads the "Blueprint" which defines the rules.
	// We use Must() to panic on initialization error for brevity.
	client := manglekit.Must(manglekit.NewClient(ctx, manglekit.WithBlueprintPath("blueprint.dl")))

	// 2. Initialize the AI Driver (Gemini via Genkit)
	gen, err := ai.NewGemini(ctx, os.Getenv("GEMINI_API_KEY"), "gemini-2.5-flash")
	if err != nil {
		log.Fatal(err)
	}

	// 3. Create & Protect the Action
	// Wrap the basic LLM generator in a "Supervised Action"
	llmAction, err := ai.NewLLMAction("jester", gen)
	if err != nil {
		log.Fatal(err)
	}
	safeAction := client.Protect(llmAction)

	// 4. Register for the Loop
	// Registering allows the engine to route and retry automatically
	client.RegisterAction("jester", safeAction)

	// 5. Execute with Self-Correction
	// The Supervisor will check the output against blueprint.dl.
	// If it violates policy, it sends feedback to Gemini and retries automatically.
	result, err := client.ExecuteByName(ctx, "jester", "Tell me a joke about security.")
	if err != nil {
		log.Fatalf("❌ Task Failed: %v", err)
	}

	fmt.Printf("✅ Result: %s\n", result.Payload)
}
```

### Defining The Blueprint (`blueprint.dl`)

Manglekit uses **Datalog** to define the logic. It's like SQL but for rules.

```prolog
// Allow requests by default
allow(Req) :- request(Req).

// The "Quality Control" Rule
// If the joke contains "password", reject it and ask for a fix.
deny(Req) :-
    request(Req),
    req_payload(Req, Text),
    fn:contains(Text, "password").
    
violation_msg("Do not mention passwords in jokes.") :- deny(Req).
```

## 📦 Architecture

Manglekit v1.0 is a **Neuro-Symbolic AI Kernel** built on three core layers:

### Layer 1: The Kernel (Client)

*   **Role**: Orchestrates the entire governance flow
*   **Responsibilities**: Holds configuration, manages the Blueprint Engine, and coordinates observability
*   **Entry Point**: `manglekit.NewClient()` initializes the kernel with policy rules

### Layer 2: The Runtime Supervisor (SupervisedAction)

*   **Role**: An intelligent orchestration layer that binds logic to execution.
*   **Lifecycle**: `Trace → Align → Run → Steer`
    *   **Trace**: Start an observable Logical Span (OpenTelemetry).
    *   **Align**: Ensure input context matches the Blueprint prerequisites (Input Alignment).
    *   **Run**: Execute the capability (LLM, Vector Search, API Call).
    *   **Steer**: Evaluate output against the Blueprint to trigger **Self-Correction (Retry)** or **Routing (Next Step)**.
*   **Pattern**: Middleware / Decorator for `core.Action`.

### Layer 3: The Engine (Datalog Runtime)

*   **Role**: The deterministic reasoning layer
*   **Components**:
    *   **Solver**: Evaluates Datalog blueprints against facts
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

## 📂 Directory Structure

```text
manglekit/
├── adapters/           # Drivers for External Systems (AI, MCP, Vector)
│   ├── ai/             # Google Genkit & LLM Adapters
│   ├── knowledge/      # N-Quads/RDF Knowledge Loaders
│   ├── mcp/            # Model Context Protocol Tools
│   └── ...
├── cmd/                # CLI Tools
│   └── mkit/           # The 'mkit' Developer Utility
├── config/             # Configuration Loading
├── core/               # Public Interfaces & Types (Action, Envelope)
├── docs/               # Architecture Documentation
├── internal/           # Private Implementation
│   ├── engine/         # The Datalog Logic Engine (Solver, Runtime)
│   ├── supervisor/     # The Governance Interceptor
│   └── ...
├── sdk/                # The User-Facing API (Client, Loop)
└── examples/           # Runnable Demo Projects
```

## 📚 Documentation

*   **[Concept & Philosophy (CSD)](docs/CSD.md)**: Read about the "Dual-Brain Architecture" and "Stochastic Paradox".
*   **[Internal Architecture (Context)](docs/CONTEXT.md)**: Deep dive into the codebase map.
*   **[Configuration](docs/CONFIG.md)**: YAML setup and environment variables.

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📜 License

Apache 2.0
