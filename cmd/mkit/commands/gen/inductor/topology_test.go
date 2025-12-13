package inductor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInferFromFile_Graph_TopologyHints(t *testing.T) {
	content := `<http://s> <http://works_in> <http://dept_engineering> .
_:b1 <http://spending_limit> "5000" .`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "topology.nq")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	hint, err := InferFromFile(path)
	if err != nil {
		t.Fatalf("InferFromFile failed: %v", err)
	}

	// We expect comments with samples
	// Decl works_in(S, O).       % Sample: works_in("s", "dept_engineering")
	// Decl spending_limit(S, O). % Sample: spending_limit("b1", "5000")

    // Note: sanitization might change "s" or "dept_engineering" depending on full URI.
    // In the input <http://s>, sanitize returns "s".
    // <http://dept_engineering> -> "dept_engineering".

	foundWorksIn := false
	foundSpending := false

	for _, decl := range hint.Declarations {
		if strings.Contains(decl, "Decl works_in(S, O).") {
			if strings.Contains(decl, "% Sample: works_in(") {
				foundWorksIn = true
			}
		}
		if strings.Contains(decl, "Decl spending_limit(S, O).") {
			if strings.Contains(decl, "% Sample: spending_limit(") {
				foundSpending = true
			}
		}
	}

	if !foundWorksIn {
		t.Errorf("missing works_in declaration with sample. Got: %v", hint.Declarations)
	}
	if !foundSpending {
		t.Errorf("missing spending_limit declaration with sample. Got: %v", hint.Declarations)
	}
}
