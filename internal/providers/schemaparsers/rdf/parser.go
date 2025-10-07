package rdf

import (
	"fmt"
	"io"
	"strings"

	"github.com/duynguyendang/manglekit"
	"github.com/google/mangle/ast"
	"github.com/knakk/rdf"
)

func init() {
	manglekit.RegisterSchemaParser("rdf", New)
}

// RDFParser is a schema parser for RDF files.
type RDFParser struct{}

// New creates a new RDFParser. This constructor is compatible with the Manglekit registry.
func New(params map[string]any) (any, error) {
	return &RDFParser{}, nil
}

// Predicates returns the predicate symbols for the facts that this parser can generate.
func (p *RDFParser) Predicates() []ast.PredicateSym {
	return []ast.PredicateSym{
		{Symbol: "triple", Arity: 3},
	}
}

// Parse reads RDF data from the source and converts it into Mangle facts.
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