# AGENTS.md — Manglekit Examples Agent Guide

Last updated: 2025-11-04

---

Purpose
- Give coding agents a minimal, reliable path to create and run examples under `examples/` without touching core SDK code.
- Examples are config‑first and should remain self‑contained.

Scope & Constraints
- Work only under `examples/`. Do not modify `core/`, `pipeline/`, `internal/providers/`, `builder.go`, or `registry.go` from here.
- No architecture/docs sync is required for example‑only changes. If you ever change areas listed above, follow the root `AGENTS.md` docs sync rules.

Quick Start Template
- Create a new folder: `examples/01-rag-sandwich` (or the next number + name).
- Add two files:
  - `main.go` — boot, load YAML, execute query
  - `config.yaml` — components and orchestrator wiring

main.go (minimal)
```go
package main

import (
    "context"
    "fmt"
    "os"
    "github.com/duynguyendang/manglekit/core"
    "github.com/duynguyendang/manglekit/sdk"
)

func main() {
    ctx := context.Background()
    data, err := os.ReadFile("config.yaml")
    if err != nil { panic(err) }

    orch, err := sdk.Load(ctx, data)
    if err != nil { panic(err) }
    defer orch.Close(ctx)

    prompt := "What is Manglekit?"
    if len(os.Args) > 1 { prompt = os.Args[1] }

    ans, err := orch.Execute(ctx, "demo-session", core.Query{Text: prompt})
    if err != nil { panic(err) }
    fmt.Println(ans.Text)
}
```

YAML Config Template
- The SDK expects a root with `components` and an `orchestrator` name. Use provider names from the codebase (see notes below).

```yaml
# examples/01-rag-sandwich/config.yaml
orchestrator: rag

components:
  # LLM — Google Gemini (set env: GOOGLE_API_KEY)
  - name: llm
    kind: llm
    type: google
    params:
      APIKey: ${GOOGLE_API_KEY}
      Model: gemini-2.5-flash

  # Retriever — In-memory docs for a zero-dependency demo
  - name: kb
    kind: retriever
    type: in-memory
    params:
      Documents:
        - ID: doc-1
          Text: "Manglekit is a type-safe, orchestrator-driven SDK."
        - ID: doc-2
          Text: "It composes retrievers, rerankers and LLMs via config."

  # Orchestrator — Sandwich: deterministic RAG flow
  - name: rag
    kind: orchestrator
    type: sandwich
    params:
      llm: llm
      retriever: kb
      top_k: 3
```

Provider Name Notes
- Kinds: `retriever`, `reranker`, `llm`, `embedder`, `state_provider`, `schema_parser`, `orchestrator`.
- Common `type` values available in this repo (registered via `providers/all`):
  - Retrievers: `bm25`, `dense`, `hybrid`, `in-memory`
  - Reranker: `cosine`
  - LLMs: `openai`, `google`
  - Embedders: `openai`, `groq`, `google`
  - Orchestrators: `sandwich`, `declarative`
- Field casing in YAML params follows struct field names when no `yaml:"..."` tag exists. For Google LLM and embedders, prefer `APIKey` and `Model` keys. For others with YAML tags (e.g., `base_url`, `vectorStore`, `top_k`), use the tagged name.

Common Variations
- Add reranking (cosine): requires an embedder.
```yaml
  - name: embedder
    kind: embedder
    type: google
    params:
      APIKey: ${GOOGLE_API_KEY}
      Model: text-embedding-004

  - name: rerank
    kind: reranker
    type: cosine
    params:
      Embedder: embedder

  - name: rag
    kind: orchestrator
    type: sandwich
    params:
      llm: llm
      retriever: kb
      reranker: rerank
```

- Hybrid retriever (RRF over sub-retrievers):
```yaml
  - name: bm25
    kind: retriever
    type: bm25
    params:
      path: ./data/docs
      topK: 10

  - name: hybrid
    kind: retriever
    type: hybrid
    params:
      retrievers: [bm25]    # add others if available
      rrf_k: 60

  - name: rag
    kind: orchestrator
    type: sandwich
    params:
      llm: llm
      retriever: hybrid
```

Run & Verify
- Run from the example folder:
  - `go run . "What is Manglekit architecture?"`
- Ensure necessary env vars are set (e.g., `export GOOGLE_API_KEY=...`).

Do / Don’t
- Do keep examples config‑first and self‑contained.
- Do use provider names and kinds that exist in this repo.
- Don’t print in core code paths; printing in examples is fine to show output.
- Don’t edit core SDK, providers, or orchestrators from `examples/`.

Troubleshooting
- “unknown type/kind”: check `kind` and `type` values match registered providers.
- Google auth failures: confirm `APIKey` field in YAML or `GOOGLE_API_KEY` env var.
- Param not applied: verify the field name vs. YAML tag (some structs use `APIKey`/`Model` without yaml tags).

Reference
- See `examples/README.md` for the catalog and goals.
- Provider registrations: `providers/all/all.go`.
- Config schema: `config/types.go`.
