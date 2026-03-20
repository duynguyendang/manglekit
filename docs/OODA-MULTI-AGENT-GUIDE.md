# How to Build Multi-Agent Systems with Manglekit Using OODA

**A practical guide to implementing agentic systems with the OODA (Observe → Orient → Decide → Act → Verify → Refine) framework.**

---

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture](#2-architecture)
3. [Step-by-Step Implementation](#3-step-by-step-implementation)
4. [Agent Configuration](#4-agent-configuration)
5. [Workflow Definition](#5-workflow-definition)
6. [Knowledge Base Integration](#6-knowledge-base-integration)
7. [Validation & Quality Gates](#7-validation--quality-gates)
8. [Domain-Specific Extensions](#8-domain-specific-extensions)
9. [Best Practices](#9-best-practices)
10. [Examples](#10-examples)

---

## 1. Overview

### What is OODA?

OODA (Observe → Orient → Decide → Act → Verify → Refine) is a decision-making framework originally developed for military strategy. In the context of multi-agent systems, it provides a structured approach to task execution:

| Phase | Purpose | Agent Role | Description |
|-------|---------|------------|-------------|
| **Observe** | Gather context | `observer` | Collect and parse input from users, systems, or external sources |
| **Orient** | Understand context | `orienter` | Classify intent, retrieve relevant knowledge, synthesize understanding |
| **Decide** | Plan action | `planner` | Create execution plan, assess risks, select approach |
| **Act** | Execute | `executor` | Perform the planned actions, generate output |
| **Verify** | Validate | `reviewer` | Check output quality, rule compliance, completeness |
| **Refine** | Improve | `refiner` | Apply corrections, optimize, iterate if needed |

### Why OODA for Multi-Agent?

- **Structured**: Provides clear separation of concerns
- **Extensible**: Each phase can be customized or extended
- **Observable**: Easy to track and debug
- **Resilient**: Loop-back mechanism for error recovery

---

## 2. Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                      OODA Loop Controller                        │
├─────────────────────────────────────────────────────────────────┤
│  Observe ──→ Orient ──→ Decide ──→ Act ──→ Verify ──→ Refine    │
│     ↑                                                           │
│     └──────────────────── Loop Back ────────────────────────────┘
└─────────────────────────────────────────────────────────────────┘
         │           │           │           │           │
         ▼           ▼           ▼           ▼           ▼
    ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐
    │Observer │ │Orienter │ │ Planner │ │Executor │ │Reviewer │
    │ Agent   │ │ Agent   │ │ Agent   │ │ Agent   │ │ Agent   │
    └─────────┘ └─────────┘ └─────────┘ └─────────┘ └─────────┘
         │           │           │           │           │
         └───────────┴───────────┴───────────┴───────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │  Knowledge Base │
                    │   (Datalog)     │
                    └─────────────────┘
```

### Knowledge Base Structure

```
kb/
├── ooda-phases.dl     # Phase definitions, capabilities, transitions
├── agents/
│   └── registry.dl    # Agent roles, capabilities, instances
├── workflows/
│   ├── ooda_workflow.dl   # Generic OODA pipeline
│   └── registry.dl        # Additional workflows
├── tools/
│   └── registry.dl        # Tool definitions
├── validation/
│   └── rules.dl           # Quality rules
├── patterns/
│   └── registry.dl        # Reusable patterns
└── templates/
    └── registry.dl        # Output templates
```

---

## 3. Step-by-Step Implementation

### Step 1: Define Your Agents

Create or modify the agent registry at `kb/agents/registry.dl`:

```datalog
% ==========================================
% 1. DEFINE AGENT ROLES (aligned with OODA)
% ==========================================

agent_role("observer").     % Observe phase
agent_role("orienter").     % Orient phase
agent_role("planner").      % Decide phase
agent_role("executor").     % Act phase
agent_role("reviewer").     % Verify phase
agent_role("refiner").      % Refine phase
agent_role("coordinator").  % Orchestration (spans all phases)

% ==========================================
% 2. MAP ROLES TO OODA PHASES
% ==========================================

role_ooda_phase("observer", "observe").
role_ooda_phase("orienter", "orient").
role_ooda_phase("planner", "decide").
role_ooda_phase("executor", "act").
role_ooda_phase("reviewer", "verify").
role_ooda_phase("refiner", "refine").
role_ooda_phase("coordinator", "observe").

% ==========================================
% 3. DEFINE CAPABILITIES PER ROLE
% ==========================================

% Observer capabilities
role_capability("observer", "information_gathering").
role_capability("observer", "entity_extraction").
role_capability("observer", "context_capture").
role_capability("observer", "semantic_search").
role_capability("observer", "input_validation").

% Orienter capabilities
role_capability("orienter", "semantic_analysis").
role_capability("orienter", "pattern_recognition").
role_capability("orienter", "classification").
role_capability("orienter", "knowledge_retrieval").
role_capability("orienter", "context_synthesis").

% Planner capabilities
role_capability("planner", "task_decomposition").
role_capability("planner", "plan_generation").
role_capability("planner", "risk_assessment").
role_capability("planner", "resource_planning").

% Executor capabilities
role_capability("executor", "action_execution").
role_capability("executor", "content_generation").
role_capability("executor", "tool_invocation").
role_capability("executor", "code_generation").

% Reviewer capabilities
role_capability("reviewer", "validation").
role_capability("reviewer", "quality_check").
role_capability("reviewer", "rule_validation").
role_capability("reviewer", "approval").

% Refiner capabilities
role_capability("refiner", "error_correction").
role_capability("refiner", "quality_optimization").
role_capability("refiner", "iterative_improvement").

% ==========================================
% 4. CREATE AGENT INSTANCES
% ==========================================

agent("observer-001", "observer").
agent("orienter-001", "orienter").
agent("planner-001", "planner").
agent("executor-001", "executor").
agent("reviewer-001", "reviewer").
agent("refiner-001", "refiner").
agent("coordinator-001", "coordinator").

% ==========================================
% 5. ASSIGN INSTANCE CAPABILITIES
% ==========================================

agent_capability("observer-001", "information_gathering").
agent_capability("observer-001", "semantic_search").
agent_capability("observer-001", "entity_extraction").

agent_capability("executor-001", "content_generation").
agent_capability("executor-001", "code_generation").
agent_capability("executor-001", "tool_invocation").

% ==========================================
% 6. CONFIGURE AGENT PARAMETERS
% ==========================================

agent_config("observer-001", "model", "gpt-4o").
agent_config("observer-001", "temperature", "0.5").
agent_config("observer-001", "max_tokens", "4000").

agent_config("executor-001", "model", "gpt-4o").
agent_config("executor-001", "temperature", "0.7").
agent_config("executor-001", "timeout_ms", "60000").
agent_config("executor-001", "max_retries", "3").

agent_config("reviewer-001", "model", "gpt-4o").
agent_config("reviewer-001", "temperature", "0.2").
agent_config("reviewer-001", "strict_mode", "true").
```

### Step 2: Define OODA Phases

Create or update `kb/ooda-phases.dl`:

```datalog
% ==========================================
% PHASE ORDER (core loop)
% ==========================================

phase_order(1, "observe").
phase_order(2, "orient").
phase_order(3, "decide").
phase_order(4, "act").
phase_order(5, "verify").
phase_order(6, "refine").

% ==========================================
% PHASE DEFINITIONS
% ==========================================

phase("observe", "Context Gathering", "Collect and ingest input from various sources").
phase("orient", "Semantic Synthesis", "Process, classify, and understand the context").
phase("decide", "Strategic Planning", "Select approach and plan execution path").
phase("act", "Execution", "Perform the planned actions").
phase("verify", "Validation", "Check output quality and correctness").
phase("refine", "Iteration", "Self-correct and optimize based on feedback").

% ==========================================
% PHASE ACTIONS
% ==========================================

% Observe actions
action("observe", "gather_input", "Gather input from user or system").
action("observe", "parse_request", "Parse and normalize request").
action("observe", "extract_entities", "Extract named entities").
action("observe", "validate_format", "Validate input format").

% Orient actions
action("orient", "classify_intent", "Classify user intent").
action("orient", "retrieve_knowledge", "Retrieve relevant knowledge").
action("orient", "map_patterns", "Map to known patterns").

% Decide actions
action("decide", "select_playbook", "Select appropriate playbook").
action("decide", "plan_tasks", "Create task execution plan").
action("decide", "assess_risks", "Identify and assess risks").

% Act actions
action("act", "execute_task", "Execute planned task").
action("act", "generate_content", "Generate requested content").
action("act", "invoke_tool", "Invoke external tool").

% Verify actions
action("verify", "check_quality", "Verify output quality").
action("verify", "validate_rules", "Validate against rules").
action("verify", "check_completeness", "Verify completeness").

% Refine actions
action("refine", "apply_fixes", "Apply corrections").
action("refine", "optimize_output", "Optimize output quality").
action("refine", "prepare_retry", "Prepare for retry if needed").

% ==========================================
% PHASE TRANSITIONS
% ==========================================

phase_transition("observe", "orient").
phase_transition("orient", "decide").
phase_transition("decide", "act").
phase_transition("act", "verify").
phase_transition("verify", "refine").
phase_transition("refine", "observe"). % Loop back

% Conditional transitions
conditional_transition("verify", "act", "needs_retry").
conditional_transition("verify", "observe", "needs_more_context").
conditional_transition("refine", "decide", "needs_new_strategy").

% ==========================================
% PHASE CONFIGURATION
% ==========================================

phase_config("observe", "timeout_ms", "5000").
phase_config("observe", "retry_on_failure", "true").

phase_config("decide", "max_plan_depth", "10").
phase_config("decide", "timeout_ms", "15000").
phase_config("decide", "allow_fallback", "true").

phase_config("act", "timeout_ms", "60000").
phase_config("act", "allow_parallel", "true").

phase_config("verify", "strict_mode", "false").
phase_config("verify", "fail_threshold", "0.8").

phase_config("refine", "max_iterations", "3").
phase_config("refine", "improvement_threshold", "0.1").
```

### Step 3: Define the Workflow

Create or update `kb/workflows/ooda_workflow.dl`:

```datalog
% ==========================================
% WORKFLOW DEFINITION
% ==========================================

workflow("ooda_pipeline", "OODA Loop Pipeline", "v1.0").
workflow_description("ooda_pipeline", "Generic OODA loop for agentic task execution").

% ==========================================
% WORKFLOW NODES (grouped by phase)
% ==========================================

% OBSERVE Phase
workflow_node("ooda_pipeline", "observe_input", "agent", "observer").
workflow_node("ooda_pipeline", "observe_extract", "action", "extract_entities").
workflow_node("ooda_pipeline", "observe_validate", "action", "validate_input").

% ORIENT Phase
workflow_node("ooda_pipeline", "orient_classify", "agent", "orienter").
workflow_node("ooda_pipeline", "orient_retrieve", "action", "retrieve_knowledge").
workflow_node("ooda_pipeline", "orient_synthesize", "action", "synthesize_context").

% DECIDE Phase
workflow_node("ooda_pipeline", "decide_select", "agent", "planner").
workflow_node("ooda_pipeline", "decide_plan", "action", "create_plan").
workflow_node("ooda_pipeline", "decide_assess", "action", "assess_risks").

% ACT Phase
workflow_node("ooda_pipeline", "act_execute", "agent", "executor").
workflow_node("ooda_pipeline", "act_generate", "action", "generate_output").

% VERIFY Phase
workflow_node("ooda_pipeline", "verify_check", "agent", "reviewer").
workflow_node("ooda_pipeline", "verify_validate", "action", "validate_rules").

% REFINE Phase
workflow_node("ooda_pipeline", "refine_decide", "agent", "refiner").
workflow_node("ooda_pipeline", "refine_fix", "action", "apply_fixes").
workflow_node("ooda_pipeline", "refine_complete", "action", "finalize").

% ==========================================
% NODE CONFIGURATION
% ==========================================

node_config("ooda_pipeline", "observe_input", "max_size", "100000").
node_config("ooda_pipeline", "orient_classify", "confidence_threshold", "0.7").
node_config("ooda_pipeline", "decide_plan", "max_depth", "10").
node_config("ooda_pipeline", "act_execute", "allow_parallel", "true").
node_config("ooda_pipeline", "verify_check", "strict_mode", "false").
node_config("ooda_pipeline", "refine_decide", "max_iterations", "3").

% ==========================================
% WORKFLOW EDGES (sequential flow)
% ==========================================

% Observe → Orient
workflow_edge("ooda_pipeline", "observe_input", "observe_extract").
workflow_edge("ooda_pipeline", "observe_extract", "observe_validate").
workflow_edge("ooda_pipeline", "observe_validate", "orient_classify").

% Orient → Decide
workflow_edge("ooda_pipeline", "orient_classify", "orient_retrieve").
workflow_edge("ooda_pipeline", "orient_retrieve", "orient_synthesize").
workflow_edge("ooda_pipeline", "orient_synthesize", "decide_select").

% Decide → Act
workflow_edge("ooda_pipeline", "decide_select", "decide_plan").
workflow_edge("ooda_pipeline", "decide_plan", "decide_assess").
workflow_edge("ooda_pipeline", "decide_assess", "act_execute").

% Act → Verify
workflow_edge("ooda_pipeline", "act_execute", "act_generate").
workflow_edge("ooda_pipeline", "act_generate", "verify_check").

% Verify → Refine
workflow_edge("ooda_pipeline", "verify_check", "verify_validate").
workflow_edge("ooda_pipeline", "verify_validate", "refine_decide").

% Refine → Loop/Complete
workflow_edge("ooda_pipeline", "refine_decide", "refine_fix").
workflow_edge("ooda_pipeline", "refine_fix", "refine_complete").

% ==========================================
% CONDITIONAL EDGES (error recovery)
% ==========================================

% Verification passed → complete
conditional_edge("ooda_pipeline", "refine_decide", "refine_complete", 
    "verification_passed(context)").

% Verification failed → retry (up to 3 times)
conditional_edge("ooda_pipeline", "refine_decide", "refine_fix", 
    "verification_failed(context), iteration_count(Count), Count < 3").

% Needs new strategy → go back to decide
conditional_edge("ooda_pipeline", "refine_decide", "decide_select", 
    "needs_new_strategy(context)").

% ==========================================
% ERROR HANDLING
% ==========================================

error_edge("ooda_pipeline", "observe_input", "refine_complete", "input_error(context)").
error_edge("ooda_pipeline", "decide_plan", "decide_select", "planning_failed(context)").
error_edge("ooda_pipeline", "act_execute", "decide_select", "execution_failed(context)").
```

### Step 4: Register Tools

Add tools to `kb/tools/registry.dl`:

```datalog
% ==========================================
% TOOL DEFINITIONS
% ==========================================

tool("semantic_search", "search").
tool("llm_generate", "generation").
tool("entity_extractor", "analysis").
tool("schema_validator", "validation").
tool("formatter", "transformation").

% ==========================================
% TOOL CAPABILITIES
% ==========================================

tool_capability("semantic_search", "natural_language_query").
tool_capability("semantic_search", "vector_similarity").

tool_capability("llm_generate", "text_generation").
tool_capability("llm_generate", "structured_output").
tool_capability("llm_generate", "code_generation").

tool_capability("entity_extractor", "ner").
tool_capability("entity_extractor", "relationship_extraction").

tool_capability("schema_validator", "json_validation").
tool_capability("schema_validator", "yaml_validation").

% ==========================================
% TOOL → OODA PHASE MAPPING
% ==========================================

tool_ooda_phase("semantic_search", "observe").
tool_ooda_phase("entity_extractor", "observe").

tool_ooda_phase("llm_generate", "act").

tool_ooda_phase("schema_validator", "verify").
tool_ooda_phase("formatter", "refine").

% ==========================================
% TOOL CONFIGURATION
% ==========================================

tool_config("llm_generate", "model", "gpt-4o").
tool_config("llm_generate", "temperature", "0.7").
tool_config("llm_generate", "max_tokens", "4000").
tool_config("llm_generate", "timeout_ms", "60000").

tool_config("semantic_search", "top_k", "10").
tool_config("semantic_search", "min_score", "0.7").
```

### Step 5: Define Validation Rules

Add quality rules to `kb/validation/rules.dl`:

```datalog
% ==========================================
% VALIDATION RULES
% ==========================================

validation_rule("output_not_empty", "Output must not be empty").
validation_rule("output_within_size_limit", "Output must be within configured size limit").
validation_rule("has_required_sections", "All required sections must be present").
validation_rule("no_placeholder_text", "No placeholder or TODO text in output").
validation_rule("no_secrets_exposed", "No secrets or credentials in output").
validation_rule("internal_consistency", "Internal references must be consistent").

% ==========================================
% VALIDATION SEVERITY
% ==========================================

validation_severity("output_not_empty", "critical").
validation_severity("output_within_size_limit", "error").
validation_severity("has_required_sections", "error").
validation_severity("no_placeholder_text", "error").
validation_severity("no_secrets_exposed", "critical").
validation_severity("internal_consistency", "warning").

% ==========================================
% VALIDATION ACTIONS
% ==========================================

validation_action("critical", "fail").
validation_action("error", "fail").
validation_action("warning", "warn").
validation_action("info", "log").
```

---

## 4. Agent Configuration

### Basic Agent Setup

```go
// agent.go
package agent

type Agent struct {
    ID          string
    Role        string      // observer, orienter, planner, etc.
    Model       string
    Temperature float64
    MaxTokens   int
    Tools       []string
    Status      string
}

func NewAgent(role string, config map[string]string) *Agent {
    return &Agent{
        ID:          fmt.Sprintf("%s-001", role),
        Role:        role,
        Model:       config["model"],
        Temperature: parseFloat(config["temperature"]),
        MaxTokens:   parseInt(config["max_tokens"]),
        Status:      "available",
    }
}
```

### Coordinator Agent

The coordinator orchestrates the OODA loop:

```go
type Coordinator struct {
    *Agent
    currentPhase string
    context      map[string]interface{}
}

func (c *Coordinator) ExecuteOodaLoop(input interface{}) (interface{}, error) {
    phases := []string{"observe", "orient", "decide", "act", "verify", "refine"}
    
    for {
        c.currentPhase = phases[0]
        
        switch c.currentPhase {
        case "observe":
            if err := c.observe(input); err != nil {
                return nil, err
            }
        case "orient":
            if err := c.orient(); err != nil {
                return nil, err
            }
        case "decide":
            if err := c.decide(); err != nil {
                return nil, err
            }
        case "act":
            if err := c.act(); err != nil {
                return nil, err
            }
        case "verify":
            if err := c.verify(); err != nil {
                return nil, err
            }
        case "refine":
            result, err := c.refine()
            if err != nil {
                return nil, err
            }
            if c.isComplete() {
                return result, nil
            }
        }
        
        // Rotate phases
        phases = append(phases[1:], phases[0])
    }
}
```

### Agent Selection by Phase

```go
func SelectAgentForPhase(phase string, kb *KnowledgeBase) (*Agent, error) {
    query := fmt.Sprintf(`
        role_ooda_phase(Role, "%s"),
        agent(Agent, Role),
        agent_status(Agent, "available")
    `, phase)
    
    results, err := kb.Query(query)
    if err != nil {
        return nil, err
    }
    
    if len(results) == 0 {
        return nil, fmt.Errorf("no available agent for phase: %s", phase)
    }
    
    return results[0].Agent, nil
}
```

---

## 5. Workflow Definition

### Loading Workflows from KB

```go
type WorkflowEngine struct {
    kb *KnowledgeBase
}

func (we *WorkflowEngine) LoadWorkflow(name string) (*Workflow, error) {
    query := fmt.Sprintf(`workflow("%s", Name, Version)`, name)
    results, err := we.kb.Query(query)
    if err != nil {
        return nil, err
    }
    
    workflow := &Workflow{
        Name:    results[0].Name,
        Version: results[0].Version,
        Nodes:   make([]*Node, 0),
        Edges:   make([]*Edge, 0),
    }
    
    // Load nodes
    nodeQuery := fmt.Sprintf(`workflow_node("%s", NodeId, Type, Role)`, name)
    nodes, _ := we.kb.Query(nodeQuery)
    for _, n := range nodes {
        workflow.Nodes = append(workflow.Nodes, &Node{
            ID:   n.NodeId,
            Type: n.Type,
            Role: n.Role,
        })
    }
    
    // Load edges
    edgeQuery := fmt.Sprintf(`workflow_edge("%s", From, To)`, name)
    edges, _ := we.kb.Query(edgeQuery)
    for _, e := range edges {
        workflow.Edges = append(workflow.Edges, &Edge{
            From: e.From,
            To:   e.To,
        })
    }
    
    return workflow, nil
}
```

### Execute Workflow

```go
func (we *WorkflowEngine) Execute(wf *Workflow, input interface{}) (interface{}, error) {
    state := &ExecutionState{
        CurrentNode: "observe_input",
        Context:     map[string]interface{}{"input": input},
    }
    
    for {
        node := wf.GetNode(state.CurrentNode)
        
        // Execute node
        result, err := node.Execute(state)
        if err != nil {
            // Handle error edges
            if edge := wf.GetErrorEdge(state.CurrentNode); edge != nil {
                state.CurrentNode = edge.To
                continue
            }
            return nil, err
        }
        
        // Store result
        state.Context[node.ID] = result
        
        // Check conditional edges
        if edge := wf.GetConditionalEdge(state.CurrentNode, state.Context); edge != nil {
            state.CurrentNode = edge.To
        } else if edge := wf.GetEdge(state.CurrentNode); edge != nil {
            state.CurrentNode = edge.To
        } else {
            break // Workflow complete
        }
    }
    
    return state.Context["refine_complete"], nil
}
```

---

## 6. Knowledge Base Integration

### Initialize KB with OODA

```go
func InitKnowledgeBase(kb *KnowledgeBase) error {
    files := []string{
        "kb/ooda-phases.dl",
        "kb/agents/registry.dl",
        "kb/workflows/ooda_workflow.dl",
        "kb/tools/registry.dl",
        "kb/validation/rules.dl",
    }
    
    for _, file := range files {
        if err := kb.Load(file); err != nil {
            return fmt.Errorf("failed to load %s: %w", file, err)
        }
    }
    
    return nil
}
```

### Query KB for Agent Selection

```go
func GetAgentsForTask(task string, kb *KnowledgeBase) ([]*Agent, error) {
    query := fmt.Sprintf(`
        task_requires_capability("%s", Capability),
        role_capability(Role, Capability),
        agent(Agent, Role),
        agent_status(Agent, "available")
    `, task)
    
    return kb.QueryAgents(query)
}
```

### Get Phase Configuration

```go
func GetPhaseConfig(phase string, key string, kb *KnowledgeBase) (string, error) {
    query := fmt.Sprintf(`phase_config("%s", "%s", Value)`, phase, key)
    results, err := kb.Query(query)
    if err != nil {
        return "", err
    }
    return results[0].Value, nil
}
```

---

## 7. Validation & Quality Gates

### Run Validation

```go
type Validator struct {
    kb *KnowledgeBase
}

func (v *Validator) Validate(output interface{}) (*ValidationResult, error) {
    result := &ValidationResult{
        Passed: true,
        Issues: []ValidationIssue{},
    }
    
    // Get all validation rules
    rules, _ := v.kb.Query(`validation_rule(Rule, Description)`)
    
    for _, rule := range rules {
        // Check rule against output
        passed, err := v.checkRule(rule.Rule, output)
        if err != nil {
            continue
        }
        
        if !passed {
            severity, _ := v.kb.GetSeverity(rule.Rule)
            result.Issues = append(result.Issues, ValidationIssue{
                Rule:       rule.Rule,
                Description: rule.Description,
                Severity:   severity,
            })
            
            if severity == "critical" || severity == "error" {
                result.Passed = false
            }
        }
    }
    
    return result, nil
}

func (v *Validator) checkRule(rule, output interface{}) (bool, error) {
    switch rule {
    case "output_not_empty":
        return output != nil && output != "", nil
    case "no_placeholder_text":
        return !containsPlaceholder(output), nil
    case "no_secrets_exposed":
        return !containsSecrets(output), nil
    default:
        return true, nil
    }
}
```

### Integration with Refine Phase

```go
func (r *Refiner) Refine(output interface{}, validation *ValidationResult) (interface{}, bool, error) {
    if validation.Passed {
        return output, true, nil // Done
    }
    
    // Check iteration count
    if r.iterationCount >= r.maxIterations {
        return output, true, nil // Max iterations reached
    }
    
    r.iterationCount++
    
    // Apply fixes for each issue
    for _, issue := range validation.Issues {
        if issue.Severity == "critical" || issue.Severity == "error" {
            output, err := r.applyFix(issue, output)
            if err != nil {
                return nil, false, err
            }
        }
    }
    
    return output, false, nil // Continue iteration
}
```

---

## 8. Domain-Specific Extensions

### Create Domain-Specific KB

```
kb/
└── domains/
    └── architect/
        ├── agents/
        │   └── architect.dl
        ├── workflows/
        │   └── csd_workflow.dl
        └── patterns/
            └── architecture.dl
```

### Example: Architect Agent Extension

```datalog
% kb/domains/architect/agents/architect.dl

% ==========================================
% ARCHITECT AGENT (extends generic)
% ==========================================

agent_role("architect").  % Additional role

role_capability("architect", "read_code").
role_capability("architect", "analyze_arch").
role_capability("architect", "generate_csd").
role_capability("architect", "review_design").

% Architect inherits from executor
role_inherits("architect", "executor").

% Agent instances
agent("architect-001", "architect").
agent_capability("architect-001", "read_code").
agent_capability("architect-001", "analyze_arch").
agent_capability("architect-001", "generate_csd").

agent_config("architect-001", "model", "gpt-4o").
agent_config("architect-001", "temperature", "0.3").
agent_config("architect-001", "max_tokens", "8000").
```

### Example: CSD Workflow Extension

```datalog
% kb/domains/architect/workflows/csd_workflow.dl

workflow("csd_wf", "Conceptual Solution Design", "v1.0").

% Uses generic OODA nodes as base
workflow_node("csd_wf", "research", "agent", "architect").
workflow_node("csd_wf", "analyze", "agent", "architect").
workflow_node("csd_wf", "design", "agent", "architect").
workflow_node("csd_wf", "review", "agent", "architect").

% Specific node configuration
node_config("csd_wf", "research", "depth", "comprehensive").
node_config("csd_wf", "research", "file_extensions", ".go,.ts,.js,.py").
node_config("csd_wf", "design", "include_diagrams", "true").
node_config("csd_wf", "review", "strict_mode", "true").

% Workflow edges
workflow_edge("csd_wf", "research", "analyze").
workflow_edge("csd_wf", "analyze", "design").
workflow_edge("csd_wf", "design", "review").
```

---

## 9. Best Practices

### 1. Define Clear Phase Boundaries

```datalog
% DO: Clear separation
phase_boundary("ooda_pipeline", "observe").
phase_boundary("ooda_pipeline", "orient").
phase_boundary("ooda_pipeline", "decide").

% DON'T: Mixed concerns
action("observe", "decide_plan", "Create plan").  % Wrong phase
```

### 2. Use Meaningful Agent IDs

```datalog
% DO: Descriptive IDs
agent("planner-001", "planner").
agent("executor-content-001", "executor").

% DON'T: Generic IDs
agent("agent-1", "planner").
agent("agent-2", "executor").
```

### 3. Configure Appropriate Timeouts

```datalog
% Observe: Quick input gathering
phase_config("observe", "timeout_ms", "5000").

% Decide: May need more time for planning
phase_config("decide", "timeout_ms", "15000").

% Act: Generation can take time
phase_config("act", "timeout_ms", "60000").

% Verify: Depends on output size
phase_config("verify", "timeout_ms", "30000").
```

### 4. Set Realistic Retry Limits

```datalog
% DON'T: Infinite retries
phase_config("refine", "max_iterations", "100").  % Too high

% DO: Reasonable limit
phase_config("refine", "max_iterations", "3").    % Industry standard
```

### 5. Document Custom Extensions

```datalog
% DO: Add descriptions
phase("custom_phase", "Custom Purpose", "Explains why this exists").
domain_phase_extension("my_domain", "decide", "custom_action").

% Keep extension points documented
% See: docs/DOMAIN-EXTENSION-GUIDE.md
```

### 6. Validate Early, Fail Fast

```datalog
% DO: Validate input in Observe
workflow_node("ooda_pipeline", "observe_validate", "action", "validate_input").

% Validate before expensive operations
conditional_edge("ooda_pipeline", "observe_validate", "refine_complete", 
    "input_invalid(context)").
```

---

## 10. Examples

### Example 1: Simple Document Generator

```datalog
% Document Generator Workflow
workflow("doc_gen", "Document Generation", "v1.0").

% Simplified OODA
workflow_node("doc_gen", "parse_req", "agent", "observer").
workflow_node("doc_gen", "classify", "agent", "orienter").
workflow_node("doc_gen", "plan", "agent", "planner").
workflow_node("doc_gen", "write", "agent", "executor").
workflow_node("doc_gen", "review", "agent", "reviewer").

workflow_edge("doc_gen", "parse_req", "classify").
workflow_edge("doc_gen", "classify", "plan").
workflow_edge("doc_gen", "plan", "write").
workflow_edge("doc_gen", "write", "review").
```

### Example 2: Multi-Agent Code Review

```datalog
% Code Review Workflow
workflow("code_review", "Code Review Pipeline", "v1.0").

% Phase: Observe
workflow_node("code_review", "fetch_code", "agent", "observer").
workflow_node("code_review", "parse_code", "action", "parse_code").

% Phase: Orient  
workflow_node("code_review", "identify_lang", "agent", "orienter").
workflow_node("code_review", "load_rules", "action", "retrieve_knowledge").

% Phase: Decide
workflow_node("code_review", "plan_review", "agent", "planner").

% Phase: Act
workflow_node("code_review", "static_analysis", "agent", "executor").
workflow_node("code_review", "check_style", "action", "lint_code").
workflow_node("code_review", "check_security", "action", "security_scan").

% Phase: Verify
workflow_node("code_review", "aggregate", "agent", "reviewer").

% Phase: Refine
workflow_node("code_review", "format_report", "action", "format_output").
```

### Example 3: Query-Based Agent Selection

```go
// Select the best agent for a task
func SelectBestAgent(task string, kb *KnowledgeBase) (*Agent, error) {
    // Query for agents with required capabilities
    query := `
        task_requires_capability($task, $capability),
        agent($agent, $role),
        role_capability($role, $capability),
        agent_capability($agent, $capability),
        agent_status($agent, "available")
    `
    
    results, err := kb.Query(query, {"task": task})
    if err != nil {
        return nil, err
    }
    
    // Score and rank agents
    var bestAgent *Agent
    var bestScore float64
    
    for _, result := range results {
        agent := result.Agent
        score := calculateScore(agent, task)
        if score > bestScore {
            bestScore = score
            bestAgent = agent
        }
    }
    
    return bestAgent, nil
}
```

---

## Quick Reference

### Essential Datalog Facts

| Pattern | Example |
|---------|---------|
| Define fact | `agent("executor-001", "executor").` |
| Query facts | `agent(A, "executor").` |
| With condition | `agent(A, R), role_ooda_phase(R, "act").` |
| Find all | `findall(A, agent(A, _), Agents).` |

### Common Queries

```datalog
% Get all agents for a phase
role_ooda_phase(Role, "act"), agent(Agent, Role).

% Get all tools for a phase
tool_ooda_phase(Tool, "verify"), tool_available(Tool, "available").

% Get workflow nodes
workflow_node("ooda_pipeline", Node, Type, Role).

% Get validation rules
validation_rule(Rule, Desc), validation_severity(Rule, "error").
```

### File Checklist

- [ ] `kb/ooda-phases.dl` - Core phases
- [ ] `kb/agents/registry.dl` - Agent definitions
- [ ] `kb/workflows/ooda_workflow.dl` - OODA pipeline
- [ ] `kb/tools/registry.dl` - Tool definitions
- [ ] `kb/validation/rules.dl` - Quality rules
- [ ] `kb/INDEX.md` - Documentation

---

**For more details, see:**
- [OODA Phases](../kb/ooda-phases.dl)
- [Agent Registry](../kb/agents/registry.dl)
- [Workflow Registry](../kb/workflows/registry.dl)
- [Validation Rules](../kb/validation/rules.dl)
