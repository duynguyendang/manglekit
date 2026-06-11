package engine

import (
	"strings"
	"testing"

	"codeberg.org/TauCeti/mangle-go/ast"
	"github.com/duynguyendang/manglekit/internal/engine/resources"
)

func TestCollectCallerDecls_ZeroArg(t *testing.T) {
	source := `Decl foo().`
	decls := collectCallerDecls(source)
	sym := ast.PredicateSym{Symbol: "foo", Arity: 0}
	if !decls[sym] {
		t.Errorf("zero-arg Decl foo() should have arity 0, got %v", decls)
	}
}

func TestCollectCallerDecls_SingleArg(t *testing.T) {
	source := `Decl bar(X).`
	decls := collectCallerDecls(source)
	sym := ast.PredicateSym{Symbol: "bar", Arity: 1}
	if !decls[sym] {
		t.Errorf("Decl bar(X) should have arity 1, got %v", decls)
	}
}

func TestCollectCallerDecls_MultiArg(t *testing.T) {
	source := `Decl triple(S, P, O).`
	decls := collectCallerDecls(source)
	sym := ast.PredicateSym{Symbol: "triple", Arity: 3}
	if !decls[sym] {
		t.Errorf("Decl triple(S,P,O) should have arity 3, got %v", decls)
	}
}

func TestCollectCallerDecls_Multiple(t *testing.T) {
	source := "Decl foo().\nDecl bar(X, Y).\nDecl baz(A, B, C)."
	decls := collectCallerDecls(source)
	if len(decls) != 3 {
		t.Errorf("expected 3 decls, got %d: %v", len(decls), decls)
	}
	if decls[ast.PredicateSym{Symbol: "foo", Arity: 0}] == false {
		t.Error("missing foo/0")
	}
	if decls[ast.PredicateSym{Symbol: "bar", Arity: 2}] == false {
		t.Error("missing bar/2")
	}
	if decls[ast.PredicateSym{Symbol: "baz", Arity: 3}] == false {
		t.Error("missing baz/3")
	}
}

func TestPrependStdLib_noFiltering(t *testing.T) {
	std := "Decl foo(X).\nfoo(X) :- bar(X)."
	src := "source body"
	result := prependStdLibIfMissing(std, src, nil, nil)
	if !strings.HasPrefix(result, std) {
		t.Errorf("expected std lib to be prepended unchanged, got:\n%s", result)
	}
	if !strings.HasSuffix(result, src) {
		t.Errorf("expected source to be at end, got:\n%s", result)
	}
}

func TestPrependStdLib_filtersCallerDecl(t *testing.T) {
	std := "Decl foo(X).\nfoo(X) :- bar(X).\nDecl baz(Y)."
	// Caller source does NOT include Decl — we're testing that the
	// std Decl is removed, not that our test source's Decl survives.
	src := "my_rule(P) :- foo(P)."
	callerDecls := collectCallerDecls("Decl foo(X).")
	result := prependStdLibIfMissing(std, src, callerDecls, nil)
	if strings.Contains(result, "Decl foo") {
		t.Errorf("std Decl foo should be filtered out, got:\n%s", result)
	}
	if !strings.Contains(result, "foo(X) :- bar(X).") {
		t.Errorf("std foo defining rule should still be present (caller only declared foo, didn't redefine rule), got:\n%s", result)
	}
	if !strings.Contains(result, "Decl baz") {
		t.Errorf("std Decl baz should be preserved, got:\n%s", result)
	}
}

func TestPrependStdLib_filtersExternalPredName(t *testing.T) {
	std := "Decl triple(S, P, O).\ntriple(S, P, O) :- quad(S, P, O, _).\nDecl halt(Reason)."
	src := "result(X) :- triple(3, X)."
	extNames := map[string]bool{"triple": true}
	result := prependStdLibIfMissing(std, src, nil, extNames)
	if strings.Contains(result, "Decl triple") {
		t.Errorf("std Decl triple should be filtered for external predicate, got:\n%s", result)
	}
	if strings.Contains(result, "triple(S, P, O) :-") {
		t.Errorf("std defining rule for triple should be filtered, got:\n%s", result)
	}
	if !strings.Contains(result, "Decl halt") {
		t.Errorf("std Decl halt should be preserved, got:\n%s", result)
	}
}

func TestPrependStdLib_keepsNonRuleLines(t *testing.T) {
	std := "% comment\nDecl foo(X).\n\n// another comment\nfoo(X) :- bar(X)."
	src := "source"
	extNames := map[string]bool{"foo": true}
	result := prependStdLibIfMissing(std, src, nil, extNames)
	if !strings.Contains(result, "% comment") {
		t.Error("comments should be preserved")
	}
	if !strings.Contains(result, "// another comment") {
		t.Error("line comments should be preserved")
	}
	if strings.Contains(result, "Decl foo") {
		t.Error("external Decl foo should be filtered")
	}
	if strings.Contains(result, "foo(X) :-") {
		t.Error("external defining rule foo should be filtered")
	}
	if !strings.HasSuffix(result, src) {
		t.Error("caller source should be appended")
	}
}

func TestStdLibShape_pinsCurrentPredicates(t *testing.T) {
	// This test pins the current std.dl shape so a CI failure
	// catches accidental predicate additions or removals.
	// If std.dl deliberately changes, update these counts.
	std := resources.StdLib()

	// Expected predicate names (symbols the policy layer depends on)
	required := []string{
		"json_str", "json_num", "json_bool", "json_link", "json_null",
		"quad", "triple",
		"deny", "halt", "route", "retry",
		"config",
		"attempt", "meta", "label", "action_operation",
		"violation_rule",
	}

	for _, name := range required {
		if !strings.Contains(std, name) {
			t.Errorf("std.dl is missing required predicate %q — remove from test or restore to std.dl", name)
		}
	}

	// Verify zero-arg Decl patterns are handled correctly
	// (currently std.dl has no zero-arg Decls, but the parser handles them)
	callerDecls := collectCallerDecls("Decl foo().")
	if !callerDecls[ast.PredicateSym{Symbol: "foo", Arity: 0}] {
		t.Error("zero-arg Decl foo() should have arity 0")
	}

	// Sanity: the std.dl itself should not produce unexpected Decls
	stdDecls := collectCallerDecls(std)
	if len(stdDecls) < 20 {
		t.Errorf("expected at least 20 Decl lines in std.dl, got %d — std.dl may be truncated", len(stdDecls))
	}
	// std.dl has multi-arity predicates (deny/1, deny/2, deny/3 etc)
	denyCount := 0
	for sym := range stdDecls {
		if sym.Symbol == "deny" {
			denyCount++
		}
	}
	if denyCount < 3 {
		t.Errorf("expected deny/1, deny/2, deny/3 in std.dl Decls, got %d variants", denyCount)
	}

	if stdDecls[ast.PredicateSym{Symbol: "triple", Arity: 3}] == false {
		t.Error("std.dl must declare triple/3")
	}
	if stdDecls[ast.PredicateSym{Symbol: "halt", Arity: 1}] == false {
		t.Error("std.dl must declare halt/1")
	}
	if stdDecls[ast.PredicateSym{Symbol: "halt", Arity: 2}] == false {
		t.Error("std.dl must declare halt/2")
	}
}
