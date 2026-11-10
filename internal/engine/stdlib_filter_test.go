package engine

import (
	"strings"
	"testing"

	"codeberg.org/TauCeti/mangle-go/ast"
	"codeberg.org/TauCeti/mangle-go/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/duynguyendang/manglekit/internal/engine/resources"
)

func TestCollectCallerDecls(t *testing.T) {
	src := `
Decl foo(X, Y).
Decl bar().
Decl baz(A) descr [doc("annotated")].
result(X) :- foo(X, _).
`
	decls := collectCallerDecls(src)
	assert.True(t, decls[ast.PredicateSym{Symbol: "foo", Arity: 2}])
	assert.True(t, decls[ast.PredicateSym{Symbol: "bar", Arity: 0}], "zero-arg Decl must be arity 0, not 1")
	assert.True(t, decls[ast.PredicateSym{Symbol: "baz", Arity: 1}], "Decl with annotation clause must be recognized")
	assert.Len(t, decls, 3)
}

func TestPrependStdLib_FiltersExternalClauses(t *testing.T) {
	stdlib := `Decl trip(S, P, O).
trip(S, P, O) :- quad(S, P, O, _).
trip("a", "b", "c").
trip(S, P, O) :-
    quad(S, P, O, _),
    other(S).
keep(X) :- trip2(X).
`
	out := prependStdLibIfMissing(stdlib, "result(X) :- trip(1, X, _).", nil, map[string]bool{"trip": true})
	assert.NotContains(t, out, "Decl trip(S, P, O)", "Decl for external predicate must be dropped")
	assert.NotContains(t, out, `trip("a"`, "fact for external predicate must be dropped")
	assert.NotContains(t, out, "quad(S, P, O, _),", "multi-line rule body must be dropped with its head")
	assert.NotContains(t, out, "other(S).", "last body line of multi-line rule must be dropped")
	assert.Contains(t, out, "keep(X) :- trip2(X).", "unrelated clauses must survive")
	assert.Contains(t, out, "result(X) :- trip(1, X, _).", "caller source must be appended")
}

func TestFilterStdLibUnit(t *testing.T) {
	stdSrc := `
Decl meta(K, V).
Decl triple(S, P, O).
triple(S, P, O) :- quad(S, P, O, _).
triple("a", "b", "c").
Decl halt(Reason).
`
	stdUnit, err := parse.Unit(strings.NewReader(stdSrc))
	require.NoError(t, err)

	caller := map[ast.PredicateSym]bool{{Symbol: "meta", Arity: 2}: true}
	ext := map[string]bool{"triple": true}

	filtered := filterStdLibUnit(stdUnit, caller, ext)

	// meta/2 Decl should be omitted because caller declared it
	for _, d := range filtered.Decls {
		assert.NotEqual(t, "meta", d.DeclaredAtom.Predicate.Symbol)
		assert.NotEqual(t, "triple", d.DeclaredAtom.Predicate.Symbol)
	}

	// triple clauses should be omitted because triple is external
	for _, c := range filtered.Clauses {
		assert.NotEqual(t, "triple", c.Head.Predicate.Symbol)
	}

	// halt/1 should survive
	foundHalt := false
	for _, d := range filtered.Decls {
		if d.DeclaredAtom.Predicate.Symbol == "halt" {
			foundHalt = true
			break
		}
	}
	assert.True(t, foundHalt, "halt Decl should survive")
}

// TestStdLibShape pins the assumptions the line-based std.mg filter
// makes about the embedded standard library: every non-comment
// statement is single-line (terminates with "." on the same line) and
// every Decl is recognized by declLineRE. If std.mg gains a multi-line
// rule or an exotic Decl form, this fails before the filter silently
// corrupts the merged program.
func TestStdLibShape(t *testing.T) {
	for i, line := range strings.Split(resources.StdLib(), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "%") || strings.HasPrefix(trimmed, "//") {
			continue
		}
		require.True(t, strings.HasSuffix(trimmed, "."),
			"std.mg line %d does not end with '.': %q — the filter assumes single-line statements", i+1, line)
		if strings.HasPrefix(trimmed, "Decl ") {
			_, ok := declSym(trimmed)
			require.True(t, ok, "std.mg line %d: Decl not recognized by declLineRE: %q", i+1, line)
		}
	}
}
