package reflection_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/core/reflection"
	"github.com/google/mangle/ast"
)

// Custom types for testing
type Role string
type DeviceOS string

// Nested structs for testing
type Device struct {
	OS DeviceOS `mangle:"os"`
}

type Metadata struct {
	Device   Device
	Settings map[string]string `mangle:"settings"`
}

type User struct {
	Name      string    `mangle:"user_name"`
	Role      Role      `mangle:"user_role"`
	Tags      []string  `mangle:"user_tags"`
	Meta      Metadata  `mangle:"user_meta"`
	Pointer   *****int  `mangle:"user_pointer"`
	NilField  *string   `mangle:"user_nil"`
	Interface interface{} `mangle:"user_interface"`
}

// Helper to sort facts for deterministic comparison.
func sortFacts(facts []ast.Atom) {
	sort.Slice(facts, func(i, j int) bool {
		return strings.Compare(facts[i].String(), facts[j].String()) < 0
	})
}

func TestToFacts_Comprehensive(t *testing.T) {
	// Pointer Hell setup
	p1 := 100
	p2 := &p1
	p3 := &p2
	p4 := &p3
	p5 := &p4

	user := User{
		Name: "Alice",
		Role: "admin",
		Tags: []string{"dev", "ops"},
		Meta: Metadata{
			Device: Device{
				OS: "linux",
			},
			Settings: map[string]string{
				"theme": "dark",
				"lang":  "en",
			},
		},
		Pointer:   &p5,
		NilField:  nil, // This should be ignored
		Interface: "interface_value",
	}

	facts, err := reflection.ToFacts("u1", &user)
	if err != nil {
		t.Fatalf("ToFacts failed: %v", err)
	}

	expectedFacts := []ast.Atom{
		ast.NewAtom("user_name", ast.String("u1"), ast.String("Alice")),
		ast.NewAtom("user_role", ast.String("u1"), ast.String("admin")),
		ast.NewAtom("user_tags", ast.String("u1"), ast.String("dev")),
		ast.NewAtom("user_tags", ast.String("u1"), ast.String("ops")),
		ast.NewAtom("user_meta.device.os", ast.String("u1"), ast.String("linux")),
		ast.NewAtom("user_meta.settings.theme", ast.String("u1"), ast.String("dark")),
		ast.NewAtom("user_meta.settings.lang", ast.String("u1"), ast.String("en")),
		ast.NewAtom("user_pointer", ast.String("u1"), ast.Number(100)),
		ast.NewAtom("user_interface", ast.String("u1"), ast.String("interface_value")),
	}

	sortFacts(facts)
	sortFacts(expectedFacts)

	if len(facts) != len(expectedFacts) {
		t.Errorf("Expected %d facts, got %d", len(expectedFacts), len(facts))
		for _, f := range facts {
			t.Logf("Got fact: %s", f.String())
		}
		return
	}

	for i := range facts {
		if facts[i].String() != expectedFacts[i].String() {
			t.Errorf("Fact mismatch at index %d:\ngot:  %s\nwant: %s", i, facts[i].String(), expectedFacts[i].String())
		}
	}
}

func TestToFacts_NilInput(t *testing.T) {
	facts, err := reflection.ToFacts("id", nil)
	if err != nil {
		t.Fatalf("ToFacts with nil input failed: %v", err)
	}
	if len(facts) != 0 {
		t.Errorf("Expected 0 facts for nil input, got %d", len(facts))
	}

	var nilPtr *User = nil
	facts, err = reflection.ToFacts("id", nilPtr)
	if err != nil {
		t.Fatalf("ToFacts with nil pointer failed: %v", err)
	}
	if len(facts) != 0 {
		t.Errorf("Expected 0 facts for nil pointer, got %d", len(facts))
	}
}

func TestToFacts_EmptyStruct(t *testing.T) {
	type Empty struct{}
	facts, err := reflection.ToFacts("id", Empty{})
	if err != nil {
		t.Fatalf("ToFacts with empty struct failed: %v", err)
	}
	if len(facts) != 0 {
		t.Errorf("Expected 0 facts for empty struct, got %d", len(facts))
	}
}

func TestToFacts_SnakeCaseConversion(t *testing.T) {
	type MyStruct struct {
		JSONField string
		APIKey    string
	}
	s := MyStruct{JSONField: "value1", APIKey: "value2"}
	facts, err := reflection.ToFacts("s1", s)
	if err != nil {
		t.Fatalf("ToFacts failed: %v", err)
	}

	expected := []ast.Atom{
		ast.NewAtom("json_field", ast.String("s1"), ast.String("value1")),
		ast.NewAtom("api_key", ast.String("s1"), ast.String("value2")),
	}

	sortFacts(facts)
	sortFacts(expected)

	if len(facts) != len(expected) {
		t.Fatalf("Expected %d facts, got %d", len(expected), len(facts))
	}
	if facts[0].String() != expected[0].String() || facts[1].String() != expected[1].String() {
		t.Errorf("Snake case conversion failed.\nGot: %v\nWant: %v", facts, expected)
	}
}
