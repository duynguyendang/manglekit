package domain

import (
	"time"

	"github.com/duynguyendang/manglekit-wip/core"
	"github.com/google/uuid"
)

// ProofNode represents a step in the Datalog proof tree.
type ProofNode struct {
	Rule    string
	Matches map[string]string
	Pass    bool
}

// AuditResult represents the outcome of a verification phase.
type AuditResult struct {
	Pass          bool       `json:"pass"`
	ViolationTier TrustTier  `json:"violation_tier"`
	TierID        string     `json:"tier_id"`
	ConflictPath  string     `json:"conflict_path"` // "safety.dl:42"
	ProofTree     *ProofNode `json:"proof_tree"`    // "Why" it failed
	EntropyDelta  float64    `json:"entropy_delta"` // Feedback for EAST
}

// EASTState captures the current Entropic Activation Steering metrics.
type EASTState struct {
	LogicSuccess       float64 `json:"logic_success"`       // L (0.0 - 1.0)
	EntropyCoefficient float64 `json:"entropy_coefficient"` // N (Novelty)
	SteeringMagnitude  float64 `json:"steering_magnitude"`  // P = exp(1-L) / N
}

// Paradox injection triggers when P > ParadoxInjectionThreshold
const ParadoxInjectionThreshold = 0.8

// CognitiveFrame represents the state of a single reasoning Epoch.
// It is the central data structure carrying context and traces through the OODA loop.
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

// DecisionOutput is a placeholder for the final structured output from the agent.
type DecisionOutput struct {
	Action string
	Params map[string]any
}

// Envelope is an alias to the core Envelope type for internal domain usage.
type Envelope = core.Envelope
