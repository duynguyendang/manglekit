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

func TestInferFromFile_JSONNested(t *testing.T) {
	content := `{
		"meta": {
			"id": "123",
			"active": true
		},
		"deployment": {
			"replicas": 3,
			"resources": {
				"cpu": 1.5
			}
		}
	}`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_nested.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	hint, err := InferFromFile(path)
	if err != nil {
		t.Fatalf("InferFromFile failed: %v", err)
	}

	expectedKeys := []string{
		"meta.id (string)",
		"meta.active (boolean)",
		"deployment.replicas (number)",
		"deployment.resources.cpu (number)",
	}

	for _, expected := range expectedKeys {
		found := false
		for _, actual := range hint.JsonKeys {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing expected key hint: %s. Found: %v", expected, hint.JsonKeys)
		}
	}
}

func TestInferFromFile_JSONArrays(t *testing.T) {
	content := `{
		"deployment": {
			"env_vars": ["SECRET", "HOST"],
			"servers": [
				{"ip": "1.2.3.4", "region": "us-east"}
			]
		}
	}`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_arrays.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	hint, err := InferFromFile(path)
	if err != nil {
		t.Fatalf("InferFromFile failed: %v", err)
	}

	expectedKeys := []string{
		"deployment.env_vars (array of string)",
		"deployment.servers[].ip (string)",
		"deployment.servers[].region (string)",
	}

	for _, expected := range expectedKeys {
		found := false
		for _, actual := range hint.JsonKeys {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing expected key hint: %s. Found: %v", expected, hint.JsonKeys)
		}
	}
}

func TestInferFromFile_Graph(t *testing.T) {
	content := `<http://s> <http://p> <http://o> .
_:b1 <http://q> "lit" .
<http://s> <http://schema.org/my-prop> "val" .`
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

	expectedDecls := map[string]bool{
		"Decl p(S, O).":       false,
		"Decl q(S, O).":       false,
		"Decl my_prop(S, O).": false,
	}

	for _, d := range hint.Declarations {
		if _, exists := expectedDecls[d]; exists {
			expectedDecls[d] = true
		}
	}

	for decl, found := range expectedDecls {
		if !found {
			t.Errorf("missing declaration: %s", decl)
		}
	}
}

func TestInferFromFile_Turtle(t *testing.T) {
	content := `
	@prefix foaf: <http://xmlns.com/foaf/0.1/> .
	<http://example.org/alice> a foaf:Person ;
	    foaf:name "Alice" ;
	    foaf:knows <http://example.org/bob> .
	`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.ttl")
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

	expectedDecls := map[string]bool{
		"Decl type(S, O).":  false,
		"Decl name(S, O).":  false,
		"Decl knows(S, O).": false,
	}

	for _, d := range hint.Declarations {
		if _, exists := expectedDecls[d]; exists {
			expectedDecls[d] = true
		}
	}

	for decl, found := range expectedDecls {
		if !found {
			t.Errorf("missing declaration: %s", decl)
		}
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
