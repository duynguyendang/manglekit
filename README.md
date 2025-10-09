[![Go](https://img.shields.io/badge/Go-1.21%2B-blue?logo=go)](https://golang.org) [![License](https://img.shields.io/badge/License-Apache_2.0-yellow)](LICENSE)

# Manglekit

**MangleKit** is a lightweight, embeddable **Go** framework for building robust and controllable **Retrieval-Augmented Generation (RAG)** applications. It integrates a declarative rules engine (**Mangle**) with modern RAG components, orchestrated using a "Sandwich Pattern" (`Rules → RAG → Rules`) to ensure every response is verifiable, policy-compliant, and grounded in evidence.

It is designed for high-performance AI applications where correctness, safety, and explainability are critical.

## 🚀 Key Features

-   **Rules-Driven Control**: Use the Mangle Datalog engine to enforce policies, validate inputs, filter outputs, and modify behavior at runtime.
-   **Verifiable & Explainable**: Every answer can be traced back to the source documents and the rules that were applied.
-   **Pluggable Components**: Easily swap out components like Retrievers, Rerankers, LLMs, and Vector Stores. MangleKit provides built-in support for popular providers.
-   **Fluent Builder API**: A type-safe, chainable API for programmatically constructing your RAG pipeline.
-   **Declarative Configuration**: Define your entire pipeline in a YAML file for easy setup and environment management.
-   **High Performance**: Built in Go for low-latency, concurrent, and scalable services.

## ⚙️ Core Architecture: The Sandwich Pattern

MangleKit processes queries using a multi-stage pipeline that wraps a standard RAG flow with rule evaluation stages. This ensures that logic and policies can be applied before and after the core generation step.

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

## 🛠️ Getting Started

### 1. Prerequisites

-   Go 1.21 or later.
-   API keys for your chosen providers (e.g., OpenAI, Google).

### 2. Installation

To add MangleKit to your Go project, run:

```bash
go get github.com/duynguyendang/manglekit
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
    ```

3.  Use a library like `godotenv` to load this file when your application starts.

    ```go
    import "github.com/joho/godotenv"

    func main() {
        // Load .env file from the current directory
        _ = godotenv.Load()

        // ... rest of your application logic
    }
    ```

## 💻 Usage Examples

You can configure MangleKit either programmatically using the fluent Builder API or declaratively with a single YAML file.

---

### Example 1: Programmatic Setup with the Builder API

This is the most common and type-safe way to build a MangleKit pipeline.

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
		WithRetriever(&retrieve.BM25Options{Path: "./path/to/your/docs"}).
		WithLLM(&llm.OpenAIOptions{Model: "gpt-4o-mini"}).
		WithTopK(5).
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

### Example 2: Declarative Setup with YAML

For maximum flexibility, you can define your entire pipeline in a `config.yaml` file. This is ideal for applications where you want to change the pipeline without recompiling code.

**`config.yaml`:**

```yaml
# Set default pipeline parameters for the "sandwich" orchestrator
topK: 8
maxTokens: 512

# Configure API keys and other provider-specific settings.
# The ${VAR} syntax supports environment variable expansion.
providers:
  openai:
    apiKey: "${OPENAI_API_KEY}"
  google:
    apiKey: "${GOOGLE_API_KEY}"

# Define the components for each stage of the pipeline
embedder:
  name: "google"
  params:
    model: "text-embedding-004"

retriever:
  name: "hybrid" # Use the hybrid retriever

reranker:
  name: "cosine"
  params:
    topK: 5

llm:
  name: "openai"
  params:
    model: "gpt-4o-mini"
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

### Example 3: Using the Mangle Rules Engine

The true power of MangleKit comes from its integrated rules engine. You can define policies and logic in `.dlog` files to control the pipeline's behavior.

**`rules.dlog`:**

```prolog
// Deny any query that contains the word "secret".
deny("Query contains forbidden term") :-
  request_query_contains("secret").
```

**Go code to load the rules:**

```go
// ... (imports and main function setup from Example 1)

// Add the Mangle rules engine to the builder.
orch, err := manglekit.NewBuilder().
    WithRetriever(&retrieve.BM25Options{Path: "./path/to/your/docs"}).
    WithLLM(&llm.OpenAIOptions{Model: "gpt-4o-mini"}).
    WithRules(&core.MangleOptions{
        Path: []string{"rules.dlog"}, // Point to your Datalog file
    }).
    Build()
if err != nil {
    log.Fatalf("Failed to build orchestrator: %v", err)
}

// This query will now be denied by the rules engine.
query := core.Query{Text: "Tell me a secret about MangleKit"}
answer, err := orch.Run(ctx, query)

if err != nil {
    // The error will be of type `core.ErrDenied`.
    fmt.Println("Request denied by Mangle rules:", err)
}
```

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
| **Embedder**      | `google`      | Generates embeddings using Google's models.                  | _None_             |
|                   | `openai`      | Generates embeddings using OpenAI's models.                  | _None_             |
| **Rules Engine**  | `mangle`      | The core Datalog engine for rules-based control.             | _None_             |
| **Schema Parser** | `jsonschema`  | Parses JSON Schema files into facts for the Mangle engine.   | _None_             |
|                   | `rdf`         | Parses RDF (Turtle) files into facts for the Mangle engine.  | _None_             |
| **Vector Store**  | `localvec`    | An in-memory vector store that persists to disk.             | `Embedder`         |

## 📂 Simplified Repository Layout

This is a high-level overview of the most important directories in the MangleKit repository.

```
.
├── core/               # Core interfaces and types (Doc, Query, Answer, Retriever, etc.).
├── providers/          # The registration point for all standard provider implementations.
├── internal/           # Contains the concrete implementations of all providers.
│   ├── providers/
│   └── embedders/
├── pipeline/           # Implementations of the main orchestration logic (Sandwich, Declarative).
├── examples/           # Standalone, runnable examples demonstrating key features.
│
├── sdk.go              # The main `New()` entry point.
├── builder.go          # The fluent `NewBuilder()` API for programmatic setup.
├── config.go           # Logic for loading pipelines from YAML files.
├── registry.go         # The central registry for all component providers.
│
├── go.mod              # Go module definition.
└── README.md           # This file.
```

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to submit pull requests, report issues, and suggest features.

## 📜 License

This project is licensed under the Apache License, Version 2.0. See the [LICENSE](LICENSE) file for details.