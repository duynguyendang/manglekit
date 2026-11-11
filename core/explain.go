package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DerivationStep is one node in a proof (derivation tree) for a decision.
// It records how a single grounded fact was derived: by which rule (full
// Datalog text), with which variable bindings, from which premise facts
// (Children), and — where derivable — the governance tier the derivation
// actually carried (e.g. a bound "Tier" argument in the rule head), as
// opposed to filename/predicate heuristics.
type DerivationStep struct {
	// Fact is the grounded atom this node proves, e.g.
	// halt("Req", "prompt injection detected", "T1").
	Fact string `json:"fact"`
	// Rule is the full text of the rule that derived Fact. Empty for
	// base (EDB) facts.
	Rule string `json:"rule,omitempty"`
	// Predicate is the head predicate name of Fact.
	Predicate string `json:"predicate,omitempty"`
	// Tier is the governance tier carried by the derivation (TierUnknown
	// when no tier is derivable from the rule itself).
	Tier Tier `json:"tier,omitempty"`
	// Bindings maps variable names to their grounded values in this rule
	// firing (compatible with RuleInference.Bindings / Phase 13 audit
	// annotations).
	Bindings map[string]string `json:"bindings,omitempty"`
	// Via classifies the step: "rule", "fact", "negation", "builtin",
	// or "depth-limit" when the proof was truncated.
	Via string `json:"via,omitempty"`
	// Children are the derivations of the premises that fired the rule.
	Children []DerivationStep `json:"children,omitempty"`
}

// Derivation via kinds.
const (
	ViaRule       = "rule"
	ViaFact       = "fact"
	ViaNegation   = "negation"
	ViaBuiltin    = "builtin"
	ViaDepthLimit = "depth-limit"
)

// Explanation is the structured proof for a query or gate decision. Each
// entry in Derivations is a complete derivation tree for one matching fact.
type Explanation struct {
	// Query is the original query string.
	Query string `json:"query"`
	// Outcome is true when at least one fact matched the query.
	Outcome bool `json:"outcome"`
	// Derivations contains one derivation tree per matching fact.
	Derivations []DerivationStep `json:"derivations,omitempty"`
}

// AuditTrail converts the explanation into an AuditTrail whose
// MatchedRules carry the real rule text, bindings, and tier provenance.
// The provided engineID identifies the producing engine instance.
func (e *Explanation) AuditTrail(engineID string) *AuditTrail {
	trail := NewAuditTrail(engineID, e.Query)
	if e == nil {
		return trail
	}
	e.appendSteps(trail)
	trail.MatchedCount = len(e.Derivations)
	return trail
}

// ToExplanation converts an AuditTrail into a structured Explanation.
func (a *AuditTrail) ToExplanation() *Explanation {
	if a == nil {
		return nil
	}
	steps := make([]DerivationStep, 0, len(a.MatchedRules))
	for _, r := range a.MatchedRules {
		steps = append(steps, DerivationStep{
			Fact:      r.RuleName,
			Rule:      r.Definition,
			Predicate: r.Predicate,
			Tier:      r.Tier,
			Bindings:  r.Bindings,
			Via:       ViaRule,
		})
	}
	return &Explanation{
		Query:       a.Query,
		Outcome:     len(steps) > 0,
		Derivations: steps,
	}
}

// ProofBundle is a canonical, JSON-serializable compliance artifact combining
// decision identity, policy fingerprint, timestamp, action metadata, gate outcome,
// governance tier, and the deductive derivation proof tree (Explanation).
type ProofBundle struct {
	// DecisionID is the unique identifier for this evaluation decision.
	DecisionID string `json:"decision_id"`
	// PolicyHash is the SHA-256 hex fingerprint of the active Datalog policy.
	PolicyHash string `json:"policy_hash,omitempty"`
	// Timestamp is the UTC timestamp when the proof was constructed.
	Timestamp time.Time `json:"timestamp"`
	// ActionName is the identifier of the evaluated action/skill.
	ActionName string `json:"action_name"`
	// Outcome is "ALLOW" or "HALT".
	Outcome string `json:"outcome"`
	// Tier is the governance tier responsible for the decision (e.g. "T0", "T1").
	Tier string `json:"tier,omitempty"`
	// Explanation is the structured deductive derivation tree.
	Explanation *Explanation `json:"explanation,omitempty"`
}

