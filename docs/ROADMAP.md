# Manglekit ROADMAP - Datalog-First Cognitive Kernel

## Vision

Transform Manglekit into a **Datalog-driven cognitive kernel** where **everything** - agents, workflows, tools, policies, retries, security - is defined declaratively in Datalog. Go code becomes merely the execution engine that interprets Datalog rules.

**Core Principle**: If it can be defined in Datalog, it should be. Go code only handles what's impossible in Datalog (I/O, networking, etc.).

---

## Completed Features ✅

### 1. Agent System (Datalog-Defined)
- [x] Agent roles defined in Datalog
- [x] Role capabilities and inheritance
- [x] Agent registry with instance capabilities
- [x] Task-to-agent capability matching

### 2. Workflow Engine (Datalog + Hydration)
- [x] Workflow DAG defined in Datalog
- [x] WorkflowLoader - Hydrates Datalog to Pure Go `core.WorkflowDef`
- [x] Sequential node execution
- [x] Conditional edge evaluation
- [x] Parallel execution groups
- [x] Error edge handling
- [x] Retry logic

### 3. Audit Trail (Transparency)
- [x] `QueryWithAudit()` returns AuditTrail
- [x] RuleName, Tier (T0-T3), Bindings captured
- [x] Source file mapping
- [x] Latency tracking

### 4. OODA Chassis SDK
- [x] CognitiveFrame with Memory, Brain, Executor interfaces
- [x] Fluent Builder pattern
- [x] Observe → Orient → Decide → Act → Verify flow
- [x] State persistence via Memory.Commit()

### 5. Action Mapping (Decoupling)
- [x] Registry maps action names to Go functions
- [x] Dispatcher executes actions
- [x] Sovereign Violation handling for unknown actions

### 6. Dependency Sanitization
- [x] `core/` has zero imports from mangle-go
- [x] HydratedWorkflowExecutor doesn't hold AgentSystem reference
- [x] Clean dependency graph verified

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     DATALOG (Source of Truth)                            │
├─────────────────────────────────────────────────────────────────────────┤
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│   │   AGENTS    │  │  WORKFLOWS  │  │   TOOLS     │  │  POLICIES   │ │
│   │ agent/2     │  │ workflow/2  │  │ tool/3      │  │ allow/1    │ │
│   │ role/1      │  │ node/4      │  │ input/2     │  │ deny/1     │ │
│   │ capability/2│  │ edge/3      │  │ output/2    │  │ halt/1     │ │
│   └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘ │
│   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│   │  RETRY      │  │  APPROVAL   │  │  AUDIT      │  │ SCHEDULER  │ │
│   │ retry_for/3 │  │ requires/2  │  │ audit/3     │  │ cron/3     │ │
│   └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘ │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              LIGHTWEIGHT CHASSIS (Pure Go)                             │
│                                                                          │
│   ┌─────────────────────────────────────────────────────────────────┐    │
│   │  PolicyEngine                                                    │    │
│   │  • Query() → Results                                          │    │
│   │  • QueryWithAudit() → Results + AuditTrail                    │    │
│   │  • AddPolicy() / RegisterExternalPredicate()                    │    │
│   └─────────────────────────────────────────────────────────────────┘    │
│                                                                          │
│   ┌─────────────────────────────────────────────────────────────────┐    │
│   │  HydratedWorkflowExecutor (Zero Datalog queries during run!)   │    │
│   │  • Receives core.WorkflowDef (Pure Go struct)                 │    │
│   │  • Uses ports.ConditionEvaluator for edge conditions           │    │
│   │  • Pluggable NodeExecutor                                     │    │
│   └─────────────────────────────────────────────────────────────────┘    │
│                                                                          │
│   ┌─────────────────────────────────────────────────────────────────┐    │
│   │  OODA Chassis SDK                                              │    │
│   │  • Memory interface (Recall/Commit)                          │    │
│   │  • Brain interface (Evaluate/Verify)                         │    │
│   │  • Executor interface (Execute/Rollback)                     │    │
│   │  • Registry + Dispatcher (Action Mapping)                   │    │
│   │  • Builder pattern                                           │    │
│   └─────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Implementation Guide

