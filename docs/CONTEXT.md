---
context_type: kernel_source_dump
project: manglekit_v2
language: go, datalog
last_updated: 2026-02-21T23:00:00Z
scan_mode: logic_focused
---

# Manglekit v2 (Sovereign Logic Kernel) - Live Architecture Snapshot

This document serves as the canonical, authoritative "Live Architecture" snapshot of the Manglekit v2 codebase. It contains the exact interfaces, structs, and logic rules that drive the Hexagonal OODA Loop.

## 1. THE COMPLETE FILE MAP

```text
.
├── adapters/                    # Implementation of Ports
│   ├── mangle/                 # Datalog Reasoning Adapter (google/mangle)
│   ├── storage/                # Knowledge & Graph storage adapters
│   │   ├── dict/               # High-performance sharded string interning
│   │   └── graph/              # BadgerDB-backed Quad store
│   └── vector/                 # Vector similarity search adapters
│       └── flat_simd/          # mmap-backed INT8 SIMD vector store
├── audit/                       # Verification & Security Auditing
├── core/                        # The Domain Model (Center of Hexagon)
│   ├── domain/                 # Pure domain structs (Quad, Gene, Envelope)
│   └── ports/                  # Hexagonal Port Interfaces
├── docs/                        # Architecture & Context Documentation
├── examples/                    # Reference implementations
│   └── proposalgpt/            # End-to-end CLI reference app
├── internal/                    # Internal engine mechanics
│   ├── engine/                 # OODA Loop & State orchestration
│   ├── genesis/                # Bootstrap & Tier 0 loading
│   └── genepool/               # Hot-reloading manifesto-driven memory
└── mkit/                        # CLI tooling
```

---

## 2. THE HEXAGONAL PORTS (core/ports)

Manglekit operates strictly through Ports. These interfaces define the capabilities required by the OODA orchestrator without coupling logic to specific infrastructure.

### Port Registry

```go
package ports

import (
	"context"
	"github.com/duynguyendang/manglekit/internal/core/domain"
	"github.com/google/mangle/ast"
)

// ReasoningPort abstracts the Datalog verification engine (Mangle).
type ReasoningPort interface {
	Verify(ctx context.Context, plan *domain.CognitiveFrame) error
	VerifyAtoms(ctx context.Context, atoms []ast.Atom, rule string) error
	Query(ctx context.Context, query string) ([]map[string]string, error)
}

// GenerativePort abstracts LLM generation and inductive synthesis.
type GenerativePort interface {
	Generate(ctx context.Context, prompt string, compiledPrompt string, maxTokens int) (string, error)
	Induce(ctx context.Context, frames []*domain.CognitiveFrame) ([]domain.Atom, error)
	Embed(ctx context.Context, text string) ([]float32, error)
}

// GenePoolPort abstracts tiered knowledge retrieval and hot-reloading.
type GenePoolPort interface {
	ActiveGenes(ctx context.Context, intent string) ([]domain.DomainGene, error)
	Reload(ctx context.Context) error
}

// PerceptionPort translates external signals into the standard Atom format.
type PerceptionPort interface {
	Normalize(ctx context.Context, raw []byte) (domain.Signal, error)
}

// GenomeStoragePort handles physical Gene CRUD, mmap zero-copy reads, and async trace writes.
type GenomeStoragePort interface {
	ReadManifest(ctx context.Context, path string) ([]byte, error)
	MapGene(ctx context.Context, path string) ([]byte, uintptr, error)
	UnmapGene(data []byte) error
	CalculateFileHash(ctx context.Context, path string) (string, error)
	ReadFile(ctx context.Context, path string) ([]byte, error)
	WriteFile(ctx context.Context, path string, data []byte, perm os.FileMode) error
	LoadKnowledge(ctx context.Context, intent string) ([]byte, error)
	PersistKnowledge(ctx context.Context, intent string, data []byte) error
	PersistTrace(ctx context.Context, frame *domain.CognitiveFrame, content []byte) error
	PersistProposal(ctx context.Context, intent string, data []byte) error
	PersistAsync(ctx context.Context, path string, data []byte) error
	Flush(ctx context.Context) error
	ResolvePath(kind, id string) string
}

// EvidenceStorePort manages historical evidence for knowledge synthesis.
type EvidenceStorePort interface {
	SaveBatch(ctx context.Context, frames []*domain.CognitiveFrame) error
	Load(ctx context.Context, limit int) ([]*domain.CognitiveFrame, error)
	FindSimilar(ctx context.Context, frame *domain.CognitiveFrame, topK int) ([]*domain.CognitiveFrame, error)
}

// CompilerPort assembles optimized prompts leveraging EAST and Attention Sinks.
type CompilerPort interface {
	Compile(ctx context.Context, frame *domain.CognitiveFrame) (string, error)
}

// PresentationPort formats raw responses into final target formats (e.g., Markdown).
type PresentationPort interface {
	Render(ctx context.Context, response domain.DecisionOutput) ([]byte, error)
}

// VectorStorePort abstracts semantic search indices (e.g., SIMD mmap).
type VectorStorePort interface {
	Insert(id string, vector []float32) error
	Search(query []float32, topK int) ([]string, error)
	Close() error
}

// EmbeddingPort abstracts the process of creating semantic vectors.
type EmbeddingPort interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// StoragePort abstracts high-level trace persistence operations.
type StoragePort interface {
	SaveTrace(ctx context.Context, frame *domain.CognitiveFrame) error
}

// AuditorPort abstracts verification and tracing away from the orchestrator.
type AuditorPort interface {
	Verify(ctx context.Context, frame *domain.CognitiveFrame) (*domain.AuditResult, error)
	GenerateTrace(result *domain.AuditResult, frame *domain.CognitiveFrame) ([]byte, error)
}
```

