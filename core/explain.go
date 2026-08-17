package core

import (
	"fmt"
	"strings"
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
	Fact string
	// Rule is the full text of the rule that derived Fact. Empty for
	// base (EDB) facts.
	Rule string
	// Predicate is the head predicate name of Fact.
	Predicate string
	// Tier is the governance tier carried by the derivation (TierUnknown
	// when no tier is derivable from the rule itself).
	Tier Tier
	// Bindings maps variable names to their grounded values in this rule
	// firing (compatible with RuleInference.Bindings / Phase 13 audit
	// annotations).
	Bindings map[string]string
	// Via classifies the step: "rule", "fact", "negation", "builtin",
	// or "depth-limit" when the proof was truncated.
	Via string
	// Children are the derivations of the premises that fired the rule.
	Children []DerivationStep
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
	Query string
	// Outcome is true when at least one fact matched the query.
	Outcome bool
	// Derivations contains one derivation tree per matching fact.
	Derivations []DerivationStep
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
