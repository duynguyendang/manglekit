package core

import (
	"io"

	"github.com/google/mangle/ast"
)

// SchemaParser defines the interface for components that can parse a data schema
// definition from a source and convert it into a set of Mangle Datalog facts.
// This allows the Mangle rules engine to reason about the structure of the
// underlying data, such as database tables, columns, or API resources.
type SchemaParser interface {
	// Parse reads a schema definition from the provided io.Reader source,
	// parses it, and returns a slice of Mangle facts (ast.Atom). These facts
	// represent the structural elements of the schema (e.g., tables, columns, types).
	//
	// source is the io.Reader containing the schema definition to be parsed.
	// It returns a slice of ast.Atom representing the schema facts, or an error
	// if parsing fails.
	Parse(source io.Reader) ([]ast.Atom, error)

	// Predicates returns a slice of ast.PredicateSym, which declares the
	// Datalog predicates that this parser can generate. This is crucial for
	// the Mangle engine to understand the vocabulary of facts that can be
	// derived from the schema.
	Predicates() []ast.PredicateSym
}