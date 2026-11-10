package check

import (
	"fmt"
	"strings"

	"codeberg.org/TauCeti/mangle-go/ast"
	"codeberg.org/TauCeti/mangle-go/parse"
)

type IssueSeverity string

const (
	SeverityError   IssueSeverity = "ERROR"
	SeverityWarning IssueSeverity = "WARN"
)

// LintIssue represents a static analysis warning or error in a Datalog policy.
type LintIssue struct {
	Severity IssueSeverity `json:"severity"`
	Rule     string        `json:"rule,omitempty"`
	Message  string        `json:"message"`
}

// LintPolicy analyzes Datalog policy source for common pitfalls:
// 1. Unbound variables in negated atoms (fails stratification / mode check).
// 2. Wildcard variables in negated atoms.
// 3. Unreachable/dead rules referencing predicates that are never declared or derived.
func LintPolicy(content string) ([]LintIssue, error) {
	unit, err := parse.Unit(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	var issues []LintIssue

	headPreds := make(map[string]bool)
	declaredPreds := make(map[string]bool)

	for _, d := range unit.Decls {
		declaredPreds[d.DeclaredAtom.Predicate.Symbol] = true
	}

	for _, c := range unit.Clauses {
		headPreds[c.Head.Predicate.Symbol] = true
	}

	for _, c := range unit.Clauses {
		positiveVars := make(map[string]bool)

		// Collect variables bound by positive atoms in the body
		for _, p := range c.Premises {
			if atom, ok := p.(ast.Atom); ok {
				for _, arg := range atom.Args {
					if v, isVar := arg.(ast.Variable); isVar && v.Symbol != "_" {
						positiveVars[v.Symbol] = true
					}
				}
			}
		}

		// Inspect premises for unbound negated variables or dead references
		for _, p := range c.Premises {
			switch prem := p.(type) {
			case ast.NegAtom:
				for _, arg := range prem.Atom.Args {
					if v, isVar := arg.(ast.Variable); isVar {
						if v.Symbol == "_" {
							issues = append(issues, LintIssue{
								Severity: SeverityWarning,
								Rule:     c.String(),
								Message:  fmt.Sprintf("negated atom !%s contains wildcard '_' which may fail evaluation; define a unary helper predicate instead", prem.Atom.Predicate.Symbol),
							})
						} else if !positiveVars[v.Symbol] {
							issues = append(issues, LintIssue{
								Severity: SeverityError,
								Rule:     c.String(),
								Message:  fmt.Sprintf("variable %s in negated atom !%s is ungrounded (not bound by any positive atom in the rule body)", v.Symbol, prem.Atom.Predicate.Symbol),
							})
						}
					}
				}
			case ast.Atom:
				sym := prem.Predicate.Symbol
				// Skip builtins, standard library, and runtime-injected predicates
				if isKnownOrStandardPredicate(sym) {
					continue
				}
				if !declaredPreds[sym] && !headPreds[sym] {
					issues = append(issues, LintIssue{
						Severity: SeverityWarning,
						Rule:     c.String(),
						Message:  fmt.Sprintf("predicate %q in rule body is never declared (Decl) and never derived by any rule head", sym),
					})
				}
			}
		}
	}

	return issues, nil
}

func isKnownOrStandardPredicate(sym string) bool {
	if strings.HasPrefix(sym, ":") || strings.HasPrefix(sym, "fn:") {
		return true
	}
	switch sym {
	case "action_operation", "meta", "label", "request", "policy_anchor",
		"halt", "deny", "route", "attempt", "threshold", "threshold_pm":
		return true
	default:
		return false
	}
}
