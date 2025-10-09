package core

import (
	"io"

	"github.com/google/mangle/ast"
)

// SchemaParser defines the interface for components that can parse a data schema
// definition from a source and convert it into a set of Mangle Datalog facts.
// This allows the Mangle rules engine to reason about the structure of
// external data models, such as JSON schemas, database schemas, or API
// specifications. Implementations of this interface act as translators,
// making external knowledge available to the rules engine.
type SchemaParser interface {
	// Parse reads a schema definition from the provided io.Reader source,
	// parses it, and returns a slice of Mangle facts (ast.Atom). These facts
	// represent the structural elements of the schema, such as tables, columns,
	// types, or API endpoints, in a format that Mangle can query.
	//
	// source is the io.Reader containing the schema definition to be parsed.
	// It returns a slice of ast.Atom representing the schema facts, or an error
	// if parsing fails.
	Parse(source io.Reader) ([]ast.Atom, error)

	// Predicates returns a slice of ast.PredicateSym, which declares the
	// Datalog predicates that this parser can generate. This is a crucial
	// part of the Mangle contract, as it allows the engine's static analysis
	// to validate rules against the known set of possible facts.
	Predicates() []ast.PredicateSym
}
