package check

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckValidPolicy(t *testing.T) {
	dir := t.TempDir()
	policy := filepath.Join(dir, "policy.dl")
	if err := os.WriteFile(policy, []byte(`Decl blocked(R).
deny(R, "blocked by policy") :- blocked(R).
`), 0o600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	CheckCmd.SetOut(&out)
	CheckCmd.SetErr(&out)
	CheckCmd.SetArgs([]string{policy})
	err := CheckCmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, out.String(), "OK")
}

func TestCheckInvalidPolicy(t *testing.T) {
	dir := t.TempDir()
	policy := filepath.Join(dir, "policy.dl")
	// 'denyy' is a typo: undeclared predicate usage should fail compilation.
	if err := os.WriteFile(policy, []byte(`deny(R, "typo") :- denyy(R).
`), 0o600); err != nil {
		t.Fatal(err)
	}

	CheckCmd.SetArgs([]string{policy})
	err := CheckCmd.Execute()
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "denyy"),
		"expected policy validation error, got: %v", err)
}

func TestCheckMissingFile(t *testing.T) {
	CheckCmd.SetArgs([]string{filepath.Join(t.TempDir(), "nope.dl")})
	assert.Error(t, CheckCmd.Execute())
}

func TestCheckRequiresExactlyOneArg(t *testing.T) {
	CheckCmd.SetArgs([]string{})
	assert.Error(t, CheckCmd.Execute())
}

func TestLint_UngroundedNegatedVariable(t *testing.T) {
	dir := t.TempDir()
	policy := filepath.Join(dir, "bad_negation.dl")
	// X in !auth(Req, X) is not bound positively in the body!
	policySrc := `
Decl auth(R, User).
Decl op(R).

halt("Req", "bad_rule", "T1") :- op("Req"), !auth("Req", X).
`
	require.NoError(t, os.WriteFile(policy, []byte(policySrc), 0o600))

	var out, errOut bytes.Buffer
	LintCmd.SetOut(&out)
	LintCmd.SetErr(&errOut)
	LintCmd.SetArgs([]string{policy})

	err := LintCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, errOut.String(), "ungrounded")
}

func TestLint_NegatedWildcardWarning(t *testing.T) {
	policySrc := `
Decl auth(R, User).
Decl op(R).

halt("Req", "wildcard_rule", "T1") :- op("Req"), !auth("Req", _).
`
	issues, err := LintPolicy(policySrc)
	require.NoError(t, err)
	foundWarning := false
	for _, issue := range issues {
		if issue.Severity == SeverityWarning && strings.Contains(issue.Message, "wildcard") {
			foundWarning = true
			break
		}
	}
	assert.True(t, foundWarning, "expected wildcard warning in negated atom")
}
