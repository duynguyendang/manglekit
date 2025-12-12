package inductor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInferFromFile_JSON(t *testing.T) {
	content := `{"amount": 100, "desc": "test", "active": true}`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	hint, err := InferFromFile(path)
	if err != nil {
		t.Fatalf("InferFromFile failed: %v", err)
	}

	if hint.FileType != "json" {
		t.Errorf("expected json, got %s", hint.FileType)
	}

	expectedKeys := map[string]string{
		"amount": "(number)",
		"desc":   "(string)",
		"active": "(boolean)",
	}

	for _, k := range hint.JsonKeys {
		// k is like "amount (number)"
		// naive check
		found := false
		for key, typeSuffix := range expectedKeys {
			if k == key+" "+typeSuffix {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unexpected key hint: %s", k)
		}
	}
}

func TestInferFromFile_Graph(t *testing.T) {
	content := `<http://s> <http://p> <http://o> .
_:b1 <http://q> "lit" .`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.nq")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	hint, err := InferFromFile(path)
	if err != nil {
		t.Fatalf("InferFromFile failed: %v", err)
	}

	if hint.FileType != "graph" {
		t.Errorf("expected graph, got %s", hint.FileType)
	}

	expectedDecl1 := "Decl <http://p>(S, O)."
	expectedDecl2 := "Decl <http://q>(S, O)."

	foundP := false
	foundQ := false

	for _, d := range hint.Declarations {
		if d == expectedDecl1 {
			foundP = true
		}
		if d == expectedDecl2 {
			foundQ = true
		}
	}

	if !foundP {
		t.Errorf("missing declaration for <http://p>")
	}
	if !foundQ {
		t.Errorf("missing declaration for <http://q>")
	}
}

func TestInferFromFile_Unsupported(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(path, []byte("text"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := InferFromFile(path)
	if err == nil {
		t.Error("expected error for unsupported file type")
	}
}
