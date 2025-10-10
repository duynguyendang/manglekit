[![Go](https://img.shields.io/badge/Go-1.24%2B-blue?logo=go)](https://golang.org) [![License](https://img.shields.io/badge/License-Apache_2.0-yellow)](LICENSE)

# Manglekit

**Manglekit** is a lightweight, embeddable Go framework for building **neuro-symbolic AI applications**. It integrates Genkit’s neural components—retrievers, rerankers, and LLMs—with Mangle’s symbolic reasoning engine of declarative rules and ontology.

Through both declarative and programmable orchestration, Manglekit lets developers define and control AI pipelines as logical flows. From simple “Sandwich Patterns” (Rules → RAG → Rules → LLM) to fully rule-driven workflows, every response remains grounded, explainable, and policy-compliant.

It is designed for high-performance AI applications where correctness, safety, and explainability are critical.

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
-   API keys for your chosen providers (e.g., OpenAI, Google).

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
    # For OpenAI models (LLM or Embedder)
    OPENAI_API_KEY="sk-..."

    # For Google models (LLM or Embedder)
    GOOGLE_API_KEY="AIza..."

    # For Groq's fast inference API
    GROQ_API_KEY="gsk_..."
    ```

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

### Example 1: Programmatic Setup (Sandwich Pattern)

This is the most common and type-safe way to build a MangleKit pipeline. The example below wires the Sandwich flow with Mangle pre/post rules, a BM25 retriever, and an OpenAI LLM.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/joho/godotenv"

	// This single blank import registers all standard MangleKit providers.
	_ "github.com/duynguyendang/manglekit/providers/all"
)

func main() {
	_ = godotenv.Load()
	ctx := context.Background()

	// Use the fluent builder to construct the orchestrator.
	// Components are configured with simple, type-safe options structs.
	orch, err := manglekit.NewBuilder().
		WithRules(&core.MangleOptions{
			Path:              []string{"./rules/pre.dlog", "./rules/post.dlog"},
			DefaultConverters: true,
		}).
		WithRetriever(&retrieve.BM25Options{Path: "./data"}).
		WithLLM(&llm.OpenAIOptions{Model: "gpt-4o-mini"}).
		WithTopK(6). // Global default; individual retrievers can override this.
		Build()
	if err != nil {
		log.Fatalf("Failed to build orchestrator: %v", err)
	}

	// Run a query through the pipeline.
	query := core.Query{Text: "What is MangleKit?"}
	answer, err := orch.Run(ctx, query)
	if err != nil {
		log.Fatalf("Orchestrator run failed: %v", err)
	}

	fmt.Println("Answer:", answer.Text)
}
```

---

### Example 2: Declarative Setup with YAML (Sandwich Pattern)

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
  openai:
    apiKey: "${OPENAI_API_KEY}"

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

    "github.com/duynguyendang/manglekit"
    "github.com/duynguyendang/manglekit/core"
    "github.com/joho/godotenv"

    // This single blank import registers all standard MangleKit providers.
    _ "github.com/duynguyendang/manglekit/providers/all"
)

func main() {
    _ = godotenv.Load()

    // Create a builder instance directly from the YAML file.
    builder, err := manglekit.NewBuilderFromYAML("config.yaml")
    if err != nil {
        log.Fatalf("Failed to create builder from YAML: %v", err)
    }

    // Build the orchestrator from the configuration.
    orch, err := builder.Build()
    if err != nil {
        log.Fatalf("Failed to build orchestrator: %v", err)
    }

    // Run a query.
    query := core.Query{Text: "What is MangleKit?"}
    answer, err := orch.Run(context.Background(), query)
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
  flowName: "main_flow"

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
|                   | `openai`      | Integrates with OpenAI's models (e.g., GPT-4).               | _None_             |
|                   | `groq`        | Integrates with Groq's fast inference API.                   | _None_             |
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
├── internal/               # Concrete providers (bm25, hybrid, llm, vectorstores, etc.)
│   ├── embedders/
│   ├── logger/
│   ├── providers/
│   └── vectorstores/
├── providers/              # Public registration helpers (e.g., providers/all)
├── examples/               # Runnable guides and sample data
└── docs/                   # CONTEXT.md, HLD/LLD/CSD, reviews
```

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to submit pull requests, report issues, and suggest features.

## 📜 License

This project is licensed under the Apache License, Version 2.0. See the [LICENSE](LICENSE) file for details.