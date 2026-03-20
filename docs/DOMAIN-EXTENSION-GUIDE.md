# Domain Extension Guide

**How to extend the generic OODA framework for specific domains.**

---

## Overview

The generic OODA framework is designed to be domain-agnostic. This guide shows how to extend it for specific use cases like:
- Code analysis
- Document generation
- Data processing
- API design

---

## Directory Structure

```
kb/
├── ooda-phases.dl           # Core generic phases
├── agents/
│   └── registry.dl          # Generic agents
├── workflows/
│   └── ooda_workflow.dl     # Generic pipeline
├── tools/
│   └── registry.dl          # Generic tools
├── validation/
│   └── rules.dl             # Generic rules
└── domains/
    └── {domain-name}/
        ├── agents/
        │   └── {domain}.dl
        ├── workflows/
        │   └── {domain}-workflow.dl
        ├── tools/
        │   └── {domain}-tools.dl
        ├── validation/
        │   └── {domain}-rules.dl
        ├── patterns/
        │   └── {domain}-patterns.dl
        └── templates/
            └── {domain}-templates.dl
```

---

## Step 1: Create Domain Directory

```bash
mkdir -p kb/domains/code-analyzer/{agents,workflows,tools,validation}
```

---

## Step 2: Extend Agents

```datalog
% kb/domains/code-analyzer/agents/code-analyzer.dl

% ==========================================
% DOMAIN-SPECIFIC AGENTS
% ==========================================

agent_role("code_analyzer").
agent_role("test_writer").
agent_role("security_expert").

% Map to OODA phases
role_ooda_phase("code_analyzer", "act").
role_ooda_phase("test_writer", "act").
role_ooda_phase("security_expert", "verify").

% Domain capabilities
role_capability("code_analyzer", "parse_code").
role_capability("code_analyzer", "analyze_complexity").
role_capability("code_analyzer", "detect_patterns").
role_capability("code_analyzer", "suggest_refactor").

role_capability("test_writer", "generate_tests").
role_capability("test_writer", "mock_dependencies").

role_capability("security_expert", "detect_vulnerabilities").
role_capability("security_expert", "check_dependencies").
role_capability("security_expert", "validate_auth").

% Agent instances
agent("code-analyzer-001", "code_analyzer").
agent("test-writer-001", "test_writer").
agent("security-expert-001", "security_expert").

agent_capability("code-analyzer-001", "parse_code").
agent_capability("code-analyzer-001", "analyze_complexity").
agent_capability("code-analyzer-001", "detect_patterns").
agent_capability("code-analyzer-001", "suggest_refactor").

agent_capability("test-writer-001", "generate_tests").
agent_capability("test-writer-001", "mock_dependencies").

agent_capability("security-expert-001", "detect_vulnerabilities").
agent_capability("security-expert-001", "check_dependencies").
agent_capability("security-expert-001", "validate_auth").

% Configuration
agent_config("code-analyzer-001", "model", "gpt-4o").
agent_config("code-analyzer-001", "temperature", "0.3").
agent_config("code-analyzer-001", "max_tokens", "8000").

agent_config("security-expert-001", "model", "gpt-4o").
agent_config("security-expert-001", "temperature", "0.2").
agent_config("security-expert-001", "strict_mode", "true").
```

---

## Step 3: Extend Tools

