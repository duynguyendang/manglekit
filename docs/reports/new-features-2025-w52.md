# New Features Report - Week 52, 2025

> **Report Date:** 2026-01-02
> **Coverage Period:** 2025-12-26 to 2026-01-01
> **Status:** ✅ Complete

---

## Executive Summary

This report documents the major new features introduced to Manglekit during Week 52 of 2025. The week saw significant enhancements across policy definition, state management, access control, and developer experience. Four major features were delivered:

1. **Gherkin-based Policy Engine** - BDD-style policy definitions
2. **Durable State Manager v2.0** - Session state checkpointing and hydration
3. **Enhanced Access Control** - Transitive relationships and security labels
4. **Multi-file Policy Loading** - Support for modular policy organization

---

## 1. Gherkin-based Policy Engine

**Commit:** `8b7d094` (2025-12-31)
**Status:** ✅ Implemented

### Overview

A new Behavior-Driven Development (BDD) approach to defining Manglekit governance policies using Gherkin syntax. This feature enables non-technical stakeholders to define policies in natural language that compile to Datalog.

### Key Components

| Component | Location | Description |
|-----------|-----------|-------------|
| **Gherkin Parser** | `internal/engine/parse/gherkin.go` | Parses `.feature` files and extracts scenarios |
| **Gherkin Compiler** | `internal/engine/compiler.go` | Translates scenarios to Datalog rules |
| **Step Definitions** | `internal/engine/stepdefs.go` | Maps Gherkin steps to Datalog predicates |
| **Integration Tests** | `internal/engine/gherkin_integration_test.go` | End-to-end policy enforcement tests |

### Feature Highlights

#### Supported Gherkin Keywords

| Keyword | Mapping | Datalog Equivalent |
|---------|---------|-------------------|
| **Feature** | Policy Set | Logical grouping of rules |
| **Scenario** | Rule/Clause | Single Datalog clause |
| **Given** | Preconditions | Fact queries (labels, metadata) |
| **When** | Trigger | `action_operation(Entity, Name)` |
| **Then** | Outcome | `halt(R)`, `retry(H)`, or `route(T)` |

#### Example Policy

**Gherkin Input:**
```gherkin
Feature: PII Protection
  Prevents personally identifiable information leakage

  Scenario: Block PII to LLM
    Given the user has "pii" label
    When calling "llm_generate"
    Then halt with "PII leakage detected"
```

**Generated Datalog:**
```datalog
halt(Req, "PII leakage detected") :-
    action_operation(Req, "llm_generate"),
    label("pii").
```

### Example Policies Included

1. **PII Protection** (`examples/bdd_policies/pii_protection.feature`)
   - Block LLM calls with PII labels
   - Block LLM calls with sensitive labels
   - Allow LLM calls with public data

2. **Access Control** (`examples/bdd_policies/access_control.feature`)
   - Admin access to sensitive operations
   - User permission restrictions
   - Guest read-only access

3. **Data Governance** (`examples/bdd_policies/data_governance.feature`)
   - Cross-border data transfer prevention
   - Encryption requirements
   - High-risk operation auditing

### API Usage

```go
// Load Gherkin policy
content, err := os.ReadFile("policy.feature")
if err != nil {
    panic(err)
}

err = policyEngine.LoadGherkinPolicy(ctx, string(content))
```

### Testing Coverage

- ✅ Gherkin parsing tests (`gherkin_test.go`)
- ✅ Step definition matching tests (`stepdefs_test.go`)
- ✅ Compiler tests (`compiler_test.go`)
- ✅ Integration tests (`gherkin_integration_test.go`)

---

## 2. Durable State Manager v2.0

**Commit:** `4afe6fd` (2025-12-31)
**Design Doc:** `docs/designs/durable_state_manager.md`
**Status:** ✅ Implemented

### Overview

A new orchestration layer that enhances Manglekit's resilience by providing checkpointing and hydration capabilities for session state. This enables recovery from process failures without losing execution context.

### Architecture

The Durable State Manager follows a **Layered Approach**:

