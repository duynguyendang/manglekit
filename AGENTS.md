# AGENTS.md — Manglekit (Go)

> **Audience:** AI coding agents (and humans). This file tells agents exactly how to build, run, test, and extend a Go project that uses **Mangle** and **Genkit**.
>
> **Core libs:**
>
> * Mangle (rules engine): [https://github.com/google/mangle](https://github.com/google/mangle)
> * Genkit (agentic flows): [https://genkit.dev/](https://genkit.dev/)

---

## 1) Project Summary

* **What:** A lightweight Go framework that combines a rules engine (Mangle) with agentic orchestration (Genkit) using the **Sandwich** pattern: `Mangle‑Pre → Retrieval → Mangle‑Post → LLM`.
* **Why:** Keep answers correct and policy‑compliant while still using semantic retrieval and generation.

---

## 2) Tech Stack

* **Language:** Go 1.21+
* **Rules:** `github.com/google/mangle`
* **Orchestration:** Genkit (Go SDK or HTTP endpoints)
* **LLM:** OpenAI (or provider of your choice)
* **Search:** Optional Vector DB (Qdrant/Milvus) + keyword index, or simple in‑memory stub for local dev

---

## 3) Setup Commands

```bash
# Initialize module (only if you're creating a brand-new fork without go.mod)
go mod init github.com/yourorg/manglekit

# Dependencies (adjust as needed)
go get github.com/google/mangle
go get github.com/sashabaranov/go-openai
# Example: vector DB client (optional)
go get github.com/qdrant/go-client/qdrant

# Build all
make build        # or: go build ./...

# Lint & fmt
make lint         # or: golangci-lint run
go fmt ./...

# Run tests
make test         # or: go test ./...
```

**Required environment:**

```
GO111MODULE=on
OPENAI_API_KEY=<set-if-using-openai>
# Optional vector DB
QDRANT_URL=http://localhost:6333
QDRANT_API_KEY=<optional>
```

---

## 4) Build & Run

```bash
# Run the demo agent HTTP server
make run          # or: go run ./cmd/agent

# Example curl
curl -s -X POST localhost:8080/answer \
  -H 'content-type: application/json' \
  -d '{"user":"u1","query":"App v2.1 on Ubuntu crashes on PDF export"}'
```

**Expected behavior:**

1. Agent normalizes & scopes query with **Mangle‑Pre** (see `internal/agent/processor.go`).
2. Performs retrieval (stub or vector DB) via `internal/agent/retrieval.go`.
3. Applies **Mangle‑Post** policy/compat filters (`internal/agent/postprocessor.go`).
4. Calls LLM to synthesize final answer (with citations) using `internal/llm`.

**Configuration knobs:**

`config.yaml` controls retrieval depth and safety thresholds. Wire new code to the following keys whenever possible:

* `retrieval.hybrid.bm25.must` / `should` — keyword filters emitted by Mangle‑Pre.
* `retrieval.hybrid.dense.topK` — ANN candidate pool for the fine reranker.
* `retrieval.rerank.mrl.topK` — number of chunks that survive the re‑rank stage.
* `llm.fallbackThreshold` — confidence score that triggers deterministic fallback text.

---

## 5) Repo Layout (expected)

```
./
├─ cmd/agent/main.go            # HTTP entrypoint
├─ internal/agent/agent.go      # Sandwich orchestration
├─ internal/agent/retrieval.go  # Hybrid search (stub/real)
├─ internal/llm/llm.go          # LLM wrapper
├─ internal/mangle/rules.mng    # Mangle rules (Datalog-like)
├─ internal/mangle/facts.json   # Seed facts for local dev
├─ docs/                        # HLD, diagrams, etc.
└─ go.mod
```

---

## 6) Code Style (Go)

* `gofmt`, `go vet`, `golangci-lint`.
* Error handling: wrap with context (`fmt.Errorf("...: %w", err)`).
* Keep packages small and focused.
* No global mutable state; pass dependencies via interfaces/structs.

---

## 7) How Agents Should Work (Rules)

**Golden flow:**

1. **Mangle‑Pre**

   * Normalize/validate entities (`product`, `version`, `os`, `feature`).
   * Enforce policy filters (tenant/visibility/region).
   * (Optional) Expansion via aliases/components (bounded terms & depth).
2. **Retrieval**

   * Use metadata filters from step 1.
   * Prefer hybrid (vector + keyword). Keep `topK ≤ 40`.
3. **Mangle‑Post**

   * Hard filter: entitlement/PII/region/compatibility.
   * Attach `explain` why documents were dropped.
4. **LLM**

   * Generate answer using **only vetted context**; cite `doc_id`s.

**Do:** keep prompts short, deterministic; log trace spans.
**Don’t:** bypass Mangle checks; leak secrets/PII.

---

## 8) Minimal Example (compilable Go)

> This is a self‑contained example that compiles with a stubbed Mangle and retrieval. Replace the placeholders with real Mangle/Genkit calls.

```go
// cmd/agent/main.go
package main

import (
  "log"
  "net/http"

  agent "github.com/yourorg/manglekit/internal/agent"
)

func main() {
  svc := agent.NewServer()
  if err := http.ListenAndServe(":8080", svc); err != nil {
    log.Fatalf("server exited: %v", err)
  }
}
```

```go
// internal/agent/server.go
package agent

import (
  "net/http"

  "github.com/yourorg/manglekit/internal/agent/orchestrator"
)

func NewServer() http.Handler {
  orch := orchestrator.New()
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // sandwich flow: PreProcess → Retrieve → PostProcess → LLM
    answer, err := orch.Answer(r.Context(), r.Body)
    if err != nil {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
    }
    w.Write(answer)
  })
}
```

```go
// internal/agent/orchestrator/orchestrator.go
package orchestrator

import "github.com/yourorg/manglekit/internal/llm"

type Orchestrator struct {
  pre  *PreProcessor
  retr *Retriever
  post *PostProcessor
  llm  llm.Client
}

func New() *Orchestrator {
  return &Orchestrator{
    pre:  NewPreProcessor(),
    retr: NewRetrieverFromConfig(),
    post: NewPostProcessor(),
    llm:  llm.NewFromEnv(),
  }
}

// Answer wires the sandwich together; see internal/agent for the real implementation.
```

> Replace stubs with real **Mangle** rule evaluation, retrieval, and Genkit flows using the actual `internal/agent` packages.

---

## 9) Agent Guidelines (Do/Don’t)

**Do**

* Always run `Mangle‑Pre` before retrieval and `Mangle‑Post` before answering.
* Prefer metadata filters and keep `topK` small; re‑rank if available.
* Cite sources (doc IDs/titles). Return explanations for dropped docs.

**Don’t**

* Don’t access external URLs or services unless configured in environment.
* Don’t emit answers that aren’t grounded in vetted context.

---

## 10) Common Tasks

```bash
# Format, lint, test
go fmt ./... && golangci-lint run && go test ./...

# Update deps
go get -u ./... && go mod tidy

# Run dev server
go run ./cmd/agent
```

---

## 11) Troubleshooting

* **401 from LLM**: set `OPENAI_API_KEY`.
* **No results**: relax filters or add seed docs in `internal/mangle/facts.json`.
* **Long answers / token overrun**: reduce `topK` or truncate doc chunks before LLM call.

---

## 12) Security Notes

* Never log secrets or raw user inputs containing PII.
* Enforce tenant/visibility checks in **Mangle‑Post**.
* Treat `AGENTS.md` as code: review changes via PR.

---

*This AGENTS.md is intentionally concise for agents. Human‑oriented details belong in `README.md` and `docs/`.*
