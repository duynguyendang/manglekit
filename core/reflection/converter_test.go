package reflection_test

import (
	"testing"

	"github.com/duynguyendang/manglekit/core/reflection"
	"github.com/google/mangle/ast"
)

type TestUser struct {
	Name    string `mangle:"user_name"`
	Age     int    `mangle:"user_age"`
	Active  bool   `mangle:"user_active"`
	Ignored string
	Score   *int `mangle:"user_score"`
}

func TestToFacts(t *testing.T) {
	score := 100
	user := TestUser{
		Name:    "Alice",
		Age:     30,
		Active:  true,
		Ignored: "secret",
		Score:   &score,
	}

	facts, err := reflection.ToFacts("u1", user)
	if err != nil {
		t.Fatalf("ToFacts failed: %v", err)
	}

	expectedCount := 4
	if len(facts) != expectedCount {
		t.Errorf("Expected %d facts, got %d", expectedCount, len(facts))
	}

	// Helper to find fact by predicate
	findFact := func(pred string) *ast.Atom {
		for _, f := range facts {
			if f.Predicate.Symbol == pred {
				return &f
			}
		}
		return nil
	}

	// Verify user_name
	fName := findFact("user_name")
	if fName == nil {
		t.Error("Missing user_name fact")
	} else {
		// user_name("u1", "Alice")
		// Args[0] is entityID, Args[1] is value
		if len(fName.Args) != 2 {
			t.Errorf("user_name expected 2 args, got %d", len(fName.Args))
		} else {
			if fName.Args[1].String() != "\"Alice\"" { // ast.String quotes the string
				t.Errorf("user_name value mismatch: got %s, want \"Alice\"", fName.Args[1].String())
			}
		}
	}

	// Verify user_age
	fAge := findFact("user_age")
	if fAge == nil {
		t.Error("Missing user_age fact")
	} else {
		if fAge.Args[1].String() != "30" {
			t.Errorf("user_age value mismatch: got %s, want 30", fAge.Args[1].String())
		}
	}

	// Verify user_active
	fActive := findFact("user_active")
	if fActive == nil {
		t.Error("Missing user_active fact")
	} else {
		if fActive.Args[1].String() != "\"true\"" {
			t.Errorf("user_active value mismatch: got %s, want \"true\"", fActive.Args[1].String())
		}
	}

	// Verify user_score (pointer)
	fScore := findFact("user_score")
	if fScore == nil {
		t.Error("Missing user_score fact")
	} else {
		if fScore.Args[1].String() != "100" {
			t.Errorf("user_score value mismatch: got %s, want 100", fScore.Args[1].String())
		}
	}

	// Verify Ignored field
	// We can't easily check for *absence* without iterating all, but we checked count.
	// 4 facts: name, age, active, score. "Ignored" should not be there.
}

func TestToFacts_Pointers(t *testing.T) {
	score := 50
	user := TestUser{
		Name:  "Bob",
		Age:   25,
		Score: &score,
	}

	// Test passing pointer to struct
	facts, err := reflection.ToFacts("u2", &user)
	if err != nil {
		t.Fatalf("ToFacts with pointer failed: %v", err)
	}

	if len(facts) != 4 { // Name, Age, Active(false), Score
		t.Errorf("Expected 4 facts, got %d", len(facts))
	}

	// Test nil pointer field
	user.Score = nil
	facts, err = reflection.ToFacts("u3", &user)
	if err != nil {
		t.Fatalf("ToFacts with nil field pointer failed: %v", err)
	}
	if len(facts) != 3 { // Name, Age, Active(false). Score is nil -> skipped
		t.Errorf("Expected 3 facts (score skipped), got %d", len(facts))
	}
}

func TestToFacts_InvalidInput(t *testing.T) {
	_, err := reflection.ToFacts("id", "not a struct")
	if err == nil {
		t.Error("Expected error for non-struct input, got nil")
	}

	var nilPtr *TestUser = nil
	_, err = reflection.ToFacts("id", nilPtr)
	if err == nil {
		t.Error("Expected error for nil pointer input, got nil")
	}
}

func TestToFacts_UnsupportedType(t *testing.T) {
	type BadStruct struct {
		Data []string `mangle:"bad_data"`
	}
	s := BadStruct{Data: []string{"a", "b"}}
	_, err := reflection.ToFacts("id", s)
	if err == nil {
		t.Error("Expected error for unsupported field type (slice), got nil")
	}
}
