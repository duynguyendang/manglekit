# Manglekit Examples

This directory contains canonical examples demonstrating the major capabilities of the **Manglekit SDK (v0.5.0)**.  
Each example is **self-contained**, **config-first**, and can be run independently using a local YAML configuration.

Manglekit now provides a **type-safe, declarative, and orchestrator-driven architecture**.  
All examples load their configuration through:

```go
orch, _ := sdk.FromConfig(ctx, "config.yaml")
defer orch.Close(ctx)
ans, _ := orch.Execute(ctx, "your prompt here")
````

---

## 🧱 Example Catalog

| #      | Folder                                                           | Orchestrator  | Purpose                                                                                                                                                                              |
| ------ | ---------------------------------------------------------------- | ------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **01** | [01-programmatic-setup](./01-programmatic-setup)        | `sandwich`    | This example demonstrates how to build and run a Manglekit pipeline programmatically using the `sdk.NewBuilder()` API. |
| **02** | [02-rag-chat-app](./02-rag-chat-app)                             | `sandwich`    | Stateful chat application that can answer questions over your documents. Demonstrates how **StateProvider** integrates with the pipeline for multi-turn context.                     |
| **03** | [03-neuro-symbolic-declarative](./03-neuro-symbolic-declarative) | `declarative` | Showcases the **Declarative Orchestrator** for neuro-symbolic composition: combining rules, reasoner, retriever, and LLM stages with policy guards and logic control.                |
| **04** | [04-schema-validation](./04-schema-validation)                   | `sandwich`    | Demonstrates the use of **SchemaParser** providers for validating input/output structure and enforcing contracts around the LLM.                                                     |
| **05** | [05-rdf-knowledge-store](./05-rdf-knowledge-store)               | `declarative` | Integrates an RDF or graph knowledge base via the **KnowledgeStore** provider. Enables hybrid retrieval with symbolic reasoning over structured data.                                |
| **06** | [06-genkit-integration](./06-genkit-integration)                 | —             | Demonstrates how Manglekit can act as a **Genkit Tool** or sub-pipeline, showing interoperability between Manglekit and Genkit frameworks.                                           |
| **07** | [07-custom-prompts](./07-custom-prompts)                         | `sandwich`    | Illustrates prompt customization and templating. Shows how to pass a custom `prompt_template` through YAML to influence LLM behavior.                                                |
| **08** | [08-tool-calling-declarative](./08-tool-calling-declarative)     | `declarative` | Demonstrates **policy-aware tool invocation** (HTTP/OpenAPI). Rulesets guard external calls and planners decide which tools to use under explicit constraints.                       |

---

## 🧩 Architectural Highlights

Each example demonstrates one or more of the following SDK features:

| Feature                        | Description                                                                                                                             |
| ------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------- |
| **Config-First Construction**  | All pipelines are built declaratively from YAML using `sdk.FromConfig`. No manual builder wiring.                                       |
| **Typed Dependency Injection** | Components request their dependencies via the `diapi.Builder` interface.                                                                |
| **Orchestrators**              | `sandwich` for deterministic RAG; `declarative` for logic-rich and policy-aware flows.                                                  |
| **Provider Composition**       | Providers (retriever, reranker, rule, reasoner, tool, etc.) register themselves via `providers/all` and are referenced by name in YAML. |
| **Observability & Lifecycle**  | Logs, metrics, and graceful shutdown (`Close()`) are managed automatically by the framework.                                            |
| **Policy & Schema Layers**     | Declarative rule and schema enforcement before and after LLM execution.                                                                 |
| **Neuro-Symbolic Integration** | Blends neural (LLM, embedder, retriever) and symbolic (rules, reasoner, planner) components.                                            |

---

## 🧠 Running the Examples

### Prerequisites

* Go 1.22+
* Environment variables for providers (e.g., `OPENAI_API_KEY`)
* Example-specific dependencies (local data, RDF files, etc.)

### Common Run Pattern

```bash
cd examples/01-rag-sandwich
go run . "What is Manglekit architecture?"
```

or for declarative flow:

```bash
cd examples/03-neuro-symbolic-declarative
go run . "Detect anomalies and call audit tool if allowed."
```

### Environment Configuration

Each example may reference environment variables in YAML, e.g.:

```yaml
options:
  api_key: env:OPENAI_API_KEY
```

Ensure they are exported before running.

---

## 🔬 Testing & Extension

You can write end-to-end tests using the same `sdk.FromConfig` API:

```go
orch, _ := sdk.FromConfig(ctx, "../examples/01-rag-sandwich/config.yaml")
out, _ := orch.Execute(ctx, "What is Manglekit?")
```

New examples can be added by copying one folder and editing only the **config.yaml** and **main.go**.

---

## 🗺 Example Roadmap

| Theme                     | Planned Examples                                                   |
| ------------------------- | ------------------------------------------------------------------ |
| **Performance & Caching** | Demonstrate warm-up, connection pooling, and token budgets.        |
| **Multi-Tenant Configs**  | Load profiles dynamically (`config.dev.yaml`, `config.prod.yaml`). |
| **Benchmark Suite**       | Compare retriever configurations (RRF, top_k) from YAML only.      |
| **WASM/Plugin Sandbox**   | Upcoming architecture extension (see ADR roadmap).                 |

---

### License

All examples are MIT-licensed and intended for educational and experimental use within the Manglekit ecosystem.
