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
	ruleSet, err := mangle.New(context.Background(), opts)
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
	ruleSet, err := mangle.New(context.Background(), opts)
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
