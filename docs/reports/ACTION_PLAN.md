# ADR R14 Compliance Scan - Action Plan

**Date:** November 8, 2025  
**Scan Status:** ✅ Complete  
**Finding:** Documentation inaccuracy (code is compliant)

---

## Executive Summary

The comprehensive scan of all 10 component handlers confirms **100% compliance with ADR R14**. However, the COMPREHENSIVE_EVALUATION.md report contains an inaccurate statement that should be corrected.

**No code changes are required.** Only documentation updates are needed.

---

## Issues Identified

### Issue #1: Inaccurate Statement in COMPREHENSIVE_EVALUATION.md

**Location:** `docs/reports/COMPREHENSIVE_EVALUATION.md:113`

**Current Text:**
```markdown
**Weaknesses:**
- Some providers still accept generic `diapi.Builder` instead of typed deps 
  (partially resolved per ADR R14)
```

**Problem:** This statement is factually incorrect. All providers receive typed dependency structs, not the builder.

**Impact:** Misleads readers about the actual compliance status of the codebase.

**Severity:** Medium (documentation only, no code impact)

---

## Action Items

### Action 1: Update COMPREHENSIVE_EVALUATION.md (Priority 1)

**File:** `docs/reports/COMPREHENSIVE_EVALUATION.md`

**Changes Required:**

#### 1.1 Update Line 113 (Section 1.3 Weaknesses)

**Current:**
```markdown
**Weaknesses:**
- Some providers still accept generic `diapi.Builder` instead of typed deps (partially resolved per ADR R14)
- Limited documentation on extending DI for custom component kinds
- No validation that all required dependencies are satisfied before factory invocation
```

**Replace With:**
```markdown
**Strengths:**
- ✅ All providers receive typed dependency structs (full ADR R14 compliance)
- ✅ Handlers correctly resolve dependencies from diapi.Builder interface
- ✅ Factories never receive the builder directly
- ✅ Type-safe DI eliminates entire classes of runtime errors
- ✅ No runtime type assertions in factory implementations

**Weaknesses:**
- Limited documentation on extending DI for custom component kinds
- No validation that all required dependencies are satisfied before factory invocation
```

#### 1.2 Update Section 1.3 Title and Rating

**Current:**
```markdown
### 1.3 Dependency Injection System

**Rating: ⭐⭐⭐⭐ (Very Good)**
```

**Replace With:**
```markdown
### 1.3 Dependency Injection System

**Rating: ⭐⭐⭐⭐⭐ (Excellent)**
```

#### 1.3 Add Detailed Explanation

**Add after the "Strengths" section:**

