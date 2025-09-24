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
# Initialize module (if not already)
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

1. Agent normalizes & scopes query with **Mangle‑Pre**.
2. Performs retrieval (stub or vector DB).
3. Applies **Mangle‑Post** policy/compat filters.
4. Calls LLM to synthesize final answer (with citations).

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
  "encoding/json"
  "log"
  "net/http"
)

type Req struct { User string `json:"user"`; Query string `json:"query"` }

type Agent interface { Answer(user, query string) (string, error) }

func main() {
  ag := NewDemoAgent() // defined below
  http.HandleFunc("/answer", func(w http.ResponseWriter, r *http.Request) {
    var req Req
    _ = json.NewDecoder(r.Body).Decode(&req)
    ans, err := ag.Answer(req.User, req.Query)
    if err != nil { http.Error(w, err.Error(), 500); return }
    _ = json.NewEncoder(w).Encode(map[string]string{"answer": ans})
  })
  log.Println("listening on :8080")
  log.Fatal(http.ListenAndServe(":8080", nil))
}
```

```go
// internal/agent/agent.go
package main

import (
  "context"
  "strings"
)

type DemoAgent struct{ llm LLM }

func NewDemoAgent() *DemoAgent { return &DemoAgent{ llm: NewLLMFromEnv() } }

func (a *DemoAgent) Answer(user, query string) (string, error) {
  // Mangle‑Pre (stub): normalize + filters
  norm := strings.ToLower(query)
  filters := map[string]string{"visibility": "public_or_tenant"}

  // Retrieval (stub)
  docs := retrieve(norm, filters)

  // Mangle‑Post (stub): drop disallowed docs
  vetted := []string{}
  for _, d := range docs { if allowDoc(user, d) { vetted = append(vetted, d.Text) } }

  // LLM synthesis
  return a.llm.Answer(context.Background(), query, vetted)
}

type doc struct{ ID, Tenant, Visibility, Title, Text string }

func retrieve(q string, filters map[string]string) []doc {
  corpus := []doc{{ID:"d1",Tenant:"t42",Visibility:"tenant",Title:"Fix PDF export crash",Text:"Workaround for PDF export crash on Ubuntu 22.04"},
                  {ID:"d2",Tenant:"public",Visibility:"public",Title:"General export guide",Text:"How export works"}}
  out := []doc{}
  for _, d := range corpus {
    if strings.Contains(strings.ToLower(d.Title+" "+d.Text), q) { out = append(out, d) }
  }
  return out
}

func allowDoc(user string, d doc) bool { return d.Visibility == "public" || d.Tenant == "t42" }
```

```go
// internal/llm/llm.go
package main

import (
  "context"
  "os"
  openai "github.com/sashabaranov/go-openai"
)

type LLM interface { Answer(ctx context.Context, question string, contextDocs []string) (string, error) }

type OpenAIClient struct{ api *openai.Client }

func NewLLMFromEnv() *OpenAIClient { return &OpenAIClient{ api: openai.NewClient(os.Getenv("OPENAI_API_KEY")) } }

func (c *OpenAIClient) Answer(ctx context.Context, question string, contextDocs []string) (string, error) {
  // Keep prompt compact
  prompt := "Use only the provided context to answer. If uncertain, say so.\n\nContext:\n" + strings.Join(contextDocs, "\n---\n") + "\n\nQuestion: " + question
  resp, err := c.api.CreateChatCompletion(ctx, openai.ChatCompletionRequest{ Model: openai.GPT4oMini, Messages: []openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: prompt}}, })
  if err != nil { return "", err }
  if len(resp.Choices) == 0 { return "", nil }
  return resp.Choices[0].Message.Content, nil
}
```

> Replace stubs with real **Mangle** rule evaluation and **Genkit** flows.

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
