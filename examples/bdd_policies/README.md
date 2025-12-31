# BDD Policy Examples

This directory contains example Gherkin-based policy definitions for Manglekit.

## Overview

Gherkin policies provide a human-readable, BDD-style approach to defining governance rules. These `.feature` files are automatically compiled to Datalog and enforced by the PolicyEngine.

## Example Policies

### 1. PII Protection (`pii_protection.feature`)

Prevents personally identifiable information (PII) from being sent to external services.

**Key Scenarios:**
- Block LLM calls with PII labels
- Block LLM calls with sensitive labels
- Allow LLM calls with public data

### 2. Access Control (`access_control.feature`)

Implements role-based access control (RBAC) for system operations.

**Key Scenarios:**
- Admin access to sensitive operations
- User permission restrictions
- Guest read-only access

### 3. Data Governance (`data_governance.feature`)

Comprehensive data governance including encryption, cross-border transfers, and audit requirements.

**Key Scenarios:**
- Cross-border data transfer prevention
- Encryption requirements
- High-risk operation auditing

## Usage

### Loading a Gherkin Policy

```go
import (
    "context"
    "os"
    "github.com/duynguyendang/manglekit/internal/engine"
)

// Create engine
policyEngine, err := engine.New()
if err != nil {
    panic(err)
}

// Load Gherkin policy
content, err := os.ReadFile("examples/bdd_policies/pii_protection.feature")
if err != nil {
    panic(err)
}

err = policyEngine.LoadGherkinPolicy(context.Background(), string(content))
if err != nil {
    panic(err)
}
```

### Testing Enforcement

```go
import "github.com/duynguyendang/manglekit/core"

// Create an envelope with PII label
input := core.NewEnvelope(map[string]string{"user": "john@example.com"})
input.AddLabel("pii")

// Test enforcement
err := policyEngine.Assess(
    context.Background(),
    core.ActionMetadata{Name: "llm_generate"},
    input,
)

if core.IsAlignmentError(err) {
    // Policy violation detected
    alignErr := err.(*core.AlignmentError)
    fmt.Println("Blocked:", alignErr.Message)
}
```

## Gherkin Syntax

### Supported Step Patterns

#### Given (Preconditions)
- `Given the user has "{label}" label`
- `Given the entity is labeled "{label}"`
- `Given the metadata "{key}" is "{value}"`

#### When (Action Triggers)
- `When calling "{action_name}"`
- `When calling the action "{action_name}"`

#### Then (Outcomes)
- `Then halt with "{reason}"` - Block the request
- `Then retry with "{feedback}"` - Request correction
- `Then route to "{target}"` - Route to specific service
- `Then allow the request` - Explicitly allow

### Example Scenario

```gherkin
Feature: My Policy
  Policy description

  Scenario: Descriptive name
    Given the user has "admin" label
    When calling "sensitive_operation"
    Then route to "admin_service"
```

## Generated Datalog

Each Gherkin scenario is compiled to a Datalog rule. For example:

**Gherkin:**
```gherkin
Scenario: Block PII to LLM
  Given the user has "pii" label
  When calling "llm_generate"
  Then halt with "PII leakage detected"
```

**Compiled Datalog:**
```datalog
halt(Req, "PII leakage detected") :-
    action_operation(Req, "llm_generate"),
    label("pii").
```

## Best Practices

1. **Use descriptive scenario names** - They appear in logs and error messages
2. **Group related scenarios** - Use features to organize related policies
3. **Test incrementally** - Start with simple scenarios and add complexity
4. **Document intent** - Use feature descriptions to explain policy goals
5. **Combine with Datalog** - Use Gherkin for common patterns, Datalog for complex logic

## See Also

- [BDD Policy Blueprint Design](../../docs/designs/bdd_policy_blueprint.md)
- [Manglekit Documentation](../../README.md)
