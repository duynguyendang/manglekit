package ooda

import (
	"time"

	"github.com/google/uuid"
)

// IntentStr represents the goal or objective of the execution
type IntentStr string

// Phase represents the phases in the OODA loop
type Phase string

const (
	PhaseObserve Phase = "observe"
	PhaseOrient  Phase = "orient"
	PhaseDecide  Phase = "decide"
	PhaseVerify  Phase = "verify"
	PhaseAct     Phase = "act"
)

// TrustTier represents the 4-level system of logical axiom trust
type TrustTier string

const (
	Tier0Kernel TrustTier = "TIER_0" // Immutable Core Axioms (Hard Logic - FP32)
	Tier1Admin  TrustTier = "TIER_1" // Human Operator / Governance
	Tier2AI     TrustTier = "TIER_2" // Induced / Learned Logic (Soft Logic - INT8)
	Tier3User   TrustTier = "TIER_3" // Untrusted External Input
)

// VerifyStatus represents the result of the Datalog verification bridge
type VerifyStatus string

const (
	VerifyStatusPending VerifyStatus = "PENDING"
	VerifyStatusPassed  VerifyStatus = "FP32_PASSED"     // Hard logic passed
	VerifyStatusFailed  VerifyStatus = "LOGIC_VIOLATION" // Critical failure, halts
	VerifyStatusWarning VerifyStatus = "WARNING"         // Soft logic warning, continues
)

// TaskType represents the operational mode for this epoch
type TaskType string

const (
	TaskTypeInduction  TaskType = "INDUCTION"  // Learning from raw input
	TaskTypeGeneration TaskType = "GENERATION" // Creating structured output
	TaskTypeAudit      TaskType = "AUDIT"      // System verification
	TaskTypeRecovery   TaskType = "RECOVERY"   // Error remediation
)

// OutputType represents what the generative port should produce
type OutputType string

const (
	OutputTypePlan OutputType = "PLAN" // Structured JSON or Markdown
	OutputTypeRule OutputType = "RULE" // Datalog logic rules for crystallization
)

// Atom is the smallest unit of knowledge in Kronos's streaming architecture
type Atom struct {
	Predicate    string    `json:"predicate"`
	Subject      string    `json:"subject"`
	Object       string    `json:"object"`
	Weight       float64   `json:"weight"` // 1.0 (Fact) to 0.1 (Guess)
	OriginIntent IntentStr `json:"origin_intent,omitempty"`
}

// DomainGene is a crystallized unit of Datalog logic with trust tiering
type DomainGene struct {
	Name         string    `json:"name"`
	Tier         TrustTier `json:"tier"`
	TierID       string    `json:"tier_id"`
	Rules        []byte    `json:"rules"`     // Compiled Datalog content
	Signature    [32]byte  `json:"signature"` // SHA256 integrity hash
	MMapAddr     uintptr   `json:"-"`         // Zero-copy mmap pointer
	Capabilities []string  `json:"capabilities"`
	Intents      []string  `json:"intents"`
	FactPath     string    `json:"fact_path,omitempty"`
	SourcePath   string    `json:"source_path,omitempty"`
	IsUnverified bool      `json:"is_unverified"`
}

// AuditResult represents the verification trace produced by the ReasoningPort
type AuditResult struct {
	Pass          bool      `json:"pass"`
	ViolationTier TrustTier `json:"violation_tier"` // Which tier was violated
	TierID        string    `json:"tier_id"`
	ConflictPath  string    `json:"conflict_path"` // "safety.dl:42"
	ProofTree     any       `json:"proof_tree"`    // "Why" it failed
	EntropyDelta  float64   `json:"entropy_delta"` // Feedback for EAST
}

// EASTState tracks the Cognitive Pressure metrics for steering LLM temperature
type EASTState struct {
	LogicSuccess       float64 `json:"logic_success"`       // L (0.0 - 1.0)
	EntropyCoefficient float64 `json:"entropy_coefficient"` // N (Novelty)
	SteeringMagnitude  float64 `json:"steering_magnitude"`  // P = exp(1-L) / N
}

// CognitiveFrame is the complete state of a single reasoning epoch.
// It replaces the generic loosely typed `State` from earlier versions.
type CognitiveFrame struct {
	ID        uuid.UUID
	Timestamp time.Time
	Intent    IntentStr
	Phase     Phase

	// Task Metadata
	TaskType   TaskType
	OutputType OutputType

	// Input Stimulus
	Input string

	// Memory & Logic
	Context       []Atom         // Soft Logic (INT8) - Observed facts, pruneable
	AttentionSink []Atom         // Hard Logic (FP32) - Immutable Axioms (Tier 0), never pruned
	ActiveGenes   []DomainGene   // Logic Pinning - crystallized rules for this epoch
	RawContext    map[string]any // Legacy escape hatch for transitional state

	// Reasoning
	Draft  any          // Neural proposal: *Plan for PLAN output, []byte for RULE output
	Proof  *AuditResult // Verification trace
	Status VerifyStatus

	// Telemetry
	TraceID        string
	SessionHistory []AuditResult
	EAST           EASTState

	// Staging
	IsProposal bool
}

// NewCognitiveFrame initializes a fresh cognitive epoch.
func NewCognitiveFrame(input string, intent IntentStr, taskType TaskType) *CognitiveFrame {
	return &CognitiveFrame{
		ID:         uuid.New(),
		Timestamp:  time.Now(),
		Phase:      PhaseObserve,
		Input:      input,
		Intent:     intent,
		TaskType:   taskType,
		Status:     VerifyStatusPending,
		RawContext: make(map[string]any),
	}
}
