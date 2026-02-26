# Manglekit (Sovereign Logic Kernel) — Low-Level Design (LLD)

## 1. Introduction

This Low-Level Design provides developers with exact struct definitions, memory layouts, concurrency patterns, and implementation specifications powering Manglekit v2. Synthesizing the high-performance memory-mapped layers of MEB and the Zero-Trust cognitive loops of Kronos v1, this document defines everything needed for implementation.

---

## 2. Core Primitives & Domain Types

### 2.1 Fact (Quad)

```go
package core

type Quad struct {
    Subject   string  // Source entity (e.g., "User:Bob")
    Predicate string  // Relationship (e.g., "has_role")
    Object    string  // Target value (e.g., "Admin", "42")
    Graph     string  // Namespace/Context (e.g., "global")
}
```

### 2.2 Atom (Streaming Knowledge Unit)

```go
type Atom struct {
    Predicate    string    `json:"predicate"`
    Subject      string    `json:"subject"`
    Object       string    `json:"object"`
    Weight       float64   `json:"weight"` // 1.0 (Fact) to 0.1 (Guess)
    OriginIntent IntentStr `json:"origin_intent,omitempty"`
}

// Payload is a zero-allocation stream of Atoms.
type Payload iter.Seq[Atom]
```

### 2.3 Signal (OODA Trigger)

```go
type Signal struct {
    ID         string    `json:"id"`
    Source     PortType  `json:"source"`
    Timestamp  time.Time `json:"timestamp"`
    RawContent string    `json:"raw_content"`
    Intent     IntentStr `json:"intent,omitempty"`
    IntentHint string    `json:"intent_hint,omitempty"`
    IsProposal bool      `json:"is_proposal,omitempty"`
}
```

### 2.4 Envelope (Execution Carrier)

```go
type Envelope struct {
    ID           string          // Trace ID / span identifier
    Payload      any             // Structured Arguments sent to LLM/Action
    Metadata     map[string]any  // Telemetry and routing tags
    ContextFacts []Quad          // Flattened state of the system
    Violations   []ViolationRule // GenePool axiom violations after Shadow Audit
}

type ViolationRule struct {
    RuleID      string
    Description string
    Severity    int // 0 = Halt, 1 = Retry, 2 = Warn
}
```

### 2.5 DomainGene (Crystallized Logic)

```go
type DomainGene struct {
    Name         string    `json:"name"`
    Tier         TrustTier `json:"tier"`       // TIER_0 .. TIER_3
    TierID       string    `json:"tier_id"`
    Rules        []byte    `json:"rules"`      // Compiled Datalog content
    Signature    [32]byte  `json:"signature"`  // SHA256 integrity hash
    MMapAddr     uintptr   `json:"-"`          // Zero-copy mmap pointer
    Capabilities []string  `json:"capabilities"`
    Intents      []string  `json:"intents"`
    FactPath     string    `json:"fact_path,omitempty"`
    SourcePath   string    `json:"source_path,omitempty"`
    IsUnverified bool      `json:"is_unverified"`
}
```

### 2.6 Trust Tiers (4-Level System)

```go
const (
    Tier0Kernel TrustTier = "TIER_0" // Immutable Core Axioms
    Tier1Admin  TrustTier = "TIER_1" // Human Operator / Governance
    Tier2AI     TrustTier = "TIER_2" // Induced / Learned Logic
    Tier3User   TrustTier = "TIER_3" // Untrusted External Input
)
```

### 2.7 CognitiveFrame (Epoch State)

The unit of work for a single OODA epoch.

```go
type CognitiveFrame struct {
    ID        uuid.UUID
    Timestamp time.Time
    Intent    IntentStr

    // Task Metadata (Datalog-Driven from strategy.dl)
    TaskType   TaskType   // INDUCTION, GENERATION, AUDIT, RECOVERY
    OutputType OutputType // PLAN (structured JSON) or RULE (Datalog rules)

    // Memory & Logic
    Context       []Atom       // Soft Logic (INT8) - Observed facts, pruneable
    AttentionSink []Atom       // Hard Logic (FP32) - Immutable Axioms (Tier 0), never pruned
    ActiveGenes   []DomainGene // Logic Pinning - active rules for this epoch

    // Reasoning
    Draft  interface{}  // Neural proposal: *Plan or []byte
    Proof  *AuditResult // Verification trace
    Status VerifyStatus // PENDING, FP32_PASSED, LOGIC_VIOLATION, WARNING

    // Telemetry
    TraceID        string
    SessionHistory []AuditResult // Temporal conversation trace
    EAST           EASTState     // Cognitive Pressure metrics

    // Staging
    IsProposal bool
}
```