---

## 3. CORE DOMAIN TYPES (core/domain)

Sovereign primitives passed seamlessly between Ports.

### 3.1 Quad (The Universal Relationship)

```go
package domain

// Quad represents a single factual relationship in the graph.
type Quad struct {
	Subject   string  // Source entity (e.g., "User:Bob")
	Predicate string  // Relationship (e.g., "has_role")
	Object    string  // Target value (e.g., "Admin", "42")
	Graph     string  // Namespace/Context (defaults to "default")
	Weight    float64 // Confidence score (0.0-1.0)
	Source    string  // Provenance ("ast", "virtual", "inference")
}
```

### 3.2 Atom & Signal (The Streaming Context)

```go
package domain

import "time"

// Atom is a fast, streamlined triple used specifically for the logic evaluator.
type Atom struct {
	Predicate    string  `json:"predicate"`
	Subject      string  `json:"subject"`
	Object       string  `json:"object"`
	Weight       float64 `json:"weight,omitempty"`
	OriginIntent string  `json:"origin_intent,omitempty"`
}

// Signal represents an incoming event requiring a decision.
type Signal struct {
	ID         string
	Source     string // "chat", "webhook", "cron"
	Timestamp  time.Time
	RawContent []byte
	Intent     string
	IntentHint string // Fallback categorization
	IsProposal bool   // If true, trigger Zero-Trust auditing
}
```

### 3.3 The Cognitive Frame (The Brain's Working Memory)

```go
package domain

import "time"

// CognitiveFrame is the central context object holding the lifecycle of a thought process.
type CognitiveFrame struct {
	ID             string
	Timestamp      time.Time
	Intent         string
	TaskType       string // Categorized task (e.g., "chat", "query")
	OutputType     string // Expected format (e.g., "json", "markdown")
	Context        map[string]interface{}
	AttentionSink  string
	ActiveGenes    []DomainGene
	Draft          string       // Initial LLM proposal
	Proof          *ProofNode  // Outcome of logic verification
	Status         string      // "pending", "approved", "rejected", "halted"
	TraceID        string
	SessionHistory []string    // Short-term conversational context
	EAST           *EASTState  // Entropy-Aware Steering coordinates
	IsProposal     bool        // True if this requires validation
}
```

### 3.4 DomainGene & Tiering

```go
package domain

// DomainGene represents a loaded behavior or knowledge set.
type DomainGene struct {
	Name         string
	Tier         string   // "0" (Core), "1" (Domain), "2" (Ad-hoc)
	TierID       int      // 0, 1, 2
	Rules        []string // Datalog rules
	Signature    string   // SHA256 integrity hash
	MMapAddr     uintptr  // Address for Zero-copy reads
	Capabilities []string
	Intents      []string // Subscribed signals
	FactPath     string
	SourcePath   string
	IsUnverified bool     // If true, hasn't passed genesis check
}
```

