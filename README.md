[![Go](https://img.shields.io/badge/Go-1.21%2B-blue?logo=go)](https://golang.org) [![License](https://img.shields.io/badge/License-Apache_2.0-yellow)](LICENSE)

# Manglekit

**MangleKit** is a lightweight, embeddable **Go** framework for building robust and controllable **Retrieval-Augmented Generation (RAG)** applications. It integrates a declarative rules engine (**Mangle**) with modern RAG components, orchestrated using a "Sandwich Pattern" (`Rules → RAG → Rules → LLM`) to ensure every response is verifiable, policy-compliant, and grounded in evidence.

It is designed for high-performance AI applications where correctness, safety, and explainability are critical.

## 🚀 Key Features

-   **Rules-Driven Control**: Use the Mangle Datalog engine to enforce policies, validate inputs, filter outputs, and modify behavior at runtime.
-   **Verifiable & Explainable**: Every answer can be traced back to the source documents and the rules that were applied.
-   **Pluggable Components**: Easily swap out components like Retrievers, Rerankers, LLMs, and Vector Stores. MangleKit provides built-in support for popular providers.
-   **Fluent Builder API**: A type-safe, chainable API for programmatically constructing your RAG pipeline.
-   **YAML Configuration**: Alternatively, define your entire pipeline in a YAML file for easy setup and environment management.
-   **High Performance**: Built in Go for low-latency, concurrent, and scalable services.

## ⚙️ Core Architecture: The Sandwich Pattern

MangleKit processes queries using a multi-stage pipeline that wraps a standard RAG flow with rule evaluation stages.

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

### Prerequisites

-   Go 1.21+
-   API keys for your chosen providers (e.g., OpenAI, Google).

### Installation

To add MangleKit to your project, run:

```bash
go get github.com/duynguyendang/manglekit
```

### Environment Variables

Create a `.env` file in your project root or export the following environment variables:

```
# For OpenAI models (LLM or Embedder)
OPENAI_API_KEY="sk-..."

# For Google models (LLM or Embedder)
GOOGLE_API_KEY="AIza..."
```

The application will automatically load the `.env` file if it exists.

## 💻 Usage

MangleKit can be configured either programmatically using the Builder API or declaratively with a YAML file.

### Example 1: Programmatic Setup with the Builder

The `NewBuilder()` function provides a fluent, type-safe API to construct your orchestrator. This is the recommended approach for most use cases.

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

	// Blank import to register all standard providers
	_ "github.com/duynguyendang/manglekit/providers/all"
)

func main() {
	_ = godotenv.Load()
	ctx := context.Background()

	// Use the builder to construct the orchestrator
	orch, err := manglekit.NewBuilder().
		WithRetriever(&retrieve.BM25Options{Path: "mangle/knowledge_base"}).
		WithLLM(&llm.OpenAIOptions{Model: "gpt-4o-mini"}).
		WithTopK(5).
		Build()
	if err != nil {
		log.Fatalf("failed to build orchestrator: %v", err)
	}

	// Run a query
	query := core.Query{Text: "What is MangleKit?"}
	answer, err := orch.Run(ctx, query)
	if err != nil {
		log.Fatalf("orchestrator run failed: %v", err)
	}

	fmt.Println("Answer:", answer.Text)
}
```

### Example 2: Configuration with YAML

For greater flexibility, you can define your entire pipeline in a `config.yaml` file and load it with a single function call.

**`config.yaml`:**

```yaml
# Set default pipeline parameters
topK: 8
maxTokens: 512
fallbackThreshold: 0.5

# Configure API keys and other provider-specific settings
providers:
  openai:
    apiKey: "${OPENAI_API_KEY}" # Supports environment variable expansion
  google:
    apiKey: "${GOOGLE_API_KEY}"

# Define the components for each stage of the pipeline
embedder:
  name: "google"
  params:
    model: "text-embedding-004"

retriever:
  name: "bm25"
  params:
    path: "mangle/knowledge_base"
    topK: 10

reranker:
  name: "cosine"
  params:
    topK: 5
    # The 'embedder' dependency is automatically resolved by the builder

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

    // Blank imports to register providers
    _ "github.com/duynguyendang/manglekit/internal/providers/bm25"
    _ "github.com/duynguyendang/manglekit/internal/providers/llm"
    _ "github.com/duynguyendang/manglekit/internal/providers/rerank/cosine"
    _ "github.com/duynguyendang/manglekit/internal/providers/embedders"
)

