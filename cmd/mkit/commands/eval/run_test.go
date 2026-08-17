package eval

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvalRequiresDataOrKnowledge(t *testing.T) {
	dir := t.TempDir()
	policy := filepath.Join(dir, "policy.dl")
	// A minimal, valid Datalog program so engine init succeeds.
	if err := os.WriteFile(policy, []byte("halt(Req).\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	EvalCmd.SetArgs([]string{"--policy", policy, "--query", `halt("Req")`})
	err := EvalCmd.Execute()
	if err == nil {
		t.Fatal("expected error when neither --data nor --knowledge is provided")
	}
	if !strings.Contains(err.Error(), "must provide either") {
		t.Errorf("expected data/knowledge validation error, got: %v", err)
	}
}

func TestEvalJSONOutputIsCleanOnStdout(t *testing.T) {
	dir := t.TempDir()
	policy := filepath.Join(dir, "policy.dl")
	if err := os.WriteFile(policy, []byte(`Decl high(R).
high(R) :- json_str(R, "risk_level", "critical").
`), 0o600); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(dir, "data.json")
	if err := os.WriteFile(data, []byte(`{"risk_level": "critical"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	EvalCmd.SetOut(&stdout)
	EvalCmd.SetErr(&stderr)
	EvalCmd.SetArgs([]string{
		"--policy", policy,
		"--data", data,
		"--query", `high(R).`,
		"--output", "json",
	})
	if err := EvalCmd.Execute(); err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	// stdout must contain only the JSON result (pipeable to jq).
	var results []map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nstdout: %q\nstderr: %q", err, stdout.String(), stderr.String())
	}
	assert.NotEmpty(t, results)

	// progress chatter must go to stderr
	assert.Contains(t, stderr.String(), "Loaded")
	assert.NotContains(t, stdout.String(), "Loaded")
}

func TestEvalJSONOutputEmptyResults(t *testing.T) {
	dir := t.TempDir()
	policy := filepath.Join(dir, "policy.dl")
	if err := os.WriteFile(policy, []byte(`Decl high(R).
high(R) :- json_str(R, "risk_level", "critical").
`), 0o600); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(dir, "data.json")
	if err := os.WriteFile(data, []byte(`{"risk_level": "low"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	EvalCmd.SetOut(&stdout)
	EvalCmd.SetErr(&stderr)
	EvalCmd.SetArgs([]string{
		"--policy", policy,
		"--data", data,
		"--query", `high(R).`,
		"--output", "json",
	})
	if err := EvalCmd.Execute(); err != nil {
		t.Fatalf("eval failed: %v", err)
	}

	var results []map[string]string
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("stdout is not valid JSON for empty results: %v\nstdout: %q", err, stdout.String())
	}
	assert.Empty(t, results)
}

func TestEvalInvalidOutputModeIsUsageError(t *testing.T) {
	dir := t.TempDir()
	policy := filepath.Join(dir, "policy.dl")
	if err := os.WriteFile(policy, []byte("halt(Req).\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	EvalCmd.SetArgs([]string{
		"--policy", policy,
		"--data", policy,
		"--query", `halt("Req")`,
		"--output", "xml",
	})
	err := EvalCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid --output value")
	}
	assert.Contains(t, err.Error(), `must be "json" or "table"`)
}
