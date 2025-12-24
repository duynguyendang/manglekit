package engine

import (
	"testing"
)

type TestAddress struct {
	City string `mangle:"city"`
	Zip  int    `mangle:"zip"`
}

type TestUser struct {
	Name    string       `mangle:"name"`
	Age     int          `mangle:"age"`
	Active  bool         `mangle:"active"`
	Address *TestAddress `mangle:"address"`
	Tags    []string     `mangle:"tags"`
	Friends []TestUser   `mangle:"friends"`
}

func TestFromFacts(t *testing.T) {
	facts := []string{
		`json_str("user_01", "name", "Alice")`,
		`json_num("user_01", "age", "30")`,
		`json_bool("user_01", "active", "true")`,
		`json_link("user_01", "address", "addr_01")`,
		`json_str("addr_01", "city", "Wonderland")`,
		`json_num("addr_01", "zip", "12345")`,
		`json_str("user_01", "tags", "admin")`,
		`json_str("user_01", "tags", "editor")`,
		`json_link("user_01", "friends", "user_02")`,
		`json_str("user_02", "name", "Bob")`,
	}

	user, err := FromFacts[TestUser]("user_01", facts)
	if err != nil {
		t.Fatalf("FromFacts failed: %v", err)
	}

	if user.Name != "Alice" {
		t.Errorf("Expected Name 'Alice', got '%s'", user.Name)
	}
	if user.Age != 30 {
		t.Errorf("Expected Age 30, got %d", user.Age)
	}
	if !user.Active {
		t.Errorf("Expected Active true, got false")
	}
	if user.Address == nil {
		t.Fatalf("Expected Address to be not nil")
	}
	if user.Address.City != "Wonderland" {
		t.Errorf("Expected City 'Wonderland', got '%s'", user.Address.City)
	}
	if user.Address.Zip != 12345 {
		t.Errorf("Expected Zip 12345, got %d", user.Address.Zip)
	}
	if len(user.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(user.Tags))
	}

	if len(user.Friends) != 1 {
		t.Fatalf("Expected 1 friend, got %d", len(user.Friends))
	}
	if user.Friends[0].Name != "Bob" {
		t.Errorf("Expected Friend Name 'Bob', got '%s'", user.Friends[0].Name)
	}
}

func TestFromFacts_Graph(t *testing.T) {
	// Test triple predicate support
	type GraphNode struct {
		Label string `mangle:"label"`
		Next  string `mangle:"next"`
	}

	facts := []string{
		`triple("node_a", "label", "Start")`,
		`triple("node_a", "next", "End")`,
	}

	node, err := FromFacts[GraphNode]("node_a", facts)
	if err != nil {
		t.Fatalf("FromFacts failed: %v", err)
	}

	if node.Label != "Start" {
		t.Errorf("Expected Label 'Start', got '%s'", node.Label)
	}
	if node.Next != "End" {
		t.Errorf("Expected Next 'End', got '%s'", node.Next)
	}
}

func TestParseAtomContent_CommaInQuotes(t *testing.T) {
    // This tests the improvement I made to parse/atom.go

    type Data struct {
        Value string `mangle:"val"`
    }

    facts := []string{
        `json_str("id1", "val", "Hello, World")`,
    }

    d, err := FromFacts[Data]("id1", facts)
    if err != nil {
        t.Fatalf("FromFacts failed: %v", err)
    }

    if d.Value != "Hello, World" {
        t.Errorf("Expected 'Hello, World', got '%s'", d.Value)
    }
}
