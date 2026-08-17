package engine

import (
	"context"
	"fmt"

	"codeberg.org/TauCeti/mangle-go/ast"
	mangleengine "codeberg.org/TauCeti/mangle-go/engine"
	"codeberg.org/TauCeti/mangle-go/factstore"
	"codeberg.org/TauCeti/mangle-go/parse"
	"github.com/duynguyendang/manglekit/core"
)

// Bounds on proof reconstruction so a pathological program cannot make
// explanation itself unbounded.
const (
	maxExplainDepth = 32
	maxExplainSteps = 1000
)

// Explain evaluates the query against the current policy plus the given
// temporary facts and returns a structured derivation tree (proof) for
// every matching fact. For a deny/halt decision the derivation reproduces
// the exact rule instantiations (grounded atoms per hop) rather than
// filename or predicate-name heuristics.
func (e *PolicyEngine) Explain(ctx context.Context, facts []string, queryStr string) (*core.Explanation, error) {
	if e.runtime == nil {
		return nil, fmt.Errorf("runtime not initialized")
	}

	var atomFacts []ast.Atom
	for _, f := range facts {
		atom, err := e.parseCachedAtom(f)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fact '%s': %w", f, err)
		}
		atomFacts = append(atomFacts, atom)
	}

	return e.runtime.Explain(ctx, atomFacts, queryStr)
}

