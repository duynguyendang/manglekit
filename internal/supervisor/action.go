package supervisor

import (
	"context"
	"fmt"
	"reflect"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/core/domain"
	"github.com/duynguyendang/manglekit/internal/core/ports"
)

// Action defines the inner, unprotected execution capability.
type Action interface {
	Execute(ctx context.Context, input domain.Envelope) (domain.Envelope, error)
}

// SupervisedAction wraps an inner capability with the Zero-Trust Gatekeeper.
// LLD 6.1: Intercepts execution, shadows audit, and reflects on output.
type SupervisedAction struct {
	inner    Action
	verifier ports.ReasoningPort
	logger   core.Logger
}

// New wraps an inner capability with the Zero-Trust Gatekeeper.
// The pre-flight gate is always fail-closed: verifier errors and Tier-0/1
// violations block execution.
func New(inner Action, verifier ports.ReasoningPort) *SupervisedAction {
	return &SupervisedAction{
		inner:    inner,
		verifier: verifier,
		logger:   core.NopLogger{},
	}
}

// executeInternal wraps the payload in an Envelope and performs the shadow audit.
func (g *SupervisedAction) ExecuteInternal(ctx context.Context, intent domain.IntentStr, input any) (domain.Envelope, error) {
	// 1. Flatten payload to facts using the engine's entity ID
	facts := g.flattenToQuads("Req", input)

	envelope := domain.Envelope{
		Payload:      input,
		ContextFacts: facts,
	}

	// 2. Assess (Shadow Audit)
	// Instead of verifying a Plan, we verify the literal execution payload facts
	// against the loaded system axioms to ensure a catastrophic safety policy
	// isn't violated before we touch the external system.

	// Convert Quads to Atoms for Mangle verification (temp adapter)
	atoms := make([]domain.Atom, len(facts))
	for i, q := range facts {
		atoms[i] = domain.Atom{
			Subject:   q.Subject,
			Predicate: q.Predicate,
			Object:    q.Object,
			Weight:    1.0,
		}
	}

	res, err := g.verifier.VerifyAtoms(ctx, atoms)
	if err != nil {
		return domain.Envelope{}, &core.SupervisorError{Reason: err}
	}

	// Capture the audit trail from the per-call result into a local.
	// This must happen before inner.Execute (which may take seconds for LLMs),
	// ensuring the trail stays on the call stack, not on a shared object.
	preFlightTrail := res.Trail

	if !res.Pass && (res.ViolationTier == domain.Tier0Kernel || res.ViolationTier == domain.Tier1Admin) {
		envelope.Violations = append(envelope.Violations, core.ViolationRule{
			RuleID:      res.ConflictPath,
			Description: "Supervisor Pre-Flight check failed.",
			Severity:    0,
		})
		return envelope, policyViolationFromResult(ctx, res, "Supervisor Pre-Flight check failed.")
	}

	// 3. Act (Execute Inner)
	result, err := g.inner.Execute(ctx, envelope)
	if err != nil {
		return domain.Envelope{}, err
	}

	// Propagate the audit trail from the pre-flight check to the result.
	// preFlightTrail is a local variable captured before inner.Execute —
	// no shared state, no race, no cross-request leakage.
	if preFlightTrail != nil {
		if result.Metadata == nil {
			result.Metadata = make(map[string]any)
		}
		result.Metadata["manglekit.audit_trail"] = preFlightTrail
	}

	// 4. Reflect (Post-Execution Validation)
	// Flatten the result output to facts using the engine's output entity ID
	outFacts := g.flattenToQuads(core.EntityOutput, result.Payload)
	// Convert output quads to atoms for verification
	outAtoms := make([]domain.Atom, len(outFacts))
	for i, q := range outFacts {
		outAtoms[i] = domain.Atom{
			Subject:   q.Subject,
			Predicate: q.Predicate,
			Object:    q.Object,
			Weight:    1.0,
		}
	}

	// Run post-check: thread the output entity ID so VerifyAtoms
	// constructs action_operation("Output", ActionName) instead of
	// action_operation("Req", ActionName).
	postCheckPC := preCheckFromContext(ctx)
	if postCheckPC != nil {
		// Clone for post-check with output entity
		postCheckPC = &preCheckContext{
			actionName: postCheckPC.actionName,
			metadata:   postCheckPC.metadata,
			labels:     postCheckPC.labels,
			facts:      postCheckPC.facts,
			entityID:   core.EntityOutput,
		}
		postCtx := withPreCheckContext(ctx, postCheckPC)
		// Post-check (Reflect) is fail-closed, consistent with the
		// pre-check: a verifier error blocks the result instead of
		// silently passing it through (ADR-001; enforcement contract).
		res, err := g.verifier.VerifyAtoms(postCtx, outAtoms)
		if err != nil {
			return domain.Envelope{}, &core.SupervisorError{
				Reason: fmt.Errorf("post-check (Reflect) verifier error: %w", err),
			}
		}
		if res != nil && !res.Pass {
			// Tier semantics mirror the pre-flight gate (P0.7): explicit
			// Tier-2/3 (playbook/user) rules are advisory — they do not
			// block. Unknown/absent tiers block (fail-closed default).
			// Verifier errors above always block (ADR-001).
			if res.ViolationTier == domain.Tier0Kernel || res.ViolationTier == domain.Tier1Admin {
				envelope.Violations = append(envelope.Violations, core.ViolationRule{
					RuleID:      res.ConflictPath,
					Description: "Supervisor post-check (Reflect) failed.",
					Severity:    0,
				})
				return domain.Envelope{}, policyViolationFromResult(ctx, res, "Supervisor post-check (Reflect) failed.")
			}
			if g.logger != nil {
				g.logger.Warn("post-check soft violation (advisory tier)",
					"tier", string(res.ViolationTier), "rule", res.ConflictPath)
			}
		}
	}

	result.ContextFacts = append(result.ContextFacts, outFacts...)

	return result, nil
}

