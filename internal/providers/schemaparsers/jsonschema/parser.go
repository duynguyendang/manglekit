package jsonschema

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/duynguyendang/manglekit"
	"github.com/google/mangle/ast"
)

func init() {
	manglekit.RegisterSchemaParser("jsonschema", New)
}

// JSONSchemaParser implements the core.SchemaParser interface for parsing
// JSON Schema files into Mangle Datalog facts.
type JSONSchemaParser struct{}

// New is the constructor function for the JSONSchemaParser, registered with the
// MangleKit registry for the "jsonschema" parser type.
func New(params map[string]any) (any, error) {
	return &JSONSchemaParser{}, nil
}

// schema represents the structure of a JSON Schema file.
type schema struct {
	ID         string              `json:"$id"`
	Type       string              `json:"type"`
	Properties map[string]property `json:"properties"`
	Required   []string            `json:"required"`
}

type property map[string]any

// Predicates returns the Mangle Datalog predicate declarations for the facts
// that this parser can generate. This is required for the Mangle engine to
// validate the program. It declares facts about schemas, fields, and constraints.
func (p *JSONSchemaParser) Predicates() []ast.PredicateSym {
	return []ast.PredicateSym{
		{Symbol: "schema", Arity: 1},
		{Symbol: "field", Arity: 3},
		{Symbol: "field_required", Arity: 2},
		{Symbol: "field_format", Arity: 3},
		{Symbol: "field_constraint", Arity: 4},
	}
}

// Parse reads a JSON Schema definition from an io.Reader and converts it into
// a slice of Mangle Datalog facts. This allows the rules engine to reason about
// the data model defined in the schema.
// It generates facts for the schema's ID, its fields, types, and constraints
// like 'required', 'format', 'minimum', 'maximum', etc.
// This method satisfies the core.SchemaParser interface.
func (p *JSONSchemaParser) Parse(source io.Reader) ([]ast.Atom, error) {
	data, err := io.ReadAll(source)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema source: %w", err)
	}

	var s schema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json schema: %w", err)
	}

	if s.ID == "" {
		return nil, fmt.Errorf("json schema must have an '$id' field to identify the schema")
	}
	schemaID := ast.String(s.ID)

	var facts []ast.Atom

	// Use the correct symbol name (without arity).
	facts = append(facts, ast.NewAtom("schema", schemaID))

	for fieldName, prop := range s.Properties {
		fieldTerm := ast.String(fieldName)
		fieldType, _ := prop["type"].(string)

		if fieldType != "" {
			facts = append(facts, ast.NewAtom("field", schemaID, fieldTerm, ast.String(fieldType)))
		}

		if format, ok := prop["format"].(string); ok {
			facts = append(facts, ast.NewAtom("field_format", schemaID, fieldTerm, ast.String(format)))
		}

		for key, val := range prop {
			switch key {
			case "minimum", "maximum", "minLength", "maxLength":
				if num, ok := val.(float64); ok {
					facts = append(facts, ast.NewAtom("field_constraint", schemaID, fieldTerm, ast.String(key), ast.Number(int64(num))))
				}
			}
		}
	}

	requiredSet := make(map[string]bool)
	for _, fieldName := range s.Required {
		requiredSet[fieldName] = true
	}

	for fieldName := range s.Properties {
		if requiredSet[fieldName] {
			facts = append(facts, ast.NewAtom("field_required", schemaID, ast.String(fieldName)))
		}
	}

	return facts, nil
}