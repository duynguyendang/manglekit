# AGENTS.md — Manglekit Coding Agent Configuration (2025.10)

*Last updated: 2025-10-23*

---

## Overview

This document defines how coding agents interact with the **Manglekit** codebase.
It specifies conventions, automated tasks, and safety rules to ensure that all AI-driven modifications stay consistent with the system design and context documentation.

Agents use this file as an **operational manual** to perform reasoning, refactoring, documentation updates, and observability instrumentation within the Go SDK.

---

## 1. Agent Goals

Coding agents working in this repository should be capable of:

* Understanding the full architecture from `docs/HLD.md`.
* Updating context and documentation automatically when code changes in `docs/CONTEXT.md`.
* Maintaining observability and performance best practices.
* Identifying gaps and inconsistencies in provider registration, orchestrator wiring, or options mapping.

---

## 2. File Structure Awareness

Agents must recognize the following key locations and their purposes (aligned with repo layout):

* `core/`: Contracts, types, diapi, handler/factory interfaces, observability.
* Root: `builder.go`, `registry.go` (builder + registry), `go.mod`.
* `sdk/`: Config→builder bridge (`sdk.FromConfig`).
* `config/`: YAML parsing, normalization, validation.
* `pipeline/`: Orchestrators and stages (Sandwich, Declarative).
* `internal/providers/`: Families of retrievers, rerankers, LLMs, rules, state, schema parsers.
* `internal/embedders/`, `internal/vectorstores/`: Embedder and vector store providers and handlers.
* `providers/all/`: Convenience registrar for standard providers.
* `docs/`: Technical docs and architecture rules (`docs/rules/manglekit-arch.yml`).

---

## 3. Context-Aware Reasoning

Before performing any modification, agents must:

1. Load and parse `docs/CONTEXT.md` completely.
2. Use it as a single source of truth for architecture, module dependencies, and known gaps.
3. Reflect any significant change in the code (e.g., API signature, module behavior, new provider) back into the `CONTEXT.md` file.

Agents should avoid scanning the entire repo blindly; instead, rely on structured summaries and cross-references within `CONTEXT.md`.

---

## 4. Editing Policies

When proposing or committing changes:

* Never remove sections or metadata blocks from `CONTEXT.md`.
* Maintain the YAML front-matter.
* Use stability icons (✅ / ⚠️ / ❌) consistently.
* Preserve Markdown headings, tables, and code references.
* Prefer minimal, targeted updates with accurate summaries.

---

## 5. Commit Conventions

Agents should follow semantic commit conventions:

```
feat: introduce new feature or provider
fix: resolve bug or panic
refactor: improve structure without changing behavior
docs: update documentation (includes CONTEXT.md)
chore(context): auto-update CONTEXT.md after code change
test: add or fix unit tests
```

---

## 6. Automatic Context Update (New)

### Keeping the Project Context Updated (Automated Task) ✅

Every time the source code changes in a way that may affect architecture or runtime behavior,
the coding agent must **refresh the project context** in `docs/CONTEXT.md`.

#### 1. When to Trigger an Update

Run or trigger the **context refresh** automatically whenever any of these happen:

* A Go file under `builder`, `pipeline`, `core`, or `internal/providers` is modified.
* A new provider, retriever, reranker, or LLM client is added.
* A field or contract is changed in `core/types.go` or `core/options.go`.
* A new configuration key or environment variable is introduced.
* Known gaps, lifecycle contracts, or observability hooks change.
* The orchestration pattern (Sandwich, Declarative, etc.) is modified.

#### 2. The Command Interface

```bash
# TOOL: update-context
#
# DESCRIPTION:
#   Analyzes the diff or file list and regenerates the relevant sections of `docs/CONTEXT.md`.
#   The agent must summarize architecture, affected modules, and known gaps.
#
# SYNTAX:
#   update-context --files "<list-of-files>" --summary "<short summary>" [--auto]
#
# OPTIONS:
#   --files    Files that changed (space-separated)
#   --summary  One-line human-readable summary of what changed
#   --auto     (optional) When specified, run automatically after commits or code edits
#
# EXAMPLES:
#   agent_tool_call: update-context --files "builder.go sdk.go" --summary "Refactored builder to support dynamic provider resolution."
#   agent_tool_call: update-context --files "internal/providers/dense/chroma.go" --summary "Added new Chroma vector retriever."
```

#### 3. Agent Automation Logic

When operating as a coding agent (e.g., in Cursor, Claude Projects, Copilot Studio, or Cody):

1. **Monitor** recent file edits, commits, or merges.
2. If relevant files match the patterns above, automatically:

   * Extract module/function names from diffs.
   * Summarize what was added, removed, or refactored.
   * Update the matching sections of `docs/CONTEXT.md` (Architecture, Components, Known Gaps, etc.).
   * Bump the `last_updated:` timestamp in the YAML frontmatter.
3. **Validate** that the machine-readable tables remain consistent (e.g., status icons, Known Gaps table).
4. **Commit the update** with message:

   ```
   chore(context): auto-update CONTEXT.md to reflect recent code changes
   ```

#### 4. Agent Safety Rules

* Never hallucinate: only update based on code actually present in the repo.
* Never delete large sections of CONTEXT.md; update only what has changed.
* Always preserve Markdown formatting and YAML metadata.
* Include file paths and line references when adding new Known Gaps.

---

## 7. Auto-Update Hook

Agents operating inside this repo must run:

