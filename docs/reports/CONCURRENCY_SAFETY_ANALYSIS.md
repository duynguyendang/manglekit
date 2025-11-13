# Concurrency Safety Analysis - Manglekit

**Date:** November 13, 2025  
**Analysis Level:** 8/10 ⚠️ Good but with identified gaps  
**Risk Assessment:** LOW-MEDIUM (non-blocking for production, but improvements recommended)

---

## Executive Summary

The Manglekit codebase demonstrates **good concurrency safety practices** with proper mutex usage in critical areas. However, the rating is **8/10 instead of 9-10** due to **5 identified concurrency gaps**:

| Gap | Severity | Location | Impact |
|-----|----------|----------|--------|
| 1. Registry not thread-safe | 🟡 MEDIUM | `registry.go` | Concurrent registration during init |
| 2. Builder maps not protected | 🟡 MEDIUM | `builder.go` | Concurrent builds could race |
| 3. Global registry access unprotected | 🟡 MEDIUM | `internal/registry/registry.go` | Race on global state |
| 4. No protection for map writes | 🟡 MEDIUM | Multiple locations | Standard Go race conditions |
| 5. Orchestrator state mutation | 🔴 CRITICAL in theory | `pipeline/sandwich` | Potential mutation if poorly used |

**Current Status:** ✅ Safe for current usage patterns (single builder, single orchestrator instance)  
**Production Risk:** ⚠️ Medium (if multi-threaded scenarios are introduced)

---

## 1. Registry Not Thread-Safe ⚠️

**Location:** `registry.go:8-26`

### Current Code:
```go
type Registry struct {
	factories         map[core.Kind]map[string]core.GenericFactory
	handlers          map[core.Kind]core.ComponentHandler
	OptionsTypeToName map[reflect.Type]string
	OptionsTypeToKind map[reflect.Type]core.Kind
}

// NO MUTEX - NOT PROTECTED
```

### Issue:
- Registry maps are **unprotected** by any mutex
- If multiple goroutines register providers concurrently, **race condition**
- Map writes in Go are not atomic

### Scenario Where This Fails:
```go
// Goroutine 1
go func() {
    manglekit.Register(reg, &googleEmbedderOpts, ...)  // Writes to registry.factories
}()

// Goroutine 2
go func() {
    manglekit.Register(reg, &openaiEmbedderOpts, ...)  // Writes to registry.factories
}()

// Result: DATA RACE ⚠️
```

### Current Mitigation:
✅ Registration only happens **during init** (single-threaded)  
✅ All providers registered before runtime starts  
⚠️ But not **enforced** - users could register during runtime

### Recommendation:
```go
// FIXED VERSION:
type Registry struct {
	factories         map[core.Kind]map[string]core.GenericFactory
	handlers          map[core.Kind]core.ComponentHandler
	OptionsTypeToName map[reflect.Type]string
	OptionsTypeToKind map[reflect.Type]core.Kind
	mu                sync.RWMutex  // ✅ ADD THIS
}

func (r *Registry) Register[T any, D any, O core.ProviderOptions](
	optsSample O,
	fn func(ctx context.Context, deps D, cfg O) (T, error),
) error {
	r.mu.Lock()  // ✅ Protect map writes
	defer r.mu.Unlock()
	
	// ... existing logic
}

func (r *Registry) Get(kind core.Kind, name string) (core.GenericFactory, error) {
	r.mu.RLock()  // ✅ Protect map reads
	defer r.mu.RUnlock()
	
	// ... existing logic
}
```

**Effort:** 30-45 minutes  
**Risk:** None (backward compatible)  
**Benefit:** Safe concurrent registration

---

## 2. Builder Component Maps Not Protected ⚠️

**Location:** `builder.go:28-42`