```datalog
% kb/domains/code-analyzer/tools/code-analyzer-tools.dl

% ==========================================
% DOMAIN-SPECIFIC TOOLS
% ==========================================

tool("code_parser", "analysis").
tool("complexity_analyzer", "analysis").
tool("test_generator", "generation").
tool("dependency_checker", "validation").
tool("security_scanner", "validation").

% Tool capabilities
tool_capability("code_parser", "syntax_parsing").
tool_capability("code_parser", "ast_generation").
tool_capability("code_parser", "import_resolution").

tool_capability("complexity_analyzer", "cyclomatic_complexity").
tool_capability("complexity_analyzer", "coupling_analysis").
tool_capability("complexity_analyzer", "cohesion_metrics").

tool_capability("test_generator", "unit_test_generation").
tool_capability("test_generator", "integration_test_generation").
tool_capability("test_generator", "mock_generation").

tool_capability("security_scanner", "owasp_check").
tool_capability("security_scanner", "dependency_audit").
tool_capability("security_scanner", "secret_detection").

% Tool configuration
tool_config("code_parser", "supported_languages", "go,python,javascript,typescript").
tool_config("code_parser", "max_file_size", "1000000").

tool_config("test_generator", "framework", "gock").
tool_config("test_generator", "coverage_target", "80").

tool_config("security_scanner", "ruleset", "owasp-top-10").
tool_config("security_scanner", "fail_on_high", "true").

% Map to OODA phases
tool_ooda_phase("code_parser", "observe").
tool_ooda_phase("complexity_analyzer", "orient").
tool_ooda_phase("test_generator", "act").
tool_ooda_phase("dependency_checker", "verify").
tool_ooda_phase("security_scanner", "verify").
```

---

## Step 4: Extend Validation Rules

```datalog
% kb/domains/code-analyzer/validation/code-analyzer-rules.dl

% ==========================================
% CODE-SPECIFIC VALIDATION RULES
% ==========================================

validation_rule("code_compiles", "Code must compile without errors").
validation_rule("has_tests", "Code must have associated tests").
validation_rule("no_high_complexity", "Cyclomatic complexity must be below threshold").
validation_rule("no_security_issues", "No critical security vulnerabilities").
validation_rule("proper_naming", "Code follows naming conventions").
validation_rule("has_documentation", "Public APIs have documentation").
validation_rule("tests_pass", "All tests must pass").
validation_rule("no_hardcoded_secrets", "No hardcoded credentials or secrets").

% Severity levels
validation_severity("code_compiles", "critical").
validation_severity("has_tests", "error").
validation_severity("no_high_complexity", "warning").
validation_severity("no_security_issues", "critical").
validation_severity("proper_naming", "warning").
validation_severity("has_documentation", "warning").
validation_severity("tests_pass", "error").
validation_severity("no_hardcoded_secrets", "critical").
```

---

## Step 5: Create Domain Workflow

