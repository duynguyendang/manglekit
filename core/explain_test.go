package core

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProofBundle_DeterministicJSON(t *testing.T) {
	fixedTime := time.Date(2026, 11, 11, 8, 0, 0, 0, time.UTC)
	expl := &Explanation{
		Query:   `halt("Req", Reason, Tier)`,
		Outcome: true,
		Derivations: []DerivationStep{
			{
				Fact:      `halt("Req", "unauthorized transfer", "T1")`,
				Rule:      `halt(Req, Reason, "T1") :- action(Req, "transfer"), amount(Req, A), A > 1000.`,
				Predicate: "halt",
				Tier:      TierT1_Governance,
				Bindings: map[string]string{
					"Req":    "Req",
					"Reason": "unauthorized transfer",
					"Tier":   "T1",
					"A":      "5000",
				},
				Via: ViaRule,
				Children: []DerivationStep{
					{
						Fact:      `action("Req", "transfer")`,
						Predicate: "action",
						Via:       ViaFact,
					},
					{
						Fact:      `amount("Req", 5000)`,
						Predicate: "amount",
						Via:       ViaFact,
					},
				},
			},
		},
	}

	bundle := &ProofBundle{
		DecisionID:  "00000000-0000-0000-0000-000000000001",
		PolicyHash:  "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		Timestamp:   fixedTime,
		ActionName:  "financial_transfer",
		Outcome:     AuditOutcomeHalt,
		Tier:        "T1",
		Explanation: expl,
	}

	// Verify deterministic serialization across repeated calls
	firstJSON, err := bundle.ToJSON()
	require.NoError(t, err)

	for i := 0; i < 50; i++ {
		nextJSON, err := bundle.ToJSON()
		require.NoError(t, err)
		assert.True(t, bytes.Equal(firstJSON, nextJSON), "ProofBundle JSON export must be 100% deterministic")
	}

	// Verify unmarshal round-trip
	parsed, err := ProofBundleFromJSON(firstJSON)
	require.NoError(t, err)
	assert.Equal(t, bundle.DecisionID, parsed.DecisionID)
	assert.Equal(t, bundle.PolicyHash, parsed.PolicyHash)
	assert.Equal(t, bundle.ActionName, parsed.ActionName)
	assert.Equal(t, bundle.Outcome, parsed.Outcome)
	assert.Equal(t, bundle.Tier, parsed.Tier)
	require.NotNil(t, parsed.Explanation)
	assert.Equal(t, bundle.Explanation.Query, parsed.Explanation.Query)
	require.Len(t, parsed.Explanation.Derivations, 1)
	assert.Equal(t, "unauthorized transfer", parsed.Explanation.Derivations[0].Bindings["Reason"])
	assert.Equal(t, "5000", parsed.Explanation.Derivations[0].Bindings["A"])
}

func TestAuditTrail_ToExplanation(t *testing.T) {
	trail := NewAuditTrail("engine-01", `halt("Req", Msg, Tier)`)
	trail.AddRule("rule_prompt_inject", `halt(Req, "injection", "T0") :- prompt(Req, P).`, "", "halt", TierT0_Axiom, map[string]string{
		"Req": "Req",
		"Msg": "injection",
	})

	expl := trail.ToExplanation()
	require.NotNil(t, expl)
	assert.Equal(t, trail.Query, expl.Query)
	assert.True(t, expl.Outcome)
	require.Len(t, expl.Derivations, 1)
	assert.Equal(t, "rule_prompt_inject", expl.Derivations[0].Fact)
	assert.Equal(t, TierT0_Axiom, expl.Derivations[0].Tier)
	assert.Equal(t, "injection", expl.Derivations[0].Bindings["Msg"])

	// Empty trail
	emptyTrail := NewAuditTrail("engine-01", "query")
	emptyExpl := emptyTrail.ToExplanation()
	require.NotNil(t, emptyExpl)
	assert.False(t, emptyExpl.Outcome)
	assert.Empty(t, emptyExpl.Derivations)

	// Nil trail
	var nilTrail *AuditTrail
	assert.Nil(t, nilTrail.ToExplanation())
}

func TestAuditRecord_SerializationAndSinkFunc(t *testing.T) {
	record := AuditRecord{
		Phase:       PhasePreCheck,
		ActionName:  "shell_exec",
		Outcome:     AuditOutcomeAllow,
		DecisionID:  "dec-123",
		PolicyHash:  "abc123hash",
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Tier:        "",
		RuleID:      "",
		Reasons:     []string{"pre-check passed"},
		ProofBundle: nil,
	}

	data, err := record.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, string(data), `"phase":"pre-check"`)
	assert.Contains(t, string(data), `"outcome":"ALLOW"`)
	assert.Contains(t, string(data), `"action_name":"shell_exec"`)

	// Test AuditSinkFunc
	var captured AuditRecord
	sink := AuditSinkFunc(func(ctx context.Context, rec AuditRecord) error {
		captured = rec
		return nil
	})

	err = sink.Record(context.Background(), record)
	require.NoError(t, err)
	assert.Equal(t, "shell_exec", captured.ActionName)
}
