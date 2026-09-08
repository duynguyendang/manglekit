package run

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/cmd/mkit/commands/exitcode"
	"github.com/spf13/cobra"
)

func TestRunRequiresFlags(t *testing.T) {
	RunCmd.SetArgs([]string{})
	err := RunCmd.Execute()
	if err == nil {
		t.Fatal("expected error when required flags are missing")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected required-flag error, got: %v", err)
	}
}

func TestRunFlagsAreMarkedRequired(t *testing.T) {
	for _, name := range []string{"policy", "data", "target", "output"} {
		flag := RunCmd.Flags().Lookup(name)
		if flag == nil {
			t.Fatalf("flag %q not registered", name)
			continue
		}
		if flag.Annotations[cobra.BashCompOneRequiredFlag][0] != "true" {
			t.Errorf("flag %q should be marked required", name)
		}
	}
}

func TestRunFormatJSON(t *testing.T) {
	dir := t.TempDir()
	policy := filepath.Join(dir, "policy.dl")
	if err := os.WriteFile(policy, []byte("Decl high(S).\nhigh(S) :- json_str(S, \"risk_level\", \"critical\").\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(dir, "data.json")
	if err := os.WriteFile(data, []byte(`{"risk_level": "critical"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.json")

	RunCmd.SetArgs([]string{"--policy", policy, "--data", data, "--target", "high", "--output", out, "--format", "json"})
	if err := RunCmd.Execute(); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	var facts []struct {
		Subject   string `json:"subject"`
		Predicate string `json:"predicate"`
		Object    string `json:"object"`
	}
	if err := json.Unmarshal(raw, &facts); err != nil {
		t.Fatalf("output is not valid JSON: %v (%s)", err, raw)
	}
	if len(facts) != 1 {
		t.Fatalf("want 1 derived fact, got %d: %s", len(facts), raw)
	}
	if facts[0].Subject != "root" || facts[0].Predicate != "high" || facts[0].Object != "true" {
		t.Errorf("unexpected fact %+v", facts[0])
	}
}

func TestRunFormatNQuadsUnchanged(t *testing.T) {
	dir := t.TempDir()
	policy := filepath.Join(dir, "policy.dl")
	if err := os.WriteFile(policy, []byte("Decl high(S).\nhigh(S) :- json_str(S, \"risk_level\", \"critical\").\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(dir, "data.json")
	if err := os.WriteFile(data, []byte(`{"risk_level": "critical"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.nq")

	RunCmd.SetArgs([]string{"--policy", policy, "--data", data, "--target", "high", "--output", out, "--format", "nquads"})
	if err := RunCmd.Execute(); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "<root> <high> \"true\" .") {
		t.Errorf("unexpected nquads output: %s", raw)
	}
}

func TestRunRejectsUnknownFormat(t *testing.T) {
	RunCmd.SetArgs([]string{"--policy", "p.dl", "--data", "d.json", "--target", "x", "--output", "o", "--format", "yaml"})
	err := RunCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "invalid --format") {
		t.Fatalf("expected invalid-format usage error, got %v", err)
	}
	if code := exitcode.CodeFor(err); code != exitcode.Usage {
		t.Errorf("exit code = %d, want %d", code, exitcode.Usage)
	}
}
