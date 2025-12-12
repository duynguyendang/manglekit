package knowledge

import (
	"testing"
)

func TestSmartCast(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Basic Numbers
		{"15000", "15000"},
		{"3.14", "3.14"},
		{"0", "0"},
		{"0.5", "0.5"},

		// Strings
		{"abc", "\"abc\""},
		{"\"abc\"", "\"abc\""},

		// IDs with leading zeros (should be strings)
		{"0123", "\"0123\""},
		{"007", "\"007\""},

		// RDF Type Suffixes
		{`"15000"^^<http://www.w3.org/2001/XMLSchema#integer>`, "15000"},
		{`"3.14"^^<http://www.w3.org/2001/XMLSchema#double>`, "3.14"},

		// Whitespace
		{" 15000 ", "15000"},
		{" \"15000\" ", "15000"},
	}

	for _, tc := range tests {
		got := smartCast(tc.input)
		if got != tc.expected {
			t.Errorf("smartCast(%q) = %s; want %s", tc.input, got, tc.expected)
		}
	}
}

func TestTriplesToFacts(t *testing.T) {
	triples := []Triple{
		{Subject: "<:req1>", Predicate: "cost", Object: `"15000"^^<xs:int>`},
		{Subject: "<:req2>", Predicate: "zip", Object: `"01234"`},
		{Subject: "<:req3>", Predicate: "active", Object: "true"},
	}

	facts := TriplesToFacts(triples)

	expected := []string{
		`cost(":req1", 15000).`,
		`zip(":req2", "01234").`,
		`active(":req3", "true").`, // "true" isn't a float, so it remains a string unless we handle bools explicitly. Smart cast logic parses float. "true" fails parse float.
	}

	if len(facts) != len(expected) {
		t.Fatalf("expected %d facts, got %d", len(expected), len(facts))
	}

	for i, f := range facts {
		if f != expected[i] {
			t.Errorf("fact %d: got %s, want %s", i, f, expected[i])
		}
	}
}