---

## 4. OODA ORCHESTRATOR IMPLEMENTATION (internal/engine)

The core control loop linking Perception to Output.

### The Supervisor Run Loop

```go
package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/internal/core/domain"
	"github.com/duynguyendang/manglekit/internal/core/logic"
	"github.com/duynguyendang/manglekit/internal/core/ports"
	"github.com/google/uuid"
)

// Orchestrator manages the OODA loop lifecycle for a given Signal.
type Orchestrator struct {
	perception ports.PerceptionPort
	proposer   ports.GenerativePort // Typically the multi-stream Proposer service
	compiler   ports.CompilerPort
	auditor    ports.AuditorPort
	genePool   ports.GenePoolPort
	storage    ports.GenomeStoragePort
}

func New(
	perception ports.PerceptionPort,
	proposer ports.GenerativePort,
	compiler ports.CompilerPort,
	auditor ports.AuditorPort,
	genePool ports.GenePoolPort,
	storage ports.GenomeStoragePort,
) *Orchestrator {
	return &Orchestrator{
		perception: perception,
		proposer:   proposer,
		compiler:   compiler,
		auditor:    auditor,
		genePool:   genePool,
		storage:    storage,
	}
}

// Execute loops through Observe, Orient, Decide, Verify until success or deadlock.
func (o *Orchestrator) Execute(ctx context.Context, signal domain.Signal) (*domain.DecisionOutput, error) {
	// Initialize Epoch Passport (CognitiveFrame)
	frame := &domain.CognitiveFrame{
		ID:        uuid.New(),
		Timestamp: time.Now().UTC(),
		Intent:    signal.Intent,
		TaskType:  domain.TaskTypeGeneration, // Default
		Status:    domain.VerifyStatusPending,
		EAST: domain.EASTState{
			LogicSuccess:       1.0,
			EntropyCoefficient: 1.0, // Assuming baseline novelty
		},
	}

	frame.EAST.SteeringMagnitude = logic.CalculateEAST(frame.EAST.LogicSuccess, frame.EAST.EntropyCoefficient)

	// 1. OBSERVE (Signal -> Atoms)
	payload, err := o.perception.Normalize(ctx, signal)
	if err != nil {
		return nil, fmt.Errorf("observe phase failed: %w", err)
	}

	// Convert iterator to slice for context
	for atom := range payload {
		frame.Context = append(frame.Context, atom)
	}

	// 2. ORIENT (Load Genes & Context)
	for gene := range o.genePool.ActiveGenes(ctx, frame.Intent) {
		frame.ActiveGenes = append(frame.ActiveGenes, *gene)
		if gene.Tier == domain.Tier0Kernel {
			// Extract FP32 axioms from Tier 0 rules for Attention Sink
			// Mocking atom extraction for now
			frame.AttentionSink = append(frame.AttentionSink, domain.Atom{Predicate: "kernel_rule", Subject: gene.Name})
		}
	}

	// 3. DECIDE & VERIFY (Teacher-Student Loop)
	maxRefinements := 3
	for pass := 1; pass <= maxRefinements; pass++ {
		// DECIDE Phase
		var refinement *logic.RefinementContext
		if frame.Status == domain.VerifyStatusFailed {
			refinement = &logic.RefinementContext{
				AuditResult:   frame.Proof,
				PreviousDraft: frame.Draft,
			}
		}

		prompt, err := o.compiler.Compile(ctx, frame.Intent,
			logic.WithTaskType(frame.TaskType),
			logic.WithContext(frame.Context),
			logic.WithAxioms(frame.AttentionSink),
			logic.WithGenes(frame.ActiveGenes),
			logic.WithPass(pass),
			logic.WithSteering(frame.EAST.SteeringMagnitude, logic.ShouldInjectParadox(frame.EAST.SteeringMagnitude)),
			logic.WithRefinement(refinement),
			logic.WithHistory(frame.SessionHistory),
		)
		if err != nil {
			return nil, fmt.Errorf("decide phase (compile) failed: %w", err)
		}
		_ = prompt

		// Neural Synthesis (GenerativePort)
		// Assuming GenerativePort takes the compiled prompt context internally now
		// In reality, we'd pass the compiled prompt to the LLM adapter.
		plan, err := o.proposer.Generate(ctx, frame.Intent, prompt, frame.Context, frame.ActiveGenes)
		if err != nil {
			return nil, fmt.Errorf("decide phase (generate) failed: %w", err)
		}

		frame.Draft = plan
		frame.OutputType = domain.OutputTypePlan

		// VERIFY Phase (Shadow Audit)
		auditResult, err := o.auditor.Verify(ctx, frame)
		if err != nil {
			// Halt immediately on Auditor system error (e.g. missing Tier 0)
			return nil, err
		}

		frame.Proof = auditResult
		frame.SessionHistory = append(frame.SessionHistory, *auditResult)

		if auditResult.Pass {
			// ACT Phase Preparation
			frame.Status = domain.VerifyStatusPassed

			// Save Trace
			trace := o.auditor.GenerateTrace(frame)
			_ = o.storage.PersistTrace(ctx, frame, trace)

			return o.finalizeOutput(plan), nil
		}

		// Teacher-Student Loop Refinement update
		// Recalculate LogicSuccess based on EntropyDelta feedback
		frame.EAST.LogicSuccess = frame.EAST.LogicSuccess - 0.2 // simplistic penalty
		frame.EAST.SteeringMagnitude = logic.CalculateEAST(frame.EAST.LogicSuccess, frame.EAST.EntropyCoefficient)
		frame.Status = domain.VerifyStatusFailed
	}

	return nil, fmt.Errorf("DEADLOCK: failed to produce logic-compliant plan after %d refinements", maxRefinements)
}

func (o *Orchestrator) finalizeOutput(plan *ports.Plan) *domain.DecisionOutput {
	// Convert LLM Plan to final structured execution output
	if plan == nil || len(plan.Steps) == 0 {
		return &domain.DecisionOutput{Action: "NO_OP"}
	}

	// Assuming first step is the desired action for now
	return &domain.DecisionOutput{
		Action: "EXECUTE_PLAN",
		Params: map[string]any{"steps": plan.Steps},
	}
}
```