func main() {
    builder, err := manglekit.NewBuilderFromYAML("config.yaml")
    if err != nil {
        log.Fatalf("failed to create builder from YAML: %v", err)
    }

    orch, err := builder.Build()
    if err != nil {
        log.Fatalf("failed to build orchestrator: %v", err)
    }

    // Run a query
    query := core.Query{Text: "What is MangleKit?"}
    answer, err := orch.Run(context.Background(), query)
    if err != nil {
        log.Fatalf("orchestrator run failed: %v", err)
    }

    fmt.Println("Answer:", answer.Text)
}
```

## 📦 Available Components

MangleKit includes a suite of built-in providers that can be configured in the builder or YAML file.

| Component Type    | Provider Name | Description                                                  |
| ----------------- | ------------- | ------------------------------------------------------------ |
| **Retriever**     | `bm25`        | Keyword-based search (Okapi BM25) over local files.          |
|                   | `dense`       | Vector search using a configured embedder and vector store.  |
|                   | `hybrid`      | Fuses results from `bm25` and `dense` using RRF.             |
|                   | `inmemory`    | A simple, updatable in-memory store for testing.             |
| **Reranker**      | `cosine`      | Re-ranks documents based on cosine similarity of embeddings. |
| **LLM**           | `google`      | Integrates with Google's generative models via Genkit.       |
|                   | `openai`      | Integrates with OpenAI's models (e.g., GPT-4).               |
|                   | `groq`        | Integrates with Groq's fast inference API.                   |
| **Embedder**      | `google`      | Generates embeddings using Google's models.                  |
|                   | `openai`      | Generates embeddings using OpenAI's models.                  |
| **Rules Engine**  | `mangle`      | The core Datalog engine for rules-based control.             |
| **Schema Parser** | `jsonschema`  | Parses JSON Schema files into facts for the Mangle engine.   |
| **Vector Store**  | `localvec`    | An in-memory vector store that persists to disk.             |


## 📂 Repository Layout

```
.
├── cmd/
│   └── agent/
│       └── main.go
├── core/
│   ├── rules.go
│   ├── schema.go
│   └── types.go
├── docs/
│   ├── CONTEXT.md
│   ├── CSD.md
│   ├── HLD.md
│   └── LLD.md
├── embed/
│   └── options.go
├── examples/
│   ├── 01-basic-rag/
│   │   ├── data/
│   │   │   └── mangle.md
│   │   └── main.go
│   ├── 02-logic-layer-mode/
│   │   ├── rules.dlog
│   │   └── main.go
│   ├── 03-custom-prompt/
│   │   └── main.go
│   ├── 04-hot-reload/
│   │   └── main.go
│   ├── 06-schema-validation/
│   │   ├── user.schema.json
│   │   ├── validation.dlog
│   │   └── main.go
│   └── custom-prompt/
│       └── main.go
├── internal/
│   ├── embedders/
│   │   ├── google/
│   │   │   ├── google.go
│   │   │   └── google_test.go
│   │   └── openai/
│   │       └── openai.go
│   ├── logger/
│   │   └── logger.go
│   ├── providers/
│   │   ├── bm25/
│   │   │   ├── bm25.go
│   │   │   └── bm25_test.go
│   │   ├── dense/
│   │   │   ├── dense.go
│   │   │   └── dense_test.go
│   │   ├── hybrid/
│   │   │   ├── hybrid.go
│   │   │   └── hybrid_test.go
│   │   ├── llm/
│   │   │   ├── google.go
│   │   │   └── openai.go
│   │   ├── mangle/
│   │   │   ├── converters/
│   │   │   │   ├── document.go
│   │   │   │   ├── query.go
│   │   │   │   └── user_context.go
│   │   │   ├── rules.go
│   │   │   └── rules_test.go
│   │   ├── rerank/
│   │   │   └── cosine/
│   │   │       ├── cosine.go
│   │   │       └── cosine_test.go
│   │   ├── retrievers/
│   │   │   └── inmemory/
│   │   │       └── inmemory.go
│   │   ├── schemaparsers/
│   │   │   └── jsonschema/
│   │   │       └── parser.go
│   │   └── ...
│   └── vectorstores/
│       └── localvec/
│           └── localvec.go
├── llm/
│   ├── google.go
│   ├── llm.go
│   ├── openai.go
│   ├── options.go
│   └── prompt.go
├── mangle/
│   ├── main.dlog
│   ├── knowledge_base/
│   │   ├── aliases.dlog
│   │   └── stopwords.dlog
│   ├── pipelines/
│   │   └── retrieval_pipeline.dlog
│   └── policies/
│       └── access_control.dlog
├── pipeline/
│   ├── sandwich.go
│   └── sandwich_test.go
├── providers/
│   └── all/
│       └── all.go
├── rerank/
│   ├── options.go
│   └── rerank.go
├── retrieve/
│   ├── bm25.go
│   ├── embed.go
│   ├── hybrid.go
│   ├── inmemory.go
│   ├── options.go
│   └── retrieve.go
├── .env.example
├── .gitignore
├── AGENTS.md
├── CONTRIBUTING.md
├── LICENSE
├── Makefile
├── README.md
├── TODO.md
├── builder.go
├── config.go
├── config.yaml
├── go.mod
├── go.sum
├── registry.go
├── sdk.go
└── typemap.go
```

## 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to submit pull requests, report issues, and suggest features.

## 📜 License

This project is licensed under the Apache License, Version 2.0. See the [LICENSE](LICENSE) file for details.