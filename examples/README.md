## Manglekit Canonical Examples

This directory contains the **7 Core Patterns** of Manglekit. These examples are designed to demonstrate the transition from "Probabilistic AI" to **"Deterministic Neuro-Symbolic Agents"**.

-----

### 🟢 Tier I: The Foundation (Input/Output Governance)

*(Start here to understand how Manglekit bridges the gap between Go Types and LLM Text.)*

| Example | Pattern | Key Concept |
| :--- | :--- | :--- |
| **`structured_data_bridge`** | **Type-Safe Extraction** | Demonstrates the **Extractor Adapter**. Converts raw unstructured text into strict Go Structs and Datalog Facts, enforcing type safety at the edge. *(Merges old `extractor_demo` & `dynamic_pricing`)* |
| **`config_driven_bot`** | **Zero-Code Config** | Shows how to initialize Agents, Providers, and Policies entirely from `mangle.yaml`, decoupling deployment from logic. |

-----

### 🟡 Tier II: The Cognitive Loop (Neuro-Symbolic Type 3)

*(The Core Value: Using Logic to plan, verify, and correct the Neural Intuition.)*

| Example | Pattern | Key Concept |
| :--- | :--- | :--- |
| **`infrastructure_copilot`** | **The Repair Loop** | **[Must See]** An SRE Agent that generates Terraform plans. If the plan violates budget/security policies, the Engine triggers a **RETRY with Feedback**, forcing the LLM to self-correct until compliant. *(Upgraded `sre_guardrail`)* |
| **`logistics_optimizer`** | **System 2 Verification** | Solves complex constraint problems (e.g., seating charts, scheduling). The LLM proposes candidates ("Intuition"), and Datalog validates constraints ("Math"). If invalid, it loops back. *(Upgraded `seating_solver`)* |
| **`autonomous_router`** | **State-Aware Routing** | Replaces complex `if/else` chains with Datalog rules. Routes user requests to specific workers (Human, Smart Model, Cheap Model) based on dynamic context (Sentiment, Tier, System Load). *(Upgraded `steering`)* |

-----

### 🔴 Tier III: Enterprise Security & Compliance

*(Applying mathematical certainty to high-risk business processes.)*

| Example | Pattern | Key Concept |
| :--- | :--- | :--- |
| **`fintech_underwriter`** | **Recursive Reasoning** | A loan approval system handling complex nested ownership structures. Demonstrates Datalog's power in **Recursive Rules** (Graph Traversal) which Standard SQL/LLMs struggle with. |
| **`data_firewall`** | **Taint Analysis** | A security gateway that tracks **Information Flow**. It labels data (e.g., "PII", "Secret") and uses policy to prevent sensitive data from leaking into public LLM prompts or logs. |

-----

## 🚀 How to Run

Prerequisites: **Go 1.24+** and an API Key (Google AI or OpenAI).

```bash
# Example: Running the Self-Healing Infrastructure Copilot
cd examples/infrastructure_copilot
export GOOGLE_API_KEY="your_key"
go run .
```