### Current Code:
```go
type builder struct {
	// ... other fields ...
	embedders      map[string]ai.Embedder      // NO MUTEX
	vectorStores   map[string]core.VectorStore // NO MUTEX
	retrievers     map[string]core.Retriever   // NO MUTEX
	rerankers      map[string]core.Reranker    // NO MUTEX
	rules          map[string]core.RuleSet     // NO MUTEX
	llms           map[string]core.LLMClient   // NO MUTEX
	stateProviders map[string]core.StateProvider // NO MUTEX
	// ... 8 more maps
}
```

### Issue:
- Builder is created **per orchestrator** (not global)
- But if builder is shared across goroutines, **race condition on map access**
- Both reads (`GetEmbedder`) and writes (`addEmbedder`) unprotected

### Current Mitigation:
✅ Builder is typically **not shared** across goroutines  
✅ Each request creates new builder → orchestrator  
⚠️ But the code doesn't enforce single-builder-per-thread pattern

### Scenario Where This Fails:
```go
// If someone tries to share builder:
builder := manglekit.NewBuilder(...)

// Goroutine 1: Building components
go func() {
    handler.BuildEmbedder(ctx, builder, cfg)  // Writes to builder.embedders map
}()

// Goroutine 2: Accessing components
go func() {
    handler.BuildRetriever(ctx, builder, cfg)  // Reads/writes to builder.retrievers map
}()

// Result: DATA RACE ⚠️
```

### Recommendation:
Since builders are typically single-threaded, **document this requirement clearly**:

```go
// In builder.go - ADD COMMENT:
//
// builder is NOT thread-safe and should only be used from a single goroutine.
// For concurrent setup, create separate builder instances per goroutine.
// All component maps (embedders, retrievers, etc.) are unprotected.
type builder struct {
	// ... fields ...
}

// Alternative: If concurrency needed, protect maps:
type builder struct {
	// ... fields ...
	mu sync.RWMutex  // Protects all component maps
	embedders      map[string]ai.Embedder
	// ... etc
}

func (b *builder) AddEmbedder(name string, emb ai.Embedder) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.embedders[name] = emb
}

func (b *builder) GetEmbedder(name string) (ai.Embedder, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return getComponent(b.embedders, name)
}
```

**Effort:** 1-2 hours (if adding mutex) OR 30 minutes (if just documenting)  
**Risk:** Low (builders already single-threaded in practice)  
**Benefit:** Prevent accidental misuse

---

## 3. Global Registry Access Unprotected ⚠️

**Location:** `internal/registry/registry.go:11-28`

### Current Code:
```go
var (
	globalRegistry *manglekit.Registry  // Global variable
	mu             sync.Mutex           // Exists but NOT USED for Get!
)

func init() {
	globalRegistry = manglekit.NewRegistry()
}

func resetLocked() {
	mu.Lock()
	defer mu.Unlock()
	globalRegistry = manglekit.NewRegistry()  // ✅ Protected
}

func Global() *manglekit.Registry {
	return globalRegistry  // ⚠️ NOT PROTECTED - Returns pointer
}
```

### Issue:
- `Global()` returns **unprotected pointer** to global registry
- Callers can access registry concurrently without locking
- The mutex exists but isn't used for `Get()` calls
- **Example race condition:**

```go
// Thread 1: Resetting registry
go func() {
    internal_registry.resetLocked()  // Takes lock, replaces globalRegistry
}()

// Thread 2: Reading from registry
go func() {
    reg := internal_registry.Global()  // ⚠️ NO LOCK
    factory := reg.Get("llm", "openai")  // May be reading from old/new registry
}()
```

### Current Mitigation:
✅ `resetLocked()` **is** protected (test hooks only)  
✅ In production, registry only reset once during init  
⚠️ But reads via `Global()` are unprotected

### Recommendation:

**Option A: Document the threading model clearly**
```go
// Global() returns the global registry. The registry itself is thread-safe
// for concurrent reads. Writes (registration) must happen before runtime
// and are protected by the mutex in internal/registry via resetLocked().
func Global() *manglekit.Registry {
	return globalRegistry
}
```

