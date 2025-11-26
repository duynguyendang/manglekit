# Manglekit Examples

This directory contains canonical examples demonstrating the major capabilities of the **Manglekit SDK**.

Manglekit supports two primary construction patterns:

1.  **Declarative (Config-First):** Defining the pipeline in YAML and loading it via `sdk.FromConfig` (Recommended for production).
2.  **Programmatic:** Building the pipeline via the Go `sdk.NewBuilder` API (Useful for dynamic construction, testing, or advanced integration).

Most examples in this directory utilize the **declarative** approach for clarity and reproducibility.

-----

## 🧱 Example Catalog

| \# | Folder | Orchestrator | Purpose |
|---|---|---|---|
| **01** | [01-programmatic-setup](https://www.google.com/search?q=./01-programmatic-setup) | `sandwich` | **Programmatic Construction & Hybrid Search.**<br>Demonstrates using `sdk.NewBuilder()` to wire a Hybrid Retriever (BM25 + Genkit) + Google Embedder + LLM without YAML. |
| **02** | [02-rag-chat-app](https://www.google.com/search?q=./02-rag-chat-app) | `sandwich` | **Stateful RAG Chat.**<br>A chat application answering questions over documents. Demonstrates how **StateProvider** integrates with the pipeline for multi-turn context. |
| **03** | [03-neuro-symbolic-declarative](https://www.google.com/search?q=./03-neuro-symbolic-declarative) | `declarative` | **Logic-Driven Workflow.**<br>Showcases the **Declarative Orchestrator**: combining rules, reasoner, retriever, and LLM stages with policy guards defined in Datalog. |
| **04** | [04-schema-validation](https://www.google.com/search?q=./04-schema-validation) | `sandwich` | **Structured Data Guardrails.**<br>Demonstrates using **SchemaParser** providers to validate input/output structures and enforce API contracts around the LLM. |
| **05** | [05-rdf-knowledge-store](https://www.google.com/search?q=./05-rdf-knowledge-store) | `declarative` | **Knowledge Graph Integration.**<br>Integrates an RDF/Graph knowledge base via the **KnowledgeStore** provider. Enables hybrid retrieval with symbolic reasoning. |
| **06** | [06-genkit-integration](https://www.google.com/search?q=./06-genkit-integration) | — | **Genkit Interoperability.**<br>Demonstrates how Manglekit can act as a **Genkit Tool** or sub-pipeline, leveraging the broader Genkit ecosystem. |
| **07** | [07-custom-prompts](https://www.google.com/search?q=./07-custom-prompts) | `sandwich` | **Prompt Engineering.**<br>Illustrates prompt customization and templating. Shows how to pass custom `prompt_template` via YAML configuration. |
| **08** | [08-tool-calling-declarative](https://www.google.com/search?q=./08-tool-calling-declarative) | `declarative` | **Agentic Tool Use.**<br>Demonstrates **policy-aware tool invocation** (HTTP/OpenAPI). Rulesets guard external calls and planners decide usage. |

-----

## 🧩 Architectural Highlights

Each example demonstrates one or more of the following SDK features:

| Feature | Description |
|---|---|
| **Config-First Construction** | Pipelines built declaratively from YAML using `sdk.FromConfig`. |
| **Programmatic Builder** | Dynamic pipeline construction using `sdk.NewBuilder` (See Example 01). |
| **Typed Dependency Injection** | Components request dependencies via `diapi.Builder` (e.g., Hybrid retriever requesting sub-retrievers). |
| **Orchestrators** | `sandwich` for linear RAG; `declarative` for complex, rule-driven flows. |
| **Provider Composition** | Providers (retriever, rules, llm, etc.) register via `providers/all` and are referenced by name. |
| **Neuro-Symbolic Integration** | Blends neural components (Embedders, LLMs) with symbolic ones (Rules, Reasoners). |

-----

## 🧠 Running the Examples

### Prerequisites

  * Go 1.24+
  * **API Keys:** Most examples require a `GOOGLE_API_KEY` or `OPENAI_API_KEY`.
  * **Setup:**
    ```bash
    cp .env.example .env
    # Edit .env with your keys
    ```

### Running an Example

Navigate to the specific example directory and run the `main.go` file.

**Example 01 (Programmatic):**

```bash
cd examples/01-programmatic-setup
go run .
```

**Example 02 (Declarative with YAML):**

```bash
cd examples/02-rag-chat-app
go run . "How do I configure the state provider?"
```

### Common Configuration Pattern (YAML)

For examples 02-08, the `config.yaml` defines the entire stack:

```yaml
orchestrator:
  type: sandwich

providers:
  google:
    api_key: ${GOOGLE_API_KEY}

retriever:
  name: hybrid
  params:
    retrievers: ["bm25", "semantic"]

llm:
  name: google
```

-----

## 🔬 Testing & Extension

To create a new example:

1.  Copy an existing folder (e.g., `01-programmatic-setup` for code-heavy or `02-rag-chat-app` for config-heavy).
2.  If using YAML, update `config.yaml`.
3.  If using Programmatic builder, update the chain in `main.go`.
4.  Ensure `providers/all` is imported to register standard components.

<!-- end list -->

```go
import _ "github.com/duynguyendang/manglekit/providers/all"
```

-----

### License

All examples are MIT-licensed and intended for educational and experimental use within the Manglekit ecosystem.