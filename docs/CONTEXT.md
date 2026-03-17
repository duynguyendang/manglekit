---
context_type: kernel_source_dump
project: manglekit_v2
language: go, datalog
last_updated: 2026-03-17
scan_mode: logic_focused
---

# Manglekit v2 (Sovereign Logic Kernel) - Live Architecture Snapshot

This document serves as the canonical, authoritative "Live Architecture" snapshot of the Manglekit v2 codebase. It contains the exact interfaces, structs, and logic rules that drive the Datalog-first Multi-Agent Cognitive Kernel.

## 1. THE COMPLETE FILE MAP

```
manglekit/
├── adapters/                    # Implementation of Ports
├── core/                       # The Domain Model (Center of Hexagon)
│   ├── types.go              # Core types (Envelope, Decision, AuditTrail, ActionEnvelope)
│   ├── governance.go         # Governance interfaces
│   └── workflow.go            # NEW: Pure Go Workflow Definition
├── docs/
│   ├── CONTEXT.md            # This file
│   └── ROADMAP.md            # Multi-Agent Roadmap
├── internal/
│   └── engine/               # Policy Engine & Runtime
│       ├── solver.go         # PolicyEngine with QueryWithAudit
│       └── audit_trail_test.go
├── kb/                        # Knowledge Base (Datalog)
│   └── agents/registry.dl
├── multiagent/                # Datalog-Driven Multi-Agent System
│   ├── agent_system.go       # Agent System
│   ├── loader.go             # NEW: DatalogWorkflowLoader (Hydration)
│   ├── hydrated_executor.go  # NEW: Datalog-agnostic executor
│   └── ...
├── sdk/
│   ├── ooda/                 # NEW: OODA Chassis SDK
│   │   ├── domain.go         # CognitiveFrame + Builder
│   │   ├── frame.go          # OODA Loop execution
│   │   ├── registry.go       # Action Registry + Dispatcher
│   │   └── interfaces.go      # Observer, Orienter, Decider, etc.
│   └── ports/
│       └── interfaces.go      # WorkflowLoader, ConditionEvaluator, AgentFinder
└── go.mod
```

---

## 2. HIGH-LEVEL ARCHITECTURE

### 2.1 Datalog-First Multi-Agent Design

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     DATALOG (Source of Truth)                            │
├─────────────────────────────────────────────────────────────────────────┤
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │
│   │   AGENTS    │  │  WORKFLOWS  │  │   TOOLS     │  │  POLICIES   │  │
│   │ agent/2     │  │ workflow/2  │  │ tool/3      │  │ allow/1    │  │
│   │ role/1      │  │ node/4      │  │ input/2     │  │ deny/1     │  │
│   │ capability/2│  │ edge/3      │  │ output/2    │  │ halt/1     │  │
│   └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘  │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  │
│   │  RETRY      │  │  APPROVAL   │  │  AUDIT     │  │ SCHEDULER  │  │
│   │ retry_for/3 │  │ requires/2  │  │ audit/3     │  │ cron/3     │  │
│   └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘  │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    GO EXECUTION ENGINE (Lightweight Chassis)            │
│                                                                          │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  PolicyEngine                                                    │   │
│   │  • Query() / QueryWithAudit() → Results + AuditTrail            │   │
│   │  • AddPolicy() / RegisterExternalPredicate()                    │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  AgentSystem                                                     │   │
│   │  • GetWorkflow() → Hydrates to core.WorkflowDef                │   │
│   │  • FindAgentsForTask() / GetRoleCapabilities()                   │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  HydratedWorkflowExecutor (NEW - Datalog-agnostic)              │   │
│   │  • Receives core.WorkflowDef (Pure Go struct)                  │   │
│   │  • Does NOT query Datalog during execution                     │   │
│   │  • Uses ports.ConditionEvaluator for edge conditions             │   │
│   └─────────────────────────────────────────────────────────────────┘   │
│                                                                          │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │  OODA Chassis SDK (NEW)                                         │   │
│   │  • CognitiveFrame with Memory, Brain, Executor                │   │
│   │  • Builder pattern for assembly                                 │   │
│   │  • Action Registry for decoupling decisions from Go tools       │   │
│   └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. CORE DATA STRUCTURES