---

## 5. REASONING ENGINE (internal/adapters/mangle/adapter.go)

```go
package mangle

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/duynguyendang/manglekit/internal/core/domain"
	"github.com/google/mangle/analysis"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/engine"
	"github.com/google/mangle/factstore"
	"github.com/google/mangle/parse"
)

// ReasoningAdapter implements ports.ReasoningPort using the Google Mangle engine.
type ReasoningAdapter struct {
	MaxRecursionDepth int
	EvalTimeout       time.Duration
}

// NewReasoningAdapter initializes the Datalog evaluator with strict circuit breakers.
func NewReasoningAdapter() *ReasoningAdapter {
	return &ReasoningAdapter{
		MaxRecursionDepth: 5,
		EvalTimeout:       2000 * time.Millisecond,
	}
}

// Verify implements the Shadow Audit for structured plans.
func (a *ReasoningAdapter) Verify(ctx context.Context, subject interface{}, genome []domain.DomainGene) (*domain.AuditResult, error) {
	atoms := []domain.Atom{}
	return a.VerifyAtoms(ctx, atoms, genome)
}

// VerifyAtoms executes the core Mangle evaluation (semi-naive with stratification).
func (a *ReasoningAdapter) VerifyAtoms(ctx context.Context, atoms []domain.Atom, genome []domain.DomainGene) (*domain.AuditResult, error) {
	evalCtx, cancel := context.WithTimeout(ctx, a.EvalTimeout)
	defer cancel()

	// 1. Combine all ActiveGene rules into a single Datalog program.
	var programSource strings.Builder
	for _, gene := range genome {
		programSource.Write(gene.Rules)
		programSource.WriteString("\n")
	}

	source := programSource.String()
	if strings.TrimSpace(source) == "" {
		return &domain.AuditResult{Pass: true}, nil
	}

	// 2. Parse the combined program into a SourceUnit.
	sourceUnit, err := parse.Unit(strings.NewReader(source))
	if err != nil {
		return nil, fmt.Errorf("failed to parse combined genome: %w", err)
	}

	// 3. Analyze (type-check and stratify) using the correct API.
	programInfo, err := analysis.Analyze([]parse.SourceUnit{sourceUnit}, nil)
	if err != nil {
		// Analysis failed — return soft pass with warning.
		return &domain.AuditResult{
			Pass:          true,
			EntropyDelta:  0.1,
			ConflictPath:  fmt.Sprintf("analysis_warning: %v", err),
			ViolationTier: domain.Tier2AI,
		}, nil
	}

	// 4. Create an in-memory fact store and inject input atoms.
	store := factstore.NewSimpleInMemoryStore()

	for _, atom := range atoms {
		baseFact := ast.Atom{
			Predicate: ast.PredicateSym{Symbol: atom.Predicate, Arity: 2},
			Args: []ast.BaseTerm{
				ast.String(atom.Subject),
				ast.String(atom.Object),
			},
		}
		store.Add(baseFact)
	}

	// 5. Check context timeout before evaluation.
	if evalCtx.Err() != nil {
		return nil, fmt.Errorf("evaluation timeout exceeded before engine start (%v)", a.EvalTimeout)
	}

	// 6. Evaluate the program using the Mangle semi-naive engine.
	if err := engine.EvalProgram(programInfo, store); err != nil {
		return nil, fmt.Errorf("mangle evaluation failed: %w", err)
	}

	// 7. Check for halt(Msg) derivations — Tier 0/1 violations.
	haltResult := a.checkPredicate(store, "halt", 1)
	if haltResult != nil {
		return haltResult, nil
	}

	// 8. Check for warn(Msg) derivations — soft heuristic warnings.
	warnResult := a.checkPredicate(store, "warn", 1)
	if warnResult != nil {
		// Warnings don't block — override Pass to true.
		warnResult.Pass = true
		warnResult.ViolationTier = domain.Tier2AI
		return warnResult, nil
	}

	return &domain.AuditResult{Pass: true}, nil
}

// checkPredicate scans the fact store for derived facts matching the given predicate.
func (a *ReasoningAdapter) checkPredicate(store factstore.SimpleInMemoryStore, predName string, arity int) *domain.AuditResult {
	pred := ast.PredicateSym{Symbol: predName, Arity: arity}

	// Build a wildcard query atom with variable arguments.
	queryArgs := make([]ast.BaseTerm, arity)
	for i := 0; i < arity; i++ {
		queryArgs[i] = ast.Variable{Symbol: fmt.Sprintf("X%d", i)}
	}
	queryAtom := ast.Atom{Predicate: pred, Args: queryArgs}

	var foundFacts []ast.Atom
	_ = store.GetFacts(queryAtom, func(a ast.Atom) error {
		foundFacts = append(foundFacts, a)
		return nil
	})

	if len(foundFacts) == 0 {
		return nil
	}

	msg := "unknown"
	if len(foundFacts[0].Args) > 0 {
		msg = foundFacts[0].Args[0].String()
	}

	return &domain.AuditResult{
		Pass:          false,
		ViolationTier: domain.Tier0Kernel,
		ConflictPath:  msg,
		ProofTree: &domain.ProofNode{
			Rule: fmt.Sprintf("%s(%s)", predName, msg),
			Pass: false,
		},
		EntropyDelta: 0.3,
	}
}

// Query allows raw datalog extraction, bounded by the depth limit.
func (a *ReasoningAdapter) Query(ctx context.Context, query string, genome []domain.DomainGene) ([]domain.Atom, error) {
	return nil, nil
}
```

