# Manglekit Context Engine (MCE): Groundbreaking Architecture Specification

## 1. Vision: The Synthetic Cognitive Organism

Standard RAG systems are **reactive**—they wait for a query and fetch text. The **Manglekit Context Engine** is **proactive** and **autonomous**. It treats context as a living resource that is continuously refined, consolidated, and protected by **Deterministic Logic** (Datalog).

---

## 2. Tiered Cognitive Memory (TCM) Architecture

To solve the memory decay and token budget issues, we implement a four-tier architecture inspired by human neurobiology.

| Layer | Type | Mechanism | Manglekit Implementation |
| --- | --- | --- | --- |
| **L1: Working** | **Hot** | **Rolling Stream** (Ring Buffer) | Transient history and logs stored in `internal/engine/memory`. |
| **L2: Semantic** | **Warm** | **Flat INT8 + IVF-PQ (Local) or External HNSW (Remote)** | Default: Zero-allocation mmap with Inverted File Index for low-RAM embedded scale (100k-1M). Scale-out: Delegated to external Vector DB via gRPC. |
| **L3: Episodic** | **Cold** | **Long-Term Fact Store** | Consolidated facts from past sessions stored in Redis or Graph databases. |
| **L4: Anchors** | **Sink** | **Global Store** (Immutable) | System prompts, core business rules, and "Sticky Facts" (Promoted by Datalog). |

---

## 3. The Neuro-Symbolic Pipeline: "Why-How-What"

### 3.1. Retrieval & Logic-Aware Selection

* **WHY**: Raw vector retrieval is purely statistical; it often includes irrelevant or deprecated information.
* **HOW**: Use **Hybrid Attention Selection**. We combine Neural (Vector) scores with Symbolic (Datalog) Priority.
* **WHAT**:
* **Retrieval**: `HNSWAdapter` fetches  candidates.
* **Datalog Filter**: The Engine runs `context_policy(ID, Outcome)`.
* **Outcome**: If `Outcome == "DENY"`, the fragment is immediately purged, ensuring **Hard Security Guardrails**.



### 3.2. Context Compression & Fact Flattener

* **WHY**: Large JSON/Struct objects waste tokens and confuse the attention mechanism.
* **HOW**: Use the `ToFacts` and `Flatten` mechanisms to convert data into its atomic logical form.
* **WHAT**: Instead of sending a 2KB JSON user object, Manglekit sends 5-10 Datalog quads (e.g., `user_role("u1", "admin")`), reducing context size by up to 90% while maintaining 100% semantic fidelity.

---

## 4. Advanced SOTA Techniques

### 4.1. The "Dreaming" Process (Memory Consolidation)

To achieve truly groundbreaking memory management, Manglekit implements an asynchronous **Dreamer Service**.

* **Mechanism**: A background Goroutine scans the `L1 Rolling Stream` during idle cycles.
* **Consolidation**: It uses a small LLM (e.g., Gemini Flash) to abstract raw chat logs into **Datalog Facts**.
* **Promotion**: If a fact is repeated or deemed critical by logic (e.g., `is_critical(Fact)`), it is **Promoted** to the `L4 Global Store` via **Sticky Attention**.
* **Result**: The Agent "grows wiser" over time without increasing RAM usage.

### 4.2. Positional Attention Optimization (The Prompt Sandwich)

To combat the "Lost-in-the-Middle" phenomenon, MCE automatically reorders the context window.

* **Top (Sink)**: **Global Store Anchors** (System instructions, Critical facts).
* **Middle**: Summarized background knowledge (Low priority).
* **Bottom (Recency)**: The Top-1 relevant fact and the User Query.
* **Benefit**: Exploits the **Primacy** and **Recency** bias of Transformer models.

### 4.3. Speculative Inference (Prefetching)

* **Mechanism**: The `PolicyEngine` uses Datalog to predict the most likely next requirement using `requires(Req, "capability")`.
* **Action**: If logic predicts a high probability for a tool call (e.g., "Check Bank Balance"), the `CandidateGatherer` pre-fetches account metadata into the L1 cache *before* the LLM finishes processing.

---

## 5. Implementation: The Context Manager

```go
// internal/attention/manager.go

type ContextManager struct {
    engine   core.Evaluator
    memory   *TieredMemory
    budget   int // Token limit
}

func (m *ContextManager) Assemble(ctx context.Context, query string) (string, error) {
    // 1. Speculative Prefetching (Phase 4.3)
    go m.prefetchLikelyRequirements(ctx, query)

    // 2. Hybrid Gathering (Phase 3.1)
    candidates := m.memory.GatherCandidates(ctx, query)

    // 3. Logic-Aware Selection (Bucketing)
    anchors := make([]Fragment, 0)
    relevant := make([]Fragment, 0)
    
    for _, frag := range candidates {
        // Datalog check: context_policy(ID, Outcome)
        outcome, _ := m.engine.AssessFragment(ctx, frag)
        switch outcome {
        case "ANCHOR": // Sticky Attention
            anchors = append(anchors, frag)
        case "ALLOW":
            relevant = append(relevant, frag)
        case "DENY":
            continue // Hard Filter
        }
    }

    // 4. Positional Assembly (Phase 4.2)
    return m.formatPrompt(anchors, relevant, query), nil
}

```

---

## 6. Datalog Governance Rules (`std.dl`)

```prolog
% Logic-Driven Attention Control

% Rule 1: Security Taint (Access Control)
context_policy(Frag, "DENY") :- 
    label(Frag, "internal_only"), 
    not(meta("user_clearance", "admin")).

% Rule 2: Deprecation Check
context_policy(Frag, "DENY") :- 
    json_str(Frag, "status", "deprecated").

% Rule 3: Sticky Fact Promotion (Attention Sink)
is_sticky(Frag) :- 
    quad(Frag, "has_role", _, _).
is_sticky(Frag) :- 
    violation_rule(Frag, _).

% Rule 4: Critical Priority
anchor(Frag) :- is_sticky(Frag).

```

---

## 7. Summary of Groundbreaking Impact

1. **Infinite Memory with Zero Bloat**: Through the **Dreaming Process**, the agent compresses a year of conversation into a few hundred KB of Datalog facts.
2. **Unbreakable Guardrails**: No matter how the LLM is prompted, it cannot "see" denied context because the **PolicyEngine** filters it at the byte level before prompt assembly.
3. **Low-Latency Cognitive Response**: **Speculative Inference** ensures the data is in L1 RAM before the AI even "thinks" about asking for it.
4. **Neuro-Symbolic Self-Correction**: When the `Reflect` phase detects a hallucination, MCE automatically injects "High-Priority" clarifying facts into the next retry loop.