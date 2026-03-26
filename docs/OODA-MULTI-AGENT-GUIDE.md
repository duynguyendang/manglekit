# How to Build Multi-Agent Systems with Manglekit Using OODA

**A practical guide to implementing agentic systems with the OODA (Observe → Orient → Decide → Act → Verify) SDK.**

**Last Updated:** 2026-03-24 | **SDK Version:** `sdk/ooda` v2

---

## Table of Contents

1. [Overview](#1-overview)
2. [Core SDK Architecture](#2-core-sdk-architecture)
3. [CognitiveFrame & Builder](#3-cognitiveframe--builder)
4. [Interfaces](#4-interfaces)
5. [The OODA Loop (`ooda.Run`)](#5-the-ooda-loop-oodarun)
6. [Action Registry & Dispatcher](#6-action-registry--dispatcher)
7. [Dual Memory Architecture](#7-dual-memory-architecture)
8. [Audit Trail & Trust Tiers](#8-audit-trail--trust-tiers)
9. [Datalog Knowledge Base Integration](#9-datalog-knowledge-base-integration)
10. [Reference Implementation: Architect Agent](#10-reference-implementation-architect-agent)
11. [Best Practices](#11-best-practices)

---

## 1. Overview

### What is the OODA SDK?

The `sdk/ooda` package provides a structured decision-making framework for building agentic systems. It wraps a **CognitiveFrame** (the state of a reasoning epoch) through five phases:

| Phase | What Happens | Key Interface |
|-------|-------------|---------------|
| **Observe** | Capture raw input, normalize | Default (customizable) |
| **Orient** | Hydrate context from Memory (MEB + Session) | `Memory`, `KnowledgeStore`, `TransientStore` |
| **Decide** | Evaluate with Brain (Datalog policy engine) | `Brain` → `core.Decision` + `core.AuditTrail` |
| **Act** | Execute action via Dispatcher or Executor | `Dispatcher`, `Executor` |
| **Verify** | Post-act validation | `Brain.Verify`, `TransientStore` |
| **Commit** | Persist results to Memory | `Memory.Commit` |

### Why OODA?

- **Observable**: Every phase produces `AuditTrail` with matched rules and tier
- **Resilient**: Built-in retry with exponential backoff
- **Decoupled**: `Registry` maps action names to Go functions — Datalog decides, Go executes
- **Dual Memory**: Long-term (MEB) + short-term (session) for multi-turn workflows

---

## 2. Core SDK Architecture

```
sdk/ooda/
├── domain.go       # CognitiveFrame, Builder, Atom, Phase, TrustTier
├── frame.go        # Run(), Memory, Brain, Executor interfaces + phase implementations
├── registry.go     # Registry, Dispatcher, ActionEnvelope, SafeStop
├── interfaces.go   # Observer, Orienter, Decider, Verifier, Actor (optional contracts)
└── frame_test.go   # Tests
```

### Dependencies

```
sdk/ooda/
  ├── core/           # Decision, AuditTrail, ActionEnvelope, WorkflowInstance
  └── sdk/ports/      # KnowledgeStore, TransientStore interfaces
```

The SDK is **Datalog-agnostic** at the interface level. Datalog integration happens via the `Brain` interface implementation (in `internal/engine/`).

---

## 3. CognitiveFrame & Builder

### CognitiveFrame

The `CognitiveFrame` is the complete state of a single reasoning epoch:

```go
type CognitiveFrame struct {
    ID        uuid.UUID
    Timestamp time.Time
    Intent    IntentStr    // Goal or objective
    Phase     Phase        // Current OODA phase
    TaskType  TaskType     // INDUCTION, GENERATION, AUDIT, RECOVERY
    Input     string       // Raw input stimulus

    // Memory & Logic
    Context       []Atom         // Soft Logic (INT8) — observed facts, pruneable
    AttentionSink []Atom         // Hard Logic (FP32) — immutable axioms (Tier 0)
    ActiveGenes   []DomainGene   // Crystallized rules for this epoch
    RawContext    map[string]any // Legacy escape hatch

    // Reasoning
    Draft  any              // Neural proposal
    Proof  *AuditResult    // Verification trace
    Status VerifyStatus    // PENDING, FP32_PASSED, LOGIC_VIOLATION, WARNING

    // OODA Components (set via Builder)
    Memory     Memory       // Recall/Commit interface
    Brain      Brain        // Evaluate/Verify interface
    Executor   Executor     // Legacy executor
    Dispatcher *Dispatcher  // Action dispatcher

    // Dual Memory Architecture
    KnowledgeStore  ports.KnowledgeStore  // MEB-backed (persistent, long-term)
    TransientStore  ports.TransientStore   // Session (RAM-only, short-term)

    // Session State
    SessionID        string
    WorkflowID       string
    WorkflowInstance *core.WorkflowInstance

    // Configuration
    MaxRetries int
    Timeout    time.Duration
    RetryCount int
    PhaseDurations map[Phase]time.Duration
}
```

### Builder (Fluent API)

```go
import "github.com/duynguyendang/manglekit/sdk/ooda"

frame := ooda.NewBuilder().
    WithInput("Generate ADD document for project X").
    WithIntent("architecture_documentation").
    WithTaskType(ooda.TaskTypeGeneration).
    WithMemory(mebAdapter).           // Long-term knowledge (MEB)
    WithBrain(policyEngine).          // Datalog policy evaluation
    WithRegistry(registry).           // Action → Go function mapping
    WithSessionID("session-001").     // For multi-turn continuity
    WithWorkflowID("architect-csd").  // Workflow graph scope
    WithMaxRetries(3).
    WithTimeout(5 * time.Minute).
    Build()
```

### Quick Start (Minimal)

```go
// Minimal: just input, no components
frame := ooda.NewBuilder().
    WithInput("Hello world").
    WithBrain(myBrain).
    Build()

result, err := ooda.Run(context.Background(), frame)
```

---

## 4. Interfaces

### Memory

```go
type Memory interface {
    Recall(ctx context.Context, input string) ([]Atom, error)
    Commit(ctx context.Context, frame *CognitiveFrame) error
    Store(ctx context.Context, atom Atom) error
    Query(ctx context.Context, predicate string) ([]Atom, error)
}
```

- **Recall**: Called during Orient to hydrate the frame with relevant facts
- **Commit**: Called after Verify to persist the outcome
- **Store/Query**: Direct atom manipulation

### Brain

```go
type Brain interface {
    Evaluate(ctx context.Context, frame *CognitiveFrame) (*core.Decision, error)
    Verify(ctx context.Context, frame *CognitiveFrame) (*core.AuditTrail, error)
    LoadPolicy(ctx context.Context, rules string) error
}
```

- **Evaluate**: Called during Decide — returns `Decision` with `AuditTrail`
- **Verify**: Called during Verify — validates action result against policies
- **LoadPolicy**: Load additional Datalog rules at runtime

### Executor (Legacy)

```go
type Executor interface {
    Execute(ctx context.Context, frame *CognitiveFrame, decision *core.Decision) (any, error)
    Rollback(ctx context.Context, frame *CognitiveFrame, result any) error
}
```

> **Note**: Prefer `Dispatcher` over `Executor` for new implementations. The Dispatcher uses `Registry` to map action names to Go functions, which is more flexible.

### Optional: Observer, Orienter, Decider, Verifier, Actor

```go
// interfaces.go — optional contracts for custom phase implementations
type Observer interface { Observe(ctx context.Context, frame *CognitiveFrame) error }
type Orienter interface { Orient(ctx context.Context, frame *CognitiveFrame) error }
type Decider interface { Decide(ctx context.Context, frame *CognitiveFrame) error }
type Verifier interface { Verify(ctx context.Context, frame *CognitiveFrame) error }
type Actor interface    { Act(ctx context.Context, frame *CognitiveFrame) error }
```

---

## 5. The OODA Loop (`ooda.Run`)

```go
func Run(ctx context.Context, frame *CognitiveFrame) (*CognitiveFrame, error)
```

### Phase Flow

```
Observe → Orient → Decide → Act (with retry) → Verify → Commit
                                                       │
                                                       ▼
                                              Memory.Commit(ctx, frame)
```

### Phase Details

| Phase | Default Behavior | Customizable Via |
|-------|-----------------|-----------------|
| **Observe** | Captures `frame.Input` as `Atom{Predicate: "raw_input"}` | Override in your orchestrator before calling `Run()` |
| **Orient** | Queries `KnowledgeStore.Recall()` + `TransientStore.ToAtoms()` + `Memory.Recall()` | Inject `KnowledgeStore`, `TransientStore`, or `Memory` |
| **Decide** | Calls `Brain.Evaluate()` → sets `frame.Decision` + `frame.AuditTrail` | Implement `Brain` interface |
| **Act** | Dispatches via `Dispatcher.Dispatch()` or falls back to `Executor.Execute()` | Register tools in `Registry` |
| **Verify** | Checks result for "error"/"failed" strings → stores in `TransientStore`; calls `Brain.Verify()` | Implement `Brain.Verify()` |
| **Commit** | Calls `Memory.Commit()` if available | Implement `Memory` |

### Retry Logic

The Act phase has built-in retry with exponential backoff:

```go
// Retries up to frame.MaxRetries times with 1s, 2s, 3s... delays
// Calls Executor.Rollback() between retries if available
result, err := ooda.Run(ctx, frame)
// If all retries fail, returns last error
```

---

## 6. Action Registry & Dispatcher

### Registry

Maps action names (from Datalog decisions) to Go functions:

```go
registry := ooda.NewRegistry()

// Register tools
registry.MustRegister("generate_document", func(ctx context.Context, args map[string]interface{}) (string, error) {
    docType := args["doc_type"].(string)
    return fmt.Sprintf("Generated %s document", docType), nil
})

registry.MustRegister("validate_output", func(ctx context.Context, args map[string]interface{}) (string, error) {
    return "Validation passed", nil
})

// Check availability
registry.Has("generate_document") // true
registry.List()                    // ["generate_document", "validate_output"]
```

### Dispatcher

Executes actions from the registry with safety fallback:

```go
dispatcher := ooda.NewDispatcher(registry)

// Dispatch an action
result, err := dispatcher.Dispatch(ctx, "generate_document", map[string]interface{}{
    "doc_type": "ADD",
})

// With fallback for unknown actions
dispatcher := ooda.NewDispatcher(registry).WithFallback(func(ctx context.Context, args map[string]interface{}) (string, error) {
    return "Custom fallback: " + args["reason"].(string), nil
})
```

### SafeStop

When an unknown action is dispatched and no fallback is set, the system invokes **SafeStop** — a mandatory safety mechanism:

```go
// SafeStop is called automatically when:
// 1. Action not found in registry
// 2. No fallback configured
// 3. Logs "SOVEREIGN VIOLATION" + available actions

// Custom SafeStop
ooda.SafeStop = func(ctx context.Context, args map[string]interface{}) (string, error) {
    log.Printf("SafeStop: %v", args)
    return "STOPPED", nil
}
```

### Wiring the Frame

```go
registry := ooda.NewRegistry()
registry.MustRegister("write_document", myWriteFunc)
registry.MustRegister("search_playbooks", mySearchFunc)

frame := ooda.NewBuilder().
    WithInput(input).
    WithBrain(policyEngine).
    WithRegistry(registry).  // Creates Dispatcher internally
    Build()
```

---

## 7. Dual Memory Architecture

The OODA SDK supports two memory layers for multi-turn workflows:

### KnowledgeStore (Long-Term, MEB-Backed)

```go
// ports.KnowledgeStore interface
type KnowledgeStore interface {
    Recall(ctx context.Context, input string, topK int, graphID string) ([]core.Atom, error)
    CommitFact(ctx context.Context, atom core.Atom) error
    SearchByVector(ctx context.Context, query string, topK int) ([]SearchResult, error)
}
```

- **Purpose**: Persistent knowledge (playbooks, rules, templates)
- **Backend**: MEB (BadgerDB + vector search)
- **Lifecycle**: Survives restarts
- **Used in**: Orient phase — recall relevant facts based on input

### TransientStore (Short-Term, Session)

```go
// ports.TransientStore interface
type TransientStore interface {
    Put(ctx context.Context, sessionID string, key string, fact *TransientFact) error
    Get(ctx context.Context, sessionID string, key string) (*TransientFact, error)
    ToAtoms(ctx context.Context, sessionID string) ([]core.Atom, error)
    Delete(ctx context.Context, sessionID string, key string) error
}
```

- **Purpose**: Session coordination (current_node, agent_status, action results)
- **Backend**: RAM-only (no persistence)
- **Lifecycle**: Tied to session ID
- **Used in**: Orient (recall session state), Act (store results), Verify (store status)

### Wiring

```go
frame := ooda.NewBuilder().
    WithInput(input).
    WithBrain(engine).
    WithRegistry(registry).
    WithSessionID(sessionID).
    Build()

// Orient phase automatically queries both:
// 1. KnowledgeStore.Recall() for long-term facts
// 2. TransientStore.ToAtoms() for session state
// 3. Memory.Recall() for legacy fallback
```

---

## 8. Audit Trail & Trust Tiers

### AuditTrail

Every `Brain.Evaluate()` returns a `Decision` with an `AuditTrail` explaining which rules were matched:

```go
type AuditTrail struct {
    MatchedRules []RuleInference
    Timestamp    time.Time
    EngineID     string
    Query        string
    LatencyMs    int64
}

type RuleInference struct {
    RuleName   string
    Tier       Tier              // T0_Axiom, T1_Governance, T2_Playbook, T3_User
    Definition string
    SourceFile string
    Bindings   map[string]string
    Predicate  string
}
```

### Trust Tiers (4-Level System)

| Tier | Name | Description | Example |
|------|------|-------------|---------|
| **T0** | Kernel Axiom | Immutable core logic | `document_order(Category, Doc)` |
| **T1** | Governance | Human operator rules | `quality_gate(ESTIMATION, 0.90)` |
| **T2** | Playbook | Induced/learned rules | `effective_persona(DocType, ...)` |
| **T3** | User Input | Untrusted external input | User requirements text |

### VerifyStatus

```go
const (
    VerifyStatusPending VerifyStatus = "PENDING"
    VerifyStatusPassed  VerifyStatus = "FP32_PASSED"      // Hard logic passed
    VerifyStatusFailed  VerifyStatus = "LOGIC_VIOLATION"  // Critical failure
    VerifyStatusWarning VerifyStatus = "WARNING"           // Soft logic warning
)
```

### Accessing Audit Trail

```go
result, err := ooda.Run(ctx, frame)

// Get audit summary
summary := result.GetAuditSummary()

// Get phase durations
durations := result.GetPhaseDurations()

// Total execution time
total := result.TotalDuration()
```

---

## 9. Datalog Knowledge Base Integration

### KB Structure for OODA

```
kb/
├── ooda-phases.dl           # Phase order, actions, transitions, config
├── agents/
│   └── registry.dl          # Agent roles, capabilities, instances
├── workflows/
│   ├── registry.dl          # Workflow definitions
│   └── ooda_workflow.dl     # Generic OODA pipeline
├── tools/
│   └── registry.dl          # Tool definitions + OODA phase mapping
├── validation/
│   └── rules.dl             # Quality rules + severity + actions
└── patterns/
    └── registry.dl          # Reusable patterns
```

### Agent Roles (Datalog)

```datalog
% Agent roles aligned with OODA phases
agent_role("observer").     % Observe phase
agent_role("orienter").     % Orient phase
agent_role("planner").      % Decide phase
agent_role("executor").     % Act phase
agent_role("reviewer").     % Verify phase
agent_role("refiner").      % Refine (iteration) phase
agent_role("coordinator").  % Orchestration

% Map roles to phases
role_ooda_phase("observer", "observe").
role_ooda_phase("executor", "act").
role_ooda_phase("reviewer", "verify").

% Capabilities
role_capability("executor", "content_generation").
role_capability("executor", "tool_invocation").
role_capability("reviewer", "validation").
role_capability("reviewer", "quality_check").

% Agent instances
agent("executor-001", "executor").
agent_capability("executor-001", "content_generation").
agent_config("executor-001", "model", "gpt-4o").
agent_config("executor-001", "temperature", "0.7").
```

### Phase Configuration (Datalog)

```datalog
% Phase order
phase_order(1, "observe").
phase_order(2, "orient").
phase_order(3, "decide").
phase_order(4, "act").
phase_order(5, "verify").

% Phase transitions
phase_transition("observe", "orient").
phase_transition("orient", "decide").
phase_transition("decide", "act").
phase_transition("act", "verify").
conditional_transition("verify", "act", "needs_retry").

% Phase configs
phase_config("act", "timeout_ms", "60000").
phase_config("act", "allow_parallel", "true").
phase_config("verify", "fail_threshold", "0.8").
phase_config("refine", "max_iterations", "3").
```

### Validation Rules (Datalog)

```datalog
validation_rule("output_not_empty", "Output must not be empty").
validation_rule("no_secrets_exposed", "No secrets in output").
validation_rule("has_required_sections", "All required sections present").

validation_severity("output_not_empty", "critical").
validation_severity("no_secrets_exposed", "critical").
validation_severity("has_required_sections", "error").

validation_action("critical", "fail").
validation_action("error", "fail").
validation_action("warning", "warn").
```

### Loading KB at Startup

```go
// Using Manglekit PolicyEngine as Brain
engine := manglekit.NewPolicyEngine()
engine.LoadPolicy(ctx, "kb/ooda-phases.dl")
engine.LoadPolicy(ctx, "kb/agents/registry.dl")
engine.LoadPolicy(ctx, "kb/workflows/ooda_workflow.dl")
engine.LoadPolicy(ctx, "kb/tools/registry.dl")
engine.LoadPolicy(ctx, "kb/validation/rules.dl")
```

---

## 10. Reference Implementation: Architect Agent

The [architect-agent](https://github.com/duynguyendang/architect-agent) is the primary reference implementation using the OODA SDK.

### How It Uses OODA

| Phase | Architect-Agent Implementation |
|-------|-------------------------------|
| **Observe** | `datalog.ClassifyProject()` — keyword matching via `keyword_priority` |
| **Orient** | Loads `project_category`, `risk_level`, `complexity`, `project_characteristic` from Datalog; MEB vector search for playbooks |
| **Decide** | Queries `document_order(Category, Doc)` for document generation sequence |
| **Act** | `guidance.Assembler.Assemble()` → `guidance.BuildPrompt()` → LLM → PostProcessor |
| **Verify** | `agent.VerifyContent()` — 7 Datalog-backed checks (quality gates, sections, patterns) |
| **Refine** | `DetectOverfitting()` + `ValidateTaskEffort()` with Datalog thresholds; max 3 retries |

### Guidance Assembly Pattern

The architect-agent uses a shared `guidance.Assembler` backed by a `guidance.Querier` interface:

```go
// Each OODA caller wraps its native query mechanism
type Querier interface {
    Query(ctx context.Context, query string) ([]map[string]string, error)
}

// Adapters: mangleQuerier, siloQuerier, dagQuerier, agentSystemQuerier
asm := guidance.New(querier, projectType)
g := asm.Assemble(ctx, docType) // → *types.Guidance
system, user := guidance.BuildPrompt(g, requirements, projectName, riskPct)
```

### Datalog-Backed Verification

```go
// 7 checks against Datalog rules
result := agent.VerifyContent(ctx, querier, docType, content)
// - Quality gate thresholds (effective_quality_gate)
// - Section completeness (document_section)
// - Section consistency (structural)
// - Generic patterns (generic_pattern)
// - Prohibited terms (prohibited_term)
// - Validation rules (effective_validation)
// - Composite score: completeness*0.4 + consistency*0.3 + (1-generic)*0.3
```

### Key Datalog Rules (Architect-Agent)

| File | Predicates |
|------|-----------|
| `01-project.dl` | `project_keyword`, `keyword_priority`, `project_category`, `risk_level`, `project_characteristic` |
| `04-guidance.dl` | `effective_persona`, `effective_instruction`, `effective_scope` (derived predicates) |
| `06-validation.dl` | `quality_gate`, `effective_quality_gate`, `generic_pattern`, `prohibited_term` |
| `08-dag.dl` | `dag_node`, `dag_phase`, `dag_parallel_group`, `dag_agent_assignment` |
| `11-models.dl` | `model_for_phase`, `model_for_doc`, `model_for_category`, `model_default` |

---

## 11. Best Practices

### 1. Use Builder, Not Manual Construction

```go
// DO: Use Builder
frame := ooda.NewBuilder().
    WithInput(input).
    WithBrain(brain).
    WithRegistry(registry).
    WithMaxRetries(3).
    Build()

// DON'T: Manual construction (easy to miss fields)
frame := &ooda.CognitiveFrame{...}
```

### 2. Register All Actions Before Run

```go
// DO: Register before Run
registry.MustRegister("write_document", writeFunc)
registry.MustRegister("validate_output", validateFunc)
frame := ooda.NewBuilder().WithRegistry(registry).Build()
ooda.Run(ctx, frame)

// DON'T: Register after Run (race condition)
```

### 3. Use TransientStore for Session State

```go
// DO: Use TransientStore for workflow coordination
frame.TransientStore.Put(ctx, sessionID, "current_phase", &ports.TransientFact{...})

// DON'T: Use global variables for session state
```

### 4. Handle SafeStop Gracefully

```go
// DO: Configure SafeStop for your domain
ooda.SafeStop = func(ctx context.Context, args map[string]interface{}) (string, error) {
    log.Printf("SafeStop: action=%s reason=%s", args["action"], args["reason"])
    metrics.Increment("safe_stop_total")
    return "STOPPED: logged and alerted", nil
}
```

### 5. Set Appropriate Timeouts

```go
// Quick tasks (< 30s)
frame := ooda.NewBuilder().WithTimeout(30 * time.Second).Build()

// Long generation (5min)
frame := ooda.NewBuilder().WithTimeout(5 * time.Minute).Build()

// Production (balanced)
frame := ooda.NewBuilder().WithTimeout(2 * time.Minute).WithMaxRetries(3).Build()
```

### 6. Use AuditTrail for Debugging

```go
result, err := ooda.Run(ctx, frame)
if err != nil {
    // AuditTrail shows which rules matched before failure
    fmt.Println(result.GetAuditSummary())
    
    // Phase durations show where time was spent
    for phase, dur := range result.GetPhaseDurations() {
        fmt.Printf("  %s: %v\n", phase, dur)
    }
}
```

---

## Quick Reference

### Core Types

| Type | Package | Purpose |
|------|---------|---------|
| `CognitiveFrame` | `sdk/ooda` | Complete state of a reasoning epoch |
| `Builder` | `sdk/ooda` | Fluent API for frame construction |
| `Atom` | `sdk/ooda` | Smallest unit of knowledge |
| `Registry` | `sdk/ooda` | Action name → Go function mapping |
| `Dispatcher` | `sdk/ooda` | Executes actions with safety fallback |
| `Decision` | `core` | Outcome from Brain evaluation |
| `AuditTrail` | `core` | Matched rules + tier + bindings |
| `ActionEnvelope` | `core` | Action name + arguments |

### Key Functions

| Function | Purpose |
|----------|---------|
| `ooda.Run(ctx, frame)` | Execute full OODA loop |
| `ooda.NewBuilder()` | Create frame builder |
| `ooda.NewRegistry()` | Create action registry |
| `ooda.NewDispatcher(registry)` | Create action dispatcher |
| `frame.GetAuditSummary()` | Human-readable audit trail |
| `frame.TotalDuration()` | Total execution time |

### File Checklist

- [ ] `kb/ooda-phases.dl` — Phase definitions
- [ ] `kb/agents/registry.dl` — Agent roles and capabilities
- [ ] `kb/workflows/ooda_workflow.dl` — OODA pipeline
- [ ] `kb/tools/registry.dl` — Tool definitions
- [ ] `kb/validation/rules.dl` — Quality rules

---

**For more details, see:**
- [OODA Quick Start](./OODA-QUICKSTART.md)
- [Domain Extension Guide](./DOMAIN-EXTENSION-GUIDE.md)
- [Architect Agent (reference implementation)](https://github.com/duynguyendang/architect-agent)