### 3.1 Pure Go Workflow Definition (core/workflow.go)

```go
package core

// WorkflowDef is a Datalog-agnostic workflow representation.
// Hydrated from Datalog and used for stateless execution.
type WorkflowDef struct {
    ID         string
    Name       string
    Version    string
    RootNodeID string
    Nodes      map[string]NodeDef
    Edges      []EdgeDef
}

type NodeDef struct {
    ID        string
    AgentRole string
    TaskType  string
    Config    map[string]interface{}
}

type EdgeDef struct {
    From      string
    To        string
    Condition string // Evaluated by ConditionEvaluator port
}
```

### 3.2 Decision with ActionEnvelope (core/types.go)

```go
package core

// Decision includes Action for dispatching to Go tools
type Decision struct {
    Outcome    string
    Target    string
    Reasons   []string
    Meta      map[string]string
    AuditTrail *AuditTrail   // NEW: Detailed trace
    Action    *ActionEnvelope // NEW: Action to execute
}

// ActionEnvelope decouples Manglekit decisions from Go implementations
type ActionEnvelope struct {
    Name      string                 // e.g., "write_document"
    Arguments map[string]interface{} // e.g., {"type": "ADD"}
}

// AuditTrail provides transparency into which rules were matched
type AuditTrail struct {
    MatchedRules []RuleInference
    Timestamp    time.Time
    EngineID     string
    Query        string
    LatencyMs    int64
}

type RuleInference struct {
    RuleName   string
    Tier       Tier               // T0_Axiom, T1_Governance, T2_Playbook, T3_User
    Definition string
    SourceFile string
    Bindings   map[string]string
    Predicate  string
}

type Tier string

const (
    TierT0_Axiom     Tier = "T0"
    TierT1_Governance Tier = "T1"
    TierT2_Playbook  Tier = "T2"
    TierT3_User      Tier = "T3"
)
```

---

## 4. PORTS (sdk/ports/interfaces.go)

```go
package ports

// WorkflowLoader hydrates Datalog to Pure Go
type WorkflowLoader interface {
    LoadWorkflow(ctx context.Context, workflowID string) (*core.WorkflowDef, error)
}

// ConditionEvaluator evaluates edge conditions
type ConditionEvaluator interface {
    EvaluateCondition(ctx context.Context, condition string, facts map[string]interface{}) (bool, error)
}

// AgentFinder finds available agents
type AgentFinder interface {
    FindAgentsByRole(ctx context.Context, role string) ([]string, error)
}
```

---

## 5. OODA CHASSIS SDK (sdk/ooda)

### 5.1 Memory, Brain, Executor Interfaces

```go
package ooda

// Memory - Recall/Commit for context hydration
type Memory interface {
    Recall(ctx context.Context, input string) ([]Atom, error)
    Commit(ctx context.Context, frame *CognitiveFrame) error
}

// Brain - Policy evaluation with AuditTrail
type Brain interface {
    Evaluate(ctx context.Context, frame *CognitiveFrame) (*core.Decision, error)
    Verify(ctx context.Context, frame *CognitiveFrame) (*core.AuditTrail, error)
}

// Executor - Action execution
type Executor interface {
    Execute(ctx context.Context, frame *CognitiveFrame, decision *core.Decision) (any, error)
    Rollback(ctx context.Context, frame *CognitiveFrame, result any) error
}
```

### 5.2 Action Registry & Dispatcher

```go
package ooda

// Registry maps action names to Go functions
type Registry struct {
    tools map[string]ToolFunc
}

type ToolFunc func(ctx context.Context, args map[string]interface{}) (string, error)

// Dispatcher executes actions from registry
type Dispatcher struct {
    registry *Registry
    fallback ToolFunc
}

// Example usage:
func Example() {
    registry := ooda.NewRegistry()
    registry.Register("generate_csd", func(ctx context.Context, args map[string]interface{}) (string, error) {
        return "CSD generated for " + args["project"].(string), nil
    })

    dispatcher := ooda.NewDispatcher(registry)
    result, _ := dispatcher.Dispatch(ctx, "generate_csd", map[string]interface{}{
        "project": "my-app",
    })
}
```