// Explain runs queryStr against the evaluated store (base facts + IDB +
// the request-scoped facts) and reconstructs a derivation tree for each
// matching fact by backward-chaining over the loaded rules.
func (r *MangleRuntime) Explain(ctx context.Context, facts []ast.Atom, queryStr string) (*core.Explanation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	evalOpts := r.buildEvalOptionsLockedSnapshot()
	if r.maxCreatedFacts > 0 {
		evalOpts = append(evalOpts, mangleengine.WithCreatedFactLimit(r.maxCreatedFacts))
	}

	workingStore, err := r.evaluatedWorkingStore(ctx, facts, evalOpts)
	if err != nil {
		return nil, err
	}

	// Snapshot the program (replaced wholesale on reload, never mutated
	// in place), so the proof is consistent for the duration of the walk.
	r.mu.RLock()
	programInfo := r.programInfo
	r.mu.RUnlock()
	if programInfo == nil {
		return nil, fmt.Errorf("runtime not initialized")
	}

	queryAtom, err := parse.Atom(queryStr)
	if err != nil {
		return nil, err
	}

	expl := &core.Explanation{Query: queryStr}
	p := &prover{store: *workingStore, rules: programInfo.Rules}

	err = workingStore.GetFacts(ast.NewQuery(queryAtom.Predicate), func(factAtom ast.Atom) error {
		if len(factAtom.Args) != len(queryAtom.Args) {
			return nil
		}
		for i, qa := range queryAtom.Args {
			if _, isVar := qa.(ast.Variable); !isVar && !qa.Equals(factAtom.Args[i]) {
				return nil
			}
		}
		if step, ok := p.prove(factAtom, 0); ok {
			expl.Outcome = true
			expl.Derivations = append(expl.Derivations, step)
		} else {
			// The fact is in the store even if reconstruction failed
			// (e.g. derived via a builtin we do not model); still report
			// it as an underived match.
			expl.Outcome = true
			expl.Derivations = append(expl.Derivations, core.DerivationStep{
				Fact:      factAtom.String(),
				Predicate: factAtom.Predicate.Symbol,
				Tier:      tierFromSubst(factAtom.Args, nil),
				Via:       core.ViaFact,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("explain query failed: %w", err)
	}

	return expl, nil
}

// prover reconstructs why-provenance for facts in an evaluated (fixpoint)
// store: since the semi-naive evaluation already computed the IDB, proving
// a fact reduces to finding a rule whose head unifies and whose premises
// are all derivable in the same store.
type prover struct {
	store factstore.SimpleInMemoryStore
	rules []ast.Clause
	steps int
}

// prove returns a derivation tree for fact, or ok=false when no rule
// fires and the fact is not a base fact.
func (p *prover) prove(fact ast.Atom, depth int) (core.DerivationStep, bool) {
	p.steps++
	if p.steps > maxExplainSteps {
		return core.DerivationStep{Fact: fact.String(), Predicate: fact.Predicate.Symbol, Via: core.ViaDepthLimit}, true
	}
	if depth >= maxExplainDepth {
		return core.DerivationStep{Fact: fact.String(), Predicate: fact.Predicate.Symbol, Via: core.ViaDepthLimit}, true
	}

	for _, rule := range p.rules {
		if rule.Head.Predicate != fact.Predicate || len(rule.Head.Args) != len(fact.Args) {
			continue
		}
		if len(rule.Premises) == 0 {
			continue // fact clauses are covered by the base-fact path
		}
		subst := make(map[ast.Variable]ast.BaseTerm)
		if !unifyAtomArgs(rule.Head.Args, fact.Args, subst) {
			continue
		}
		children, ok := p.solvePremises(rule.Premises, 0, subst, depth)
		if !ok {
			continue
		}
		return core.DerivationStep{
			Fact:      fact.String(),
			Rule:      rule.String(),
			Predicate: fact.Predicate.Symbol,
			Tier:      tierFromSubst(rule.Head.Args, subst),
			Bindings:  bindingsToStrings(subst),
			Via:       core.ViaRule,
			Children:  children,
		}, true
	}

	// Base fact (EDB): present in the store without a firing rule.
	if p.store.Contains(fact) {
		return core.DerivationStep{
			Fact:      fact.String(),
			Predicate: fact.Predicate.Symbol,
			Tier:      tierFromSubst(fact.Args, nil),
			Via:       core.ViaFact,
		}, true
	}

	return core.DerivationStep{}, false
}

// solvePremises solves premises[i:] under subst, returning the grounded
// child derivations. Backtracks over candidate facts in the store; subst
// copies keep failed branches from polluting each other.
func (p *prover) solvePremises(premises []ast.Term, i int, subst map[ast.Variable]ast.BaseTerm, depth int) ([]core.DerivationStep, bool) {
	if i >= len(premises) {
		return nil, true
	}

	switch premise := premises[i].(type) {
	case ast.Atom:
		ground := applySubstToArgs(premise.Args, subst)

		// Fully ground premise: containment check, then recurse.
		if isGround(ground) {
			pattern := ast.Atom{Predicate: premise.Predicate, Args: ground}
			if !p.store.Contains(pattern) {
				return nil, false
			}
			child, ok := p.prove(pattern, depth+1)
			if !ok {
				return nil, false
			}
			rest, ok := p.solvePremises(premises, i+1, subst, depth)
			if !ok {
				return nil, false
			}
			return append([]core.DerivationStep{child}, rest...), true
		}

		// Open premise: try every store fact of this predicate.
		solved := false
		var children []core.DerivationStep
		candidateErr := p.store.GetFacts(ast.NewQuery(premise.Predicate), func(candidate ast.Atom) error {
			if solved || len(candidate.Args) != len(premise.Args) {
				return nil
			}
			trial := copySubst(subst)
			if !unifyAtomArgs(ground, candidate.Args, trial) {
				return nil
			}
			child, ok := p.prove(candidate, depth+1)
			if !ok {
				return nil
			}
			rest, ok := p.solvePremises(premises, i+1, trial, depth)
			if !ok {
				return nil
			}
			children = append([]core.DerivationStep{child}, rest...)
			solved = true
			return nil
		})
		if candidateErr != nil || !solved {
			return nil, false
		}
		return children, true

	case ast.NegAtom:
		ground := ast.NegAtom{Atom: ast.Atom{Predicate: premise.Atom.Predicate, Args: applySubstToArgs(premise.Atom.Args, subst)}}
		if ground.IsGround() && p.store.Contains(ground.Atom) {
			return nil, false
		}
		child := core.DerivationStep{Fact: ground.String(), Predicate: ground.Atom.Predicate.Symbol, Via: core.ViaNegation}
		rest, ok := p.solvePremises(premises, i+1, subst, depth)
		if !ok {
			return nil, false
		}
		return append([]core.DerivationStep{child}, rest...), true

	case ast.Eq:
		left := applySubstTerm(premise.Left, subst)
		right := applySubstTerm(premise.Right, subst)
		if isGroundTerm(left) && isGroundTerm(right) {
			if !left.Equals(right) {
				return nil, false
			}
		}
		child := core.DerivationStep{Fact: fmt.Sprintf("%s = %s", left, right), Via: core.ViaBuiltin}
		rest, ok := p.solvePremises(premises, i+1, subst, depth)
		if !ok {
			return nil, false
		}
		return append([]core.DerivationStep{child}, rest...), true

	case ast.Ineq:
		left := applySubstTerm(premise.Left, subst)
		right := applySubstTerm(premise.Right, subst)
		if isGroundTerm(left) && isGroundTerm(right) {
			if left.Equals(right) {
				return nil, false
			}
		}
		child := core.DerivationStep{Fact: fmt.Sprintf("%s != %s", left, right), Via: core.ViaBuiltin}
		rest, ok := p.solvePremises(premises, i+1, subst, depth)
		if !ok {
			return nil, false
		}
		return append([]core.DerivationStep{child}, rest...), true

	default:
		// ApplyFn / temporal literals and other constructs: record as an
		// opaque builtin step and continue; the fact was already derived
		// by the fixpoint evaluation, so the remainder of the proof is
		// still valid.
		child := core.DerivationStep{Fact: premises[i].String(), Via: core.ViaBuiltin}
		rest, ok := p.solvePremises(premises, i+1, subst, depth)
		if !ok {
			return nil, false
		}
		return append([]core.DerivationStep{child}, rest...), true
	}
}

// unifyAtomArgs unifies pattern args against ground fact args, extending
// subst. Variables in pattern are bound; a repeated variable must bind
// consistently.
func unifyAtomArgs(pattern, actual []ast.BaseTerm, subst map[ast.Variable]ast.BaseTerm) bool {
	for i, pa := range pattern {
		if !unifyTerm(pa, actual[i], subst) {
			return false
		}
	}
	return true
}

func unifyTerm(pattern, actual ast.BaseTerm, subst map[ast.Variable]ast.BaseTerm) bool {
	if v, ok := pattern.(ast.Variable); ok {
		if bound, ok := subst[v]; ok {
			return bound.Equals(actual)
		}
		subst[v] = actual
		return true
	}
	return pattern.Equals(actual)
}

func copySubst(subst map[ast.Variable]ast.BaseTerm) map[ast.Variable]ast.BaseTerm {
	out := make(map[ast.Variable]ast.BaseTerm, len(subst))
	for k, v := range subst {
		out[k] = v
	}
	return out
}

func applySubstToArgs(args []ast.BaseTerm, subst map[ast.Variable]ast.BaseTerm) []ast.BaseTerm {
	out := make([]ast.BaseTerm, len(args))
	for i, a := range args {
		out[i] = applySubstTerm(a, subst)
	}
	return out
}

func applySubstTerm(t ast.BaseTerm, subst map[ast.Variable]ast.BaseTerm) ast.BaseTerm {
	if v, ok := t.(ast.Variable); ok {
		if bound, ok := subst[v]; ok {
			return bound
		}
		return v
	}
	return t
}

func isGround(args []ast.BaseTerm) bool {
	for _, a := range args {
		if !isGroundTerm(a) {
			return false
		}
	}
	return true
}

func isGroundTerm(t ast.BaseTerm) bool {
	_, isVar := t.(ast.Variable)
	return !isVar
}

// tierFromSubst extracts real tier provenance from a rule instantiation:
// if the rule head carries a Tier argument (the halt/3, retry/3, route/3
// convention) as a variable that the firing bound, or directly as a tier
// constant (e.g. halt(Req, M, "T1") :- ...), that tier is the derivation's
// tier. Returns core.TierUnknown otherwise.
func tierFromSubst(headArgs []ast.BaseTerm, subst map[ast.Variable]ast.BaseTerm) core.Tier {
	for _, a := range headArgs {
		var candidate ast.Constant
		switch arg := a.(type) {
		case ast.Variable:
			if arg.Symbol != "Tier" {
				continue
			}
			bound, ok := subst[arg].(ast.Constant)
			if !ok {
				continue
			}
			candidate = bound
		case ast.Constant:
			candidate = arg
		default:
			continue
		}
		tier := core.Tier(candidate.Symbol)
		switch tier {
		case core.TierT0_Axiom, core.TierT1_Governance, core.TierT2_Playbook, core.TierT3_User:
			return tier
		}
	}
	return core.TierUnknown
}

// bindingsToStrings renders a substitution as string bindings, compatible
// with RuleInference.Bindings and Phase 13 audit annotations.
func bindingsToStrings(subst map[ast.Variable]ast.BaseTerm) map[string]string {
	if len(subst) == 0 {
		return nil
	}
	out := make(map[string]string, len(subst))
	for v, t := range subst {
		out[v.Symbol] = t.String()
	}
	return out
}
