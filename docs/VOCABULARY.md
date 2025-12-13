# **Manglekit Blueprint Standard Vocabulary**

This document defines the reserved predicates (keywords) and standard configuration keys used in Manglekit Blueprints (`.dl`). Sticking to this vocabulary ensures portability across different Agents and consistency in the Neuro-Symbolic Engine.

---

##1. System Predicates (Reserved)These predicates control the **Flow** and **Lifecycle** of the Agent. They are interpreted directly by the Engine's State Machine.

| Predicate | Signature | Role | Behavior |
| --- | --- | --- | --- |
| **`deny`** | `deny(Reason)` | **Blocking** | Stops execution immediately. Returns an `AlignmentError` with the `Reason`. |
| **`route`** | `route(NextStep)` | **Steering** | Completes the current step and routes execution to `NextStep` (Action Name). *Replaces legacy `next_step`.* |
| **`retry`** | `retry(Feedback)` | **Correction** | Triggers a self-correction loop. Sends `Feedback` to the AI and retries the current action. *Replaces legacy `correction`.* |
| **`config`** | `config(Key, Value)` | **Config** | Injects dynamic configuration into the Action's context. *Mapped to `action_config` in code.* |

---

##2. Standard Configuration Keys (`config`)When using `config(Key, Value)`, use these standard keys to ensure your Adapters (Genkit/OpenAI) understand the intent.

###2.1. Prompting & Persona (`prompt.*`)Controls *how* the AI speaks or behaves.

| Key | Values (Examples) | Description |
| --- | --- | --- |
| `prompt.tone` | `"formal"`, `"friendly"`, `"pirate"` | Sets the tone of voice. |
| `prompt.lang` | `"en"`, `"vi"`, `"json"` | Forces output language or format. |
| `prompt.strategy` | `"cot"` (Chain of Thought), `"react"` | Enforces a reasoning strategy. |
| `prompt.template_id` | `"tpl_sales_v1"`, `"tpl_support_angry"` | IDs mapping to external text files/CMS. |
| `prompt.safety` | `"strict"`, `"lenient"` | Adjusts safety guardrails in the prompt. |

###2.2. Model Parameters (`llm.*`)Controls the *hyperparameters* of the underlying model.

| Key | Values | Description |
| --- | --- | --- |
| `llm.temperature` | `"0.0"` - `"2.0"` | Creativity vs. Determinism. |
| `llm.model` | `"gpt-4"`, `"gemini-pro"` | Hot-swaps the underlying model. |
| `llm.max_tokens` | `"1024"`, `"4096"` | Limits output length. |
| `llm.json_mode` | `"true"`, `"false"` | Forces JSON mode (if supported by provider). |

###2.3. System & Network (`sys.*`)Controls runtime behavior.

| Key | Values | Description |
| --- | --- | --- |
| `sys.timeout` | `"30s"`, `"5m"` | Execution timeout override. |
| `sys.cache` | `"true"`, `"false"` | Enable/Disable semantic caching for this request. |
| `sys.priority` | `"high"`, `"low"` | Priority for job queues (if async). |

---

##3. Input Facts (Context)These predicates are automatically injected by the Engine *into* the Blueprint. You use them in the `BODY` of your rules to make decisions.

| Predicate | Signature | Description |
| --- | --- | --- |
| **`input`** | `input.field_name` | Access fields of the input payload (via Reflection). E.g., `input.amount > 500`. |
| **`meta`** | `meta(Key, Val)` | Metadata from the Envelope (e.g., User ID, Session ID). |
| **`attempt`** | `attempt(N)` | Current retry count (starts at 0). Useful for escalating logic on failures. |
| **`label`** | `label(Tag)` | Security taint labels (e.g., `label("pii")`). |

---

##4. Blueprint Examples###Example 1: Adaptive Retry Strategy*Scenario: If the AI fails twice, switch to a smarter model and force Chain-of-Thought.*

```prolog
% Attempt > 1 -> Use Chain of Thought
config("prompt.strategy", "cot") :- attempt(N), N > 1.

% Attempt > 2 -> Escalation: Switch to smarter model
config("llm.model", "gpt-4-turbo") :- attempt(N), N > 2.

```

###Example 2: Dynamic Persona based on User Tier*Scenario: VIPs get polite, long answers. Free users get concise answers.*

```prolog
% VIP User
config("prompt.tone", "formal") :- meta("user_tier", "vip").
config("llm.max_tokens", "4000") :- meta("user_tier", "vip").

% Free User
config("prompt.tone", "concise") :- meta("user_tier", "free").
config("llm.max_tokens", "100")  :- meta("user_tier", "free").

```

###Example 3: Security Routing*Scenario: If input contains PII, verify via a secure logic step instead of direct LLM.*

```prolog
% Block unsafe usage
deny("Cannot send PII to public LLM") :- 
    label("pii"), 
    config("llm.model", "public-gpt").

% Route to secure handler
route("secure_pii_processor") :- label("pii").

```

---