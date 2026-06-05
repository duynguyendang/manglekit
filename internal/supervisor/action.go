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
	genePool ports.GenePoolPort
}

func New(inner Action, verifier ports.ReasoningPort, genePool ports.GenePoolPort) *SupervisedAction {
	return &SupervisedAction{
		inner:    inner,
		verifier: verifier,
		genePool: genePool,
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

	// 2. Load active system rules
	var activeGenes []domain.DomainGene
	for gene := range g.genePool.ActiveGenes(ctx, intent) {
		activeGenes = append(activeGenes, *gene)
	}

	// 3. Assess (Shadow Audit)
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

	res, err := g.verifier.VerifyAtoms(ctx, atoms, activeGenes)
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
		return envelope, core.NewPolicyViolationError(
			string(res.ViolationTier),
			res.ConflictPath,
			"Supervisor Pre-Flight check failed.",
			"",
		)
	}

	// 4. Act (Execute Inner)
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

	// 5. Reflect (Post-Execution Validation)
	// Flatten the result output to facts using the engine's output entity ID
	outFacts := g.flattenToQuads(core.EntityOutput, result.Payload)
	// Optional: g.verifier.VerifyAtoms(ctx, outFacts...)

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