```
┌─────────────────────────────────────────┐
│  Durable State Manager (Logic Layer)    │  ← Decides WHAT/WHEN to save
├─────────────────────────────────────────┤
│  core.StateProvider (Storage Layer)     │  ← Handles physical I/O
└─────────────────────────────────────────┘
```

### Key Components

| Component | Location | Description |
|-----------|-----------|-------------|
| **SessionState** | `core/session_state.go` | Serializable state container |
| **DurableStateManager** | `internal/statemanager/manager.go` | State orchestration logic |
| **Client Integration** | `sdk/client.go`, `sdk/loop.go` | Checkpointing in RunLoop |

### SessionState Structure

```go
type SessionState struct {
    SessionID       string            // Unique thread identifier
    ActiveEnvelope  *core.Envelope    // Current payload + metadata
    ExecutionCtx    *ExecutionParams  // Retry count, history
    LogicalFacts    []string          // Datalog facts from reflection
}
```

### Checkpointing Lifecycle

1. **Hydration (Load)** - At RunLoop start, retrieve `SessionState` and reconstruct envelope
2. **Governance Execution** - Action undergoes Assess → Execute → Reflect
3. **Atomic Checkpoint (Save)** - Only after successful Reflect, persist state
4. **Continuity** - Loop continues with safe progress on disk

### Semantic Recovery

The manager performs **Semantic Re-constitution** on process restart:

- **Type Reconstruction** - Unmarshals payload using `Envelope.ContentType`
- **Engine Priming** - Re-injects `LogicalFacts` into Mangle runtime
- **Feedback Alignment** - Restores `FeedbackHistory` for LLM context

### Client Options

```go
client, err := sdk.NewClient(
    sdk.WithStateProvider(stateProvider),
    sdk.WithCheckpointing(true),
)
```

### Testing Coverage

- ✅ Session state serialization tests (`session_state_test.go`)
- ✅ Manager checkpointing tests (`manager_test.go`)
- ✅ Hydration/recovery tests (`manager_test.go`)

---

## 3. Enhanced Access Control

**Commit:** `51a744a` (2026-01-01)
**Status:** ✅ Implemented

### Overview

Significant enhancement to the access control system with support for transitive relationships, security labels, and hierarchical document structures.

### New Capabilities

#### 1. Transitive Relationships

Support for indirect access through group membership and project ownership:

```nq
# User belongs to group
<user:alice> <has:group> <group:admin> .

# Group has permission
<group:admin> <has:permission> <permission:write> .

# Transitive: Alice has write permission
```

#### 2. Security Labels

Multi-level security classification with propagation:

```go
// Security levels: public, internal, confidential, secret
input.AddLabel("confidential")
input.AddLabel("pii")
```

#### 3. Document Nesting

Hierarchical document structures with inherited access:

```nq
# Parent-child relationships
<doc:report> <contains> <doc:appendix> .
<doc:report> <has:security> <level:confidential> .
```

### Policy Enhancements

Updated `examples/hybrid_rag/policy.dl` with:

- Complex transitive access control rules
- Security label propagation logic
- PII detection scenarios
- Document hierarchy traversal

### Example Scenarios

| Scenario | Description |
|----------|-------------|
| **Admin Override** | Admins can bypass restrictions |
| **Group Inheritance** | Users inherit group permissions |
| **Label Escalation** | Security labels propagate to children |
| **PII Blocking** | PII data requires special handling |

---

## 4. Multi-file Policy Loading

**Commit:** `3b641d1` (2025-12-26)
**Status:** ✅ Implemented

### Overview

Support for loading multiple policy files simultaneously, enabling modular policy organization and better separation of concerns.

### Changes

#### SDK API

```go
// Before: Single file
client.LoadPolicy(ctx, "policy.dl")

// After: Multiple files
client.LoadPolicy(ctx, "base.dl", "access.dl", "data.dl")
```

#### CLI Enhancement

```bash
# Multiple -p flags supported
mkit eval -p base.dl -p access.dl -p data.dl
```

### Benefits

