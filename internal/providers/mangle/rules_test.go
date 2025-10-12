package mangle_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/mangle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleSet_Evaluate_PreProcess_WithConverters(t *testing.T) {
	dir := t.TempDir()
	rules := `
		Decl expanded_query(Token, Expansion).
		Decl alias(Source, Target).
		expanded_query(Token, Token) :- query_token(Token).
		expanded_query(Token, Expansion) :- query_token(Token), alias(Token, Expansion).
	`
	facts := `
		alias("foo", "bar").
	`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "rules.dlog"), []byte(rules), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "facts.dlog"), []byte(facts), 0o600))

	// Use the correct core.MangleOptions struct.
	opts := core.MangleOptions{
		Path:              []string{dir},
		DefaultConverters: true,
	}
	// Call the refactored New function with the correct signature.
	ruleSet, err := mangle.New(context.TODO(), opts)
	require.NoError(t, err)

	query := core.Query{Text: "foo"} // The QueryConverter will process this text.
	result, err := ruleSet.Evaluate(core.Pre, query, nil)
	require.NoError(t, err)
	require.True(t, result.Allowed)
	require.NotNil(t, result.Mutate)

	// Apply the mutation to the query
	result.Mutate(&query, nil)

	require.NotNil(t, query.Meta)
	expansions, ok := query.Meta["expansion_terms"].([]string)
	require.True(t, ok)
	sort.Strings(expansions)
	assert.Equal(t, []string{"bar", "foo"}, expansions)
}

func TestRuleSet_Evaluate_PostProcess_WithConverters(t *testing.T) {
	dir := t.TempDir()
	rules := `
		Decl deny(DocID, Reason).
		deny(DocID, "sensitive content") :- doc_metadata(DocID, "category", "sensitive").
		deny(DocID, "user not allowed") :- user_attribute("role", "guest"), doc_metadata(DocID, "access", "premium").
	`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "rules.dlog"), []byte(rules), 0o600))

	// Use the correct core.MangleOptions struct.
	opts := core.MangleOptions{
		Path:              []string{dir},
		DefaultConverters: true,
	}
	// Call the refactored New function with the correct signature.
	ruleSet, err := mangle.New(context.TODO(), opts)
	require.NoError(t, err)

	query := core.Query{
		Meta: map[string]any{
			"user_context": map[string]any{"role": "guest"},
		},
	}

	answer := &core.Answer{
		Citations: []core.Citation{
			{ID: "doc1", Snippet: "regular content"},
			{ID: "doc2", Snippet: "sensitive stuff"},
			{ID: "doc3", Snippet: "premium content"},
			{ID: "doc4", Snippet: "another regular doc"},
		},
		Meta: make(map[string]any),
	}

	originalDocs := []core.Doc{
		{ID: "doc1", Text: "regular content"},
		{ID: "doc2", Text: "sensitive stuff", Meta: map[string]any{"category": "sensitive"}},
		{ID: "doc3", Text: "premium content", Meta: map[string]any{"access": "premium"}},
	}
	answer.Meta["original_docs"] = originalDocs

	result, err := ruleSet.Evaluate(core.Post, query, answer)
	require.NoError(t, err)
	require.True(t, result.Allowed)

	assert.Len(t, answer.Citations, 2)
	var remainingIDs []string
	for _, c := range answer.Citations {
		remainingIDs = append(remainingIDs, c.ID)
	}
	sort.Strings(remainingIDs)
	assert.Equal(t, []string{"doc1", "doc4"}, remainingIDs)

	require.NotNil(t, answer.Meta)
	deniedReasons, ok := answer.Meta["mangle_denied_reasons"].(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "sensitive content", deniedReasons["doc2"])
	assert.Equal(t, "user not allowed", deniedReasons["doc3"])
}