// NewProofBundle creates a ProofBundle with a new UUID DecisionID and UTC timestamp.
func NewProofBundle(actionName, outcome, tier, policyHash string, expl *Explanation) *ProofBundle {
	return &ProofBundle{
		DecisionID:  uuid.New().String(),
		PolicyHash:  policyHash,
		Timestamp:   time.Now().UTC(),
		ActionName:  actionName,
		Outcome:     outcome,
		Tier:        tier,
		Explanation: expl,
	}
}

// ToJSON returns canonical, deterministic JSON encoding for compliance workflows.
func (p *ProofBundle) ToJSON() ([]byte, error) {
	if p == nil {
		return []byte("null"), nil
	}
	return json.Marshal(p)
}

// ProofBundleFromJSON parses a JSON-encoded ProofBundle.
func ProofBundleFromJSON(data []byte) (*ProofBundle, error) {
	var pb ProofBundle
	if err := json.Unmarshal(data, &pb); err != nil {
		return nil, err
	}
	return &pb, nil
}

// Audit record phases.
const (
	PhasePreCheck  = "pre-check"
	PhasePostCheck = "post-check"

	AuditOutcomeAllow = "ALLOW"
	AuditOutcomeHalt  = "HALT"
)

// ToJSON returns canonical JSON bytes for the audit record.
func (r *AuditRecord) ToJSON() ([]byte, error) {
	if r == nil {
		return []byte("null"), nil
	}
	return json.Marshal(r)
}

// AuditSink receives structured audit records produced by supervised actions.
// Implementations MUST NOT mutate gate outcomes. Any error returned by an
// audit sink is logged without altering the gate outcome.
type AuditSink interface {
	Record(ctx context.Context, record AuditRecord) error
}

// AuditSinkFunc allows standard functions to satisfy the AuditSink interface.
type AuditSinkFunc func(ctx context.Context, record AuditRecord) error

// Record executes the underlying function.
func (f AuditSinkFunc) Record(ctx context.Context, record AuditRecord) error {
	return f(ctx, record)
}

// appendSteps recursively flattens derivation steps into RuleInference
// entries (outermost derivation first).
func (e *Explanation) appendSteps(trail *AuditTrail) {
	for _, d := range e.Derivations {
		trail.AddRule(d.Predicate, d.ruleText(), "", d.Predicate, d.Tier, d.Bindings)
		for i := range d.Children {
			sub := Explanation{Derivations: d.Children[i : i+1]}
			sub.appendSteps(trail)
		}
	}
}

// ruleText returns the rule text for a step, falling back to the fact
// itself for EDB facts.
func (d DerivationStep) ruleText() string {
	if d.Rule != "" {
		return d.Rule
	}
	return d.Fact
}

// String renders the derivation forest as a human-readable tree, e.g.:
//
//	halt("Req","blocked","T1")
//	└─ via rule [T1]: halt(Req,M,"T1") :- risky(Req), violation_msg(M).
//	   ├─ risky("Req")  [fact]
//	   └─ violation_msg("blocked")
//	      └─ via rule [Unknown]: violation_msg(M) :- ...
func (e *Explanation) String() string {
	if e == nil {
		return "<nil explanation>"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "query: %s\n", e.Query)
	if !e.Outcome || len(e.Derivations) == 0 {
		sb.WriteString("no derivations (fact not derivable)\n")
		return sb.String()
	}
	for _, d := range e.Derivations {
		renderStep(&sb, d, "")
	}
	return sb.String()
}

func renderStep(sb *strings.Builder, d DerivationStep, indent string) {
	fmt.Fprintf(sb, "%s%s\n", indent, d.Fact)
	switch d.Via {
	case ViaFact:
		fmt.Fprintf(sb, "%s└─ [base fact]\n", indent)
		return
	case ViaDepthLimit:
		fmt.Fprintf(sb, "%s└─ [proof truncated at depth limit]\n", indent)
		return
	case ViaNegation, ViaBuiltin:
		fmt.Fprintf(sb, "%s└─ [%s]\n", indent, d.Via)
		return
	}
	childIndent := indent + "  "
	if d.Rule != "" {
		fmt.Fprintf(sb, "%s└─ via rule [%s]: %s\n", childIndent, d.Tier, d.Rule)
	}
	for _, c := range d.Children {
		renderStep(sb, c, childIndent)
	}
}
