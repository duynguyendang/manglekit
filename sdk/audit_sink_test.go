package sdk_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const auditTestPolicy = `
% Declarations for dynamic EDB predicates not in std.mg
Decl amount(Req, A).
Decl secret(Output, S).

% Allow policy with pre-check and post-check guardrails
halt("Req", "blocked by pre-check", "T1") :- action_operation("Req", "bad_pre_action").
halt("Req", "amount too high", "T1") :- action_operation("Req", "transfer"), amount("Req", A), :gt(A, 1000).

halt("Output", "blocked by post-check", "T1") :- action_operation("Output", "bad_post_action").
halt("Output", "secret leaked in output", "T0") :- secret("Output", "leak").
`

type testTransferPayload struct {
	Amount int `mangle:"amount"`
}

type testOutputSecretPayload struct {
	Secret string `mangle:"secret"`
}

func writeAuditPolicy(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "policy.mg")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

type testSyncSink struct {
	mu      sync.Mutex
	records []core.AuditRecord
}

func (s *testSyncSink) Record(_ context.Context, rec core.AuditRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, rec)
	return nil
}

func (s *testSyncSink) Records() []core.AuditRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := make([]core.AuditRecord, len(s.records))
	copy(copied, s.records)
	return copied
}

func TestAuditSink_AllowedExecution(t *testing.T) {
	ctx := context.Background()
	policyPath := writeAuditPolicy(t, auditTestPolicy)

	sink := &testSyncSink{}
	client, err := sdk.NewClient(ctx,
		sdk.WithPolicyPath(policyPath),
		sdk.WithAuditSink(sink),
	)
	require.NoError(t, err)
	require.NotEmpty(t, client.PolicyHash())

	// Define allowed action
	act := manglekit.Define(client, "transfer", func(ctx context.Context, in testTransferPayload) (string, error) {
		return "success", nil
	})

	// Execute allowed transfer (Amount: 500 <= 1000)
	resp, err := act.Run(ctx, testTransferPayload{Amount: 500})
	require.NoError(t, err)
	assert.Equal(t, "success", resp)

	records := sink.Records()
	require.Len(t, records, 2, "must receive pre-check and post-check records")

	// Pre-check record
	pre := records[0]
	assert.Equal(t, core.PhasePreCheck, pre.Phase)
	assert.Equal(t, "transfer", pre.ActionName)
	assert.Equal(t, core.AuditOutcomeAllow, pre.Outcome)
	assert.NotEmpty(t, pre.DecisionID)
	assert.Equal(t, client.PolicyHash(), pre.PolicyHash)
	assert.NotEmpty(t, pre.Timestamp)

	// Post-check record
	post := records[1]
	assert.Equal(t, core.PhasePostCheck, post.Phase)
	assert.Equal(t, "transfer", post.ActionName)
	assert.Equal(t, core.AuditOutcomeAllow, post.Outcome)
	assert.Equal(t, pre.DecisionID, post.DecisionID, "same action invocation must share DecisionID")
	assert.Equal(t, client.PolicyHash(), post.PolicyHash)
}

