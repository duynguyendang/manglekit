package engine

import (
	"reflect"
	"sort"
	"testing"
)

func TestFlatten_Robustness(t *testing.T) {
	t.Run("MapStringInt", func(t *testing.T) {
		input := map[string]int{
			"one": 1,
			"two": 2,
		}
		facts, err := Flatten("root", input)
		if err != nil {
			t.Fatalf("Flatten failed: %v", err)
		}

		expected := []string{
			`json_num("root", "one", 1)`,
			`json_num("root", "two", 2)`,
		}

		sort.Strings(facts)
		sort.Strings(expected)

		if !reflect.DeepEqual(facts, expected) {
			t.Errorf("Want %v, got %v", expected, facts)
		}
	})

	t.Run("SliceString", func(t *testing.T) {
		input := []string{"a", "b"}
		facts, err := Flatten("root", input)
		if err != nil {
			t.Fatalf("Flatten failed: %v", err)
		}

		expected := []string{
			`json_str("root", "0", "a")`,
			`json_str("root", "1", "b")`,
		}

		sort.Strings(facts)
		sort.Strings(expected)

		if !reflect.DeepEqual(facts, expected) {
			t.Errorf("Want %v, got %v", expected, facts)
		}
	})

	t.Run("NestedMap", func(t *testing.T) {
		input := map[string]map[string]int{
			"outer": {
				"inner": 42,
			},
		}
		facts, err := Flatten("root", input)
		if err != nil {
			t.Fatalf("Flatten failed: %v", err)
		}

		// The node ID generation depends on iteration order and counter.
		// Since map iteration is random, node IDs might vary or order of processing might vary.
		// However, with 1 child, order is deterministic if we only have 1 key.

		// Expected:
		// json_link("root", "outer", "node_1")
		// json_num("node_1", "inner", 42)

		// We can't strict predict "node_1" if we had multiple branches, but here we have 1.

		// Let's just check for existence of substrings/patterns or use regex if needed.
		// Or simpler: check length and content.

		if len(facts) != 2 {
			t.Fatalf("Expected 2 facts, got %d: %v", len(facts), facts)
		}

		hasLink := false
		hasNum := false

		for _, f := range facts {
			if f == `json_num("node_1", "inner", 42)` {
				hasNum = true
			}
			if f == `json_link("root", "outer", "node_1")` {
				hasLink = true
			}
		}

		if !hasLink || !hasNum {
			t.Errorf("Missing expected facts. Got: %v", facts)
		}
	})
}