```datalog
% kb/domains/code-analyzer/workflows/code-analyzer-workflow.dl

% ==========================================
% CODE ANALYSIS WORKFLOW
% ==========================================

workflow("code_analyzer_wf", "Code Analysis Pipeline", "v1.0").
workflow_description("code_analyzer_wf", "Analyzes code for quality, complexity, and security").

% ==========================================
% WORKFLOW NODES (OODA-aligned)
% ==========================================

% OBSERVE: Gather code
workflow_node("code_analyzer_wf", "fetch_code", "agent", "observer").
workflow_node("code_analyzer_wf", "parse_code", "tool", "code_parser").
workflow_node("code_analyzer_wf", "validate_input", "action", "validate_input").

% ORIENT: Understand structure
workflow_node("code_analyzer_wf", "analyze_structure", "agent", "code_analyzer").
workflow_node("code_analyzer_wf", "detect_patterns", "tool", "pattern_matcher").
workflow_node("code_analyzer_wf", "check_dependencies", "tool", "dependency_checker").

% DECIDE: Plan analysis
workflow_node("code_analyzer_wf", "plan_review", "agent", "planner").
workflow_node("code_analyzer_wf", "assess_risks", "action", "assess_risks").

% ACT: Generate analysis
workflow_node("code_analyzer_wf", "analyze_complexity", "tool", "complexity_analyzer").
workflow_node("code_analyzer_wf", "generate_report", "agent", "code_analyzer").
workflow_node("code_analyzer_wf", "generate_tests", "agent", "test_writer").

% VERIFY: Validate results
workflow_node("code_analyzer_wf", "security_scan", "tool", "security_scanner").
workflow_node("code_analyzer_wf", "review_results", "agent", "reviewer").
workflow_node("code_analyzer_wf", "validate_rules", "action", "validate_rules").

% REFINE: Improve output
workflow_node("code_analyzer_wf", "refine_report", "agent", "refiner").
workflow_node("code_analyzer_wf", "finalize", "action", "finalize").

% ==========================================
% NODE CONFIGURATION
% ==========================================

node_config("code_analyzer_wf", "parse_code", "languages", "go,python,js,ts").
node_config("code_analyzer_wf", "parse_code", "max_file_size", "1000000").

node_config("code_analyzer_wf", "analyze_complexity", "max_complexity", "10").
node_config("code_analyzer_wf", "analyze_complexity", "include_coverage", "true").

node_config("code_analyzer_wf", "security_scan", "ruleset", "owasp-top-10").
node_config("code_analyzer_wf", "security_scan", "fail_on_critical", "true").

node_config("code_analyzer_wf", "generate_tests", "coverage_target", "80").
node_config("code_analyzer_wf", "generate_tests", "framework", "gock").

node_config("code_analyzer_wf", "refine_report", "max_iterations", "2").
node_config("code_analyzer_wf", "refine_report", "improvement_threshold", "0.1").

% ==========================================
% WORKFLOW EDGES
% ==========================================

% Observe → Orient
workflow_edge("code_analyzer_wf", "fetch_code", "parse_code").
workflow_edge("code_analyzer_wf", "parse_code", "validate_input").
workflow_edge("code_analyzer_wf", "validate_input", "analyze_structure").

% Orient → Decide
workflow_edge("code_analyzer_wf", "analyze_structure", "detect_patterns").
workflow_edge("code_analyzer_wf", "detect_patterns", "check_dependencies").
workflow_edge("code_analyzer_wf", "check_dependencies", "plan_review").

% Decide → Act
workflow_edge("code_analyzer_wf", "plan_review", "assess_risks").
workflow_edge("code_analyzer_wf", "assess_risks", "analyze_complexity").

% Act → Verify
workflow_edge("code_analyzer_wf", "analyze_complexity", "generate_report").
workflow_edge("code_analyzer_wf", "generate_report", "generate_tests").
workflow_edge("code_analyzer_wf", "generate_tests", "security_scan").

% Verify → Refine
workflow_edge("code_analyzer_wf", "security_scan", "review_results").
workflow_edge("code_analyzer_wf", "review_results", "validate_rules").
workflow_edge("code_analyzer_wf", "validate_rules", "refine_report").

% Refine → Complete
workflow_edge("code_analyzer_wf", "refine_report", "finalize").

% ==========================================
% CONDITIONAL EDGES
% ==========================================

% Validation passed → complete
conditional_edge("code_analyzer_wf", "validate_rules", "refine_report",
    "validation_passed(context)").

% High complexity → flag warning
conditional_edge("code_analyzer_wf", "analyze_complexity", "generate_report",
    "complexity_within_threshold(context)").

conditional_edge("code_analyzer_wf", "analyze_complexity", "review_results",
    "complexity_exceeded(context)").

% Security issues → fail fast
conditional_edge("code_analyzer_wf", "security_scan", "finalize",
    "no_critical_issues(context)").

conditional_edge("code_analyzer_wf", "security_scan", "refine_report",
    "critical_issues_found(context)").

% ==========================================
% ERROR HANDLING
% ==========================================

error_edge("code_analyzer_wf", "fetch_code", "finalize", "fetch_failed(context)").
error_edge("code_analyzer_wf", "parse_code", "finalize", "parse_failed(context)").
error_edge("code_analyzer_wf", "analyze_complexity", "generate_report", "analysis_timeout(context)").
```

---

## Step 6: Load Domain Extension

