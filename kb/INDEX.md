# Manglekit Knowledge Base

**Generic, domain-agnostic knowledge base for agentic systems.**

---

## OODA Framework

The KB is structured around the **OODA Loop** (Observe → Orient → Decide → Act → Verify → Refine):

| Phase | Purpose | Registry | Key Files |
|-------|---------|----------|-----------|
| **Observe** | Context gathering | `agents/`, `tools/` | observer roles, semantic_search |
| **Orient** | Semantic synthesis | `patterns/` | classification, retrieval |
| **Decide** | Strategic planning | `workflows/` | ooda_workflow.dl, planner roles |
| **Act** | Execution | `templates/` | generators, executors |
| **Verify** | Quality assurance | `validation/` | rules.dl, reviewer roles |
| **Refine** | Iteration | `workflows/` | refinement actions |

---

## Directory Structure

```
kb/
├── agents/           # Agent roles and capabilities
│   ├── registry.dl  # Generic OODA-aligned agents
│   └── architect.dl # Domain-specific (architect agent)
├── patterns/        # Reusable patterns
│   └── registry.dl # Pattern definitions
├── templates/       # Document templates
│   └── registry.dl # Template definitions
├── tools/           # Tool registry
│   └── registry.dl # Tool definitions
├── validation/      # Quality rules
│   └── rules.dl    # Validation rules
├── workflows/       # Workflow definitions
│   ├── ooda_workflow.dl   # Generic OODA pipeline
│   ├── csd_workflow.dl    # Domain-specific (CSD)
│   └── registry.dl       # Workflow registry
├── ooda-phases.dl   # Core OODA phase definitions
└── INDEX.md         # This file
```

---

## Core Files

### `ooda-phases.dl`
Core OODA phase definitions including:
- Phase order and descriptions
- Phase capabilities and actions
- Phase tools and metrics
- Phase configuration
- Domain-specific extensions

### `agents/registry.dl`
Generic agent registry aligned with OODA:
- **Observer** → Observe phase
- **Orienter** → Orient phase
- **Planner** → Decide phase
- **Executor** → Act phase
- **Reviewer** → Verify phase
- **Refiner** → Refine phase
- **Coordinator** → Orchestration

### `workflows/ooda_workflow.dl`
Generic OODA loop pipeline:
- 6 phases with multiple nodes each
- Conditional transitions
- Error handling
- Parallel execution support
- Session state tracking

### `validation/rules.dl`
Generic quality validation rules:
- Output validation
- Completeness checks
- Consistency validation
- Quality metrics
- Security checks
- Compliance validation

---

## Extension Points

### Domain-Specific KBs
Extend the generic KB for specific domains:

```
kb/
└── domains/
    └── architect/
        ├── agents/architect.dl
        ├── workflows/csd_workflow.dl
        └── patterns/architecture-patterns.dl
```

### Custom Phases
Add custom OODA phases:

```datalog
phase_order(7, "custom_phase").
phase("custom_phase", "Custom Purpose", "Custom description").
```

### Custom Tools
Register new tools:

```datalog
tool("custom_tool", "custom_category").
tool_capability("custom_tool", "custom_capability").
```

---

## Usage

### Query Available Agents
```datalog
select_agent_for_phase("decide", Agent).
```

### Get Tools for Phase
```datalog
tools_for_phase("act", Tools).
```

### Validate Output
```datalog
validation_rule(Rule, Description),
validation_severity(Rule, "error").
```

### Execute OODA Workflow
```datalog
workflow("ooda_pipeline", Name, Version),
workflow_node("ooda_pipeline", Node, Type, Role).
```

---

**Status:** Generic Foundation Ready ✅
