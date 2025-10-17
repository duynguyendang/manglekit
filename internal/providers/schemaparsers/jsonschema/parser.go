// Package jsonschema provides a schema parser that can read a JSON Schema
// definition and convert it into a set of Mangle Datalog facts.
package jsonschema

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/google/mangle/ast"
)

// Options defines the configuration for a JSONSchemaParser.
// It is currently empty but is defined for future use and consistency.
type Options struct{}

func (o Options) ProviderName() string { return "jsonschema" }
func (o Options) ProviderKind() core.Kind   { return core.KindSchemaParser }

func Register(r *manglekit.Registry) {
	manglekit.Register(r, Options{},
		func(ctx context.Context, deps diapi.NoopDeps, cfg Options) (core.SchemaParser, error) {
			return New(nil)
		},
	)
}

// JSONSchemaParser implements the `core.SchemaParser` interface for parsing
// JSON Schema files into Mangle Datalog facts. This allows the rules engine to
// reason about the structure and constraints of a JSON data model.
type JSONSchemaParser struct{}

// New is the constructor for the JSONSchemaParser. It is registered with the
// MangleKit registry for the "jsonschema" parser type.
func New(params map[string]any) (core.SchemaParser, error) {
	return &JSONSchemaParser{}, nil
}

// schema represents a simplified view of a JSON Schema file, focusing on the
// properties needed to generate relevant Datalog facts.
type schema struct {
	ID         string              `json:"$id"`
	Type       string              `json:"type"`
	Properties map[string]property `json:"properties"`
	Required   []string            `json:"required"`
}

// property is a generic map representing the attributes of a single property
// within a JSON schema.
type property map[string]any

// Predicates returns the Mangle Datalog predicate declarations for the facts
// that this parser can generate. This is required by the `core.SchemaParser`
// interface for Mangle's static analysis. The generated predicates are:
//
//   - `schema(SchemaID)`: Declares the existence of a schema.
//   - `field(SchemaID, FieldName, FieldType)`: Declares a field within a schema.
//   - `field_required(SchemaID, FieldName)`: Marks a field as required.
//   - `field_format(SchemaID, FieldName, Format)`: Specifies a format constraint (e.g., "date-time").
//   - `field_constraint(SchemaID, FieldName, Constraint, Value)`: Specifies other constraints like "minimum" or "maxLength".
func (p *JSONSchemaParser) Predicates() []ast.PredicateSym {
	return []ast.PredicateSym{
		{Symbol: "schema", Arity: 1},
		{Symbol: "field", Arity: 3},
		{Symbol: "field_required", Arity: 2},
		{Symbol: "field_format", Arity: 3},
		{Symbol: "field_constraint", Arity: 4},
	}
}

// Parse reads a JSON Schema definition from an `io.Reader`, unmarshals it, and
// converts its structure into a slice of Mangle Datalog facts. This allows the
// rules engine to query the data model defined in the schema.
//
// It generates facts for the schema's ID, its fields, their types, and common
// constraints like 'required', 'format', 'minimum', 'maximum', etc.
// This method satisfies the `core.SchemaParser` interface.
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