func TestAuditSink_PreCheckHaltedExecution(t *testing.T) {
	ctx := context.Background()
	policyPath := writeAuditPolicy(t, auditTestPolicy)

	sink := &testSyncSink{}
	client, err := sdk.NewClient(ctx,
		sdk.WithPolicyPath(policyPath),
		sdk.WithAuditSink(sink),
	)
	require.NoError(t, err)

	act := manglekit.Define(client, "transfer", func(ctx context.Context, in testTransferPayload) (string, error) {
		return "should not be called", nil
	})

	// Execute transfer that violates pre-check (Amount: 2500 > 1000)
	_, err = act.Run(ctx, testTransferPayload{Amount: 2500})
	require.Error(t, err)

	var polErr *core.PolicyViolationError
	require.True(t, errors.As(err, &polErr))
	assert.Equal(t, "T1", polErr.Tier)

	records := sink.Records()
	require.Len(t, records, 1, "only pre-check record must be emitted on pre-check halt")

	pre := records[0]
	assert.Equal(t, core.PhasePreCheck, pre.Phase)
	assert.Equal(t, "transfer", pre.ActionName)
	assert.Equal(t, core.AuditOutcomeHalt, pre.Outcome)
	assert.Equal(t, "T1", pre.Tier)
	assert.NotEmpty(t, pre.RuleID)
	assert.NotEmpty(t, pre.DecisionID)
	assert.Equal(t, client.PolicyHash(), pre.PolicyHash)

	// Verify ProofBundle is attached
	require.NotNil(t, pre.ProofBundle)
	assert.Equal(t, pre.DecisionID, pre.ProofBundle.DecisionID)
	assert.Equal(t, client.PolicyHash(), pre.ProofBundle.PolicyHash)
	assert.Equal(t, "transfer", pre.ProofBundle.ActionName)
	assert.Equal(t, core.AuditOutcomeHalt, pre.ProofBundle.Outcome)
	assert.Equal(t, "T1", pre.ProofBundle.Tier)

	// Verify ProofBundle deterministic JSON
	jsonBytes, err := pre.ProofBundle.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"outcome":"HALT"`)
	assert.Contains(t, string(jsonBytes), `"tier":"T1"`)
}

func TestAuditSink_PostCheckHaltedExecution(t *testing.T) {
	ctx := context.Background()
	policyPath := writeAuditPolicy(t, auditTestPolicy)

	sink := &testSyncSink{}
	client, err := sdk.NewClient(ctx,
		sdk.WithPolicyPath(policyPath),
		sdk.WithAuditSink(sink),
	)
	require.NoError(t, err)

	// Action passes pre-check but produces output violating post-check
	act := manglekit.Define(client, "bad_post_action", func(ctx context.Context, in string) (testOutputSecretPayload, error) {
		return testOutputSecretPayload{Secret: "leak"}, nil
	})

	_, err = act.Run(ctx, "normal-input")
	require.Error(t, err)

	var polErr *core.PolicyViolationError
	require.True(t, errors.As(err, &polErr))

	records := sink.Records()
	require.Len(t, records, 2, "pre-check pass + post-check halt must be emitted")

	// Pre-check passed
	assert.Equal(t, core.PhasePreCheck, records[0].Phase)
	assert.Equal(t, core.AuditOutcomeAllow, records[0].Outcome)

	// Post-check halted
	post := records[1]
	assert.Equal(t, core.PhasePostCheck, post.Phase)
	assert.Equal(t, core.AuditOutcomeHalt, post.Outcome)
	assert.Equal(t, records[0].DecisionID, post.DecisionID)
	require.NotNil(t, post.ProofBundle)
	assert.Equal(t, core.AuditOutcomeHalt, post.ProofBundle.Outcome)
}

func TestAuditSink_PanicOrErrorDoesNotMutateGateOutcome(t *testing.T) {
	ctx := context.Background()
	policyPath := writeAuditPolicy(t, auditTestPolicy)

	// Sink that panics on every call
	panickingSink := core.AuditSinkFunc(func(ctx context.Context, rec core.AuditRecord) error {
		panic("sink exploded intentionally")
	})

	client, err := sdk.NewClient(ctx,
		sdk.WithPolicyPath(policyPath),
		sdk.WithAuditSink(panickingSink),
	)
	require.NoError(t, err)

	act := manglekit.Define(client, "transfer", func(ctx context.Context, in testTransferPayload) (string, error) {
		return "ok", nil
	})

	// Execution should still succeed without crashing
	resp, err := act.Run(ctx, testTransferPayload{Amount: 50})
	require.NoError(t, err)
	assert.Equal(t, "ok", resp)

	// And denied execution still fails closed with PolicyViolationError
	_, err = act.Run(ctx, testTransferPayload{Amount: 5000})
	require.Error(t, err)
	var polErr *core.PolicyViolationError
	require.True(t, errors.As(err, &polErr))
}

func TestClient_ExplainAction(t *testing.T) {
	ctx := context.Background()
	policyPath := writeAuditPolicy(t, auditTestPolicy)

	client, err := sdk.NewClient(ctx, sdk.WithPolicyPath(policyPath))
	require.NoError(t, err)

	// Test action that halts
	bundle, err := client.ExplainAction(ctx, "bad_pre_action", nil)
	require.NoError(t, err)
	require.NotNil(t, bundle)
	assert.Equal(t, "bad_pre_action", bundle.ActionName)
	assert.Equal(t, core.AuditOutcomeHalt, bundle.Outcome)
	assert.Equal(t, "T1", bundle.Tier)
	assert.Equal(t, client.PolicyHash(), bundle.PolicyHash)
	require.NotNil(t, bundle.Explanation)
	assert.True(t, bundle.Explanation.Outcome)

	// Test action that passes
	allowBundle, err := client.ExplainAction(ctx, "safe_action", nil)
	require.NoError(t, err)
	require.NotNil(t, allowBundle)
	assert.Equal(t, "safe_action", allowBundle.ActionName)
	assert.Equal(t, core.AuditOutcomeAllow, allowBundle.Outcome)
}