- **Modularity** - Organize policies by domain (access, data, compliance)
- **Reusability** - Share common rule sets across projects
- **Maintainability** - Smaller, focused files are easier to manage
- **Cross-file Resolution** - Engine correctly resolves predicates across files

### Implementation Details

- Updated `sdk.Client.LoadPolicy` to accept variadic file paths
- Concatenates multiple files into a single policy string
- Custom `stringSlice` flag type for CLI
- Verified cross-file predicate resolution

---

## 5. Additional Improvements

### 5.1 Architecture Documentation Updates

**Commit:** `5950869` (2025-12-31)

- Updated README.md with improved architecture description
- Enhanced error handling terminology in documentation
- Updated supervisor error messages for clarity

### 5.2 Code Quality Improvements

**Commit:** `5950869` (2025-12-31)

- Removed unused test file (`adapters/memory/inmem_hybrid_test.go`)
- Updated error wrapping in `core/errors.go`
- Improved supervisor error handling consistency

---

## Feature Impact Matrix

| Feature | Impact Area | Complexity | Lines Added | Test Coverage |
|---------|-------------|------------|--------------|---------------|
| Gherkin Policy Engine | Developer Experience | High | ~1,350 | ✅ Comprehensive |
| Durable State Manager | Resilience | Medium | ~1,020 | ✅ Comprehensive |
| Enhanced Access Control | Security | Medium | ~200 | ✅ Example-based |
| Multi-file Loading | Developer Experience | Low | ~50 | ✅ Verified |

---

## Documentation Updates

| Document | Status | Changes |
|----------|--------|---------|
| `docs/CONTEXT.md` | ✅ Updated | Architecture snapshot, new contracts |
| `docs/designs/bdd_policy_blueprint.md` | ✅ New | Gherkin integration HLD |
| `docs/designs/durable_state_manager.md` | ✅ New | State manager v2.0 HLD |
| `examples/bdd_policies/README.md` | ✅ New | BDD policy usage guide |
| `examples/hybrid_rag/policy.dl` | ✅ Updated | Enhanced access control rules |

---

## Breaking Changes

**None.** All new features are additive and backward compatible.

---

## Migration Guide

### For Existing Users

No migration required. All new features are opt-in:

1. **Gherkin Policies** - Continue using `.dl` files or adopt `.feature` files
2. **Durable State** - Enable via `WithStateProvider` option
3. **Multi-file Loading** - Use variadic `LoadPolicy` when needed
4. **Enhanced Access Control** - Update policies to use new predicates

### New Project Setup

```go
// Example: Full feature stack
client, err := sdk.NewClient(
    sdk.WithLLM(llm),
    sdk.WithStateProvider(stateProvider),
    sdk.WithCheckpointing(true),
)

// Load modular policies
err = client.LoadPolicy(
    ctx,
    "policies/base.dl",
    "policies/access.dl",
    "policies/bdd_policies/pii_protection.feature",
)
```

---

## Future Considerations

### Potential Enhancements

1. **Gherkin IDE Plugin** - VS Code extension for `.feature` files
2. **State Provider Backends** - Redis, PostgreSQL, S3 support
3. **Policy Testing Framework** - Automated policy validation
4. **Access Control UI** - Visual policy management interface

### Known Limitations

1. **Gherkin Step Extensibility** - Custom step definitions require code changes
2. **State Serialization** - Complex nested structures may need custom marshaling
3. **Transitive Query Performance** - Deep hierarchies may impact evaluation speed

---

## Conclusion

Week 52 of 2025 was a highly productive period for Manglekit, delivering four major features that significantly enhance the platform's capabilities:

- **Gherkin-based Policy Engine** makes policy definition accessible to non-technical stakeholders
- **Durable State Manager** provides production-grade resilience with checkpointing
- **Enhanced Access Control** supports complex enterprise security requirements
- **Multi-file Policy Loading** improves policy organization and maintainability

All features include comprehensive test coverage, documentation, and examples. No breaking changes were introduced, ensuring smooth adoption for existing users.

---

**Report Generated:** 2026-01-02
**Next Review:** 2026-01-09
