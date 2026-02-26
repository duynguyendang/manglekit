[![Go](https://img.shields.io/badge/Go-1.24%2B-blue?logo=go)](https://golang.org) [![License](https://img.shields.io/badge/License-Apache_2.0-yellow)](LICENSE)

# Manglekit

**Manglekit** is the **Sovereign Neuro-Symbolic Logic Kernel** for Go.

It solves the **Stochastic Runtime Paradox** of modern AI: applications require **Deterministic Reliability** (strict protocols, type safety, logic), but LLMs are inherently **Probabilistic** (creative, non-deterministic).

Manglekit bridges this gap by formalizing the agent lifecycle into an **OODA Loop** (Observe, Orient, Decide, Verify, Act) protected by a **Zero-Trust Supervisor** architecture:
1.  **The Brain (Symbolic)**: The Datalog Engine and **Tiered GenePool** that handle verifiable reasoning and Shadow Audits.
2.  **The Planner (Neural)**: The Execution Runtime (Genkit) that drafts generative plans.
3.  **The Memory (Silo)**: A persistent BadgerDB storage layer for SPO facts and SQ8-compressed vectors.

## 🚀 Core Capabilities

1.  **OODA Loop Execution**: Orchestrates AI workflows using a structural Observe, Orient, Decide, Verify, Act pipeline.
2.  **Shadow Audit (Self-Correction)**: The *Verify* step mathematically proves AI-generated plans against Tier 0 Axioms in the GenePool using Datalog *before* execution. If a policy is violated, the loop self-corrects using real-time generative feedback.
3.  **The Silo (Persistent Knowledge)**: Native BadgerDB integration providing high-performance SPOg (Subject-Predicate-Object-Graph) quad indexing and SQ8 vector storage for long-term memory.
4.  **Source-to-Knowledge Pipeline**: Built-in extractors capable of ingesting Markdown/Code and dynamically inducing Tier 2 Datalog policies.
5.  **Deep Observability**: Fully integrated trace rendering that links Genkit spans directly to logic rules, showing exactly *why* a decision was made.

## 🛠️ System Building Blocks

| Component | Role | Responsibility |
| :--- | :--- | :--- |
| **SDK** | **Client** | The entry point. Developers use `client.Supervise()` to wrap capabilities. |
| **GenePool** | **Logic Store** | Datalog files (`.dl`) defining the Tier 0, 1, and 2 "Standard Operating Procedures". |
| **The Silo** | **Persistent Memory**| BadgerDB backed SPOg quad fact and SQ8 vector storage. |
| **Supervisor** | **Interceptor** | The zero-trust gateway that enforces the GenePool on every action. |
| **Integrations** | **Drivers** | Universal adapters for LLMs (Genkit), Extractors, Tools (MCP), and Functions. |

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

	_ "github.com/duynguyendang/manglekit/providers/google" // Auto-registers the "google" provider
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()
	_ = godotenv.Load() // Load GOOGLE_API_KEY from .env

	// 1. Initialize Client from YAML Configuration
	// This loads the Blueprint policy and configures the LLM action.
	client, err := sdk.NewClientFromFile(ctx, "mangle.yaml")
	if err != nil {
		log.Fatalf("❌ Client Init Failed: %v", err)
	}
	defer client.Shutdown(ctx)

	// 2. Execute with Self-Correction
	// client.Action("jester") returns a handle usable as core.Action.
	// The Supervisor automatically checks output against blueprint.dl.
	// If it violates policy, it sends feedback to Gemini and retries.
	result, err := client.Action("jester").Execute(ctx, sdk.NewEnvelope("Tell me a joke about security."))
	if err != nil {
		log.Fatalf("❌ Task Failed: %v", err)
	}

	fmt.Printf("✅ Result: %s\n", result.Payload)
}
```

### Configuration File (`mangle.yaml`)

```yaml
actions:
  jester:
    type: "llm"
    provider: "google"
    options:
      model: "gemini-2.5-flash"
      prompt: "You are a comedian who tells jokes."
policy:
  path: "blueprint.dl"
failure_mode: "closed"
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

Manglekit v2.0 is a **Sovereign Logic Kernel** built on four core layers:

### Layer 1: The Client (SDK)

*   **Role**: Orchestrates the entire governance flow
*   **Responsibilities**: Holds configuration, manages the Cognitive Loop, and coordinates observability.
*   **Entry Point**: `manglekit.NewClient()` initializes the kernel with policy rules.

### Layer 2: The Cognitive Loop (OODA)

*   **Role**: An intelligent orchestration layer that binds logic to execution.
*   **Lifecycle**: `Observe → Orient → Decide → Verify → Act`
    *   **Observe**: Ingest raw signals and extract logical quad facts (SPOg) and embeddings into The Silo.
    *   **Orient**: Align input context against The Silo and Tiered Policy Rules.
    *   **Decide**: Generate an execution plan via the LLM Driver.
    *   **Verify**: Mathematically prove the execution plan against Datalog GenePool policies (Shadow Audit).
    *   **Act**: Safely execute capability (Tool, API Call) through the Zero-Trust Supervisor.

### Layer 3: The Zero-Trust Supervisor (Interceptor)

*   **Role**: The mechanical port that physically blocks unverified Actions. 
*   **Lifecycle**: `Trace → Check Proof → Emit Spans`
*   **Pattern**: Middleware / Decorator for execution protocols.

### Layer 4: The Brain (Memory & Logic Store)

*   **Role**: The deterministic reasoning and storage layer.
*   **Components**:
    *   **The Silo**: Persistent BadgerDB storage for metadata, vectors, and facts (Quads). Supported in `--readonly` and `--lowmem` operating modes.
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

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📜 License

Apache 2.0
