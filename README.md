# Manglekit

Manglekit is a lightweight, embeddable Go framework for building Retrieval-Augmented Generation (RAG) workflows. It integrates declarative rules via the Mangle engine, hybrid semantic and keyword search, and AI orchestration through Genkit. Designed for high-performance AI applications, it uses the Sandwich Pattern (Mangle-Pre → RAG → Mangle-Post → LLM) to ensure controlled, explainable, and policy-compliant responses.

## Overview

Manglekit addresses challenges in AI-driven knowledge management by combining:
- **Rule-Based Processing (Mangle)**: Datalog-style engine for query normalization, constraints, validation, and redaction.
- **Hybrid Retrieval**: Vector search (e.g., FAISS/Pinecone) + keyword indexing (BM25) with metadata filtering for precise results.
- **Orchestration (Genkit)**: Manages LLM flows, intent routing, and context handling with pluggable providers (e.g., OpenAI, Ollama).

Key goals include low latency (<300ms E2E), modularity for easy integration, and traceability for compliance. It's suitable for internal knowledge bases, compliant chatbots, and exploratory analytics in enterprise settings.

## Features
- **Sandwich Pattern**: Wraps RAG with pre- and post-processing rules to mitigate hallucinations and enforce policies.
- **Pluggable Components**: Swap backends for vector DBs, LLMs, and storage (in-memory/BoltDB).
- **Hybrid Search**: Combines semantic embeddings with keyword matching for high recall/precision.
- **Explainability**: Structured logging and annotations for rule firings and decisions.
- **Performance**: Go stdlib-focused, stateless design for 100+ QPS; minimal dependencies.
- **Security**: Built-in redaction, access controls, and sandboxed rule evaluation.

## Quick Start

1. **Install**:
   ```
   go mod init your-project
   go get ndduy.dev/manglekit
   ```

2. **Basic Usage** (Library Mode):
   ```go
   package main

   import (
       "context"
       "fmt"
       mkit "ndduy.dev/manglekit"
   )

   func main() {
       config := mkit.Config{
           LLM: mkit.LLMConfig{Provider: "openai", Model: "gpt-4o-mini"},
           Retrieval: mkit.RetrievalConfig{Vector: mkit.VectorConfig{EmbedModel: "all-MiniLM-L6-v2"}},
       }
       kit := mkit.New(config)

       ctx := context.Background()
       resp, err := kit.Answer(ctx, &mkit.QueryInput{Query: "What is Manglekit?"})
       if err != nil {
           fmt.Println("Error:", err)
           return
       }
       fmt.Println("Answer:", resp.Answer)
   }
   ```

3. **Run as Service**:
   - Build: `go build -o manglekit ./cmd/agent`
   - Serve: `./manglekit serve --config config.yaml`
   - Query: `curl -X POST http://localhost:8080/v1/answer -d '{"query": "example"}'`

For ingestion, use the `/v1/ingest` endpoint or SDK methods.

## Architecture

See the [High-Level Design (HLD)](docs/HLD.md) for detailed components, data flows, and deployment.

## Documentation
- [Conceptual Solution Design (CSD)](docs/CSD.md): Business requirements and use cases.
- [HLD](docs/HLD.md): Technical architecture and interfaces.
- Examples: `examples/` (TBD).

## Contributing
See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License
[MIT](LICENSE)