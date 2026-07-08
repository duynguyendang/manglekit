package engine

import (
	"context"
	"testing"

	mangleast "codeberg.org/TauCeti/mangle-go/ast"
	mangleparse "codeberg.org/TauCeti/mangle-go/parse"
)

func mustParseAtom(s string) mangleast.Atom {
	a, err := mangleparse.Atom(s)
	if err != nil {
		panic(err)
	}
	return a
}

// M.3: Comparisons (:ge, :le, :gt, :lt) — WORKS in rule bodies
func TestMangleRuntime_Comparisons(t *testing.T) {
	runtime := NewMangleRuntime()
	rules := `
		completeness_pct("BRD", 870).
		min_completeness_pct("BRD", 850).
		generic_ratio_pct("BRD", 100).
		max_generic_pct("BRD", 200).
		passes_completeness(D) :- completeness_pct(D, S), min_completeness_pct(D, M), :ge(S, M).
		fails_completeness(D) :- completeness_pct(D, S), min_completeness_pct(D, M), :lt(S, M).
		passes_generic(D) :- generic_ratio_pct(D, R), max_generic_pct(D, M), :le(R, M).
		passes_quality_gate(D) :- passes_completeness(D), passes_generic(D).
	`
	if err := runtime.LoadFromString(context.Background(), rules); err != nil {
		t.Fatal(err)
	}
	assertQueryTrue(t, runtime, `passes_completeness("BRD").`)
	assertQueryFalse(t, runtime, `fails_completeness("BRD").`)
	assertQueryTrue(t, runtime, `passes_quality_gate("BRD").`)
}

func TestMangleRuntime_QualityGateFails(t *testing.T) {
	runtime := NewMangleRuntime()
	rules := `
		completeness_pct("CSD", 700).
		min_completeness_pct("CSD", 850).
		passes_gate(D) :- completeness_pct(D, S), min_completeness_pct(D, M), :ge(S, M).
	`
	if err := runtime.LoadFromString(context.Background(), rules); err != nil {
		t.Fatal(err)
	}
	assertQueryFalse(t, runtime, `passes_gate("CSD").`)
}

func TestMangleRuntime_OverfittingDetection(t *testing.T) {
	runtime := NewMangleRuntime()
	rules := `
		overfitting_max_pct(150).
		overfitting_min_pct(5).
		task_pct("M1.1", 30). task_pct("M1.2", 200). task_pct("M1.3", 3).
		exceeds_max(T) :- task_pct(T, E), overfitting_max_pct(M), :gt(E, M).
		below_min(T) :- task_pct(T, E), overfitting_min_pct(M), :lt(E, M).
	`
	if err := runtime.LoadFromString(context.Background(), rules); err != nil {
		t.Fatal(err)
	}
	assertOneResult(t, querySolutions(t, runtime, `exceeds_max(T).`), "T", "M1.2")
	assertOneResult(t, querySolutions(t, runtime, `below_min(T).`), "T", "M1.3")
}

// M.1: Negation (!) — WORKS in rule bodies
func TestMangleRuntime_Negation(t *testing.T) {
	runtime := NewMangleRuntime()
	rules := `
		has_cap("cloud", "deploy"). has_cap("cloud", "monitor").
		needs_cap("cloud", "scale"). needs_cap("cloud", "deploy").
		missing(P, C) :- needs_cap(P, C), !has_cap(P, C).
	`
	if err := runtime.LoadFromString(context.Background(), rules); err != nil {
		t.Fatal(err)
	}
	assertOneResult(t, querySolutions(t, runtime, `missing("cloud", C).`), "C", "scale")
}