### 2.8 EAST State (Entropic Activation Steering)

```go
type EASTState struct {
    LogicSuccess       float64 `json:"logic_success"`       // L (0.0 - 1.0)
    EntropyCoefficient float64 `json:"entropy_coefficient"` // N (Novelty)
    SteeringMagnitude  float64 `json:"steering_magnitude"`  // P = exp(1-L) / N
}

// Paradox injection triggers when P > ParadoxInjectionThreshold
const ParadoxInjectionThreshold = 0.8
```

### 2.9 Audit Result

```go
type AuditResult struct {
    Pass          bool       `json:"pass"`
    ViolationTier TrustTier  `json:"violation_tier"`
    TierID        string     `json:"tier_id"`
    ConflictPath  string     `json:"conflict_path"` // "safety.dl:42"
    ProofTree     *ProofNode `json:"proof_tree"`    // "Why" it failed
    EntropyDelta  float64    `json:"entropy_delta"` // Feedback for EAST
}
```

### 2.10 Gene Manifest (Boot-Time Registry)

```go
type GeneManifest struct {
    Version string         `yaml:"version"`
    Genes   []GeneMetadata `yaml:"genes"`
}

type GeneMetadata struct {
    Name         string    `yaml:"name"`
    Tier         TrustTier `yaml:"tier"`
    Path         string    `yaml:"path"`
    FactPath     string    `yaml:"fact_path,omitempty"`
    Signature    string    `yaml:"signature"` // SHA256 hex
    Capabilities []string  `yaml:"capabilities"`
    Intents      []string  `yaml:"intents"`
}
```

---

## 3. Storage Architecture: The Silo (MEB V2)

### 3.1 Directory Layout

```text
/manglekit/silo/
├── graph/ (BadgerDB for Quads/Content) # SyncWrites=false
├── vocab/ (BadgerDB for Dictionary)    # SyncWrites=true
└── vector/ (Hybrid)
    └── vectors.bin (Memory-Mapped flat file)
```

### 3.2 Quad Storage & Prefix Encoding

Quads are mapped to strictly aligned 33-byte `[]byte` keys.

- **Byte 0:** Prefix indicating the index type
  - `0x10` = SPOg
  - `0x11` = POSg
  - `0x12` = PSOg
  - `0x13` = GSPO
- **Bytes 1-8:** Subject ID (Big-Endian uint64)
- **Bytes 9-16:** Predicate ID (Big-Endian uint64)
- **Bytes 17-24:** Object ID (Big-Endian uint64)
- **Bytes 25-32:** Graph ID (Big-Endian uint64)

*Performance Note:* Big-Endian serialization ensures lexicographic byte-sequence sorting matches numeric ordering in BadgerDB.

**Encoding implementation:**
```go
func EncodeQuadKey(prefix byte, s, p, o, g uint64) []byte {
    key := make([]byte, 33) // QuadKeySize
    key[0] = prefix
    switch prefix {
    case 0x10: // SPOG
        binary.BigEndian.PutUint64(key[1:9], s)
        binary.BigEndian.PutUint64(key[9:17], p)
        binary.BigEndian.PutUint64(key[17:25], o)
        binary.BigEndian.PutUint64(key[25:33], g)
    case 0x11: // POSG
        binary.BigEndian.PutUint64(key[1:9], p)
        binary.BigEndian.PutUint64(key[9:17], o)
        binary.BigEndian.PutUint64(key[17:25], s)
        binary.BigEndian.PutUint64(key[25:33], g)
    case 0x13: // GSPO
        binary.BigEndian.PutUint64(key[1:9], g)
        binary.BigEndian.PutUint64(key[9:17], s)
        binary.BigEndian.PutUint64(key[17:25], p)
        binary.BigEndian.PutUint64(key[25:33], o)
    }
    return key
}
```

**System keys:**
```go
var KeyFactCount = []byte{0xFF, 0x01} // System metadata: total fact count
```

### 3.3 Vector Quantization Pipeline

