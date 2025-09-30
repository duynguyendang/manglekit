package mangle

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"ndduy.dev/manglekit/internal/types"
)

func TestProcessor(t *testing.T) {
	dir := t.TempDir()

	rules := `
		Decl query_token(Token).
		Decl expanded_query(Token, Expansion).
		Decl alias(Source, Target).
		expanded_query(Token, Token) :- query_token(Token).
		expanded_query(Token, Expansion) :- query_token(Token), alias(Token, Expansion).
	`
	facts := `
		alias("foo", "bar").
		alias("multi", "word expansion").
	`
	nestedFacts := `
		alias("baz", "qux").
	`

	if err := os.WriteFile(filepath.Join(dir, "rules.dlog"), []byte(rules), 0o600); err != nil {
		t.Fatalf("write rules.dlog: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "facts.dlog"), []byte(facts), 0o600); err != nil {
		t.Fatalf("write facts.dlog: %v", err)
	}

	nestedDir := filepath.Join(dir, "nested")
	if err := os.Mkdir(nestedDir, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "more.dlog"), []byte(nestedFacts), 0o600); err != nil {
		t.Fatalf("write nested/more.dlog: %v", err)
	}
	// This file should be ignored.
	if err := os.WriteFile(filepath.Join(nestedDir, "ignored.txt"), []byte("alias(\"a\", \"b\")."), 0o600); err != nil {
		t.Fatalf("write nested/ignored.txt: %v", err)
	}

	cfg := Config{
		RulesFile: dir,
	}

	proc, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}

	testCases := []struct {
		name  string
		query string
		want  *types.ExpandedQuery
	}{
		{
			name:  "simple expansion",
			query: "foo",
			want: &types.ExpandedQuery{
				NormalizedQuery: "foo",
				ExpansionTerms:  []string{"bar", "foo"},
				Filters:         map[string]string{},
				Explanation:     "mangle expansions applied",
			},
		},
		{
			name:  "nested expansion",
			query: "baz",
			want: &types.ExpandedQuery{
				NormalizedQuery: "baz",
				ExpansionTerms:  []string{"baz", "qux"},
				Filters:         map[string]string{},
				Explanation:     "mangle expansions applied",
			},
		},
		{
			name:  "no expansion",
			query: "unknown",
			want: &types.ExpandedQuery{
				NormalizedQuery: "unknown",
				ExpansionTerms:  []string{"unknown"},
				Filters:         map[string]string{},
				Explanation:     "mangle expansions applied",
			},
		},
		{
			name:  "multi-word expansion",
			query: "multi",
			want: &types.ExpandedQuery{
				NormalizedQuery: "multi",
				ExpansionTerms:  []string{"multi", "word expansion"},
				Filters:         map[string]string{},
				Explanation:     "mangle expansions applied",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := proc.PreProcess(&types.QueryInput{Query: tc.query})
			if err != nil {
				t.Fatalf("PreProcess() error = %v, want nil", err)
			}
			sort.Strings(got.ExpansionTerms)
			sort.Strings(tc.want.ExpansionTerms)

			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("PreProcess() = %+v, want %+v", got, tc.want)
			}
		})
	}
}