```bash
make context-refresh
```

or

```bash
agent_tool_call: update-context --auto
```

after any commit that modifies Go source files under `core/`, root `builder.go`, or `pipeline/`.
If `make context-refresh` is unavailable, prefer the agent tool call.

---

## 8. Observability and Logging Rules

Agents must preserve the following principles:

* Never print directly to stdout in production paths.
* Use `core.Logger`, `core.Tracer`, and `core.Meter` interfaces.
* Ensure proper `Close()` and cleanup of registered resources.
* Always record latencies and token usage in metrics when calling LLMs or retrievers.

---

## 9. Testing Enforcement

Agents must maintain or extend test coverage when altering code in these areas:

* Pipeline orchestrator and stages: `pipeline/sandwich_test.go`, `pipeline/sandwich_selection_test.go`, `pipeline/stage_llm_test.go`, `pipeline/pipeline_test.go`; add or update `pipeline/declarative/*_test.go` when modifying declarative.
* Provider families under `internal/providers/*`.

Test names should follow `Test<Component>_<Behavior>` naming pattern.

---

## 10. Summary

This file defines how coding agents maintain, reason about, and update Manglekit.
The automation described here ensures that `docs/CONTEXT.md` always mirrors the true state of the codebase.

> *“An agent is only as smart as its context — keep it fresh, structured, and faithful.”*

---

## 11. Architecture Rules (Enforced)

These rules codify the new Builder/Registry/Factory/Provider pattern. Agents must adhere to them for every change.

- Layered dependencies
  - core must not import internal/providers, pipeline, or the root module.
  - pipeline must not import internal/providers (or any concrete providers).
  - Providers import only core (and standard/external libs). No provider imports the builder.
- Provider registration
  - Every provider exposes an Options struct implementing `core.ProviderOptions` (name + kind) with `yaml` tags for config.
  - Register providers via the generic `manglekit.Register[T, D, O]` with a typed factory; avoid ad‑hoc maps or stringly registration.
  - Use typed Deps (from `core/diapi`) in factories. Do not accept the builder as a dependency in factories; handlers construct deps.
- Handlers per kind
  - Build logic belongs to `core.ComponentHandler` implementations, one per kind. Handlers assemble `diapi.*Deps` and call factories.
  - When adding a new kind, include: (1) a handler, (2) a build order slot in `builder.go`, and (3) provider registration.
- Orchestrators
  - Orchestrator options implement `core.ProviderOptions` with kind `core.KindOrchestrator`.
  - Register an orchestrator factory and a matching handler; selection of component names must be explicit in options (no map iteration).
  - State provider selection must be configured (not “first entry”).
- Configuration binding
  - All options must be yaml‑tagged and mapstructure‑friendly; file path fields should use `path:"resolve"` when applicable.
  - `sdk.FromConfig` is the only bridge from YAML to builder calls; do not parse config inside providers or orchestrators.
- Observability & lifecycle
  - No direct stdout prints in production paths; use `core.Logger`.
  - Collect `core.ResourceCloser` from components (via handler) and drain LIFO in orchestrators.
  - Record standard metrics for stages and LLM token usage; wire `core.Tracer` if provided.

Automated checks:
- Keep `docs/rules/manglekit-arch.yml` passing. If a rule must be relaxed, add rationale to `docs/ADR.md` and update the rule.
- If you add or modify kinds/handlers/factories, update diagrams and Known Gaps in `docs/CONTEXT.md`.

---

## 12. Implementation Checklists

When adding or changing components, follow the checklists below.

- New provider (retriever, reranker, embedder, LLM, vector store, rules, state)
  - Define `Options` implementing `core.ProviderOptions` with proper `yaml` tags.
  - If dependencies are required, declare through `diapi` dep marker interfaces (e.g., `EmbedderDep`, `VectorStoreDep`, `SubRetrieversDep`).
  - Register with `manglekit.Register[T, D, O]` using a typed factory that accepts the matching `diapi.*Deps`.
  - Ensure a handler for the kind exists (or add one under `internal/providers/<kind>/handler.go`).
  - Add tests under the provider folder; ensure determinism and lifecycle coverage.
  - Run context refresh and update Known Gaps if behavior or wiring differs from the standard.

- New orchestrator
  - Create `Options` implementing `core.ProviderOptions` (kind = `core.KindOrchestrator`).
  - Register a typed factory and a dedicated handler; avoid arbitrary dependency selection (require names in options).
  - Cover stage metrics, logging, and closers. Add tests for explicit component selection.
  - Refresh docs and Known Gaps; update HLD/LLD diagrams if flow semantics differ.

- Core or DI changes
  - Never import providers or pipeline from core.
  - If `diapi` contracts change, update all handlers and factories to keep signatures aligned.
  - Bump `last_updated` across `CONTEXT.md`, `LLD.md`, and refresh `docs/rules/` as needed.

Reject / fix anti‑patterns:
- Factories that accept the builder directly (use typed deps instead).
- Orchestrators or providers picking the “first” dependency from a map.
- init() registration inside providers (use explicit `Register` functions).
- Magic strings for execution context; use typed contexts or adapters where applicable.

---

## 13. Post‑Change Actions

- Run tests: `go test ./...` and ensure provider/orchestrator coverage.
- Run static checks that use `docs/rules/manglekit-arch.yml`.
- Trigger context update: `make context-refresh` or `agent_tool_call: update-context --auto`.
