package skill

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToSnakeAndPascal(t *testing.T) {
	cases := []struct {
		in     string
		snake  string
		pascal string
	}{
		{"pii-scan", "pii_scan", "PiiScan"},
		{"My Skill", "my_skill", "MySkill"},
		{"classify", "classify", "Classify"},
		{"__weird!!name__", "weird_name", "WeirdName"},
	}
	for _, c := range cases {
		if got := toSnake(c.in); got != c.snake {
			t.Errorf("toSnake(%q) = %q, want %q", c.in, got, c.snake)
		}
		if got := toPascal(c.in); got != c.pascal {
			t.Errorf("toPascal(%q) = %q, want %q", c.in, got, c.pascal)
		}
	}
	if got := toSnake("!!!"); got != "" {
		t.Errorf("toSnake(\"!!!\") = %q, want empty", got)
	}
}

func TestRunNew_ScaffoldsValidGo(t *testing.T) {
	dir := t.TempDir()
	flagDir = dir
	flagForce = false
	t.Cleanup(func() { flagDir = "."; flagForce = false })

	if err := runNew("pii-scan"); err != nil {
		t.Fatalf("runNew: %v", err)
	}

	outDir := filepath.Join(dir, "pii_scan")
	for _, f := range []string{"main.go", "main_test.go", "policy.dl"} {
		if _, err := os.Stat(filepath.Join(outDir, f)); err != nil {
			t.Fatalf("expected %s to exist: %v", f, err)
		}
	}

	fset := token.NewFileSet()
	for _, f := range []string{"main.go", "main_test.go"} {
		if _, err := parser.ParseFile(fset, filepath.Join(outDir, f), nil, parser.AllErrors); err != nil {
			t.Errorf("generated %s does not parse: %v", f, err)
		}
	}

	policy, err := os.ReadFile(filepath.Join(outDir, "policy.dl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(policy), `action_operation("Req", "pii_scan")`) {
		t.Error("policy.dl does not reference the scaffolded action name")
	}

	mainSrc, err := os.ReadFile(filepath.Join(outDir, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mainSrc), "PiiScanRequest") || !strings.Contains(string(mainSrc), "sdk.WithBlueprintPath") {
		t.Error("main.go missing typed request or blessed policy-loading option")
	}
}

func TestRunNew_ReusesExistingDirOnlyWithForce(t *testing.T) {
	dir := t.TempDir()
	flagDir = dir
	flagForce = false
	t.Cleanup(func() { flagDir = "."; flagForce = false })

	if err := runNew("demo"); err != nil {
		t.Fatalf("first runNew: %v", err)
	}
	if err := runNew("demo"); err == nil {
		t.Fatal("expected error for existing directory without --force")
	}
	flagForce = true
	if err := runNew("demo"); err != nil {
		t.Fatalf("runNew with --force: %v", err)
	}
}
