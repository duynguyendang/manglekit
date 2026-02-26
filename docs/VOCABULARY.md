# **Manglekit Blueprint Standard Vocabulary (v2)**

This document defines the reserved predicates (keywords) used in Manglekit Blueprints (`.dl`) under the new v2 Hexagonal OODA Architecture. Since Manglekit v2 leverages the Google Mangle Leapfrog-Triejoin evaluator and maps logic directly to prompt context via the `PromptCompiler`, adhering to this vocabulary ensures strict zero-trust governance and accurate Generative steering.

---

## 1. System Predicates (Reserved)

These predicates are evaluated by the **Auditor** (Shadow Audit Phase) and the **Prompt Compiler** (Decide Phase).

| Predicate | Signature | Role | Behavior |
| --- | --- | --- | --- |
| **`halt`** | `halt(Msg)` | **Blocking (Tier 0/1)** | If derived during the `Verify()` phase, the `SupervisedAction` strictly aborts the OODA loop. The `Msg` provides the violation reason. In Generative translation, compiler emits: `NEVER allow action where: halt(...)`. |
| **`warn`** | `warn(Msg)` | **Soft Guidance (Tier 2/3)** | Marks the state as `VerifyStatusWarning`. Does not strictly block execution, but triggers the Teacher-Student loop if generated during `Decide`. Compiler emits: `Avoid action if possible: warn(...)`. |
| **`weight`** | `weight(FactId, W)` | **Context Shaving** | Assigns a confidence float `W` (0.0 - 1.0) to a fact. The `ContextManager` uses this to perform **Intelligent Shaving**, aggressively dropping facts where `W < 0.5` if the token budget exceeds the 24,000 threshold. |

---

## 2. Dynamic Input Facts (Context)

During the **Observe** and **Orient** phases, the `PerceptionPort` and `Proposer` transform external intent into Datalog facts (Atoms) seamlessly injected into the active Frame.

| Predicate | Signature | Description |
| --- | --- | --- |
| **`semantic_similarity`** | `semantic_similarity("Intent", Object)` | MRL vectors (INT8 quantized) matched from `flat_simd` vector store for the current Intent. |
| **`structural_fact`** | `structural_fact("HardContext", Object)` | N-Quad facts pulled from BadgerDB matching the current graph sandbox. |
| **`input`** | `input_param(Predicate, Object)` | Flattened struct parameters derived via Zero-Config Reflection using the `mangle` struct tags. |

---

## 3. Cognitive Pressure (EAST)

Manglekit v2 does not use linear routing commands like the legacy `route(Target)`.
Instead, the **Orchestrator** dynamically controls the LLM's Persona and formatting based on Entropic Activation Steering (EAST).

If `halt` or `warn` constraints fail consecutively during the Teacher-Student loop:
1. `LogicSuccess` mathematical metric drops.
2. `SteeringMagnitude (P)` scales up logarithmically.
3. If `P > 0.8`, the Kernel injects **Paradox Headers**, forcing highly conservative, deterministic Chain-of-Thought output constraints.

---

## 4. Blueprint Examples

### Example 1: Tier 0 Kernel Axiom (Strict Security)
*Scenario: Under no circumstance can the system write to the root namespace.*

```prolog
% Any action attempting to modify the 'system' namespace is catastrophically blocked.
halt("Unauthorized System Mutation") :- 
    input_param("target_namespace", "system").
```

### Example 2: Tier 2 Learned Heuristic (Soft Guidance)
*Scenario: AI induced a rule from a Markdown memo: We shouldn't use overly casual tones for enterprise clients.*

```prolog
% If client is enterprise, try to avoid colloquialisms. Generates a warning if 
% the content verifier detects violations.
warn("Casual tone detected for Enterprise Client") :- 
    input_param("client_tier", "enterprise"),
    input_param("tone_detected", "slang").
```

### Example 3: Contextual Weighting
*Scenario: A memory fragment was recalled, but it's very old. Downgrade its context weight so the ContextManager shaves it first under pressure.*

```prolog
weight(FactID, 0.3) :-
    structural_fact("source", FactID),
    input_param("age_days", Age), 
    Age > 365.
```