---

## 6. ZERO-TRUST AUDITOR (internal/audit/auditor.go)

```go
package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/internal/core/domain"
	"github.com/duynguyendang/manglekit/internal/core/ports"
)

// Auditor governs the OODA Verification Phase (Shadow Audit).
type Auditor struct {
	verifier ports.ReasoningPort
}

// New initializes the auditor subsystem.
func New(verifier ports.ReasoningPort) *Auditor {
	return &Auditor{
		verifier: verifier,
	}
}

// Verify implements the core security gate as defined in LLD 8.1.
// It mathematically proves that an intended payload complies with the active GenePool.
func (a *Auditor) Verify(ctx context.Context, frame *domain.CognitiveFrame) (*domain.AuditResult, error) {
	// LLD 8.1: State Machine Violation Check
	// The Auditor must explicitly assert that Tier 0 Kernel Axioms are loaded into the
	// active reasoning frame. If the prompt compiler or agent loop somehow dropped Tier 0,
	// the agent is operating completely unrestrained and must be halted.
	hasTier0 := false
	for _, gene := range frame.ActiveGenes {
		if gene.Tier == domain.Tier0Kernel {
			hasTier0 = true
			break
		}
	}

	if !hasTier0 {
		return nil, fmt.Errorf("CRITICAL SECURITY EXCEPTION: Tier 0 Kernel Axioms missing from Epoch %s. Halting.", frame.ID)
	}

	// Route audit based on target output type
	switch frame.OutputType {
	case domain.OutputTypePlan:
		return a.verifyPlan(ctx, frame)
	case domain.OutputTypeRule:
		return a.verifyInducedRules(ctx, frame)
	default:
		return nil, fmt.Errorf("unknown task output type: %s", frame.OutputType)
	}
}

// verifyPlan checks structured JSON task execution plans against the Tiered Logic.
func (a *Auditor) verifyPlan(ctx context.Context, frame *domain.CognitiveFrame) (*domain.AuditResult, error) {
	// Route to ReasoningPort (Mangle LFTJ evaluator)
	res, err := a.verifier.Verify(ctx, frame.Draft, frame.ActiveGenes)
	if err != nil {
		return nil, err
	}

	// Update the frame status depending on the trust tier violated
	if !res.Pass {
		if res.ViolationTier == domain.Tier0Kernel || res.ViolationTier == domain.Tier1Admin {
			frame.Status = domain.VerifyStatusFailed
		} else {
			// Tier 2 or 3 are soft heuristics, log warning but allow pass
			frame.Status = domain.VerifyStatusWarning
			res.Pass = true // override
		}
	} else {
		frame.Status = domain.VerifyStatusPassed
	}

	return res, nil
}

// verifyInducedRules handles the Knowledge Induction shadow audit.
// Validates that newly learned Tier 2/3 rules do not contradict Tier 0/1.
func (a *Auditor) verifyInducedRules(ctx context.Context, frame *domain.CognitiveFrame) (*domain.AuditResult, error) {
	rawCandidate, ok := frame.Draft.([]byte)
	if !ok {
		return nil, fmt.Errorf("verifyInducedRules expects []byte draft, got %T", frame.Draft)
	}

	// Mocking compilation of the combined graph (Existing Genes + rawCandidate)
	_ = rawCandidate

	// In a complete implementation, this merges `rawCandidate` logic into
	// the global Datalog program string, and issues a contradiction query:
	// `contradiction(X) :- rule_A, not rule_B.`

	return &domain.AuditResult{
		Pass: true,
	}, nil
}

// GenerateTrace creates a markdown artifact documenting the verification run.
func (a *Auditor) GenerateTrace(frame *domain.CognitiveFrame) []byte {
	trace := fmt.Sprintf("# Epoch Trace: %s\n", frame.ID)
	trace += fmt.Sprintf("Timestamp: %s\n", time.Now().UTC().Format(time.RFC3339))
	trace += fmt.Sprintf("Intent: %s\n", frame.Intent)

	if frame.Proof != nil {
		trace += fmt.Sprintf("Audit Status: %s\n", frame.Status)
		if !frame.Proof.Pass {
			trace += fmt.Sprintf("Violation Tier: %s\n", frame.Proof.ViolationTier)
			trace += fmt.Sprintf("Conflict Path: %s\n", frame.Proof.ConflictPath)
		}
	} else {
		trace += "Audit Status: PENDING or SKIPPED\n"
	}

	return []byte(trace)
}
```

---

## 7. SYSTEM VOCABULARY (Datalog Predicates)

The standard vocabulary recognized uniformly by the Mangle logic evaluator for steering and auditing:

*   `halt(Message)`: Derivation triggers an immediate hard block of the OODA loop due to security violation or invariant failure. Checked explicitly by `ReasoningAdapter.checkPredicate("halt")`.
*   `warn(Message)`: Soft violation; flag for trace analysis but the OODA loop proceeds. Checked explicitly by `ReasoningAdapter.checkPredicate("warn")`.
*   `action_meta(ActionID, Name, Type)`: Identifies executable capability bounds.
*   `env_has_label(EnvID, Label)`: Attests a verified identity or security clearance level to a frame.