func TestRuleSet_PostRules_DropsDocuments(t *testing.T) {
	dir := t.TempDir()
	rules := `
		drop_doc(DocID, "internal_only") :- doc_metadata(DocID, "tag", "internal").
	`
	rulePath := filepath.Join(dir, "policy.dlog")
	require.NoError(t, os.WriteFile(rulePath, []byte(rules), 0o600))

	opts := core.MangleOptions{
		Path:              []string{rulePath},
		DefaultConverters: true,
	}
	ruleSet, err := mangle.New(context.TODO(), opts)
	require.NoError(t, err)

	postEval, ok := ruleSet.(core.PostRuleEvaluator)
	require.True(t, ok)

	docs := []core.Doc{
		{ID: "public", Text: "hello", Meta: map[string]any{"tag": "public"}},
		{ID: "internal", Text: "secret", Meta: map[string]any{"tag": "internal"}},
	}

	result, err := postEval.Post(context.TODO(), core.Query{}, docs, nil)
	require.NoError(t, err)
	require.False(t, result.Denied)
	require.Len(t, result.Filtered, 1)
	assert.Equal(t, "public", result.Filtered[0].ID)

	dropped, ok := result.Meta["dropped_docs"].(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "internal_only", dropped["internal"])
}

func TestRuleSet_PostRules_DenyOnLowConfidence(t *testing.T) {
	dir := t.TempDir()
	// Mangle v0.3.0's :filter(expr) for float comparisons was not parsing correctly.
	// This test has been adapted to use integers and :lt to validate the rule logic.
	rules := `
		Decl best_score(Score).
		threshold(45).
		deny("low_confidence") :-
			best_score(S),
			threshold(T),
			:lt(S, T).
	`
	rulePath := filepath.Join(dir, "policy.dlog")
	require.NoError(t, os.WriteFile(rulePath, []byte(rules), 0o600))

	opts := core.MangleOptions{
		Path:              []string{rulePath},
		DefaultConverters: true,
	}
	ruleSet, err := mangle.New(context.TODO(), opts)
	require.NoError(t, err)

	postEval, ok := ruleSet.(core.PostRuleEvaluator)
	require.True(t, ok)

	result, err := postEval.Post(context.TODO(), core.Query{}, nil, map[string]any{"best_score": 20})
	require.NoError(t, err)
	require.True(t, result.Denied)
	assert.Equal(t, "low_confidence", result.Reason)
	assert.Equal(t, "low_confidence", result.Meta["denied_reason"])
}

func TestRuleSet_PostRules_RedactsSensitiveText(t *testing.T) {
	dir := t.TempDir()
	// Mangle v0.3.0 does not support :regex:match.
	// The Go code in `applySingleRedaction` handles matching "phone" to a real regex.
	// This test verifies that the `redact("phone")` fact is correctly produced
	// by a valid rule, and that the Go code acts on it.
	rules := `
		redact("phone") :-
			doc_text(T),
			:string:contains(T, "123-456-7890").
	`
	rulePath := filepath.Join(dir, "policy.dlog")
	require.NoError(t, os.WriteFile(rulePath, []byte(rules), 0o600))

	opts := core.MangleOptions{
		Path:              []string{rulePath},
		DefaultConverters: true,
	}
	ruleSet, err := mangle.New(context.TODO(), opts)
	require.NoError(t, err)

	postEval, ok := ruleSet.(core.PostRuleEvaluator)
	require.True(t, ok)

	docs := []core.Doc{
		{ID: "doc1", Text: "Call me at 123-456-7890"},
	}

	result, err := postEval.Post(context.TODO(), core.Query{}, docs, nil)
	require.NoError(t, err)
	require.False(t, result.Denied)
	require.Len(t, result.Filtered, 1)
	assert.NotContains(t, result.Filtered[0].Text, "123-456-7890")
	assert.Contains(t, result.Filtered[0].Text, "[REDACTED]")

	redactions, ok := result.Meta["redactions"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, redactions, 1)
	assert.Equal(t, "doc1", redactions[0]["doc_id"])
	assert.Equal(t, "phone", redactions[0]["pattern"])
}
