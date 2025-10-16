// Package rdf provides a schema parser that can read RDF data (in Turtle format)
// and convert it into a set of Mangle Datalog facts.
package rdf

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/google/mangle/ast"
	"github.com/knakk/rdf"
)

// Options defines the configuration for an RDFParser.
// It is currently empty but is defined for future use and consistency.
type Options struct{}

func Register(r *manglekit.Registry) {
	r.RegisterSchemaParser("rdf", func(ctx context.Context, options any, deps manglekit.FactoryDeps) (core.SchemaParser, error) {
		return New(nil)
	})
	if err := r.RegisterOptions("rdf", (*Options)(nil)); err != nil {
		panic(err)
	}
}

// RDFParser implements the `core.SchemaParser` interface for RDF files. It uses
// the `knakk/rdf` library to decode RDF triples and represents each one as a
// Mangle `triple/3` fact.
type RDFParser struct{}

// New is the constructor for the RDFParser. It is registered with the MangleKit
// registry for the "rdf" parser type.
func New(params map[string]any) (core.SchemaParser, error) {
	return &RDFParser{}, nil
}

// Predicates returns the Mangle Datalog predicate declarations for the facts
// that this parser can generate. It declares a single predicate:
//
//   - `triple(Subject, Predicate, Object)`: Represents a single RDF triple.
func (p *RDFParser) Predicates() []ast.PredicateSym {
	return []ast.PredicateSym{
		{Symbol: "triple", Arity: 3},
	}
}

// Parse reads RDF data from an `io.Reader` (assuming Turtle format), decodes
// each triple, and converts it into a Mangle `triple(Subject, Predicate, Object)`
// fact. This allows the rules engine to query graph-structured data.
// This method satisfies the `core.SchemaParser` interface.
func (p *RDFParser) Parse(source io.Reader) ([]ast.Atom, error) {
	var facts []ast.Atom
	decoder := rdf.NewTripleDecoder(source, rdf.Turtle)

	for triple, err := decoder.Decode(); err != io.EOF; triple, err = decoder.Decode() {
		if err != nil {
			return nil, fmt.Errorf("failed to decode RDF triple: %w", err)
		}

		subjTerm, err := termToConstant(triple.Subj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert subject term: %w", err)
		}
		predTerm, err := termToConstant(triple.Pred)
		if err != nil {
			return nil, fmt.Errorf("failed to convert predicate term: %w", err)
		}
		objTerm, err := termToConstant(triple.Obj)
		if err != nil {
			return nil, fmt.Errorf("failed to convert object term: %w", err)
		}

		fact := ast.NewAtom("triple", subjTerm, predTerm, objTerm)
		facts = append(facts, fact)
	}
	return facts, nil
}

// termToConstant converts an rdf.Term to an ast.Constant, intelligently creating
// ast.Name for IRIs and ast.String for literals.
func termToConstant(term rdf.Term) (ast.Constant, error) {
	termStr := term.String()
	// Check if the term's string representation is an IRI (e.g., "<http://...>")
	// The rdf library returns the raw IRI without angle brackets for .String()
	if term.Type() == rdf.TermIRI {
		// We must pass an IRI-like string (with brackets) to our helper.
		name, err := iriToName(fmt.Sprintf("<%s>", termStr))
		if err != nil {
			// Fallback to string if conversion fails, though it shouldn't for valid IRIs.
			return ast.String(termStr), nil
		}
		return name, nil
	}
	// For all other literal types, use a simple string representation.
	return ast.String(termStr), nil
}

// iriToName converts an IRI-like string (e.g., "<http://foo.com/bar>") into
// a Mangle Name constant (e.g., /foo.com/bar).
func iriToName(iriStr string) (ast.Constant, error) {
	if !strings.HasPrefix(iriStr, "<") || !strings.HasSuffix(iriStr, ">") {
		return ast.Constant{}, fmt.Errorf("not an IRI-like string: %s", iriStr)
	}
	content := strings.TrimSuffix(strings.TrimPrefix(iriStr, "<"), ">")
	if parts := strings.SplitN(content, "://", 2); len(parts) > 1 {
		content = parts[1]
	}
	if !strings.HasPrefix(content, "/") {
		content = "/" + content
	}
	return ast.Name(content)
}