1. **Float32 Generation:** Original embedding is `[]float32` (e.g., 1536-d).
2. **Matryoshka Truncation (MRL):** Truncated to the first 64 dimensions, preserving baseline semantics while dropping size by 95%.
3. **INT8 Quantization:** `float32` compressed to `int8`, mapping to `[-128, 127]` scale dynamically.
4. **Mmap + SIMD Search:** 64-byte chunks are contiguous in `vectors.bin`. AVX2/NEON SIMD dot-product scans.
5. **Re-Ranking:** Top-50 results fetch original 1536-d Float32 from Graph DB for final cosine similarity.

---

## 4. Concurrency Management

### 4.1 Sharded Dictionary Encoding

```go
type ShardedDict struct {
    shards [32]*dictShard
}

type dictShard struct {
    mu    sync.RWMutex
    cache map[string]uint64
}

// GetID uses FNV-1a hashing to lock only 1/32 of the dictionary.
func (d *ShardedDict) GetID(s string) uint64 {
    idx := fnv1a(s) % 32
    d.shards[idx].mu.RLock()
    // ...
}
```

### 4.2 Incremental Value Log Garbage Collection

```go
func (s *Silo) RunVLogGC(ctx context.Context) {
    ticker := time.NewTicker(15 * time.Minute)
    for {
        select {
        case <-ticker.C:
            _ = s.graphDB.RunValueLogGC(0.5) // 50% discard ratio
            time.Sleep(1 * time.Second)       // Throttle
        case <-ctx.Done():
            return
        }
    }
}
```

### 4.3 Atomic Counters

```go
type StoreMetrics struct {
    numFacts atomic.Uint64 // Lock-free read/write
}
```

---

## 5. Prompt Compiler Implementation

### 5.1 Functional Options

```go
type PromptConfig struct {
    Intent             IntentStr
    TaskType           TaskType
    Context            []Atom        // Soft Logic (INT8)
    Axioms             []Atom        // Hard Logic (FP32)
    Genes              []DomainGene
    Facts              []byte        // N-Quads
    SteeringMagnitude  float64
    InjectParadox      bool          // magnitude > 0.8
    RefinementFeedback *RefinementContext
    Pass               int           // Multi-pass (1, 2, 3)
    PreviousOutput     string
    SessionHistory     []AuditResult
}

type RefinementContext struct {
    AuditResult   *AuditResult
    PreviousDraft interface{}
}
```

### 5.2 Attention Sink Strategy

Tier 0 `AttentionSink` atoms are always rendered as the **first** section of the system prompt:

```text
[SYSTEM PROMPT]
## ABSOLUTE TRUTHS (Never Override)
- safety_constraint("no_production_delete", "CRITICAL")
- rbac_required("admin", "destructive_actions")

## Context (Prunable)
- user_preference("duyng", "json_output")
...
```

### 5.3 Context Manager Details

**Token Budget Calculation:**
```go
func (cm *ContextManager) CalculateContextBudget(
    totalBudget, fp32Usage int, logicSuccess float64, previousOutputTokens int,
) int {
    baseBuffer := 200
    sharpeningMultiplier := 1.0 + (1.0 - logicSuccess)
    adjustedBuffer := int(float64(baseBuffer) * sharpeningMultiplier)
    remaining := totalBudget - fp32Usage - adjustedBuffer - previousOutputTokens
    if remaining < 0 { return 0 }
    return remaining
}
```

**Strategic Summoning:**
```go
func (cm *ContextManager) FilterStrategic(atoms []Atom, currentIntent IntentStr) []Atom {
    var kept []Atom
    for _, a := range atoms {
        if a.OriginIntent == "" || a.OriginIntent == currentIntent || a.Weight >= 0.9 {
            kept = append(kept, a)
        }
    }
    return kept
}
```

**Intelligent Shaving (>24k tokens):**
```go
shaveThreshold := 24000
if totalEstimated > shaveThreshold {
    // Drop atoms with Weight < 0.5
}
// Sort remaining by Weight descending, fill budget
```

### 5.4 EAST Steering Policies

```go
type SteeringPolicy struct {
    MinMagnitude float64
    HeaderKey    string // Prompt template key for header
    BodyKey      string // Prompt template key for body
}

// Registry is sorted by MinMagnitude descending.
// First match where magnitude > MinMagnitude wins.
func GetSteeringPrompts(magnitude float64) (headerKey, bodyKey string)
```

### 5.5 Datalog-to-Natural-Language Translation

