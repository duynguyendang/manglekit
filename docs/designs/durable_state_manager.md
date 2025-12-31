# High-Level Design (HLD): Durable State Manager (v2.0)

This HLD defines the **Durable State Manager**, an orchestration layer designed to enhance Manglekit's resilience. It acts as a high-level wrapper around the **`core.StateProvider`** interface, transforming it from a simple storage primitive into a semantic execution manager.

---

## 1. Architectural Strategy: The Layered Approach

The Durable State Manager is an **Enhancement**, not a replacement for the `core.StateProvider`. It separates **Storage Logic** from **Execution Logic**.

* **Storage Layer (`core.StateProvider`)**: Handles the physical I/O (Get/Set/Delete) of raw byte/interface data to backends like Disk or Redis.
* **Management Layer (Durable State Manager)**: Handles the "Action Sandwich" state orchestration, deciding *what* to save, *when* to save it, and *how* to recover the logic engine.

---

## 2. Data Model: The `SessionState`

To ensure full recovery, the Manager packages multiple Manglekit components into a single serializable **`SessionState`** object before passing it to the `StateProvider`.

| Field | Description | Source |
| --- | --- | --- |
| **SessionID** | Unique identifier for the persistent thread. | `ExecutionParams.SessionID` |
| **ActiveEnvelope** | The current `core.Envelope` including Payload, Metadata, and Labels. | `core.Envelope` |
| **ExecutionCtx** | Current `RetryCount`, `FeedbackHistory`, and `CurrentHistory`. | `sdk.ExecutionParams` |
| **LogicalFacts** | The Datalog facts derived during the last successful reflection. | `internal/engine` |

---

## 3. The Checkpointing Lifecycle

The Manager integrates directly into the **`SupervisedAction`** execution flow to ensure atomic state updates.

1. **Hydration (Load)**: At the start of the `RunLoop`, the Manager uses `StateProvider.Get()` to retrieve the `SessionState`. It reconstructs the `Envelope` and primes the `PolicyEngine` with stored facts.
2. **Governance execution**: The action undergoes **Assess -> Execute -> Reflect**.
3. **Atomic Checkpoint (Save)**: **Only** after the **Reflect** phase confirms the output is valid (Alignment OK), the Manager triggers `StateProvider.Set()`. This prevents "poisoned" or invalid states from being persisted.
4. **Continuity**: The loop continues to the next action, knowing the progress is safely on disk.

---

## 4. Semantic Recovery (Hydration Logic)

When a process restarts, the Durable State Manager performs more than just a data load; it performs **Semantic Re-constitution**:

* **Type Reconstruction**: It uses `Envelope.ContentType` to unmarshal the `any` payload back into the correct Go struct.
* **Engine Priming**: It re-injects all `LogicalFacts` into the Mangle runtime, ensuring that the next **Assess** phase has full awareness of the session's history.
* **Feedback Alignment**: It restores the `FeedbackHistory` so the LLM does not repeat past mistakes even after a cold start.

---

## 5. Summary of Roles

| Feature | `StateProvider` (Storage) | Durable State Manager (Logic) |
| --- | --- | --- |
| **Responsibility** | Raw Data I/O | Execution Continuity |
| **Unit of Work** | `any` state object | `SessionState` (Envelope + Params) |
| **Timing** | When called manually | Automatic Checkpointing (Post-Reflect) |
| **Context** | Stateless Key-Value | Stateful Neuro-Symbolic recovery |

---