```go
func LoadCodeAnalyzerKB(kb *knowledgebase.KnowledgeBase) error {
    // Load generic OODA base
    if err := kb.Load("kb/ooda-phases.dl"); err != nil {
        return err
    }
    if err := kb.Load("kb/agents/registry.dl"); err != nil {
        return err
    }
    
    // Load domain-specific extensions
    if err := kb.Load("kb/domains/code-analyzer/agents/code-analyzer.dl"); err != nil {
        return err
    }
    if err := kb.Load("kb/domains/code-analyzer/tools/code-analyzer-tools.dl"); err != nil {
        return err
    }
    if err := kb.Load("kb/domains/code-analyzer/validation/code-analyzer-rules.dl"); err != nil {
        return err
    }
    if err := kb.Load("kb/domains/code-analyzer/workflows/code-analyzer-workflow.dl"); err != nil {
        return err
    }
    
    return nil
}
```

---

## Step 7: Execute Domain Workflow

```go
func RunCodeAnalysis(codePath string) (*AnalysisResult, error) {
    kb := knowledgebase.New()
    if err := LoadCodeAnalyzerKB(kb); err != nil {
        return nil, err
    }
    
    engine := NewWorkflowEngine(kb)
    workflow, err := engine.LoadWorkflow("code_analyzer_wf")
    if err != nil {
        return nil, err
    }
    
    result, err := engine.Execute(workflow, map[string]interface{}{
        "code_path": codePath,
    })
    if err != nil {
        return nil, err
    }
    
    return result.(*AnalysisResult), nil
}
```

---

## Pattern: Document Generation Domain

Here's another example for document generation:

```
kb/domains/document-gen/
├── agents/document-gen.dl
├── workflows/document-gen-workflow.dl
├── templates/
│   ├── brd.dl
│   ├── csd.dl
│   └── add.dl
└── validation/document-gen-rules.dl
```

Key differences:
- **Agents**: Writer, Editor, Reviewer
- **Tools**: template_renderer, markdown_formatter, diagram_generator
- **Validation**: completeness, formatting, ID uniqueness

---

## Pattern: Data Processing Domain

```
kb/domains/data-processing/
├── agents/data-processor.dl
├── workflows/etl-workflow.dl
└── validation/data-quality-rules.dl
```

Key differences:
- **Agents**: Extractor, Transformer, Loader
- **Tools**: schema_validator, data_quality_checker, transformation_engine
- **Validation**: schema conformance, data quality metrics

---

## Best Practices

### 1. Always Extend, Never Override

```datalog
% DO: Add new capabilities
role_capability("code_analyzer", "new_capability").

% DON'T: Override generic capabilities
% role_capability("executor", "parse_code").  % Wrong
```

### 2. Keep Domain Files Isolated

```
kb/
├── generic files stay generic
└── domains/
    └── {domain}/
        └── {domain}-specific files
```

### 3. Document Domain Extensions

```datalog
% Add descriptions
workflow_description("code_analyzer_wf", "Analyzes code for quality and security").
agent_config("code-analyzer-001", "specialty", "golang").
```

### 4. Test Domain Workflows

```go
func TestCodeAnalyzerWorkflow(t *testing.T) {
    kb := knowledgebase.New()
    LoadCodeAnalyzerKB(kb)
    
    engine := NewWorkflowEngine(kb)
    workflow, _ := engine.LoadWorkflow("code_analyzer_wf")
    
    // Test all paths
    result, err := engine.Execute(workflow, testInput)
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

---

## Checklist

- [ ] Create domain directory structure
- [ ] Define domain-specific agents
- [ ] Add domain-specific tools
- [ ] Create validation rules
- [ ] Build domain workflow
- [ ] Add configuration options
- [ ] Document the domain
- [ ] Write tests
- [ ] Update main KB index

---

**See also:**
- [OODA Multi-Agent Guide](./OODA-MULTI-AGENT-GUIDE.md)
- [OODA Quick Start](./OODA-QUICKSTART.md)
- [API Reference](../api/README.md)
