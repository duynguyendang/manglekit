# AGENTS.md — Manglekit Coding Agent Configuration (2025.10)

*Last updated: 2025-10-13*

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

Agents must recognize the following key directories and their purposes:

* `core/`: Shared data structures and observability interfaces.
* `builder/`: SDK initialization, registry logic, and provider wiring.
* `pipeline/`: Execution orchestrators (Sandwich and Declarative).
* `internal/providers/`: Families of retrievers, rerankers, embedders, LLMs, and vector stores.
* `docs/`: Technical documentation and agent context files.

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

after any commit that modifies Go source files under `core/`, `builder/`, or `pipeline/`.

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

* `pipeline/sandwich_test.go`
* `pipeline/declarative/orchestrator_test.go`
* Provider families under `internal/providers/*`

Test names should follow `Test<Component>_<Behavior>` naming pattern.

---

## 10. Summary

This file defines how coding agents maintain, reason about, and update Manglekit.
The automation described here ensures that `docs/CONTEXT.md` always mirrors the true state of the codebase.

> *“An agent is only as smart as its context — keep it fresh, structured, and faithful.”*