The Compiler includes `mangleToNaturalLanguage(rules []byte) string` which converts raw Datalog rules into LLM-friendly natural language instructions. This is critical for injecting Gene constraints into prompts without confusing the LLM.

---

## 6. Execution Framework: The Supervisor

### 6.1 SupervisedAction Lifecycle

```go
type SupervisedAction struct {
    inner        core.Action
    engine       core.Evaluator
    tracer       trace.Tracer
    failureMode  string
}

func (g *SupervisedAction) executeInternal(ctx context.Context, input Envelope) (Envelope, error) {
    // Phase 1: Inject dynamic configuration from policy engine
    g.injectDynamicConfig(ctx, &input)

    // Phase 2: Pre-check (Assess / Shadow Audit)
    if err := g.performAssessment(ctx, meta, input); err != nil {
        return Envelope{}, err
    }

    // Phase 3: Execute inner action
    result, err := g.executeAction(ctx, meta, input)
    if err != nil { return Envelope{}, err }

    // Phase 4: Post-check (Reflect)
    validatedResult, err := g.performReflection(ctx, meta, result)
    if err != nil { return Envelope{}, err }

    // Phase 5: Apply steering decisions
    finalResult := g.applySteering(ctx, meta, validatedResult)
    return finalResult, nil
}
```

### 6.2 Zero-Config Reflection (Struct Flattening)

```go
type Document struct {
    Author string   `mangle:"authored_by"`
    Tags   []string `mangle:"has_tag"`
}
// Becomes:
// quad("Doc_1", "authored_by", "duyng", "context").
// quad("Doc_1", "has_tag", "engineering", "context").
```

### 6.3 Leapfrog Triejoin (LFTJ) Execution

- Instead of `O(N*M)` nested loops, LFTJ bounds query time to worst-case output size.
- **Circuit Breakers**: `depth > 5` returns `ErrMaxRecursionDepth`.
- **Timeouts**: Every derivation runs under `context.WithTimeout(ctx, 2000*time.Millisecond)`.

---

## 7. Knowledge Induction Pipeline (Inductor)

### 7.1 Service Dependencies

```go
type Inductor struct {
    llm           ports.GenerativePort
    verifier      ports.ReasoningPort
    genePool      ports.GenePoolPort
    perception    *system.MarkdownPerceptionAdapter
    storage       ports.GenomeStoragePort
    evidenceStore ports.EvidenceStorePort
    auditor       *audit.Auditor
    quorum        federated.Quorum
    compiler      *logic.Compiler
}
```

### 7.2 Process Flow

```go
func (i *Inductor) Process(ctx context.Context, signal Signal) (string, error) {
    // 1. Normalize raw content → Atoms
    // 2. Extract evidence, deduplicate via SimilarityStorePort
    // 3. Call GenerativePort.Induce() → raw Datalog string
    // 4. Sanitize: strip markdown fences, comments, invalid syntax
    // 5. Shadow Audit: merge candidate gene with Tier 0 axioms, verify
    // 6. If pass: PersistKnowledge() as .dl file
    // 7. Return crystallized Datalog rules
}
```

### 7.3 Sanitization Rules

Post-generation cleaning strips:
- Markdown code fences (`` ```datalog ... ``` ``)
- Line comments (`// ...`, `# ...`)
- Empty predicates
- Duplicate rules
- Syntax-invalid Datalog

---

## 8. Auditor Implementation

### 8.1 Verification Flow

```go
func (a *Auditor) Verify(ctx context.Context, frame *CognitiveFrame) (*AuditResult, error) {
    switch frame.OutputType {
    case OutputTypePlan:
        // Verify structured Plan against genome
        return a.verifier.Verify(ctx, plan, frame.ActiveGenes)
    case OutputTypeRule:
        // Shadow Audit: merge candidate rules with active genome
        // CRITICAL: Validate Tier 0 axioms are present in ActiveGenes
        hasTier0 := false
        for _, gene := range frame.ActiveGenes {
            if gene.Tier == Tier0Kernel { hasTier0 = true; break }
        }
        if !hasTier0 {
            return nil, fmt.Errorf("CRITICAL: Tier 0 axioms missing (state machine violation)")
        }
        // Compile and verify combined genome
    }
}
```

### 8.2 Content Verification (MangleVerifier)

Verifies generated content against Gene rules:
- **Section presence:** `section("id", "Title", "Desc")` rules checked against output.
- **Style constraints:** `style_constraint("prohibited_word", "value")` checked via case-insensitive `strings.Contains`.

