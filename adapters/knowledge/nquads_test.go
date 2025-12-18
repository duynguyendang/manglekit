package knowledge

import (
	"strings"
	"testing"
)

func TestNQuadsLoader_Parse(t *testing.T) {
	input := `<http://sub1> <http://pred1> "obj1" .
<http://sub2> <http://pred2> "obj2" <http://graphA> .
# This is a comment`

	loader := NewNQuadsLoader()
	r := strings.NewReader(input)
	facts, err := loader.Parse(r)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(facts) != 2 {
		t.Errorf("Expected 2 facts, got %d", len(facts))
	}

	expectedFact1 := `quad("http://sub1", "http://pred1", "\"obj1\"", "default")`
	if facts[0] != expectedFact1 {
		t.Errorf("Fact 1 mismatch.\nGot:  %s\nWant: %s", facts[0], expectedFact1)
	}

	expectedFact2 := `quad("http://sub2", "http://pred2", "\"obj2\"", "http://graphA")`
	if facts[1] != expectedFact2 {
		t.Errorf("Fact 2 mismatch.\nGot:  %s\nWant: %s", facts[1], expectedFact2)
	}
}

func TestNQuadsLoader_GetBaseRules(t *testing.T) {
	loader := NewNQuadsLoader()
	rules := loader.GetBaseRules()
	if !strings.Contains(rules, "triple(S, P, O) :- quad(S, P, O, _).") {
		t.Error("Base rules missing quad to triple mapping")
	}
}
