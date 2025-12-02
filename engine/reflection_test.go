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
		`tags("test", "alpha")`,
		`tags("test", "beta")`,
		`data("test", "id", "123")`,
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