**Option B: Make registry fully thread-safe (preferred)**
```go
// Ensure Registry has mutex (from fix #1)
// Then Global() is safe because Registry.Get() is protected

func Global() *manglekit.Registry {
	// Registry internally handles locking
	return globalRegistry
}

// Registry.Get() now has RWMutex protection from fix #1
```

**Effort:** 15 minutes (Option A) OR 30 minutes (Option B)  
**Risk:** Low  
**Benefit:** Clear concurrency model

---

## 4. No Protection for Map Reads/Writes in Multiple Locations ⚠️

### Locations with Unprotected Maps:

**1. InMemory State Provider** (`internal/providers/state/inmemory/provider.go`)
```go
type Provider struct {
	mu     sync.RWMutex  // ✅ HAS MUTEX
	data   map[string]interface{}  // ✅ PROTECTED
	closed bool
}

func (p *Provider) Get(ctx context.Context, sessionID string) (interface{}, error) {
	p.mu.RLock()  // ✅ GOOD
	defer p.mu.RUnlock()
	return p.data[sessionID], nil
}
```
**Status:** ✅ **SAFE** - Properly locked

---

**2. InMemory Retriever** (`internal/providers/retrievers/inmemory/inmemory.go`)
```go
type InMemoryRetriever struct {
	mu   sync.RWMutex  // ✅ HAS MUTEX
	docs map[string]core.Doc  // ✅ PROTECTED
}

func (r *InMemoryRetriever) Retrieve(...) {
	r.mu.RLock()  // ✅ GOOD
	defer r.mu.RUnlock()
	// ... search docs
}
```
**Status:** ✅ **SAFE** - Properly locked

---

**3. Config Types** (`config/types.go`)
```go
type Config struct {
	Components []ComponentConfig  // Array - immutable after parse
	TopK       int
	// ... no maps
}
```
**Status:** ✅ **SAFE** - No mutable maps

---

**4. Template Cache** (`llm/prompt.go:48`)
```go
type PromptBuilder struct {
	templateCache map[string]*template.Template  // ⚠️ IS PROTECTED
	mu            sync.RWMutex  // ✅ HAS MUTEX
}

func (pb *PromptBuilder) Build(userTemplate string, data map[string]any) (string, error) {
	// 1. Check cache with read lock
	pb.mu.RLock()  // ✅ GOOD
	tmpl, found := pb.templateCache[userTemplate]
	pb.mu.RUnlock()

	if !found {
		// 2. Double-check with write lock (pattern: check-then-act)
		pb.mu.Lock()  // ✅ GOOD
		tmpl, found = pb.templateCache[userTemplate]
		if !found {
			tmpl, _ = template.New("rag").Parse(userTemplate)
			pb.templateCache[userTemplate] = tmpl
		}
		pb.mu.Unlock()
	}
	
	// 3. Execute (outside lock - smart!)
	var buf bytes.Buffer
	tmpl.Execute(&buf, data)  // ✅ No lock needed - template.Template is thread-safe
	return buf.String(), nil
}
```
**Status:** ✅ **SAFE** - Properly protected with double-check pattern

---

**5. Mock Provider** (`internal/testproviders/mock/mock.go:202`)
```go
type MockProvider struct {
	mu        sync.Mutex  // ✅ HAS MUTEX
	lastParams map[string]interface{}  // ✅ PROTECTED
}

func (m *MockProvider) GetLastParams() map[string]interface{} {
	m.mu.Lock()  // ✅ GOOD
	defer m.mu.Unlock()
	// Return a copy to avoid race conditions on the map
	result := make(map[string]interface{})
	for k, v := range m.lastParams {
		result[k] = v
	}
	return result
}
```
**Status:** ✅ **SAFE** - Properly protected; returns copy

---

## 5. Goroutine Safety in Reranker ✅ (Already Good)

**Location:** `internal/providers/rerank/cosine/cosine.go:106-122`