### 5.3 Fluent Builder

```go
package ooda

// Build a CognitiveFrame with all components
frame := ooda.NewBuilder().
    WithInput("Generate ADD document").
    WithMemory(mebAdapter).
    WithBrain(policyEngine).
    WithRegistry(registry).
    WithMaxRetries(3).
    WithTimeout(5 * time.Minute).
    Build()

result, err := ooda.Run(ctx, frame)
```

### 5.4 OODA Loop Flow

```go
package ooda

func Run(ctx context.Context, frame *CognitiveFrame) (*CognitiveFrame, error) {
    // 1. Observe - Capture raw input
    // 2. Orient - Hydrate from Memory
    // 3. Decide - Brain.Evaluate() → Decision + AuditTrail
    // 4. Act - Dispatcher dispatches Decision.Action
    // 5. Verify - Brain.Verify()
    // 6. Commit - Memory.Commit()
}
```

---

## 6. HYDRATION MODEL (Workflow Decoupling)

### 6.1 Load Phase (Datalog → Go)

```go
// At startup or on-demand:
loader := multiagent.NewDatalogWorkflowLoader(agentSystem)
workflowDef, _ := loader.LoadWorkflow(ctx, "architect-csd-v1")
// Returns *core.WorkflowDef (Pure Go, no Datalog references)
```

### 6.2 Execute Phase (Pure Go, no Datalog)

```go
// Execute WITHOUT querying Datalog:
executor := multiagent.NewHydratedWorkflowExecutor(workflowDef).
    WithConditionEvaluator(conditionEvalAdapter).
    WithAgentFinder(agentFinderAdapter)

result, _ := executor.Execute(ctx, input)
// Zero Datalog queries during execution!
```

---

## 7. DEPENDENCY GRAPH

### Clean Dependencies (No Datalog Leaks)

- `core/` → Standard Go only (no mangle-go)
- `sdk/ooda/` → Standard Go + core
- `multiagent/hydrated_executor.go` → Uses ports interfaces only

### Adapter Layer (Datalog Here)

- `multiagent/loader.go` → Uses mangle-go internally (isolated)
- `internal/engine/` → Uses mangle-go (expected)

---

## 8. MEMORY PROFILES (MEB)

```go
// Cloud Run / Serving (Minimal RAM)
config := &store.Config{
    Profile:    "ReadOnly",
    ReadOnly:  true,
    DataDir:   os.Getenv("DATA_DIR"),
}

// Safe-Serving (1-2GB)
config := &store.Config{
    Profile:    "Safe-Serving",
    DataDir:   os.Getenv("DATA_DIR"),
}
```

---

## 9. KEY DESIGN DECISIONS

1. **Datalog-First**: Everything defined in Datalog
2. **Hydration Model**: Workflow loaded once, executed in pure Go
3. **Zero Live Queries**: Executor doesn't query Datalog during execution
4. **Action Mapping**: Registry decouples decisions from Go implementations
5. **Audit Trail**: Full traceability with RuleName, Tier, Bindings
6. **Fluent Builder**: Easy component assembly
7. **Port/Adapter**: Clean separation between Core and Datalog

---

## 10. TESTING

```bash
# All tests
go test ./... 

# OODA SDK tests (MockBrain, no Datalog)
go test ./sdk/ooda/... -v

# Multiagent tests
go test ./multiagent/... -v

# Engine with AuditTrail
go test ./internal/engine/... -v -run TestAudit
```

---

## 11. DEPENDENCIES

- **codeberg.org/TauCeti/mangle-go**: Datalog engine (adapter layer only)
- **github.com/dgraph-io/badger/v4**: Key-value storage
- **github.com/google/uuid**: UUID generation
