# OODA Quick Start Guide

**Get up and running with OODA-based multi-agent systems in 5 minutes.**

---

## TL;DR

1. Load the KB with OODA definitions
2. Create agent instances
3. Execute the OODA pipeline
4. Validate output
5. Iterate if needed

---

## 1. Minimal Setup

### Load OODA Knowledge Base

```go
kb := knowledgebase.New()

// Load core OODA definitions
kb.Load("kb/ooda-phases.dl")
kb.Load("kb/agents/registry.dl")
kb.Load("kb/workflows/ooda_workflow.dl")
kb.Load("kb/tools/registry.dl")
kb.Load("kb/validation/rules.dl")
```

### Create Agent

```go
// Use predefined agents from registry
agent := agent.New("executor-001", kb)

// Or create custom
agent := &agent.Agent{
    ID:          "my-agent",
    Role:        "executor",
    Model:       "gpt-4o",
    Temperature: 0.7,
    Tools:       []string{"llm_generate", "semantic_search"},
}
```

---

## 2. Execute OODA Loop

```go
coordinator := NewCoordinator(kb)
result, err := coordinator.ExecuteOodaLoop(userInput)
```

---

## 3. Common Patterns

### Pattern 1: Single Agent Task

```go
// Observe → Act → Verify
agent := agent.New("executor-001", kb)
output := agent.Execute("generate_content", input)
valid, _ := validator.Validate(output, kb)
```

### Pattern 2: Multi-Agent Pipeline

```go
// Each phase uses different agent
observers := agent.SelectByPhase("observe")
orienters := agent.SelectByPhase("orient")
executors := agent.SelectByPhase("act")
reviewers := agent.SelectByPhase("verify")

context := observers[0].Execute(input)
context = orienters[0].Execute(context)
context = executors[0].Execute(context)
reviewers[0].Verify(context)
```

### Pattern 3: With Error Recovery

```go
coordinator := NewCoordinator(kb)
coordinator.MaxIterations = 3
coordinator.FailFast = false

result, err := coordinator.ExecuteOodaLoop(input)
if err != nil {
    // Handle final failure
    log.Printf("Failed after %d iterations: %v", coordinator.Iterations, err)
}
```

---

## 4. Query Examples

### Get Agents by Phase

```bash
# In Datalog shell
> role_ooda_phase(Role, "act"), agent(Agent, Role).
> Agent = "executor-001"
> Agent = "executor-002"
```

### Get Tools by Phase

```bash
> tool_ooda_phase(Tool, "observe"), tool_available(Tool, "available").
> Tool = "semantic_search"
> Tool = "entity_extractor"
```

### Get Phase Config

```bash
> phase_config("act", "timeout_ms", Value).
> Value = "60000"
```

---

## 5. Configuration Templates

### Fast Response (Low Latency)

```datalog
phase_config("observe", "timeout_ms", "2000").
phase_config("orient", "timeout_ms", "3000").
phase_config("decide", "timeout_ms", "5000").
phase_config("act", "timeout_ms", "30000").
phase_config("verify", "timeout_ms", "5000").
phase_config("refine", "max_iterations", "1").
```

### High Quality (Multiple Iterations)

```datalog
phase_config("observe", "timeout_ms", "10000").
phase_config("orient", "timeout_ms", "15000").
phase_config("decide", "timeout_ms", "30000").
phase_config("act", "timeout_ms", "120000").
phase_config("verify", "timeout_ms", "60000").
phase_config("refine", "max_iterations", "5").
phase_config("refine", "improvement_threshold", "0.05").
```

### Production (Balanced)

```datalog
phase_config("observe", "timeout_ms", "5000").
phase_config("orient", "timeout_ms", "10000").
phase_config("decide", "timeout_ms", "15000").
phase_config("act", "timeout_ms", "60000").
phase_config("verify", "timeout_ms", "30000").
phase_config("refine", "max_iterations", "3").
```

---

## 6. Debugging Tips

### Enable Verbose Logging

```go
coordinator := NewCoordinator(kb)
coordinator.LogLevel = "debug"

log, _ := os.Create("ooda-debug.log")
coordinator.Logger = log
```

### Check Agent Availability

```bash
> agent(Agent, Role), agent_status(Agent, "available").
```

### Verify Workflow Loaded

```bash
> workflow("ooda_pipeline", Name, Version).
> workflow_node("ooda_pipeline", Node, Type, Role).
> workflow_edge("ooda_pipeline", From, To).
```

### Test Validation Rules

```go
validator := NewValidator(kb)
result, _ := validator.Validate(testOutput)

fmt.Printf("Passed: %v\n", result.Passed)
for _, issue := range result.Issues {
    fmt.Printf("  [%s] %s: %s\n", issue.Severity, issue.Rule, issue.Description)
}
```

---

## 7. Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `no available agent` | No agent with matching role/phase | Add agent instance to registry |
| `phase timeout` | Phase taking too long | Increase `timeout_ms` config |
| `max iterations` | Too many refinement cycles | Increase `max_iterations` or fix root cause |
| `validation failed` | Output doesn't meet rules | Add validation rules or fix generator |
| `workflow not found` | KB not loaded | Load workflow file before executing |

---

## 8. Next Steps

- Read the full [OODA Multi-Agent Guide](./OODA-MULTI-AGENT-GUIDE.md)
- Explore [domain-specific extensions](./DOMAIN-EXTENSION-GUIDE.md)
- Check [API reference](../api/README.md)
- Review [example implementations](../examples/)

---

**Questions?** Check the [troubleshooting FAQ](./FAQ.md) or open an issue.