### Current Code:
```go
// Embed all documents in parallel
docVectors := make([][]float32, len(req.Docs))
g, gCtx := errgroup.WithContext(ctx)

for i, doc := range req.Docs {
	i, doc := i, doc  // ✅ GOOD: Loop variable capture pattern
	g.Go(func() error {
		docEmbedResp, err := r.embedder.Embed(gCtx, docEmbedReq)
		docVectors[i] = docEmbedResp.Embeddings[0].Embedding  // ✅ Safe: unique index
		return nil
	})
}

if err := g.Wait(); err != nil {  // ✅ Wait for all goroutines
	return nil, fmt.Errorf("cosine: one or more goroutines failed: %w", err)
}
```

**Status:** ✅ **SAFE**
- ✅ Correct loop variable capture
- ✅ Each goroutine writes to unique index (no race)
- ✅ Proper error handling with `errgroup`
- ✅ Wait for completion before proceeding

---

## 6. Orchestrator State - Good Design ✅

**Location:** `pipeline/sandwich/sandwich.go`

### Current Code:
```go
type Sandwich struct {
	retriever     core.Retriever       // ✅ Immutable after construction
	reranker      core.Reranker        // ✅ Immutable after construction
	llm           core.LLMClient       // ✅ Immutable after construction
	stateProvider core.StateProvider   // ✅ Immutable after construction
	obs           core.Observability   // ✅ Immutable after construction
	// ... no mutating fields
}

// Execute is called concurrently from different sessions
func (s *Sandwich) Execute(ctx context.Context, sessionID string, q core.Query) (core.Answer, error) {
	// Only reads immutable fields
	result := s.retriever.Retrieve(...)  // ✅ Call is thread-safe
	// ...
}
```

**Status:** ✅ **SAFE**
- ✅ Orchestrator fields are **immutable** after construction
- ✅ Multiple goroutines can call `Execute()` safely
- ✅ Component implementations handle their own thread-safety (e.g., InMemoryState with mutex)
- ✅ No shared mutable state in orchestrator

---

## Summary Table

| Component | Has Mutex? | Protected? | Status | Risk |
|-----------|-----------|-----------|--------|------|
| Registry | ❌ NO | ❌ NO | ⚠️ Unsafe | MEDIUM |
| Builder | ❌ NO | ❌ NO | ⚠️ Unsafe* | LOW** |
| Global Registry | ✅ YES | ⚠️ PARTIAL | ⚠️ Mixed | LOW |
| InMemory State | ✅ YES | ✅ YES | ✅ Safe | NONE |
| InMemory Retriever | ✅ YES | ✅ YES | ✅ Safe | NONE |
| Template Cache | ✅ YES | ✅ YES | ✅ Safe | NONE |
| Cosine Reranker | ✅ errgroup | ✅ YES | ✅ Safe | NONE |
| Orchestrator | ✅ Immutable | ✅ YES | ✅ Safe | NONE |

*Builder is single-threaded in practice but not enforced  
**Low risk because builders aren't shared across threads in current usage

---

## Recommendations for 9/10 Score

### Priority 1: Add Mutex to Registry (30-45 min)

```go
// registry.go - Add mutex protection
type Registry struct {
	factories         map[core.Kind]map[string]core.GenericFactory
	handlers          map[core.Kind]core.ComponentHandler
	OptionsTypeToName map[reflect.Type]string
	OptionsTypeToKind map[reflect.Type]core.Kind
	mu                sync.RWMutex  // ← ADD THIS
}

func (r *Registry) Register[T, D, O](optsSample O, fn func(...) (T, error)) error {
	r.mu.Lock()      // ← ADD LOCK
	defer r.mu.Unlock()
	// ... existing code
}

func (r *Registry) Get(kind, name) (GenericFactory, error) {
	r.mu.RLock()     // ← ADD LOCK
	defer r.mu.RUnlock()
	// ... existing code
}

func (r *Registry) RegisterHandler(handler) {
	r.mu.Lock()      // ← ADD LOCK
	defer r.mu.Unlock()
	// ... existing code
}
```

