package mangle

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/mangle/ast"
	"github.com/google/mangle/factstore"
)

func TestLoadFactsFromDirectory(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"aliases.facts":      "alias(\"foo\", \"bar\").\n",
		"filters.data":       "# comment\n// another comment\ndefault_filter(\"visibility\", \"tenant\").\n",
		"nested/more.dlog":   "alias(\"bar\", \"baz\").\n",
		"nested/ignored.log": "alias(\"this\", \"should\").", // extension ignored
	}

	for name, contents := range files {
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(contents), 0o600); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}

	store, err := loadFacts(dir)
	if err != nil {
		t.Fatalf("loadFacts: %v", err)
	}

	assertHasFacts(t, store,
		[][2]string{{"foo", "bar"}, {"bar", "baz"}},
		[][2]string{{"visibility", "tenant"}},
	)
}

func TestLoadFactsFromGlob(t *testing.T) {
	dir := t.TempDir()
	paths := []string{
		filepath.Join(dir, "a.facts"),
		filepath.Join(dir, "b.facts"),
	}

	for i, p := range paths {
		content := fmt.Sprintf("alias(\"glob%c\", \"value%c\").\n", 'a'+i, 'a'+i)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
	}

	store, err := loadFacts(filepath.Join(dir, "*.facts"))
	if err != nil {
		t.Fatalf("loadFacts glob: %v", err)
	}

	assertHasFacts(t, store,
		[][2]string{{"globa", "valuea"}, {"globb", "valueb"}},
		nil,
	)
}

func assertHasFacts(t *testing.T, store factstore.SimpleInMemoryStore, aliases [][2]string, filters [][2]string) {
	t.Helper()

	aliasFacts := gatherFacts(t, store, "alias")
	for _, pair := range aliases {
		if !aliasFacts[[2]string{pair[0], pair[1]}] {
			t.Fatalf("missing alias fact %q -> %q", pair[0], pair[1])
		}
	}

	filterFacts := gatherFacts(t, store, "default_filter")
	for _, pair := range filters {
		if !filterFacts[[2]string{pair[0], pair[1]}] {
			t.Fatalf("missing filter fact %q -> %q", pair[0], pair[1])
		}
	}
}

func gatherFacts(t *testing.T, store factstore.SimpleInMemoryStore, predicate string) map[[2]string]bool {
	t.Helper()
	results := make(map[[2]string]bool)
	query := ast.NewQuery(ast.PredicateSym{Symbol: predicate, Arity: 2})
	err := store.GetFacts(query, func(atom ast.Atom) error {
		if len(atom.Args) != 2 {
			return nil
		}
		left, ok := atom.Args[0].(ast.Constant)
		if !ok {
			return nil
		}
		right, ok := atom.Args[1].(ast.Constant)
		if !ok {
			return nil
		}
		lval, err := left.StringValue()
		if err != nil {
			return err
		}
		rval, err := right.StringValue()
		if err != nil {
			return err
		}
		results[[2]string{lval, rval}] = true
		return nil
	})
	if err != nil {
		t.Fatalf("GetFacts(%s): %v", predicate, err)
	}
	return results
}
