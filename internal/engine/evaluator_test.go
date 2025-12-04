package engine

import (
	"testing"
)

func TestNewEvaluator(t *testing.T) {
	tests := []struct {
		name         string
		rule         string
		expectError  bool
		expectedHead string
	}{
		{
			name:         "Valid deny rule",
			rule:         `deny(Req) :- amount(Req, X), X > 1000.`,
			expectError:  false,
			expectedHead: "deny",
		},
		{
			name:         "Valid allow rule",
			rule:         `allow(Req) :- region(Req, "US").`,
			expectError:  false,
			expectedHead: "allow",
		},
		{
			name:         "Valid route rule with multiple args",
			rule:         `route(Req, "fraud") :- amount(Req, X), X > 5000.`,
			expectError:  false,
			expectedHead: "route",
		},
		{
			name:        "Empty rule",
			rule:        "",
			expectError: true,
		},
		{
			name:        "Invalid syntax",
			rule:        "deny(Req) :- .",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			eval, err := NewEvaluator(tc.rule)

			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if eval.ruleHead != tc.expectedHead {
				t.Errorf("Expected head '%s', got '%s'", tc.expectedHead, eval.ruleHead)
			}
		})
	}
}

func TestEvaluator_Evaluate(t *testing.T) {
	type Transaction struct {
		Amount int    `mangle:"amount"`
		Region string `mangle:"region"`
	}

	tests := []struct {
		name        string
		rule        string
		entity      Transaction
		entityID    string
		expectMatch bool
		expectError bool
	}{
		{
			name:        "UK over 1000 matches deny rule",
			rule:        `deny(Req) :- amount(Req, Amount), region(Req, "UK"), Amount > 1000.`,
			entity:      Transaction{Amount: 1500, Region: "UK"},
			entityID:    "tx1",
			expectMatch: true,
		},
		{
			name:        "UK under 1000 does not match deny rule",
			rule:        `deny(Req) :- amount(Req, Amount), region(Req, "UK"), Amount > 1000.`,
			entity:      Transaction{Amount: 500, Region: "UK"},
			entityID:    "tx2",
			expectMatch: false,
		},
		{
			name:        "US over 1000 does not match UK deny rule",
			rule:        `deny(Req) :- amount(Req, Amount), region(Req, "UK"), Amount > 1000.`,
			entity:      Transaction{Amount: 2000, Region: "US"},
			entityID:    "tx3",
			expectMatch: false,
		},
		{
			name:        "Simple amount threshold",
			rule:        `deny(Req) :- amount(Req, X), X > 100.`,
			entity:      Transaction{Amount: 150, Region: "ANY"},
			entityID:    "tx4",
			expectMatch: true,
		},
		{
			name:        "Amount at threshold boundary (not greater)",
			rule:        `deny(Req) :- amount(Req, X), X > 100.`,
			entity:      Transaction{Amount: 100, Region: "ANY"},
			entityID:    "tx5",
			expectMatch: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			eval, err := NewEvaluator(tc.rule)
			if err != nil {
				t.Fatalf("Failed to create evaluator: %v", err)
			}

			result, err := eval.Evaluate(tc.entityID, tc.entity)

			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result.Matched != tc.expectMatch {
				t.Errorf("Expected match=%v, got match=%v", tc.expectMatch, result.Matched)
			}

			if result.EntityID != tc.entityID {
				t.Errorf("Expected entityID='%s', got '%s'", tc.entityID, result.EntityID)
			}
		})
	}
}

func TestStructToFacts(t *testing.T) {
	type FullStruct struct {
		IntField    int     `mangle:"int_field"`
		StringField string  `mangle:"string_field"`
		BoolField   bool    `mangle:"bool_field"`
		FloatField  float64 `mangle:"float_field"`
		Ignored     string  `mangle:"-"`
		NoTag       int     // should use lowercase field name
		private     int     // unexported, should be skipped
	}

	entity := FullStruct{
		IntField:    42,
		StringField: "hello",
		BoolField:   true,
		FloatField:  3.14,
		Ignored:     "should not appear",
		NoTag:       100,
	}

	facts, err := structToFacts("entity1", entity)
	if err != nil {
		t.Fatalf("structToFacts failed: %v", err)
	}

	// Check we have expected number of facts (5: int, string, bool, float, notag)
	expectedCount := 5
	if len(facts) != expectedCount {
		t.Errorf("Expected %d facts, got %d", expectedCount, len(facts))
	}

	// Verify predicate names exist
	predicates := make(map[string]bool)
	for _, fact := range facts {
		predicates[fact.Predicate.Symbol] = true
	}

	expectedPredicates := []string{"int_field", "string_field", "bool_field", "float_field", "notag"}
	for _, p := range expectedPredicates {
		if !predicates[p] {
			t.Errorf("Expected predicate '%s' not found", p)
		}
	}

	// Verify ignored field is not present
	if predicates["ignored"] {
		t.Error("Field marked with mangle:\"-\" should not be in facts")
	}
}

func TestStructToFacts_InvalidInputs(t *testing.T) {
	// Test nil pointer
	var nilPtr *struct{ X int }
	_, err := structToFacts("id", nilPtr)
	if err == nil {
		t.Error("Expected error for nil pointer")
	}

	// Test non-struct
	_, err = structToFacts("id", "not a struct")
	if err == nil {
		t.Error("Expected error for non-struct")
	}

	// Test struct with no valid fields
	type EmptyStruct struct {
		private int // unexported only
	}
	_, err = structToFacts("id", EmptyStruct{})
	if err == nil {
		t.Error("Expected error for struct with no valid fields")
	}
}