func TestMangleRuntime_NegationWildcard(t *testing.T) {
	runtime := NewMangleRuntime()
	rules := `
		role_cap("writer", "doc_gen"). role_cap("reviewer", "quality").
		doc_needs("BRD", "doc_gen"). doc_needs("BRD", "quality"). doc_needs("EST", "data_analysis").
		missing(D, C) :- doc_needs(D, C), !role_cap(_, C).
	`
	if err := runtime.LoadFromString(context.Background(), rules); err != nil {
		t.Fatal(err)
	}
	assertOneResult(t, querySolutions(t, runtime, `missing("EST", C).`), "C", "data_analysis")
}

// M.4: Aggregation (fn:count, fn:sum, fn:max, fn:min) — WORKS with |> transforms
func TestMangleRuntime_Aggregation(t *testing.T) {
	runtime := NewMangleRuntime()
	rules := `
		task_effort("M1.1", 3). task_effort("M1.2", 5). task_effort("M1.3", 2).
		total(T) :- task_effort(X, E) |> do fn:group_by(X), let T = fn:sum(E).
		max_e(M) :- task_effort(X, E) |> do fn:group_by(X), let M = fn:max(E).
		min_e(M) :- task_effort(X, E) |> do fn:group_by(X), let M = fn:min(E).
	`
	if err := runtime.LoadFromString(context.Background(), rules); err != nil {
		t.Fatal(err)
	}
	r := querySolutions(t, runtime, `total(T).`)
	t.Logf("fn:sum = %v", r[0]["T"])
	r = querySolutions(t, runtime, `max_e(M).`)
	t.Logf("fn:max = %v", r[0]["M"])
	r = querySolutions(t, runtime, `min_e(M).`)
	t.Logf("fn:min = %v", r[0]["M"])
}

// End-to-end through PolicyEngine
func TestPolicyEngine_Query_AllFeatures(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	rules := `
		Decl completeness_pct(D, S).
		Decl threshold(D, M).
		Decl needs_cap(P, C).
		Decl has_cap(P, C).
		passes_gate(D) :- completeness_pct(D, S), threshold(D, M), :ge(S, M).
		missing_cap(P, C) :- needs_cap(P, C), !has_cap("writer", C).
	`
	if err := engine.LoadPolicy(ctx, rules); err != nil {
		t.Fatal(err)
	}
	facts := []string{
		`completeness_pct("BRD", 870)`, `threshold("BRD", 850)`,
		`needs_cap("cloud", "scale")`, `needs_cap("cloud", "deploy")`, `has_cap("writer", "deploy")`,
	}

	r, err := engine.Query(ctx, facts, `passes_gate("BRD").`)
	if err != nil {
		t.Fatal(err)
	}
	if len(r) == 0 {
		t.Error("expected BRD to pass")
	}

	r, err = engine.Query(ctx, facts, `missing_cap("cloud", C).`)
	if err != nil {
		t.Fatal(err)
	}
	if len(r) != 1 || r[0]["C"] != "scale" {
		t.Errorf("expected missing 'scale', got %v", r)
	}
}

func querySolutions(t *testing.T, runtime *MangleRuntime, query string) []map[string]any {
	t.Helper()
	var results []map[string]any
	if err := runtime.QueryWithSolutions(context.Background(), nil, query, func(sol map[string]any) error {
		results = append(results, sol)
		return nil
	}); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return results
}

func assertQueryTrue(t *testing.T, runtime *MangleRuntime, query string) {
	t.Helper()
	result, err := runtime.ExecuteQuery(context.Background(), nil, query)
	if err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	if !result {
		t.Errorf("expected %q true", query)
	}
}

func assertQueryFalse(t *testing.T, runtime *MangleRuntime, query string) {
	t.Helper()
	result, err := runtime.ExecuteQuery(context.Background(), nil, query)
	if err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	if result {
		t.Errorf("expected %q false", query)
	}
}

func assertOneResult(t *testing.T, results []map[string]any, key, expected string) {
	t.Helper()
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(results), results)
	}
	if results[0][key] != expected {
		t.Errorf("expected %s=%s, got %v", key, expected, results[0][key])
	}
}
