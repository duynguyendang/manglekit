package sdk

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/google/uuid"
)

// reloader is the capability a policy engine must implement to support
// atomic hot reloads (implemented by *engine.PolicyEngine).
type reloader interface {
	ReloadPolicySource(ctx context.Context, source string) error
}

// explainer is the capability a policy engine must implement to produce
// derivation trees (proofs) for queries and decisions (implemented by
// *engine.PolicyEngine).
type explainer interface {
	Explain(ctx context.Context, facts []string, queryStr string) (*core.Explanation, error)
}

// ReloadPolicySource atomically replaces the active policy with the given
// Datalog source. The new policy is fully validated (parsed, analyzed, and
// evaluated against a copy of the current base facts) before it is swapped
// in: a failed reload returns an error and leaves the previous policy
// active; a successful reload is atomic, so concurrent gate checks observe
// either the old or the new policy, never a mix. Base facts loaded via
// LoadFacts survive the reload.
func (c *Client) ReloadPolicySource(ctx context.Context, source string) error {
	if c.engine == nil {
		return fmt.Errorf("engine not initialized")
	}
	rl, ok := c.engine.(reloader)
	if !ok {
		return fmt.Errorf("engine %T does not support hot policy reload", c.engine)
	}
	if err := rl.ReloadPolicySource(ctx, source); err != nil {
		return err
	}
	c.setPolicyHashFromBytes([]byte(source))
	return nil
}

// ReloadPolicy atomically replaces the active policy with the Datalog
// program read from path. See ReloadPolicySource for the atomicity and
// failure semantics. On success the path is remembered as the client's
// policy path (used for debugging/reloading).
func (c *Client) ReloadPolicy(ctx context.Context, path string) error {
	if path == "" {
		return fmt.Errorf("policy path cannot be empty")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read policy %q: %w", path, err)
	}
	if err := c.ReloadPolicySource(ctx, string(content)); err != nil {
		return err
	}
	c.blueprintPath = path
	c.setPolicyHashFromBytes(content)
	return nil
}

// Explain evaluates queryStr against the current policy plus the given
// temporary facts and returns a structured derivation tree (proof) for
// each matching fact: the exact rule instantiations (full rule text,
// variable bindings, grounded atoms per hop) and the governance tier
// actually carried by the derivation. Use it to explain a deny:
//
//	expl, err := client.Explain(ctx, `halt("Req", Reason, Tier)`, nil)
//
// Explanation.AuditTrail converts the proof into the same AuditTrail
// structure the gate produces.
func (c *Client) Explain(ctx context.Context, queryStr string, facts []string) (*core.Explanation, error) {
	if c.engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	ex, ok := c.engine.(explainer)
	if !ok {
		return nil, fmt.Errorf("engine %T does not support explanations", c.engine)
	}
	return ex.Explain(ctx, facts, queryStr)
}

// ExplainAction evaluates halt queries for an action and returns a canonical ProofBundle.
// If the action is halted, ProofBundle.Outcome is "HALT" with the explanation tree.
// If the action is allowed, ProofBundle.Outcome is "ALLOW".
func (c *Client) ExplainAction(ctx context.Context, actionName string, facts []string) (*core.ProofBundle, error) {
	if c.engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}
	actionFact := fmt.Sprintf("action_operation(%q, %q).", core.EntityInput, actionName)
	evalFacts := make([]string, 0, len(facts)+1)
	evalFacts = append(evalFacts, facts...)
	evalFacts = append(evalFacts, actionFact)

	queryStr := fmt.Sprintf(`halt(%q, Reason, Tier)`, core.EntityInput)
	expl, err := c.Explain(ctx, queryStr, evalFacts)
	if err != nil {
		return nil, err
	}
	outcome := core.AuditOutcomeAllow
	tier := ""
	if expl != nil && expl.Outcome && len(expl.Derivations) > 0 {
		outcome = core.AuditOutcomeHalt
		tier = string(expl.Derivations[0].Tier)
	}
	return &core.ProofBundle{
		DecisionID:  uuid.New().String(),
		PolicyHash:  c.PolicyHash(),
		Timestamp:   time.Now().UTC(),
		ActionName:  actionName,
		Outcome:     outcome,
		Tier:        tier,
		Explanation: expl,
	}, nil
}