**Impact:** ✅ Registry safe for concurrent registration  
**Backward Compatibility:** ✅ 100% compatible

---

### Priority 2: Document Builder Thread-Safety (15 min)

```go
// builder.go - Add documentation
//
// builder is NOT thread-safe. Each builder instance should be used by only one goroutine.
// If concurrent setup is needed, create separate builder instances per goroutine.
//
// This is intentional for performance: no lock contention during build phase.
// After building, the resulting Orchestrator is thread-safe for concurrent Execute() calls.
type builder struct {
	// ... existing fields
}
```

**Impact:** ✅ Prevents accidental misuse  
**Backward Compatibility:** ✅ 100% compatible (doc-only)

---

### Priority 3: Ensure Registry Thread-Safety in Global Access (15 min)

```go
// internal/registry/registry.go - Clarify thread-safety contract
//
// Global() returns the global registry. The registry is thread-safe:
// - Registration (via Global().Register) is protected by internal mutex
// - Reading (via Global().Get) is protected by internal mutex
// - Safe for concurrent access from multiple goroutines
func Global() *manglekit.Registry {
	return globalRegistry  // ✅ Safe because Registry has mutex
}
```

**Impact:** ✅ Clear concurrency model  
**Backward Compatibility:** ✅ 100% compatible

---

## Test Coverage Gaps

No specific concurrency tests exist. Recommended additions:

```go
// registry_concurrent_test.go
func TestRegistry_ConcurrentRegistration(t *testing.T) {
	r := manglekit.NewRegistry()
	
	// 10 goroutines registering different providers
	for i := 0; i < 10; i++ {
		go func(idx int) {
			name := fmt.Sprintf("provider-%d", idx)
			// Register provider
		}(i)
	}
	// Wait & verify no races
}

func TestRegistry_ConcurrentReadWrite(t *testing.T) {
	r := manglekit.NewRegistry()
	
	// Some goroutines registering, others reading
	go func() {
		for i := 0; i < 100; i++ {
			r.Register(...)
		}
	}()
	
	go func() {
		for i := 0; i < 100; i++ {
			r.Get(...)
		}
	}()
}

// builder_concurrent_test.go
func TestBuilder_NotConcurrent(t *testing.T) {
	// Verify that concurrent access to builder causes issues
	// (Document as expected behavior)
}

// orchestrator_concurrent_test.go
func TestOrchestrator_ConcurrentExecute(t *testing.T) {
	// Multiple goroutines calling Execute() simultaneously
	// Should be safe because orchestrator is immutable
}
```

---

## Why 8/10 and Not Higher?

| Score | Criteria |
|-------|----------|
| 9-10/10 | ✅ All maps protected OR immutable |
|         | ✅ Clear thread-safety guarantees |
|         | ✅ Concurrent tests exist |
| **8/10** | ⚠️ Registry unprotected (fixable gap) |
|         | ⚠️ Builder not thread-safe (by design but undocumented) |
|         | ⚠️ No concurrent tests |
|         | ✅ In-use components ARE thread-safe |
|         | ✅ No actual race conditions in typical usage |

**Current state is SAFE for production** because:
- ✅ Registry only mutated during init (single-threaded)
- ✅ Builder only used per-orchestrator (not shared)
- ✅ Runtime orchestrators are immutable after construction
- ✅ Component implementations protect their own state

**But could be 9-10/10 with:**
1. Registry mutex (30 min)
2. Thread-safety documentation (15 min)
3. Concurrent tests (1-2 hours)

---

## Final Verdict

**Score: 8/10** ✅ Good, with minor improvements recommended

**Production Ready?** ✅ **YES** - Current usage patterns are safe

**Recommended Actions:**
1. ✅ Priority 1 (Registry mutex): Would improve to 9/10
2. ✅ Priority 2 (Documentation): Prevents user errors
3. ✅ Priority 3 (Tests): Ensures regressions caught

**Timeline for 10/10:** 2-3 hours total

---

**End of Analysis**
