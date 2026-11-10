package testcmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/duynguyendang/manglekit/cmd/mkit/commands/exitcode"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const samplePolicy = `
Decl amount(Req, Val).
Decl manager_approval(Req, Mgr).
Decl has_approval(Req).

has_approval(Req) :- manager_approval(Req, _).

halt("Req", "high_value_refund_requires_manager", "T1") :-
    action_operation("Req", "refund"),
    amount("Req", Amount),
    :gt(Amount, 1000),
    !has_approval("Req").
`

const sampleYAMLSuite = `
policy: "policy.mg"
tests:
  - name: "disallow refund over 1000 without manager approval"
    action: "refund"
    facts:
      - 'amount("Req", 1500).'
    expect: "halt"
    expected_reason: "high_value_refund_requires_manager"
    expected_tier: "T1"

  - name: "allow refund over 1000 with manager approval"
    action: "refund"
    facts:
      - 'amount("Req", 1500).'
      - 'manager_approval("Req", "alice").'
    expect: "allow"

  - name: "allow refund under threshold without approval"
    action: "refund"
    facts:
      - 'amount("Req", 500).'
    expect: "allow"
`

const failingYAMLSuite = `
policy: "policy.mg"
tests:
  - name: "this test is expected to fail"
    action: "refund"
    facts:
      - 'amount("Req", 1500).'
    expect: "allow"
`

func setupFixture(t *testing.T, policy, suite string) string {
	t.Helper()
	dir := t.TempDir()
	policyPath := filepath.Join(dir, "policy.mg")
	suitePath := filepath.Join(dir, "tests.yaml")

	require.NoError(t, os.WriteFile(policyPath, []byte(policy), 0o600))
	require.NoError(t, os.WriteFile(suitePath, []byte(suite), 0o600))
	return suitePath
}

func TestMkitTest_AllPassing(t *testing.T) {
	suitePath := setupFixture(t, samplePolicy, sampleYAMLSuite)

	var stdout, stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := RunTests(context.Background(), suitePath, "", false, "text", cmd)
	require.NoError(t, err)

	outStr := stdout.String()
	assert.Contains(t, outStr, "=== RUN   disallow refund over 1000 without manager approval")
	assert.Contains(t, outStr, "--- PASS: disallow refund over 1000 without manager approval")
	assert.Contains(t, outStr, "=== RUN   allow refund over 1000 with manager approval")
	assert.Contains(t, outStr, "--- PASS: allow refund over 1000 with manager approval")
	assert.Contains(t, outStr, "PASS: 3/3 tests passed")
}

func TestMkitTest_VerboseProofTree(t *testing.T) {
	suitePath := setupFixture(t, samplePolicy, sampleYAMLSuite)

	var stdout, stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := RunTests(context.Background(), suitePath, "", true, "text", cmd)
	require.NoError(t, err)

	outStr := stdout.String()
	assert.Contains(t, outStr, "proof tree:")
	assert.Contains(t, outStr, "1500 > 1000")
}

func TestMkitTest_FailingAssertion(t *testing.T) {
	suitePath := setupFixture(t, samplePolicy, failingYAMLSuite)

	var stdout, stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := RunTests(context.Background(), suitePath, "", false, "text", cmd)
	require.Error(t, err)

	assert.Equal(t, exitcode.PolicyDeny, exitcode.CodeFor(err))

	errStr := stderr.String()
	assert.Contains(t, errStr, "--- FAIL: this test is expected to fail")
	assert.Contains(t, errStr, "expected allow, but halt rule derived")
	assert.Contains(t, errStr, "FAIL: 1/1 tests failed")
}

func TestMkitTest_JSONOutput(t *testing.T) {
	suitePath := setupFixture(t, samplePolicy, sampleYAMLSuite)

	var stdout, stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := RunTests(context.Background(), suitePath, "", false, "json", cmd)
	require.NoError(t, err)

	outStr := stdout.String()
	assert.Contains(t, outStr, `"total": 3`)
	assert.Contains(t, outStr, `"passed": 3`)
	assert.Contains(t, outStr, `"failed": 0`)
}

func TestMkitTest_MissingPolicyErrors(t *testing.T) {
	dir := t.TempDir()
	suitePath := filepath.Join(dir, "no_policy.yaml")
	require.NoError(t, os.WriteFile(suitePath, []byte(`tests: [{name: "test", expect: "allow"}]`), 0o600))

	cmd := &cobra.Command{}
	err := RunTests(context.Background(), suitePath, "", false, "text", cmd)
	require.Error(t, err)
	assert.Equal(t, exitcode.Usage, exitcode.CodeFor(err))
}
