package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
