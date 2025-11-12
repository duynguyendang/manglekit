[![Go](https://img.shields.io/badge/Go-1.24%2B-blue?logo=go)](https://golang.org) [![License](https://img.shields.io/badge/License-Apache_2.0-yellow)](LICENSE)

# Manglekit

**Manglekit** is a lightweight, embeddable Go framework for building **neuro-symbolic AI applications**. It integrates Genkit’s neural components—retrievers, rerankers, and LLMs—with Mangle’s symbolic reasoning engine of declarative rules and ontology.

Through both declarative and programmable orchestration, Manglekit lets developers define and control AI pipelines as logical flows. From simple “Sandwich Patterns” (Rules → RAG → Rules → LLM) to fully rule-driven workflows, every response remains grounded, explainable, and policy-compliant.

It is designed for high-performance AI applications where correctness, safety, and explainability are critical.

> **📖 Architecture & Implementation:** For detailed information about the system design, architectural patterns, and contribution guidelines, see [AGENTS.md](AGENTS.md), [docs/HLD.md](docs/HLD.md), [docs/LLD.md](docs/LLD.md), and [docs/ADR.md](docs/ADR.md).

## 🚀 Key Features

-   **Rules-Driven Control**: Use the Mangle Datalog engine to enforce policies, validate inputs, filter outputs, and modify behavior at runtime.
-   **Verifiable & Explainable**: Every answer can be traced back to the source documents and the rules that were applied.
-   **Pluggable Components**: Easily swap out components like Retrievers, Rerankers, LLMs, and Vector Stores. MangleKit provides built-in support for popular providers.
-   **Fluent Builder API**: A type-safe, chainable API for programmatically constructing your RAG pipeline.
-   **Declarative Configuration**: Define your entire pipeline in a YAML file for easy setup and environment management.
-   **Two Orchestration Modes**: Choose between a simple, linear "Sandwich" pipeline or a powerful, dynamic "Declarative" workflow driven by rules.
-   **High Performance**: Built in Go for low-latency, concurrent, and scalable services.

## ⚙️ Core Architectures

MangleKit offers two primary orchestration models to suit different needs.

### 1. The Sandwich Pattern (Default)

This is a robust, linear pipeline that wraps a standard RAG flow with rule evaluation stages. It's easy to configure and ideal for many common use cases.

```
User Query
    │
    ▼
[Mangle Pre-Rules]
(Validate, Normalize, Scope Query)
    │
    ▼
[Retrieve] → [Rerank]
(Fetch & Re-order Documents)
    │
    ▼
[Mangle Post-Rules]
(Filter Docs by Entitlement, PII, etc.)
    │
    ▼
[LLM Generation]
(Synthesize Answer from Vetted Docs)
    │
    ▼
Final Answer
```

### 2. The Declarative Workflow

For ultimate flexibility, the declarative orchestrator uses the Mangle engine itself to define the execution flow. The pipeline is defined as a set of Datalog facts, allowing you to create complex, conditional, and dynamic workflows without changing any Go code.

See the [Declarative Workflow](#example-3-declarative-workflow-with-yaml) section for a detailed example.

## 🛠️ Getting Started

### 1. Prerequisites

-   Go 1.24 or later.
-   API keys for your chosen providers (e.g., Google, OpenAI).

### 2. Installation

To add MangleKit to your Go project, run:

```bash
go get github.com/duynguyendang/manglekit
```

You will also need to import the `providers/all` package to register the built-in components:

```go
import _ "github.com/duynguyendang/manglekit/providers/all"
```

### 3. Environment Setup

MangleKit can be configured to read API keys from environment variables. The easiest way to manage this during development is with a `.env` file.

1.  Create a file named `.env` in the root of your project.
2.  Add your API keys to the file:

    ```env
    # For Google models (LLM or Embedder)
    GOOGLE_API_KEY="AIza..."

    # For Groq's fast inference API (compatible with OpenAI provider)
    GROQ_API_KEY="gsk_..."
    ```

    **Note on Groq:** To use Groq, configure an `openai` provider with a custom `base_url` pointing to the Groq API endpoint.

3.  Use a library like `godotenv` to load this file when your application starts.

    ```go
    import "github.com/joho/godotenv"

    func main() {
        // Load .env file from the current directory.
        // It's conventional to ignore the error if the file doesn't exist.
        _ = godotenv.Load()

        // ... rest of your application logic
    }
    ```

## 💻 Usage Examples

You can configure MangleKit either programmatically using the fluent Builder API or declaratively with a single YAML file.

---

### Example 1: Declarative Setup with YAML (Sandwich Pattern)

For easy configuration changes, you can define the entire sandwich pipeline in a `config.yaml` file.

**`config.yaml`:**

```yaml
# Set default pipeline parameters for the "sandwich" orchestrator.
orchestrator:
  type: "sandwich"

topK: 6
fallbackThreshold: 0.35

# Configure API keys and other provider-specific settings. The ${VAR} syntax
# expands environment variables at load time.
providers:
  google:
    apiKey: "${GOOGLE_API_KEY}"

rules:
  name: "mangle"
  params:
    path:
      - "./rules/kb.facts"
      - "./rules/policy.dlog"
    defaultConverters: true
    fileFirst: true

# Define the components for each stage of the pipeline
retriever:
  name: "bm25"
  params:
    path: "./data"

llm:
  name: "google"
  params:
    model: "gemini-1.5-flash"
```

**Go code to load the YAML:**

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/duynguyendang/manglekit/core"
    "github.com/duynguyendang/manglekit/sdk"
    "github.com/joho/godotenv"
)

func main() {
    _ = godotenv.Load()
    ctx := context.Background()

    // Read the YAML configuration file.
    data, err := os.ReadFile("config.yaml")
    if err != nil {
        log.Fatalf("Failed to read config file: %v", err)
    }

    // Load the orchestrator directly from the YAML config.
    orch, err := sdk.Load(ctx, data)
    if err != nil {
        log.Fatalf("Failed to load orchestrator: %v", err)
    }
    defer orch.Close(ctx)

    // Run a query.
    query := core.Query{Text: "What is MangleKit?"}
    answer, err := orch.Execute(ctx, "session-1", query)
    if err != nil {
        log.Fatalf("Orchestrator run failed: %v", err)
    }

    fmt.Println("Answer:", answer.Text)
}
```

---

### Example 2: Programmatic Setup (Sandwich Pattern)

As of ADR-010, the programmatic builder is available for advanced use cases like testing or dynamic pipeline construction. The recommended production approach is the declarative, YAML-based setup.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	ctx := context.Background()

	// 1. Get a new programmatic builder instance.
	builder, err := sdk.NewBuilder(ctx)
	if err != nil {
		log.Fatalf("failed to create builder: %v", err)
	}

	// 2. Configure and add each component by name.
	builder.
		WithOptions("rules", &core.MangleOptions{
			Path: []string{"./rules/policy.dlog"},
		}).
		WithOptions("retriever", &retrieve.BM25Options{
			Path: "./testdata/docs",
		}).
		WithOptions("llm", &llm.GoogleOptions{
			Model: "gemini-2.0-flash",
		}).
		WithOptions("sandwich", &sandwich.Options{
			LLM:       "llm",
			Retriever: "retriever",
			Rules:     "rules",
		})

	// 3. Build the final orchestrator, specifying the top-level component.
	orch, _, err := builder.Build(ctx, "sandwich", "")
	if err != nil {
		log.Fatalf("Failed to build orchestrator: %v", err)
	}
	defer orch.Close(ctx)

	// 4. Run a query.
	query := core.Query{Text: "What is Mangle?"}
	answer, err := orch.Execute(ctx, "session-123", query)
	if err != nil {
		log.Fatalf("Orchestrator run failed: %v", err)
	}

	fmt.Println("Answer:", answer.Text)
}
```

---

### Example 3: Declarative Workflow with YAML

This example showcases the power of the declarative orchestrator. The workflow is defined in Datalog (`.dlog`) files, not in Go code. This allows for incredibly dynamic and complex pipelines.

**`config.yaml`:**

```yaml
orchestrator:
  type: "declarative"
  # The 'main_flow' is defined in our .dlog files.
  flowName: "main__flow"

providers:
  google:
    apiKey: "${GOOGLE_API_KEY}"

# The main rules engine that will also act as the FlowController.
rules:
  name: "mangle"
  params:
    path:
      - "./rules/flow.dlog" # Contains the workflow definition
      - "./rules/policy.dlog"

# Define a map of named "tools" that can be used by the workflow.
tools:
  google_llm:
    provider: "google"
    params:
      model: "gemini-1.5-flash"

  doc_retriever:
    provider: "bm25"
    params:
      path: "./data"
```

**`rules/flow.dlog`:**

```prolog
// Define the stages of the 'main_flow'.
// The second argument is the execution order.
flow_stage("main_flow", "1", "retrieval_stage").
flow_stage("main_flow", "2", "llm_stage").

// Assign a configured tool to each stage.
// The tool name must match a key in the 'tools' map in config.yaml.
stage_tool("retrieval_stage", "doc_retriever").
stage_tool("llm_stage", "google_llm").
```

The Go code to run this is identical to the previous YAML example. The `NewBuilderFromYAML` function handles the complexity of parsing the declarative configuration and wiring the tools.

## 📦 Available Components

MangleKit includes a suite of built-in providers that can be configured in the builder or YAML file.

| Component Type    | Provider Name | Description                                                  | Depends On         |
| ----------------- | ------------- | ------------------------------------------------------------ | ------------------ |
| **Retriever**     | `bm25`        | Keyword-based search (Okapi BM25) over local files.          | _None_             |
|                   | `dense`       | Vector search using an embedder and vector store.            | `Embedder`, `VectorStore` |
|                   | `hybrid`      | Fuses results from `bm25` and `dense` using RRF.             | `bm25`, `dense`    |
|                   | `in-memory`   | A simple, updatable in-memory store for testing.             | _None_             |
| **Reranker**      | `cosine`      | Re-ranks documents based on cosine similarity of embeddings. | `Embedder`         |
| **LLM**           | `google`      | Integrates with Google's generative models via Genkit.       | _None_             |
|                   | `openai`      | Integrates with OpenAI's models (e.g., GPT-4) and compatible APIs like Groq (via `base_url`). | _None_             |
| **Embedder**      | `google-embedder` | Generates embeddings using Google's models.                  | _None_             |
|                   | `openai-embedder` | Generates embeddings using OpenAI's models.                  | _None_             |
| **Rules Engine**  | `mangle`      | The core Datalog engine for rules-based control.             | _None_             |
| **Schema Parser** | `jsonschema`  | Parses JSON Schema files into facts for the Mangle engine.   | _None_             |
|                   | `rdf`         | Parses RDF (Turtle) files into facts for the Mangle engine.  | _None_             |
| **Vector Store**  | `localvec`    | An in-memory vector store that persists to disk.             | `Embedder`         |

## 📂 Simplified Repository Layout

This is a high-level overview of the most important directories in the MangleKit repository.

```
.
├── builder.go              # Fluent Builder API
├── config.go               # YAML/environment loading helpers
├── registry.go             # Provider registry + Must* helpers
├── sdk.go                  # New(), Options, orchestrator wiring
│
├── core/                   # Core contracts (Doc, Query, Answer, Options, rules)
├── embed/                  # Embedder option types
├── llm/                    # LLM client options and prompts
├── retrieve/               # Public retriever option types
├── rerank/                 # Reranker interfaces and options
├── pipeline/               # Sandwich orchestrator + declarative engine
│   └── declarative/
├── internal/               # Internal logic and concrete provider implementations
│   ├── providers/          # Providers organized by Kind
│   │   ├── retrievers/     # Contains bm25, dense, hybrid, etc.
│   │   └── ...
│   ├── testproviders/      # Mock providers for testing
│   └── ...                 # Other internal packages
├── providers/              # Public registration helpers (e.g., providers/all)
├── examples/               # Runnable guides and sample data
└── docs/                   # CONTEXT.md, HLD/LLD/CSD, reviews
```

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to submit pull requests, report issues, and suggest features.

### For Developers & Contributors

Before making changes to the Manglekit codebase, please review:

- **[AGENTS.md](AGENTS.md)** — Comprehensive guide for agents and developers on architectural patterns, testing strategies, and code review checklists. Documents resolved patterns, anti-patterns to avoid, and known limitations.
- **[docs/HLD.md](docs/HLD.md)** — High-level architecture, system design, and user-facing abstractions.
- **[docs/LLD.md](docs/LLD.md)** — Low-level implementation details, handler dispatch, dependency injection, and lifecycle management.
- **[docs/ADR.md](docs/ADR.md)** — Architectural decisions and rationale behind design choices.

## 📜 License

This project is licensed under the Apache License, Version 2.0. See the [LICENSE](LICENSE) file for details.
