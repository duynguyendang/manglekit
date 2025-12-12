package engine

import (
	"reflect"
	"sort"
	"testing"
)

func TestToFacts(t *testing.T) {
	type Address struct {
		Street string `mangle:"street"`
		City   string
	}

	type User struct {
		Name    string   `mangle:"username"`
		Address *Address `mangle:"location"`
		Tags    []string
		Data    map[string]any
	}

	addr := &Address{
		Street: "123 Main St",
		City:   "Anytown",
	}

	user := User{
		Name:    "John Doe",
		Address: addr,
		Tags:    []string{"alpha", "beta"},
		Data: map[string]any{
			"id":      123,
			"isAdmin": true,
		},
	}

	expectedFacts := []string{
		`username("test", "John Doe")`,
		`location_street("test", "123 Main St")`,
		`location_city("test", "Anytown")`,
		`tags("test", "0", "alpha")`,
		`tags("test", "1", "beta")`,
		`data("test", "id", 123)`,
		`data("test", "isAdmin", "true")`,
	}

	facts, err := ToFacts("test", user)
	if err != nil {
		t.Fatalf("ToFacts() returned an error: %v", err)
	}

	// Sort both slices for stable comparison
	sort.Strings(facts)
	sort.Strings(expectedFacts)

	if !reflect.DeepEqual(facts, expectedFacts) {
		t.Errorf("ToFacts() got = %v, want %v", facts, expectedFacts)
	}
}

func TestLabelsToFacts(t *testing.T) {
	labels := []string{"secret", "pii", "complex\"label\\"}
	entityID := "req\"1"

	expected := []string{
		`label("secret")`,
		`label("pii")`,
		`label("complex\"label\\")`,
	}

	facts, err := LabelsToFacts(entityID, labels)
	if err != nil {
		t.Fatalf("LabelsToFacts failed: %v", err)
	}

	if len(facts) != len(expected) {
		t.Fatalf("Expected %d facts, got %d", len(expected), len(facts))
	}

	for i, fact := range facts {
		if fact != expected[i] {
			t.Errorf("Fact %d: got %s, want %s", i, fact, expected[i])
		}
	}
}