---

## 9. Genome Storage & Memory Mapping

### 9.1 GenomeStoragePort Interface

```go
type GenomeStoragePort interface {
    ReadManifest(ctx context.Context, path string) ([]byte, error)
    MapGene(ctx context.Context, path string) ([]byte, uintptr, error)  // mmap zero-copy
    UnmapGene(data []byte) error
    CalculateFileHash(ctx context.Context, path string) (string, error) // SHA256
    ReadFile(ctx context.Context, path string) ([]byte, error)
    WriteFile(ctx context.Context, path string, data []byte, perm os.FileMode) error
    LoadKnowledge(ctx context.Context, intent string) ([]byte, error)
    PersistKnowledge(ctx context.Context, intent string, data []byte) error
    PersistTrace(ctx context.Context, frame *CognitiveFrame, content []byte) error
    PersistProposal(ctx context.Context, intent string, data []byte) error
    PersistAsync(ctx context.Context, path string, data []byte) error
    Flush(ctx context.Context) error  // Wait for async writes
    ResolvePath(kind, id string) string  // "induced"/"pending"/"trace"/"evidence"/"manifest"
}
```

### 9.2 Boot-Time Integrity Check

```go
for _, meta := range manifest.Genes {
    data, addr, err := storage.MapGene(ctx, meta.Path)
    hash, _ := storage.CalculateFileHash(ctx, meta.Path)
    if hash != meta.Signature {
        return fmt.Errorf("INTEGRITY VIOLATION: gene %s hash mismatch", meta.Name)
    }
    gene := DomainGene{
        Rules:    data,
        MMapAddr: addr,
        // ...
    }
}
```

---

## 10. Observability

### 10.1 OpenTelemetry Spans

- **Semantic Lookup Spans:** Measure latency of mmap vs DB fallback.
- **Rule Traversal Spans:** Attach literal `.dl` file line numbers to log traces showing exactly *which axiom* permitted or denied execution.
- **EAST Pressure Metric:** Expose `LogicSuccess / EntropyCoefficient` ratio to Prometheus.

### 10.2 Trace Rendering

Every OODA epoch generates a Markdown trace artifact via `PresentationPort.Render()`:
```markdown
# Trace: <TraceID>
## Intent: <Intent>
## Audit Result: FP32_PASSED
## Proof Tree:
- Rule: rbac_check("duyng", "admin") → PASS
- Rule: safety_constraint("no_delete") → PASS
```

---

## 11. Project Layout (DDD)

```text
/manglekit/
├── cmd/manglekit/         # One Binary
├── genomes/               # Knowledge Base (.dl)
│   ├── core/              # Tier 0 Axioms
│   ├── governance/        # Tier 1 Policies
│   └── induced/           # Tier 2 AI-generated
├── internal/
│   ├── core/              # INNER HEXAGON
│   │   ├── domain/        # Entities: Gene, Atom, Frame, Envelope
│   │   ├── memory/        # CognitiveFrame Pool
│   │   ├── logic/         # Compiler, ContextManager, Steering
│   │   └── ports/         # Interfaces (11 ports)
│   ├── orchestrator/      # OODA State Machine & Observer
│   ├── audit/             # Auditor & Verifier
│   ├── supervisor/        # SupervisedAction Guard
│   ├── reasoning/         # ADAPTERS (Outer Hexagon)
│   │   ├── inductor/      # Knowledge Induction
│   │   ├── proposer/      # Tri-Stream Context Recall
│   │   └── verifier/      # Content Verification
│   ├── adapters/
│   │   ├── gemini/        # GenerativePort adapter
│   │   ├── mangle/        # ReasoningPort adapter
│   │   ├── storage/       # GenomeStoragePort adapter
│   │   ├── system/        # PerceptionPort, async worker
│   │   └── vector/        # VectorStorePort adapters
│   │       ├── flat_simd/ # Built-in: mmap + IVF-PQ (Zero-allocation)
│   │       └── hnsw_ext/  # Plug-in: gRPC client for Qdrant/Milvus
│   ├── genepool/          # GenePoolPort implementation
│   ├── storage/           # MEB Silo integration
│   └── telemetry/         # OpenTelemetry setup
└── docs/
    ├── CSD.md
    └── designs/
        ├── HLD.md
        └── LLD.md
```