### 1. Hydration Model (Load Phase)

```go
// Load workflow ONCE at startup (or on-demand)
loader := multiagent.NewDatalogWorkflowLoader(agentSystem)
workflowDef, _ := loader.LoadWorkflow(ctx, "architect-csd-v1")
// Returns *core.WorkflowDef (Pure Go, no Datalog references)
```

### 2. Execution Phase (Zero Datalog)

```go
// Execute WITHOUT any Datalog queries!
executor := multiagent.NewHydratedWorkflowExecutor(workflowDef).
    WithConditionEvaluator(conditionEvalAdapter).
    WithAgentFinder(agentFinderAdapter).
    WithNodeExecutor(customExecutor)

result, _ := executor.Execute(ctx, input)
```

### 3. OODA Chassis

```go
// Assemble with Builder
frame := ooda.NewBuilder().
    WithInput("Generate ADD document").
    WithMemory(mebAdapter).
    WithBrain(policyEngine).
    WithRegistry(registry).
    WithMaxRetries(3).
    Build()

// Run OODA loop
result, _ := ooda.Run(ctx, frame)

// Manglekit returns: Decision{Action: "write_document", Arguments: {...}}
// Chassis dispatches to: Registry["write_document"](args)
```

---

## Memory Profiles (MEB)

```go
// Cloud Run / Serving (Minimal RAM ~1GB)
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

// Ingest-Heavy (8GB+)
config := &store.Config{
    Profile:    "Ingest-Heavy",
    DataDir:   os.Getenv("DATA_DIR"),
}
```

---

## Future Enhancements (Roadmap)

### Phase 1: Tool Registry (Datalog)
- [ ] Tool definitions in Datalog
- [ ] Tool schemas (input/output)
- [ ] Capability-based tool matching

### Phase 2: Advanced Policies
- [ ] Retry policies from Datalog
- [ ] Approval workflows from Datalog
- [ ] RBAC from Datalog

### Phase 3: Scheduler
- [ ] Cron-like schedules from Datalog
- [ ] Event-based triggers
- [ ] Delayed actions

### Phase 4: Observability
- [ ] Monitoring rules from Datalog
- [ ] Alert thresholds
- [ ] Log pattern detection

### Phase 5: Multi-Agent Coordination
- [ ] Message bus integration
- [ ] Agent communication protocols
- [ ] Distributed execution

---

## Files Reference

| Component | File | Status |
|-----------|------|--------|
| Workflow Definition | `core/workflow.go` | ✅ |
| Decision + AuditTrail | `core/types.go` | ✅ |
| PolicyEngine + QueryWithAudit | `internal/engine/solver.go` | ✅ |
| AgentSystem | `multiagent/agent_system.go` | ✅ |
| WorkflowLoader | `multiagent/loader.go` | ✅ |
| HydratedExecutor | `multiagent/hydrated_executor.go` | ✅ |
| OODA Frame | `sdk/ooda/frame.go` | ✅ |
| Registry + Dispatcher | `sdk/ooda/registry.go` | ✅ |
| Ports | `sdk/ports/interfaces.go` | ✅ |
| MEB Config | `meb/store/badger.go` | ✅ |

---

## Tests

```bash
# All tests pass
go test ./...

# OODA SDK (MockBrain, no Datalog)
go test ./sdk/ooda/... -v

# Multiagent
go test ./multiagent/... -v

# Engine with Audit
go test ./internal/engine/... -v -run TestAudit
```

---

## Key Achievements

1. **Hydration Model**: Workflow loaded once, executed in pure Go (zero Datalog queries during run)
2. **Action Decoupling**: Manglekit returns action names, Chassis maps to Go functions
3. **Full Auditability**: Every decision includes which rules matched and from which tier
4. **Cloud Ready**: ReadOnly mode for minimal RAM on Cloud Run
5. **Clean Dependencies**: `core/` has zero Datalog imports
