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

* **Language:** Go 1.24+
* **Rules:** `github.com/google/mangle`
* **Orchestration:** Genkit (Go SDK or HTTP endpoints)
* **LLM:** Google (or provider of your choice)
* **Search:** simple in‑memory stub and support local/remote vector db (Qdrant/Chroma/Pinecone).

---

## 3) Setup Commands

```bash
# Initialize module (only if you're creating a brand-new fork without go.mod)
go mod init github.com/duynguyendang/manglekit

# Dependencies (adjust as needed)
go get github.com/google/mangle
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
GOOGLE_API_KEY=<set-if-using-google-ai>
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

**Configuration knobs:**

`config.yaml` controls retrieval depth and safety thresholds. Wire new code to the following keys whenever possible:

* `retrieval.hybrid.bm25.must` / `should` — keyword filters emitted by Mangle‑Pre.
* `retrieval.hybrid.dense.topK` — ANN candidate pool for the fine reranker.
* `retrieval.rerank.mrl.topK` — number of chunks that survive the re‑rank stage.
* `llm.fallbackThreshold` — confidence score that triggers deterministic fallback text.

---

## 5) Repo Layout (expected)

```
github.com/duynguyendang/manglekit
├── go.mod
├── builder.go               # Fluent Builder API
├── config.go                # YAML/environment loading helpers
├── registry.go              # Provider registry + Must* helpers
├── sdk.go                   # New(), Options, orchestrator wiring
│
├── core/
│   ├── rules.go             # Rule contracts and helpers
│   ├── schema.go
│   └── types.go             # Doc/Query/Answer, Options, Observability
├── embed/
│   └── options.go
├── llm/
│   ├── google.go
│   ├── openai.go
│   ├── options.go
│   └── prompt.go
├── retrieve/
│   ├── bm25.go
│   ├── embed.go
│   ├── hybrid.go
│   └── retrieve.go
├── rerank/
│   ├── options.go
│   └── rerank.go
├── pipeline/
│   ├── sandwich.go
│   └── declarative/
│       └── orchestrator.go
│
├── internal/
│   ├── embedders/           # Google/OpenAI embedders
│   ├── logger/              # Logging adapters
│   ├── providers/           # BM25, dense, hybrid, rerank, LLM, mangle, etc.
│   └── vectorstores/        # LocalVec implementation
├── providers/
│   └── all/all.go           # Registers every bundled provider
├── cmd/
│   └── agent/main.go        # Demo HTTP server
├── examples/                # Runnable guides & fixtures
│   ├── 01-basic-rag/
│   ├── 02-logic-layer-mode/
│   ├── ...
│   └── 08-symbolic-rag/
└── docs/                    # HLD/LLD/CSD, CONTEXT snapshot, reviews
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
## 8) Testing Guidelines

* Use Go's standard `testing` package for unit and integration tests.
* Write table-driven tests for rule evaluations in Mangle-Pre/Post, retrieval, and reranking logic.
* Test the full Sandwich pipeline end-to-end using `examples/05-chat-with-data/main.go` (mock the LLM/vector DB via interfaces when running in CI).
* Cover edge cases: invalid queries, policy violations, empty retrievals, and token limits.
* Run tests with `make test` or `go test ./... -v`.
* Generate coverage: `go test ./... -coverprofile=coverage.out` then `go tool cover -html=coverage.out`.
* Aim for >80% coverage; focus on critical paths in pipeline/sandwich.go and providers.

## 9) Agent Guidelines & Core Tasks

### Golden Flow (Do's) ✅

* **Always follow the Sandwich Pattern:** Run `Mangle-Pre` before retrieval and `Mangle-Post` before the final LLM call.
* **Be efficient:** Prefer metadata filters from Mangle-Pre and keep retrieval `topK` small.
* **Cite your sources:** Always return `doc_id`s for the context used. Explain why any documents were dropped by Mangle-Post.
* **Track your work:** Update `TODO.md` as you complete implementations or discover new tasks.

### Updating Project Knowledge (Critical Task)

To ensure the project's ground truth is always current, you **must** use the `update-context` command after making significant changes. This keeps `docs/CONTEXT.md` accurate.

```bash
# TOOL DEFINITION: update-context
#
# DESCRIPTION:
#   Analyzes the provided code changes and generates a concise Markdown
#   snippet to be appended to `docs/CONTEXT.md`. This file is the
#   single source of truth for the project's architecture and state.
#
# USAGE:
#   update-context --files "<file1.go> <file2.go>" --summary "<A brief, clear summary of the change>"
#
# WHEN TO RUN THIS TOOL:
#   - A new dependency or provider is added (e.g., `qdrant/go-client`).
#   - A core data structure is modified (e.g., fields added to `Query` or `Answer`).
#   - New configuration keys are added or changed in `config.yaml`.
#   - A fundamental architectural decision is made or changed.
#
# EXAMPLE:
#   human: "@workspace Based on the last commit, please update the context."
#   agent: "Understood. Running the update command."
#   agent_tool_call: update-context --files "internal/providers/dense/chroma.go sdk.go" --summary "Added a new ChromaDB client for dense vector retrieval, integrated into the main SDK as an optional retriever."
````

### Restrictions (Don'ts) ❌

  * **Don’t go rogue:** Do not access external URLs or services unless configured in the environment.
  * **Don’t hallucinate:** Only generate answers grounded in the vetted context from the pipeline. Do not invent architectural decisions.
  * **Don’t assume:** Only document what has actually been implemented. Do not add hypothetical features to `docs/CONTEXT.md`.
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

* **401 from LLM**: set `GOOGLE_API_KEY`.
* **No results**: relax filters.
* **Long answers / token overrun**: reduce `topK` or truncate doc chunks before LLM call.

---

## 12) Security Notes

* Never log secrets or raw user inputs containing PII.
* Enforce tenant/visibility checks in **Mangle‑Post**.
* Treat `AGENTS.md` as code: review changes via PR.

---

*This AGENTS.md is intentionally concise for agents. Human‑oriented details belong in `README.md` and `docs/`.*