// flattenToQuads implements "Zero-Config Reflection" mapping arbitrary Go structs to Datalog Quads via the `mangle` tag.
func (g *SupervisedAction) flattenToQuads(subjectID string, v any) []domain.Quad {
	var quads []domain.Quad
	val := reflect.ValueOf(v)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return quads
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("mangle")
		if tag == "" {
			continue
		}

		fieldVal := val.Field(i).Interface()
		quads = append(quads, domain.Quad{
			Subject:   subjectID,
			Predicate: tag,
			Object:    fmt.Sprint(fieldVal),
			Graph:     "temporal_context",
		})
	}
	return quads
}

// coreTierName maps the gate's TrustTier back to the core tier vocabulary
// ("T0".."T3") used inside policy rules, so deny errors report one consistent
// vocabulary regardless of whether a matching trail rule is found.
func coreTierName(t domain.TrustTier) string {
	switch t {
	case domain.Tier0Kernel:
		return string(core.TierT0_Axiom)
	case domain.Tier1Admin:
		return string(core.TierT1_Governance)
	case domain.Tier2AI:
		return string(core.TierT2_Playbook)
	case domain.Tier3User:
		return string(core.TierT3_User)
	default:
		return string(t)
	}
}

// policyViolationFromResult builds the structured PolicyViolationError the
// supervisor returns on a gate block. Tier and MatchedRule always describe
// the VIOLATION's own blocking tier: when several rules of different tiers
// halt at once, the rule picked for provenance is the one whose tier maps to
// res.ViolationTier (the tier that actually caused the block) — never
// "first rule wins", which could report "blocked at tier T3" for a T1 block.
func policyViolationFromResult(ctx context.Context, res *domain.AuditResult, description string) *core.PolicyViolationError {
	err := core.NewPolicyViolationError(
		coreTierName(res.ViolationTier),
		res.ConflictPath,
		description,
		"",
	)
	if pc := preCheckFromContext(ctx); pc != nil {
		err.ActionName = pc.actionName
	}
	if res.Trail != nil {
		for _, rule := range res.Trail.MatchedRules {
			if rule.Definition != "" && rule.Tier != "" && mapCoreTier(rule.Tier) == res.ViolationTier {
				err.MatchedRule = rule.Definition
				err.Tier = string(rule.Tier)
				break
			}
		}
		if err.MatchedRule == "" {
			// No tier-matching rule (nil-trail post-checks, tierless legacy
			// halts): keep the first definition for context but preserve the
			// violation's own tier.
			for _, rule := range res.Trail.MatchedRules {
				if rule.Definition != "" {
					err.MatchedRule = rule.Definition
					break
				}
			}
		}
	}
	return err
}
