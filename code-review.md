# Manglekit Core Code Review

> **Date:** 2025-12-19  
> **Scope:** `core`, `sdk`, `adapters`, `providers`, `internal/supervisor`

---

## Summary

This review identifies **code smells** and areas for improvement across Manglekit's core codebase. Issues are categorized by severity and organized by component.

| Severity | Count | Description |
|----------|-------|-------------|
| 🔴 High | 5 | Architectural issues, potential bugs |
| 🟠 Medium | 8 | Maintainability concerns, duplication |
| 🟡 Low | 6 | Minor style issues, cleanup opportunities |

---

## 🔴 High Priority

### 1. Duplicate Fields in `sdk/client.go`

**Location:** [client.go:42-45](file:///mnt/e/manglekit-wip/sdk/client.go#L42-45)

```go
// defaultLLM is the plugged-in text generation backend.
defaultLLM core.TextGenerator
// llm is alias for defaultLLM to maintain internal compatibility.
llm core.TextGenerator
```

**Issue:** Two fields (`llm` and `defaultLLM`) serve the same purpose with sync logic in `NewClient`:
```go
if c.llm != nil && c.defaultLLM == nil {
    c.defaultLLM = c.llm
} else if c.defaultLLM != nil && c.llm == nil {
    c.llm = c.defaultLLM
}
```

**Impact:** Confusion, potential inconsistency, extra maintenance burden.

**Recommendation:** Consolidate to a single field `llm` and remove synchronization logic.

---

### 2. Long Method: `ExecuteSingleStep` in `sdk/loop.go`

**Location:** [loop.go:109-264](file:///mnt/e/manglekit-wip/sdk/loop.go#L109-264)

**Issue:** 155 lines handling 8 distinct phases:
1. Action resolution
2. Context injection (feedback, history, RAG, metadata, facts)
3. Blueprint check
4. Error handling
5. History update
6. Persistence
7. Decision evaluation
8. Memorization

**Impact:** Hard to test, understand, and maintain. Violates Single Responsibility Principle.

**Recommendation:** Extract phases into private helper methods:
```go
func (c *Client) injectContext(ctx context.Context, env *core.Envelope, params *ExecutionParams) {}
func (c *Client) handleDecision(result core.Envelope, params *ExecutionParams) error {}
func (c *Client) persistHistory(ctx context.Context, params *ExecutionParams) error {}
```

---

### 3. Long Method: `executeInternal` in `internal/supervisor/supervisor.go`

**Location:** [supervisor.go:201-325](file:///mnt/e/manglekit-wip/internal/supervisor/supervisor.go#L201-325)

**Issue:** 124 lines handling 8 lifecycle phases. Similar to `ExecuteSingleStep`.

**Recommendation:** Extract into smaller focused methods for each phase.

---

### 4. Type Assertion Fragility in `sdk/planner.go`

**Location:** [planner.go:53-59](file:///mnt/e/manglekit-wip/sdk/planner.go#L53-59)

```go
type Queryable interface {
    Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error)
}

queryable, ok := c.engine.(Queryable)
if !ok {
    return nil, fmt.Errorf("engine does not support querying")
}
```

**Issue:** Runtime type assertion instead of interface contract. `Query` is not part of `core.Evaluator`.

**Recommendation:** Add `Query` to `core.Evaluator` interface or create a separate planning interface.

---

### 5. Weak Document ID in `sdk/memory_orchestrator.go`

**Location:** [memory_orchestrator.go:103](file:///mnt/e/manglekit-wip/sdk/memory_orchestrator.go#L103)

```go
ID: fmt.Sprintf("%d", len(content)), // Simple ID, would imply hashing in real sys
```

**Issue:** Document ID based on content length will cause collisions.

**Recommendation:** Use proper UUID or content hash:
```go
ID: uuid.New().String(),
// or
ID: fmt.Sprintf("%x", sha256.Sum256([]byte(content)))[0:16],
```

---

## 🟠 Medium Priority

### 6. Magic Strings Throughout Codebase

**Locations:**
- `sdk/client.go:63`: `"closed"`
- `internal/supervisor/supervisor.go:195`: `"open"`
- `sdk/loop.go:16-18`: Default constants

**Issue:** Hardcoded strings for failure modes, decisions, etc.

**Recommendation:** Define constants:
```go
const (
    FailModeOpen   = "open"
    FailModeClosed = "closed"
)
```

---

### 7. Duplicate Comment Block in `sdk/memory_orchestrator.go`

**Location:** [memory_orchestrator.go:11-16](file:///mnt/e/manglekit-wip/sdk/memory_orchestrator.go#L11-16)

```go
// HybridMemory implements core.AgentMemory by combining:
// 1. HistoryStore (Sequential Chat Logs)
// 2. VectorStore (Semantic Search / RAG)
// HybridMemory implements core.AgentMemory by combining:
// 1. HistoryStore (Sequential Chat Logs)
// 2. VectorStore (Semantic Search / RAG)
```

**Impact:** Copy-paste error, reduces code quality perception.

**Recommendation:** Remove duplicate comment block.

---

### 8. Hardcoded Values in RAG Recall

**Location:** [memory_orchestrator.go:68](file:///mnt/e/manglekit-wip/sdk/memory_orchestrator.go#L68)

```go
docs, err := m.Vectors.Search(ctx, "default", vec, 3) // Top 3
```

**Issue:** Collection name `"default"` and `k=3` are hardcoded.

**Recommendation:** Make configurable via `HybridMemory` struct fields or options.

---

### 9. Inconsistent Nil Checks

**Location:** Multiple files

```go
// sdk/loop.go:36 - checks before use
if c.logger != nil {
    ctx = core.ContextWithLogger(ctx, c.logger)
}

// But in other places, assumes non-nil
c.logger.Warn("...")
```

**Recommendation:** Ensure `logger` is always initialized (default to `NopLogger`), then remove nil checks.

---

### 10. Similar Factory Patterns in Providers

**Location:** `providers/google/factory.go` and `providers/openai/factory.go`

Both have nearly identical structure:
1. Get global Genkit
2. Call Init
3. Create action
4. Type assert to `TextGenerator`
5. Set LLM
6. Register action

**Recommendation:** Extract shared base factory logic:
```go
func createProviderOption(initFn, modelPrefix string) sdk.ClientOption
```

---

### 11. Unused `Timeout` Field

**Location:** [sdk/options.go:268](file:///mnt/e/manglekit-wip/sdk/options.go#L268)

```go
// Timeout (unused currently) specifies the max duration for the execution.
Timeout time.Duration
```

**Recommendation:** Either implement timeout support or remove the field.

---

### 12. Commented-Out Code in Supervisor

**Location:** [supervisor.go:217-220, 273-274](file:///mnt/e/manglekit-wip/internal/supervisor/supervisor.go#L217-220)

```go
// if parentID, ok := core.GetParentID(ctx); ok {
//     // Evaluator doesn't support RecordLineage directly.
//     // g.engine.RecordLineage(ctx, input.ID.String(), parentID)
// }
```

**Recommendation:** Remove commented code or implement the feature.

---

### 13. Outcome Set Twice in Supervisor

**Location:** [supervisor.go:131-148, 162-163](file:///mnt/e/manglekit-wip/internal/supervisor/supervisor.go#L162-163)

```go
// Line 147: Already set based on decision
span.SetAttributes(map[string]any{core.AttrOutcome: core.OutcomeProceed})

// Line 163: Set again unconditionally
span.SetAttributes(map[string]any{
    core.AttrOutcome:     core.OutcomeProceed,  // DUPLICATE
    "mangle.output_id":   result.ID.String(),
})
```

**Impact:** Overwrites route/retry outcomes with `PROCEED`.

**Recommendation:** Move output_id attribute to decision switch, remove duplicate.

---

## 🟡 Low Priority

### 14. Inconsistent Error Wrapping

**Location:** Various

Some errors use `fmt.Errorf("...: %w", err)`, others don't wrap.

**Recommendation:** Consistently wrap errors for stack trace preservation.

---

### 15. Missing Interface Compile-Time Checks

**Location:** `adapters/ai/adapter.go`

```go
// Missing:
var _ core.Action = (*LLMAction)(nil)
var _ core.TextGenerator = (*LLMAction)(nil)
```

**Recommendation:** Add compile-time interface satisfaction checks.

---

### 16. Verbose Logging Patterns

**Location:** Various

```go
if c.logger != nil {
    c.logger.Info("message")
}
```

**Recommendation:** Since `NopLogger` exists, always use logger directly without nil checks.

---

### 17. String Prefix Check

**Location:** [adapters/ai/adapter.go:60](file:///mnt/e/manglekit-wip/adapters/ai/adapter.go#L60)

```go
if len(k) > len(core.PrefixPromptConfig) && k[:len(core.PrefixPromptConfig)] == core.PrefixPromptConfig
```

**Recommendation:** Use `strings.HasPrefix`:
```go
if strings.HasPrefix(k, core.PrefixPromptConfig)
```

---

### 18. Missing Godoc on Some Exported Functions

**Location:** Various small utility functions

**Recommendation:** Ensure all exported symbols have documentation.

---

### 19. Unused Import Check

**Location:** `sdk/loop.go:11`

```go
engine_memory "github.com/duynguyendang/manglekit/internal/engine/memory"
```

Only used for `VolatileStore`. Consider if this coupling is necessary.

---

## Recommendations Summary

| Priority | Action Item |
|----------|-------------|
| 🔴 | Consolidate `llm`/`defaultLLM` fields |
| 🔴 | Refactor long methods (`ExecuteSingleStep`, `executeInternal`) |
| 🔴 | Add `Query` to `core.Evaluator` interface |
| 🔴 | Fix document ID generation |
| 🟠 | Define failure mode constants |
| 🟠 | Remove duplicate comment |
| 🟠 | Make RAG parameters configurable |
| 🟠 | Extract shared provider factory logic |
| 🟠 | Fix duplicate outcome attribute |
| 🟡 | Consistent error wrapping |
| 🟡 | Add interface compile-time checks |
| 🟡 | Use `strings.HasPrefix` |

---

## Architecture Notes

The overall architecture is sound with good separation of concerns:
- `core/` defines clean interfaces
- `sdk/` provides orchestration
- `adapters/` and `providers/` handle external integrations
- `internal/supervisor/` implements the governance lifecycle

Key positive patterns observed:
- ✅ Functional options pattern
- ✅ Interface-based design
- ✅ Nop implementations for testing
- ✅ Context propagation
- ✅ Structured logging interface