```markdown
**Detailed Compliance Analysis:**

The DI system achieves full ADR R14 compliance through a consistent pattern:

1. **Handlers accept `diapi.Builder` interface** (correct - handlers are external clients)
   - All 10 handlers type-assert `builderDI` to `diapi.Builder`
   - This is the correct pattern for dependency resolution

2. **Handlers resolve specific dependencies from builder**
   - Handlers call typed methods: `b.GetRetriever()`, `b.GetEmbedder()`, etc.
   - No implicit or magical dependency injection

3. **Handlers construct typed dependency structs**
   - Each handler creates a specific `diapi.*Deps` struct
   - Examples: `LLMDeps`, `RerankerDeps`, `DenseRetrieverDeps`

4. **Factories receive ONLY typed dependencies**
   - Factories never receive `diapi.Builder` directly
   - Factories have compile-time type safety
   - No runtime type assertions needed in factory code

**Evidence:**
- All 10 handlers in `internal/providers/*` and `pipeline/*`
- Typed dependency structs in [`core/diapi/di.go`](core/diapi/di.go)
- Factory signatures accept typed deps, not Builder
- See detailed analysis in [`docs/reports/DI_COMPLIANCE_SCAN.md`](reports/DI_COMPLIANCE_SCAN.md)
```

#### 1.4 Update Section 4.2 Architecture Rules Table

**Current:**
```markdown
| R14 | No Builder in factories | ✅ Enforced |
```

**Add Note:**
```markdown
| R14 | No Builder in factories | ✅ Enforced | All 10 handlers verified compliant |
```

---

### Action 2: Create Cross-Reference in CONTEXT.md (Priority 2)

**File:** `docs/CONTEXT.md`

**Location:** Known Gaps section

**Add Entry:**
```markdown
### GAP-001: Builder Leaking into Handler (ADR 7 / R14) — ✅ RESOLVED

**Status:** Verified Compliant (November 8, 2025)

**Details:** Comprehensive scan of all 10 component handlers confirms 100% compliance with ADR R14. All handlers correctly:
- Type-assert `builderDI` to `diapi.Builder` interface
- Resolve specific dependencies from builder
- Construct typed `diapi.*Deps` structs
- Pass typed deps to factories (never the builder)

**Evidence:** See [`docs/reports/DI_COMPLIANCE_SCAN.md`](reports/DI_COMPLIANCE_SCAN.md)

**Note:** The COMPREHENSIVE_EVALUATION.md report (line 113) contained an inaccurate statement about this issue. It has been corrected.
```

---

### Action 3: Add DI Pattern Documentation (Priority 2)

**File:** `CONTRIBUTING.md` (create if doesn't exist)

**Add Section:**
```markdown
## Dependency Injection Pattern

When implementing a new provider, follow this pattern:

### 1. Define Typed Dependency Struct

```go
// In core/diapi/di.go
type MyProviderDeps struct {
    CoreDeps core.CoreDeps
    // Add other dependencies as needed
    SomeDependency SomeType
}
```

### 2. Implement Handler

```go
// In internal/providers/myprovider/handler.go
func (h *Handler) BuildComponent(
    ctx context.Context,
    builderDI any,
    factory any,
    resolved *core.Resolved,
    cfg core.ProviderOptions,
    name string,
) (core.ResourceCloser, error) {
    // Type-assert to diapi.Builder interface
    b, ok := builderDI.(diapi.Builder)
    if !ok {
        return nil, fmt.Errorf("invalid builder DI type")
    }
    
    // Resolve dependencies from builder
    someDep, err := b.GetSomeDependency(name)
    if err != nil {
        return nil, err
    }
    
    // Construct typed dependency struct
    deps := diapi.MyProviderDeps{
        CoreDeps:       b.GetCoreDeps(),
        SomeDependency: someDep,
    }
    
    // Pass typed deps to factory (NOT builder)
    built, err := f.Build(ctx, deps, cfg)
    if err != nil {
        return nil, err
    }
    
    // Type-assert result and store
    provider, ok := built.(MyProvider)
    if !ok {
        return nil, fmt.Errorf("invalid provider type")
    }
    resolved.MyProviders[name] = provider
    return core.NopCloser, nil
}
```

### 3. Implement Factory

```go
// In internal/providers/myprovider/factory.go
func NewMyProvider(
    ctx context.Context,
    deps diapi.MyProviderDeps,  // ✅ Typed deps, not Builder
    cfg MyProviderOptions,
) (MyProvider, error) {
    // Use deps.CoreDeps, deps.SomeDependency
    // Never receive the builder
    return &myProvider{...}, nil
}
```

### Key Rules

- ✅ Handlers accept `diapi.Builder` interface
- ✅ Handlers resolve specific dependencies from builder
- ✅ Handlers construct typed `diapi.*Deps` structs
- ✅ Factories receive ONLY typed dependencies
- ❌ Factories never receive `diapi.Builder` directly
- ❌ No runtime type assertions in factory code
```

---

### Action 4: Update ADR.md (Priority 3)

**File:** `docs/ADR.md`

**Location:** Section 9 (Remediation Plan)

**Update Status:**
```markdown
1. **[COMPLETED & VERIFIED]** **Remediate all instances of "Builder Leaking into Handler" (ADR R14).** 
   Comprehensive scan on November 8, 2025 verified that all 10 component handlers are fully compliant 
   with ADR R14. All handlers correctly use the `diapi.Builder` interface to resolve dependencies 
   and pass typed `diapi.*Deps` structs to factories. No violations detected.
   
   See: [`docs/reports/DI_COMPLIANCE_SCAN.md`](reports/DI_COMPLIANCE_SCAN.md)
```

---

## Implementation Checklist

### Phase 1: Documentation Corrections (Immediate)
- [ ] Update COMPREHENSIVE_EVALUATION.md line 113 (Section 1.3)
- [ ] Update COMPREHENSIVE_EVALUATION.md section 1.3 rating to ⭐⭐⭐⭐⭐
- [ ] Add detailed compliance explanation to COMPREHENSIVE_EVALUATION.md
- [ ] Update COMPREHENSIVE_EVALUATION.md section 4.2 table note
- [ ] Add cross-reference in CONTEXT.md Known Gaps section

### Phase 2: Documentation Enhancement (Next Release)
- [ ] Add DI Pattern section to CONTRIBUTING.md
- [ ] Update ADR.md section 9 with verification note
- [ ] Create "DI Pattern Guide" document
- [ ] Add code examples to developer documentation

### Phase 3: Automation (Future)
- [ ] Add static analysis rule to verify factories don't accept `diapi.Builder`
- [ ] Add linter rule to check handler patterns
- [ ] Include in CI/CD pipeline
- [ ] Add automated compliance tests

---

## Files to Modify

| File | Changes | Priority |
|------|---------|----------|
| `docs/reports/COMPREHENSIVE_EVALUATION.md` | Update lines 113, rating, add explanation | 1 |
| `docs/CONTEXT.md` | Add cross-reference in Known Gaps | 2 |
| `CONTRIBUTING.md` | Add DI Pattern section | 2 |
| `docs/ADR.md` | Update section 9 with verification | 3 |

---

## Verification Steps

After making changes:

1. ✅ Verify COMPREHENSIVE_EVALUATION.md reads correctly
2. ✅ Verify cross-references work
3. ✅ Verify code examples in CONTRIBUTING.md are accurate
4. ✅ Run any existing linters/checks
5. ✅ Review changes for consistency with other docs

---

## Related Documents

- [`docs/reports/DI_COMPLIANCE_SCAN.md`](DI_COMPLIANCE_SCAN.md) - Detailed handler analysis
- [`docs/reports/ADR_R14_FINDINGS.md`](ADR_R14_FINDINGS.md) - Findings and recommendations
- [`docs/reports/SCAN_SUMMARY.md`](SCAN_SUMMARY.md) - Executive summary
- [`core/diapi/di.go`](../../core/diapi/di.go) - Typed dependency definitions
- [`docs/ADR.md`](../ADR.md) - Architecture Decision Records

---

## Timeline

- **Immediate (This Week):** Complete Phase 1 documentation corrections
- **Next Release:** Complete Phase 2 documentation enhancements
- **Future:** Implement Phase 3 automation

---

## Success Criteria

✅ All documentation accurately reflects ADR R14 compliance status  
✅ Developers have clear guidance on DI pattern  
✅ New providers follow typed DI pattern  
✅ No future violations of ADR R14  

---

**Prepared By:** Architecture Scan (November 8, 2025)  
**Status:** Ready for Implementation  
**Estimated Effort:** 2-3 hours for Phase 1, 4-6 hours for Phase 2
