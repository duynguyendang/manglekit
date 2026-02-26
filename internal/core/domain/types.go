package domain

// IntentStr represents the derived intent of a signal.
type IntentStr string

// VerifyStatus represents the outcome of the verification phase.
type VerifyStatus string

const (
	VerifyStatusPending VerifyStatus = "PENDING"
	VerifyStatusPassed  VerifyStatus = "FP32_PASSED"
	VerifyStatusFailed  VerifyStatus = "LOGIC_VIOLATION"
	VerifyStatusWarning VerifyStatus = "WARNING"
)

// TrustTier defines the authority level of a gene or action.
type TrustTier string

const (
	Tier0Kernel TrustTier = "TIER_0" // Immutable Core
	Tier1Admin  TrustTier = "TIER_1" // Human Operator
	Tier2AI     TrustTier = "TIER_2" // Induced Logic
	Tier3User   TrustTier = "TIER_3" // Untrusted Input
)

// TaskType defines the category of cognitive work.
type TaskType string

const (
	TaskTypeInduction  TaskType = "INDUCTION"  // Learning from raw input
	TaskTypeGeneration TaskType = "GENERATION" // Creating structured output
	TaskTypeAudit      TaskType = "AUDIT"      // System verification
	TaskTypeRecovery   TaskType = "RECOVERY"   // Error remediation
)

// OutputType defines the expected output format.
type OutputType string

const (
	OutputTypePlan OutputType = "PLAN" // Structured action plan (JSON)
	OutputTypeRule OutputType = "RULE" // Datalog rules
)

// PortType identifies the source of a signal.
type PortType string
