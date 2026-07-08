# Manglekit Roadmap

This document records the aspirational **neuro-symbolic** vision behind Manglekit so
that intent is preserved even as earlier, unwired scaffolding has been removed from
the codebase.

## What ships today (live)

Manglekit is a lean **supervisor → engine** governance kernel. The following
capabilities are real and exercised by the live path (`NewClient` → `client.Supervise`):

- **Tiered GenePool (Symbolic reasoning)**: Datalog `.dl` policy files loaded in tiers
  (Tier 0 axioms, Tier 1 standard library, Tier 2 user policy) by the engine
  (`internal/engine/solver.go`). This *is* the "GenePool" — a set of Datalog policies,
  not a separate Go engine.
- **Shadow Audit (fail-closed verification)**: The supervisor enforces a real
  fail-closed pre/post gate against the active Datalog policy on every supervised
  action. A violated policy blocks execution rather than passing through.
- **OODA cognitive loop**: `sdk/ooda/` implements the live Observe–Orient–Decide–
  Verify–Act pipeline (with its own `EASTState`), independent of the removed
  `internal/kernel` scaffolding.
- **The Silo (memory)**: BadgerDB-backed SPOg quad storage and vector embeddings
  (`internal/engine` + `internal/statemanager`).

## Planned / not yet built

These ideas were previously represented by dead, mock-heavy scaffolding under
`internal/genepool`, `internal/kernel`, `internal/reasoning/*`, `internal/orchestrator`,
and `internal/adapters/mangle`. They are **not implemented** today and are recorded
here as direction, not as shipped features:

- **Teacher–Student rule induction**: the genuinely novel capability. Learn new
  Tier-2 Datalog rules from experience via LLM distillation, with a *real*
  contradiction check that induced rules do not violate Tier 0/1 axioms. Previously
  stubbed in `internal/reasoning/inductor` (empty-atom output, no contradiction check).
- **Multi-stream proposal**: a proposer that drafts several candidate plans/actions
  for the verifier to rank (previously `internal/reasoning/proposer`).
- **Gene "crystallization" & signatures**: versioning/cryptographic signatures over
  induced genes so they can be promoted from tentative to trusted
  (previously `internal/genepool` with hardcoded word-list "genes" and zero signatures).

## Why the earlier scaffolding was removed

The packages above (`internal/orchestrator`, `internal/reasoning/{proposer,verifier,
inductor}`, `internal/genepool`, `internal/kernel`, `internal/adapters/mangle`,
`internal/audit`) had **zero importers** — not even test files — and duplicated
functionality already present in the live path:

- The fake "Shadow Audit" (`internal/adapters/mangle` `Verify` returning an
  empty-atom pass) was **security theater**; removing it is a safety win because the
  live supervisor's fail-closed gate is what actually protects execution.
- The dead `GenePoolPort`/`ReasoningPort` consumers pointed at mock implementations
  rather than the real engine adapters built in
  `internal/supervisor/sdk_adapter.go`.
- The `internal/kernel` "EAST loop" and embedded `.dl` genes were an
  unwired re-implementation of `sdk/ooda` and the engine's tiered loading.

The design intent now lives in this document rather than in rotting mock code.
